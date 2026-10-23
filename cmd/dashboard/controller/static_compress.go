package controller

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

var errUnsupportedEncoding = errors.New("unsupported content encoding")

// Static frontend assets are served straight out of an embedded fs.FS (or a
// local override directory). Everything else in the dashboard is tiny, so the
// only thing that matters for perceived latency is shipping the bundle over the
// wire smaller than the build produced it.
//
// http.ServeContent is used here on purpose: it gives us Range requests,
// If-Modified-Since and If-None-Match for free. The catch is that it decides
// Content-Length (and the 206/304 paths) from the bytes it is handed, so
// compression has to happen *before* ServeContent and the encoded bytes have to
// be its input. utils.GinCustomWriter does not intercept Write, so any
// compressor installed as an outer middleware would emit a body that disagrees
// with Content-Length.
const (
	// minCompressSize keeps us from spending CPU on assets where the encoding
	// overhead dominates (~1 KiB of JSON/SVG/favicon-ish payloads).
	minCompressSize = 1024
	// compressCacheBudgetKB caps the in-memory cache of encoded assets. Embedded
	// dists are a few MB, so this exists purely as a runaway guard.
	compressCacheBudgetKB = 32 * 1024
)

var compressibleExts = map[string]struct{}{
	".js":          {},
	".mjs":         {},
	".css":         {},
	".html":        {},
	".json":        {},
	".svg":         {},
	".map":         {},
	".webmanifest": {},
	".xml":         {},
}

// encodedAsset is an immutable, fully compressed body. It is shared between
// requests, so it must never be mutated after insertion.
type encodedAsset struct {
	body []byte
}

var compressCache = struct {
	sync.RWMutex
	entries map[string]*encodedAsset
	bytes   int
}{entries: make(map[string]*encodedAsset)}

// compressible reports whether name is worth compressing based on extension.
// Already-compressed formats (images, fonts, woff2) are excluded explicitly so
// an odd allowlist entry can never double-compress them.
func compressible(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	_, ok := compressibleExts[ext]
	return ok
}

// pickEncoding returns the best Content-Encoding the client accepts, or "" when
// it accepts neither. Brotli is preferred: for JS/CSS it is typically ~15-20%
// smaller than gzip at comparable encode cost.
func pickEncoding(r *http.Request) string {
	if r == nil {
		return ""
	}
	switch acceptEncodingQ(r.Header.Values("Accept-Encoding")) {
	case "br":
		return "br"
	case "gzip":
		return "gzip"
	default:
		return ""
	}
}

func acceptEncodingQ(values []string) string {
	brQ, gzipQ := -1.0, -1.0
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			token, params, _ := strings.Cut(strings.TrimSpace(part), ";")
			token = strings.ToLower(strings.TrimSpace(token))
			q := 1.0
			if params != "" {
				for _, p := range strings.Split(params, ";") {
					k, v, ok := strings.Cut(strings.TrimSpace(p), "=")
					if !ok || !strings.EqualFold(strings.TrimSpace(k), "q") {
						continue
					}
					if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
						q = parsed
					}
				}
			}
			switch token {
			case "br":
				brQ = q
			case "gzip":
				gzipQ = q
			case "*":
				if brQ < 0 {
					brQ = q
				}
				if gzipQ < 0 {
					gzipQ = q
				}
			}
		}
	}
	if brQ > 0 && brQ >= gzipQ {
		return "br"
	}
	if gzipQ > 0 {
		return "gzip"
	}
	return ""
}

// serveCompressed serves name through http.ServeContent with a compressed body.
// The returned io.ReadSeeker is the compressed body when sent is true, and the
// original bytes (fully buffered) otherwise, so the caller has a single serve
// path for both the compressed and the plain case. etag is a content-digest
// validator (independent of encoding) so equal-length embed upgrades still bust
// stale 304s.
//
// Files opened from an embedded fs.FS are not io.ReadSeekers, so the body is
// always fully read here; for the dashboard's a few-MB dists that is cheap and
// only happens on the first request per asset, after which the encoded bytes
// come from the cache.
//
// The compress cache key includes a content+mtime version so rewriting a local
// index.html (or any same-path asset) cannot return a stale gzip/br body while
// the ETag already reflects the new bytes.
func serveCompressed(c *gin.Context, cacheKey, name, encoding string, src io.Reader, modTime time.Time) (body io.ReadSeeker, etag string, ok bool) {
	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, "", false
	}
	etag = staticETag(name, raw, modTime)
	version := assetCacheVersion(raw, modTime)

	if encoding != "" && compressible(name) && len(raw) >= minCompressSize {
		key := cacheKey + "|" + version + "|" + encoding
		asset := lookupEncodedAsset(key)
		if asset == nil {
			if encoded, encErr := encodeAsset(encoding, raw); encErr == nil && len(encoded) < len(raw) {
				asset = &encodedAsset{body: encoded}
				storeEncodedAsset(key, asset)
			}
		}
		if asset != nil {
			c.Header("Content-Encoding", encoding)
			c.Header("Vary", "Accept-Encoding")
			return bytes.NewReader(asset.body), etag, true
		}
	}

	// Still advertise the negotiation dimension on the identity response,
	// otherwise a shared cache may hand a plain body to a gzip-capable client.
	c.Header("Vary", "Accept-Encoding")
	return bytes.NewReader(raw), etag, true
}

// assetCacheVersion identifies the uncompressed bytes for the compress cache.
// Content digest covers embed.FS (mtime always zero); mtime covers local
// overrides that may be rewritten in place.
func assetCacheVersion(raw []byte, modTime time.Time) string {
	h := fnv.New64a()
	_, _ = h.Write(raw)
	return fmt.Sprintf("%x-%x", h.Sum64(), modTime.UnixNano())
}

// staticETag derives an ETag for a frontend template file from the actual
// bytes. Filename+length+mtime is not enough: embed.FS reports a zero mtime, so
// two releases that replace an equal-length file would keep the old validator
// and clients would incorrectly 304. Compression variant is deliberately
// ignored — browsers reuse If-None-Match across encodings.
func staticETag(name string, raw []byte, modTime time.Time) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	_, _ = h.Write(raw)
	return fmt.Sprintf(`W/"%x-%x-%x"`, h.Sum64(), len(raw), modTime.UnixNano())
}

func encodeAsset(encoding string, raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	var w io.WriteCloser
	switch encoding {
	case "br":
		// Level 5 is the usual size/CPU sweet spot for static text assets.
		w = brotli.NewWriterLevel(&buf, 5)
	case "gzip":
		gw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
		if err != nil {
			return nil, err
		}
		w = gw
	default:
		return nil, errUnsupportedEncoding
	}

	// Best-effort size hint; harmless when it is wrong.
	buf.Grow(len(raw) / 3)
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		return nil, err
	}
	// Close flushes the encoder's trailer; ignoring it would ship a truncated stream.
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lookupEncodedAsset(key string) *encodedAsset {
	compressCache.RLock()
	defer compressCache.RUnlock()
	return compressCache.entries[key]
}

func storeEncodedAsset(key string, asset *encodedAsset) {
	compressCache.Lock()
	defer compressCache.Unlock()
	if _, ok := compressCache.entries[key]; ok {
		return
	}
	// Reset wholesale rather than track LRU order: per-file entries are
	// re-derivable in milliseconds and the cap is far above real dist sizes.
	if compressCache.bytes+len(asset.body) > compressCacheBudgetKB*1024 {
		compressCache.entries = make(map[string]*encodedAsset)
		compressCache.bytes = 0
	}
	compressCache.entries[key] = asset
	compressCache.bytes += len(asset.body)
}

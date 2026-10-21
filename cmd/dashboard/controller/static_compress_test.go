package controller

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/service/singleton"
)

func newStaticCompressTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	t.Chdir(t.TempDir())

	originalConf := singleton.Conf
	singleton.Conf = &singleton.ConfigClass{Config: &model.Config{
		ConfigDashboard: model.ConfigDashboard{
			AdminTemplate: "admin-dist",
			UserTemplate:  "user-dist",
		},
	}}
	t.Cleanup(func() { singleton.Conf = originalConf })

	// The shared helper installs its own tiny app.js and index.html fixture.
	router := newFrontendFallbackTestRouter(t)

	// Overwrite afterwards: writeFrontendFallbackTestFile resolves relative paths
	// against the process cwd, which the shared helper has already chdir'd into
	// the fixture root.
	big := strings.Repeat("console.log('nezha asset bundle');\n", 400)
	writeFrontendFallbackTestFile(t, "admin-dist/assets/app.js", big)
	writeFrontendFallbackTestFile(t, "admin-dist/assets/logo.png", strings.Repeat("png", 2000))

	return router
}

func requestWithEncoding(t *testing.T, router *gin.Engine, target, acceptEncoding string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	router.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, encoding string, body []byte) string {
	t.Helper()
	switch encoding {
	case "gzip":
		r, err := gzip.NewReader(strings.NewReader(string(body)))
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("read gzip body: %v", err)
		}
		return string(out)
	case "br":
		out, err := io.ReadAll(brotli.NewReader(strings.NewReader(string(body))))
		if err != nil {
			t.Fatalf("read brotli body: %v", err)
		}
		return string(out)
	default:
		return string(body)
	}
}

func TestStaticAssetNegotiatesBrotli(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	w := requestWithEncoding(t, router, "/dashboard/assets/app.js", "gzip, deflate, br", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("Content-Encoding = %q, want br", got)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if decoded := decodeBody(t, "br", w.Body.Bytes()); !strings.HasPrefix(decoded, "console.log('nezha asset bundle')") {
		t.Fatalf("decoded body mismatch: %q", decoded[:40])
	}
}

func TestStaticAssetFallsBackToGzip(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	w := requestWithEncoding(t, router, "/dashboard/assets/app.js", "gzip", nil)
	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if decoded := decodeBody(t, "gzip", w.Body.Bytes()); !strings.HasPrefix(decoded, "console.log(") {
		t.Fatalf("decoded body mismatch")
	}
}

func TestStaticAssetSkipsIncompressibleAndNoAcceptEncoding(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	w := requestWithEncoding(t, router, "/dashboard/assets/logo.png", "gzip, br", nil)
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("png must not be compressed, got %q", got)
	}

	w = requestWithEncoding(t, router, "/dashboard/assets/app.js", "identity", nil)
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("identity request must not be compressed, got %q", got)
	}
}

func TestStaticAssetSkipsBelowThreshold(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	// index.html fixture is far below minCompressSize.
	w := requestWithEncoding(t, router, "/dashboard/", "gzip, br", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("tiny file must not be compressed, got %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestStaticAssetConditionalRequestReturns304(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	first := requestWithEncoding(t, router, "/dashboard/assets/app.js", "br", nil)
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	second := requestWithEncoding(t, router, "/dashboard/assets/app.js", "br", http.Header{
		"If-None-Match": []string{etag},
	})
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("304 must have empty body, got %d bytes", second.Body.Len())
	}
}

func TestStaticAssetContentLengthMatchesCompressedBody(t *testing.T) {
	router := newStaticCompressTestRouter(t)

	w := requestWithEncoding(t, router, "/dashboard/assets/app.js", "gzip", nil)
	// ServeContent sets Content-Length from the bytes it is handed; a mismatch
	// here means the connection would hang or truncate in a real client.
	if got, want := w.Header().Get("Content-Length"), w.Body.Len(); got != "" && got != strconv.Itoa(want) {
		t.Fatalf("Content-Length = %s, body = %d", got, want)
	}
}

func TestAcceptEncodingQValues(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"br;q=0.5, gzip;q=1.0", "gzip"},
		{"gzip;q=0.1, br;q=0.9", "br"},
		{"identity", ""},
		{"*", "br"},
		{"", ""},
		{"gzip;q=0, br;q=0", ""},
	}
	for _, tt := range tests {
		if tt.header == "" {
			if got := acceptEncodingQ(nil); got != tt.want {
				t.Fatalf("%q => %q, want %q", tt.header, got, tt.want)
			}
			continue
		}
		if got := acceptEncodingQ([]string{tt.header}); got != tt.want {
			t.Fatalf("%q => %q, want %q", tt.header, got, tt.want)
		}
	}
}

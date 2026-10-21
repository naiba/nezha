package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
)

type etagTestItem struct {
	model.Common
	Value string `json:"value"`
}

func (i etagTestItem) GetID() uint64                   { return i.ID }
func (i etagTestItem) GetUserID() uint64               { return 0 }
func (i etagTestItem) HasPermission(*gin.Context) bool { return true }

func newETagTestRouter(items *[]etagTestItem) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := listHandlerWithETag(func(*gin.Context) ([]etagTestItem, error) {
		return *items, nil
	})
	r.GET("/items", handler)
	r.GET("/one", commonHandlerWithETag(func(*gin.Context) (etagTestItem, error) {
		return (*items)[0], nil
	}))
	return r
}

func doRequest(r *gin.Engine, path, ifNoneMatch string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestListHandlerWithETagReturns304OnMatch(t *testing.T) {
	items := []etagTestItem{{Value: "a"}}
	r := newETagTestRouter(&items)

	first := doRequest(r, "/items", "")
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}
	if got := first.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("API responses must not be cacheable, got %q", got)
	}
	if got := first.Header().Get("Vary"); got != "Authorization, Cookie" {
		t.Fatalf("Vary = %q", got)
	}

	second := doRequest(r, "/items", etag)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("304 body = %d bytes", second.Body.Len())
	}
}

func TestListHandlerWithETagChangesWhenBodyChanges(t *testing.T) {
	items := []etagTestItem{{Value: "a"}}
	r := newETagTestRouter(&items)

	first := doRequest(r, "/items", "")
	etag := first.Header().Get("ETag")

	items = append(items, etagTestItem{Value: "b"})
	second := doRequest(r, "/items", etag)
	if second.Code != http.StatusOK {
		t.Fatalf("stale ETag must not produce 304, got %d", second.Code)
	}
	if second.Header().Get("ETag") == etag {
		t.Fatal("ETag did not change with the body")
	}
}

func TestCommonHandlerWithETagReturns304OnMatch(t *testing.T) {
	items := []etagTestItem{{Value: "a"}}
	r := newETagTestRouter(&items)

	first := doRequest(r, "/one", "")
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d", first.Code)
	}
	second := doRequest(r, "/one", first.Header().Get("ETag"))
	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", second.Code)
	}
}

func TestETagMatches(t *testing.T) {
	tests := []struct {
		header string
		etag   string
		want   bool
	}{
		{"", `W/"abc"`, false},
		{`W/"abc"`, `W/"abc"`, true},
		{`"abc"`, `W/"abc"`, true},
		{`"other", W/"abc"`, `W/"abc"`, true},
		{"*", `W/"abc"`, true},
		{`"other"`, `W/"abc"`, false},
	}
	for _, tt := range tests {
		if got := etagMatches(tt.header, tt.etag); got != tt.want {
			t.Fatalf("etagMatches(%q, %q) = %v, want %v", tt.header, tt.etag, got, tt.want)
		}
	}
}

func TestServerListRequestKeyIsolatesViewers(t *testing.T) {
	admin := &model.User{Common: model.Common{ID: 1}, Role: model.RoleAdmin}
	member := &model.User{Common: model.Common{ID: 2}, Role: model.RoleMember}

	tokA := &model.APIToken{ID: 10, UserID: 2}
	tokA.SetServerIDs([]uint64{7})
	tokB := &model.APIToken{ID: 11, UserID: 2}
	tokB.SetServerIDs([]uint64{8})

	jwtMember := serverListRequestKey(newCacheKeyCtx(t, member, nil))
	patMemberA := serverListRequestKey(newCacheKeyCtx(t, member, tokA))
	patMemberB := serverListRequestKey(newCacheKeyCtx(t, member, tokB))
	adminJWT := serverListRequestKey(newCacheKeyCtx(t, admin, nil))

	keys := map[string]string{
		"jwtMember":  jwtMember,
		"patMemberA": patMemberA,
		"patMemberB": patMemberB,
		"adminJWT":   adminJWT,
	}
	seen := map[string]string{}
	for name, key := range keys {
		if prev, ok := seen[key]; ok {
			t.Fatalf("%s and %s share a singleflight key %q", prev, name, key)
		}
		seen[key] = name
	}

	// Same viewer + same whitelist must dedupe.
	again := serverListRequestKey(newCacheKeyCtx(t, member, tokA))
	if again != patMemberA {
		t.Fatalf("same viewer produced different keys: %q vs %q", again, patMemberA)
	}
}

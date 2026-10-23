package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/service/singleton"
)

// TestListServerSingleflightConcurrentFilterDoesNotRace covers the P1 review
// finding: singleflight used to hand out one shared []*Server, and
// listHandlerWithETag's filter mutated it in place via slices.DeleteFunc.
func TestListServerSingleflightConcurrentFilterDoesNotRace(t *testing.T) {
	original := singleton.ServerShared
	sc := singleton.NewEmptyServerClassForTest()
	singleton.ServerShared = sc
	t.Cleanup(func() { singleton.ServerShared = original })

	owner := &model.User{Common: model.Common{ID: 42}, Role: model.RoleMember}
	for id := uint64(1); id <= 8; id++ {
		sc.InsertForTest(&model.Server{
			Common: model.Common{ID: id, UserID: owner.ID},
			Name:   fmt.Sprintf("srv-%d", id),
			UUID:   fmt.Sprintf("uuid-%d", id),
		})
	}

	// PAT whitelist keeps only half the inventory so filter must DeleteFunc.
	tok := &model.APIToken{ID: 99, UserID: owner.ID}
	tok.SetServerIDs([]uint64{1, 2, 3, 4})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/server", func(c *gin.Context) {
		c.Set(model.CtxKeyAuthorizedUser, owner)
		c.Set(model.CtxKeyAPIToken, tok)
		c.Set(apiTokenCtxKey, tok)
		listHandlerWithETag(listServer)(c)
	})

	const goroutines = 32
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/server", nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				errCh <- fmt.Errorf("status=%d body=%s", w.Code, w.Body.String())
				return
			}
			var resp model.CommonResponse[[]*model.Server]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				errCh <- err
				return
			}
			if !resp.Success {
				errCh <- fmt.Errorf("success=false")
				return
			}
			if len(resp.Data) != 4 {
				errCh <- fmt.Errorf("len=%d want 4", len(resp.Data))
				return
			}
			for _, s := range resp.Data {
				if s == nil {
					errCh <- fmt.Errorf("nil server in filtered response")
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

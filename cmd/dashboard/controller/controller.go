package controller

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
)

func ServeWeb(port uint) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	if dao.Conf.Debug {
		gin.SetMode(gin.DebugMode)
		pprof.Register(r)
	}
	r.Use(mygin.RecordPath)
	r.SetFuncMap(template.FuncMap{
		"tf": func(t time.Time) string {
			return t.Format("2006年1月2号 15:04:05")
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s) // #nosec
		},
		"tag": func(s string) template.HTML {
			return template.HTML(`<` + s + `>`) // #nosec
		},
		"stf": func(s uint64) string {
			return time.Unix(int64(s), 0).Format("2006年1月2号 15:04")
		},
		"sf": func(duration uint64) string {
			return time.Duration(time.Duration(duration) * time.Second).String()
		},
		"bf": func(b uint64) string {
			return bytefmt.ByteSize(b)
		},
		"ts": func(s string) string {
			return strings.TrimSpace(s)
		},
		"float32f": func(f float32) string {
			return fmt.Sprintf("%.2f", f)
		},
		"divU64": func(a, b uint64) float32 {
			if b == 0 {
				if a > 0 {
					return 100
				}
				return 0
			}
			if a == 0 {
				// 这是从未在线的情况
				return 0.00001 / float32(b) * 100
			}
			return float32(a) / float32(b) * 100
		},
		"div": func(a, b int) float32 {
			if b == 0 {
				if a > 0 {
					return 100
				}
				return 0
			}
			if a == 0 {
				// 这是从未在线的情况
				return 0.00001 / float32(b) * 100
			}
			return float32(a) / float32(b) * 100
		},
		"addU64": func(a, b uint64) uint64 {
			return a + b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"dayBefore": func(i int) string {
			year, month, day := time.Now().Date()
			today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
			return today.AddDate(0, 0, i-29).Format("1月2号")
		},
		"className": func(percent float32) string {
			if percent == 0 {
				return ""
			}
			if percent > 95 {
				return "good"
			}
			if percent > 80 {
				return "warning"
			}
			return "danger"
		},
		"statusName": func(percent float32) string {
			if percent == 0 {
				return "无数据"
			}
			if percent > 95 {
				return "良好"
			}
			if percent > 80 {
				return "低可用"
			}
			return "故障"
		},
	})
	r.Static("/static", "resource/static")
	r.LoadHTMLGlob("resource/template/**/*.html")
	routers(r)

	page404 := func(c *gin.Context) {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusNotFound,
			Title: "该页面不存在",
			Msg:   "该页面内容可能已着陆火星",
			Link:  "/",
			Btn:   "返回首页",
		}, true)
	}
	r.NoRoute(page404)
	r.NoMethod(page404)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return srv
}

func routers(r *gin.Engine) {
	// 通用页面
	cp := commonPage{r: r, terminals: make(map[string]*terminalContext), terminalsLock: new(sync.Mutex)}
	cp.serve()
	// 游客页面
	gp := guestPage{r}
	gp.serve()
	// 会员页面
	mp := &memberPage{r}
	mp.serve()
	// API
	api := r.Group("api")
	{
		ma := &memberAPI{api}
		ma.serve()
	}
}

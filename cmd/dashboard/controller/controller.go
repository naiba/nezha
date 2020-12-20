package controller

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
)

// ServeWeb ..
func ServeWeb(port uint) {
	gin.SetMode(gin.ReleaseMode)
	if dao.Conf.Debug {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.Default()
	r.Use(mygin.RecordPath)
	r.SetFuncMap(template.FuncMap{
		"tf": func(t time.Time) string {
			return t.Format("2006年1月2号")
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"tag": func(s string) template.HTML {
			return template.HTML(`<` + s + `>`)
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
	})
	r.Static("/static", "resource/static")
	r.LoadHTMLGlob("resource/template/**/*")
	routers(r)
	r.Run(fmt.Sprintf(":%d", port))
}

func routers(r *gin.Engine) {
	// 通用页面
	cp := commonPage{r}
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

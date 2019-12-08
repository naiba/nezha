package controller

import (
	"fmt"
	"html/template"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/p14yground/nezha/pkg/mygin"
	"github.com/p14yground/nezha/service/dao"
)

// ServeWeb ..
func ServeWeb() {
	r := gin.Default()
	r.Use(mygin.RecordPath)
	r.SetFuncMap(template.FuncMap{
		"tf": func(t time.Time) string {
			return t.Format("2006年1月2号")
		},
		"fs": func() string {
			if !dao.Conf.Debug {
				return ""
			}
			return fmt.Sprintf("%d", time.Now().UnixNano())
		},
	})
	r.Static("/static", "resource/static")
	r.LoadHTMLGlob("resource/template/**/*")
	routers(r)
	r.Run()
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

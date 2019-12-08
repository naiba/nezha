package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/pkg/mygin"
	"github.com/p14yground/nezha/service/dao"
)

type memberPage struct {
	r *gin.Engine
}

func (mp *memberPage) serve() {
	mr := mp.r.Group("")
	mr.Use(mygin.Authorize(mygin.AuthorizeOption{
		Member:   true,
		IsPage:   true,
		Msg:      "此页面需要登录",
		Btn:      "点此登录",
		Redirect: "/login",
	}))
	mr.GET("/server", mp.server)
}

func (mp *memberPage) server(c *gin.Context) {
	var servers []model.Server
	dao.DB.Find(&servers)
	c.HTML(http.StatusOK, "page/server", mygin.CommonEnvironment(c, gin.H{
		"Title":   "服务器管理",
		"Servers": servers,
	}))
}

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/pkg/mygin"
	"github.com/p14yground/nezha/service/dao"
)

type commonPage struct {
	r *gin.Engine
}

func (cp *commonPage) serve() {
	cr := cp.r.Group("")
	cr.Use(mygin.Authorize(mygin.AuthorizeOption{}))
	cr.GET("/", cp.home)
}

func (cp *commonPage) home(c *gin.Context) {
	var admin *model.User
	isLogin, ok := c.Get(model.CtxKeyIsUserLogin)
	if ok && isLogin.(bool) {
		admin = dao.Admin
	}
	var servers []model.Server
	dao.DB.Find(&servers)
	c.HTML(http.StatusOK, "page/home", mygin.CommonEnvironment(c, gin.H{
		"Admin":   admin,
		"Servers": servers,
	}))
}

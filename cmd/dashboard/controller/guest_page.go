package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type guestPage struct {
	r *gin.Engine
}

func (gp *guestPage) serve() {
	gr := gp.r.Group("")
	gr.Use(mygin.Authorize(mygin.AuthorizeOption{
		Guest:    true,
		IsPage:   true,
		Msg:      "您已登录",
		Btn:      "返回首页",
		Redirect: "/",
	}))

	gr.GET("/login", gp.login)

	oauth := &oauth2controller{
		oauth2Config: &oauth2.Config{
			ClientID:     dao.Conf.GitHub.ClientID,
			ClientSecret: dao.Conf.GitHub.ClientSecret,
			Scopes:       []string{},
			Endpoint:     github.Endpoint,
		},
		r: gr,
	}
	oauth.serve()
}

func (gp *guestPage) login(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/login", mygin.CommonEnvironment(c, gin.H{
		"Title": "登录",
	}))
}

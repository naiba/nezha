package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
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

	var endPoint oauth2.Endpoint

	if dao.Conf.Oauth2.Type == model.ConfigTypeGitee {
		endPoint = oauth2.Endpoint{
			AuthURL:  "https://gitee.com/oauth/authorize",
			TokenURL: "https://gitee.com/oauth/token",
		}
	} else {
		endPoint = github.Endpoint
	}

	oauth := &oauth2controller{
		oauth2Config: &oauth2.Config{
			ClientID:     dao.Conf.Oauth2.ClientID,
			ClientSecret: dao.Conf.Oauth2.ClientSecret,
			Scopes:       []string{},
			Endpoint:     endPoint,
		},
		r: gr,
	}
	oauth.serve()
}

func (gp *guestPage) login(c *gin.Context) {
	LoginType := "GitHub"
	RegistrationLink := "https://github.com/join"
	if dao.Conf.Oauth2.Type == model.ConfigTypeGitee {
		LoginType = "Gitee"
		RegistrationLink = "https://gitee.com/signup"
	}
	c.HTML(http.StatusOK, "dashboard/login", mygin.CommonEnvironment(c, gin.H{
		"Title":            "登录",
		"LoginType":        LoginType,
		"RegistrationLink": RegistrationLink,
	}))
}

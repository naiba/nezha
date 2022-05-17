package mygin

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

type AuthorizeOption struct {
	Guest    bool
	Member   bool
	IsPage   bool
	Msg      string
	Redirect string
	Btn      string
}

func Authorize(opt AuthorizeOption) func(*gin.Context) {
	return func(c *gin.Context) {
		var code = http.StatusForbidden
		if opt.Guest {
			code = http.StatusBadRequest
		}

		commonErr := ErrInfo{
			Title: "访问受限",
			Code:  code,
			Msg:   opt.Msg,
			Link:  opt.Redirect,
			Btn:   opt.Btn,
		}
		var isLogin bool

		// 用户鉴权
		token, _ := c.Cookie(singleton.Conf.Site.CookieName)
		token = strings.TrimSpace(token)
		if token != "" {
			var u model.User
			if err := singleton.DB.Where("token = ?", token).First(&u).Error; err == nil {
				isLogin = u.TokenExpired.After(time.Now())
			}
			if isLogin {
				c.Set(model.CtxKeyAuthorizedUser, &u)
			}
		}

		// API鉴权
		apiToken := c.GetHeader("Authorization")
		if apiToken != "" {
			var t model.ApiToken
			// TODO: 需要有缓存机制 减少数据库查询次数
			if err := singleton.DB.Where("token = ?", apiToken).First(&t).Error; err == nil {
				isLogin = t.TokenExpired.After(time.Now())
			}
			if isLogin {
				c.Set(model.CtxKeyAuthorizedUser, &t)
			}
		}

		// 已登录且只能游客访问
		if isLogin && opt.Guest {
			ShowErrorPage(c, commonErr, opt.IsPage)
			return
		}
		// 未登录且需要登录
		if !isLogin && opt.Member {
			ShowErrorPage(c, commonErr, opt.IsPage)
			return
		}
	}
}

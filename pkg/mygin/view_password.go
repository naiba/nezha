package mygin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/crypto/bcrypt"
)

type ValidateViewPasswordOption struct {
	IsPage        bool
	AbortWhenFail bool
}

func ValidateViewPassword(opt ValidateViewPasswordOption) gin.HandlerFunc {
	return func(c *gin.Context) {
		if singleton.Conf.Site.ViewPassword == "" {
			return
		}
		_, authorized := c.Get(model.CtxKeyAuthorizedUser)
		if authorized {
			return
		}
		viewPassword, err := c.Cookie(singleton.Conf.Site.CookieName + "-vp")
		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(viewPassword), []byte(singleton.Conf.Site.ViewPassword))
		}
		if err == nil {
			c.Set(model.CtxKeyViewPasswordVerified, true)
			return
		}
		if !opt.AbortWhenFail {
			return
		}
		if opt.IsPage {
			c.HTML(http.StatusOK, GetPreferredTheme(c, "/viewpassword"), CommonEnvironment(c, gin.H{
				"Title": singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "VerifyPassword"}),
			}))

		} else {
			c.JSON(http.StatusOK, model.Response{
				Code:    http.StatusForbidden,
				Message: "访问受限",
			})
		}
		c.Abort()
	}
}

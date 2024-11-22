package waf

import (
	_ "embed"
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

//go:embed waf.html
var errorPageTemplate string

func RealIp(c *gin.Context) {
	if singleton.Conf.RealIPHeader == "" {
		c.Next()
		return
	}

	if singleton.Conf.RealIPHeader == model.ConfigUsePeerIP {
		c.Set(model.CtxKeyRealIPStr, c.RemoteIP())
		c.Next()
		return
	}

	vals := c.Request.Header.Get(singleton.Conf.RealIPHeader)
	if vals == "" {
		c.AbortWithStatusJSON(http.StatusOK, model.CommonResponse[any]{Success: false, Error: "real ip header not found"})
		return
	}
	ip, err := netip.ParseAddrPort(vals)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, model.CommonResponse[any]{Success: false, Error: err.Error()})
		return
	}
	c.Set(model.CtxKeyRealIPStr, ip.Addr().String())
	c.Next()
}

func Waf(c *gin.Context) {
	if singleton.Conf.RealIPHeader == "" {
		c.Next()
		return
	}
	realipAddr := c.GetString(model.CtxKeyRealIPStr)
	if realipAddr == "" {
		c.Next()
		return
	}
	if err := model.CheckIP(singleton.DB, realipAddr); err != nil {
		ShowBlockPage(c, err)
		return
	}
	c.Next()
}

func ShowBlockPage(c *gin.Context, err error) {
	c.Writer.WriteHeader(http.StatusForbidden)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteString(strings.Replace(errorPageTemplate, "{error}", err.Error(), 1))
	c.Abort()
}

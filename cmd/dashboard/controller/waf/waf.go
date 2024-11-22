package waf

import (
	_ "embed"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"gorm.io/gorm"
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
	ip, err := netip.ParseAddr(vals)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, model.CommonResponse[any]{Success: false, Error: err.Error()})
		return
	}
	c.Set(model.CtxKeyRealIPStr, ip.String())
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
	var w model.WAF
	if err := singleton.DB.First(&w, "ip = ?", realipAddr).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			ShowBlockPage(c, err)
			return
		}
	}
	now := time.Now().Unix()
	if w.LastBlockTimestamp+pow(w.Count, 4) > uint64(now) {
		log.Println(w.Count, w.LastBlockTimestamp+pow(w.Count, 4)-uint64(now))
		ShowBlockPage(c, errors.New("you are blocked by nezha WAF"))
		return
	}
	c.Next()
}

func pow(x, y uint64) uint64 {
	base := big.NewInt(0).SetUint64(x)
	exp := big.NewInt(0).SetUint64(y)
	result := big.NewInt(1)
	result.Exp(base, exp, nil)
	if !result.IsUint64() {
		return ^uint64(0) // return max uint64 value on overflow
	}
	return result.Uint64()
}

func ShowBlockPage(c *gin.Context, err error) {
	c.Writer.WriteHeader(http.StatusForbidden)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteString(strings.Replace(errorPageTemplate, "{error}", err.Error(), 1))
	c.Abort()
}

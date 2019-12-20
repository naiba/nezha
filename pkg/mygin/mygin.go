package mygin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/service/dao"
)

// CommonEnvironment ..
func CommonEnvironment(c *gin.Context, data map[string]interface{}) gin.H {
	data["MatchedPath"] = c.MustGet("MatchedPath")
	data["Version"] = dao.Version
	// 站点标题
	if t, has := data["Title"]; !has {
		data["Title"] = dao.Conf.Site.Brand
	} else {
		data["Title"] = fmt.Sprintf("%s - %s", t, dao.Conf.Site.Brand)
	}
	u, ok := c.Get(model.CtxKeyAuthorizedUser)
	if ok {
		data["Admin"] = u
	}
	return data
}

// RecordPath ..
func RecordPath(c *gin.Context) {
	url := c.Request.URL.String()
	for _, p := range c.Params {
		url = strings.Replace(url, p.Value, ":"+p.Key, 1)
	}
	c.Set("MatchedPath", url)
}

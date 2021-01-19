package mygin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/dao"
)

var adminPage = map[string]bool{
	"/server":       true,
	"/monitor":      true,
	"/setting":      true,
	"/notification": true,
	"/cron":         true,
}

func CommonEnvironment(c *gin.Context, data map[string]interface{}) gin.H {
	data["MatchedPath"] = c.MustGet("MatchedPath")
	data["Version"] = dao.Version
	// 是否是管理页面
	data["IsAdminPage"] = adminPage[data["MatchedPath"].(string)]
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

func RecordPath(c *gin.Context) {
	url := c.Request.URL.String()
	for _, p := range c.Params {
		url = strings.Replace(url, p.Value, ":"+p.Key, 1)
	}
	c.Set("MatchedPath", url)
}

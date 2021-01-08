package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
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
	mr.GET("/notification", mp.notification)
	mr.GET("/setting", mp.setting)
}

func (mp *memberPage) server(c *gin.Context) {
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	c.HTML(http.StatusOK, "dashboard/server", mygin.CommonEnvironment(c, gin.H{
		"Title":   "服务器管理",
		"Servers": dao.SortedServerList,
	}))
}

func (mp *memberPage) notification(c *gin.Context) {
	var nf []model.Notification
	dao.DB.Find(&nf)
	var ar []model.AlertRule
	dao.DB.Find(&ar)
	c.HTML(http.StatusOK, "dashboard/notification", mygin.CommonEnvironment(c, gin.H{
		"Title":         "通知管理",
		"Notifications": nf,
		"AlertRules":    ar,
	}))
}

func (mp *memberPage) setting(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/setting", mygin.CommonEnvironment(c, gin.H{
		"Title": "系统设置",
		"Conf":  dao.Conf,
	}))
}

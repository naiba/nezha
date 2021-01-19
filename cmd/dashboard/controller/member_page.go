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
	mr.GET("/monitor", mp.monitor)
	mr.GET("/cron", mp.cron)
	mr.GET("/notification", mp.notification)
	mr.GET("/setting", mp.setting)
}

func (mp *memberPage) server(c *gin.Context) {
	dao.SortedServerLock.RLock()
	defer dao.SortedServerLock.RUnlock()
	c.HTML(http.StatusOK, "dashboard/server", mygin.CommonEnvironment(c, gin.H{
		"Title":   "服务器管理",
		"Servers": dao.SortedServerList,
	}))
}

func (mp *memberPage) monitor(c *gin.Context) {
	var monitors []model.Monitor
	dao.DB.Find(&monitors)
	c.HTML(http.StatusOK, "dashboard/monitor", mygin.CommonEnvironment(c, gin.H{
		"Title":    "服务监控",
		"Monitors": monitors,
	}))
}

func (mp *memberPage) cron(c *gin.Context) {
	var crons []model.Cron
	dao.DB.Find(&crons)
	c.HTML(http.StatusOK, "dashboard/cron", mygin.CommonEnvironment(c, gin.H{
		"Title": "计划任务",
		"Crons": crons,
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

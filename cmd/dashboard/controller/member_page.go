package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/singleton"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type memberPage struct {
	r *gin.Engine
}

func (mp *memberPage) serve() {
	mr := mp.r.Group("")
	mr.Use(mygin.Authorize(mygin.AuthorizeOption{
		MemberOnly: true,
		IsPage:     true,
		Msg:        singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "YouAreNotAuthorized"}),
		Btn:        singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "Login"}),
		Redirect:   "/login",
	}))
	mr.GET("/server", mp.server)
	mr.GET("/monitor", mp.monitor)
	mr.GET("/cron", mp.cron)
	mr.GET("/notification", mp.notification)
	mr.GET("/ddns", mp.ddns)
	mr.GET("/nat", mp.nat)
	mr.GET("/setting", mp.setting)
	mr.GET("/api", mp.api)
}

func (mp *memberPage) api(c *gin.Context) {
	singleton.ApiLock.RLock()
	defer singleton.ApiLock.RUnlock()
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/api", mygin.CommonEnvironment(c, gin.H{
		"title":  singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ApiManagement"}),
		"Tokens": singleton.ApiTokenList,
	}))
}

func (mp *memberPage) server(c *gin.Context) {
	singleton.SortedServerLock.RLock()
	defer singleton.SortedServerLock.RUnlock()
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/server", mygin.CommonEnvironment(c, gin.H{
		"Title":   singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ServersManagement"}),
		"Servers": singleton.SortedServerList,
	}))
}

func (mp *memberPage) monitor(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/monitor", mygin.CommonEnvironment(c, gin.H{
		"Title":    singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ServicesManagement"}),
		"Monitors": singleton.ServiceSentinelShared.Monitors(),
	}))
}

func (mp *memberPage) cron(c *gin.Context) {
	var crons []model.Cron
	singleton.DB.Find(&crons)
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/cron", mygin.CommonEnvironment(c, gin.H{
		"Title": singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ScheduledTasks"}),
		"Crons": crons,
	}))
}

func (mp *memberPage) notification(c *gin.Context) {
	var nf []model.Notification
	singleton.DB.Find(&nf)
	var ar []model.AlertRule
	singleton.DB.Find(&ar)
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/notification", mygin.CommonEnvironment(c, gin.H{
		"Title":         singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "Notification"}),
		"Notifications": nf,
		"AlertRules":    ar,
	}))
}

func (mp *memberPage) ddns(c *gin.Context) {
	var data []model.DDNSProfile
	singleton.DB.Find(&data)
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/ddns", mygin.CommonEnvironment(c, gin.H{
		"Title":        singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "DDNS"}),
		"DDNS":         data,
		"ProviderMap":  model.ProviderMap,
		"ProviderList": model.ProviderList,
	}))
}

func (mp *memberPage) nat(c *gin.Context) {
	var data []model.NAT
	singleton.DB.Find(&data)
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/nat", mygin.CommonEnvironment(c, gin.H{
		"Title": singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "NAT"}),
		"NAT":   data,
	}))
}

func (mp *memberPage) setting(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/setting", mygin.CommonEnvironment(c, gin.H{
		"Title":           singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "Settings"}),
		"Languages":       model.Languages,
		"DashboardThemes": model.DashboardThemes,
	}))
}

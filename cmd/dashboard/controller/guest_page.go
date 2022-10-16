package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/singleton"
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
	gr.GET("/chart", gp.chart)
	oauth := &oauth2controller{
		r: gr,
	}
	oauth.serve()
}

type TransferRecord struct {
	Date string `gorm:"date" json:"date"`
	In   uint64 `gorm:"in" json:"in"`
	Out  uint64 `gorm:"out" json:"out"`
}

func (gp *guestPage) chart(c *gin.Context) {
	// 获取 GET 请求参数 并转化为 uint64
	serverId, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	// 获取server_id对应的本周所有 Transfer 记录
	var trs []TransferRecord
	// 获取本月第一天
	monthFirstDay := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, singleton.Loc)

	// 获取一个月内，每天的 Transfer In Out 的总和
	singleton.DB.Model(&model.Transfer{}).
		Select(`date(created_at) as date, sum("in") as "in", sum(out) as out`).
		Where("server_id = ? AND created_at > ?", serverId, monthFirstDay).
		Group("date(created_at)").Scan(&trs)
	// 获取 Transfer 记录的 created_at in out
	recordLen := len(trs)
	xAxis := make([]string, recordLen)
	in := make([]uint64, recordLen)
	out := make([]uint64, recordLen)
	var totalIn, totalOut uint64
	for i := 0; i < recordLen; i++ {
		xAxis[i] = trs[i].Date
		in[i] = trs[i].In
		out[i] = trs[i].Out
		totalIn += trs[i].In
		totalOut += trs[i].Out
	}
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "success",
		"xAxis":    xAxis,
		"in":       in,
		"out":      out,
		"totalIn":  totalIn,
		"totalOut": totalOut,
	})
}

func (gp *guestPage) login(c *gin.Context) {
	LoginType := "GitHub"
	RegistrationLink := "https://github.com/join"
	if singleton.Conf.Oauth2.Type == model.ConfigTypeGitee {
		LoginType = "Gitee"
		RegistrationLink = "https://gitee.com/signup"
	} else if singleton.Conf.Oauth2.Type == model.ConfigTypeGitlab {
		LoginType = "Gitlab"
		RegistrationLink = "https://gitlab.com/users/sign_up"
	} else if singleton.Conf.Oauth2.Type == model.ConfigTypeJihulab {
		LoginType = "Jihulab"
		RegistrationLink = "https://jihulab.com/users/sign_up"
	} else if singleton.Conf.Oauth2.Type == model.ConfigTypeGitea {
		LoginType = "Gitea"
		RegistrationLink = fmt.Sprintf("%s/user/sign_up", singleton.Conf.Oauth2.Endpoint)
	}
	c.HTML(http.StatusOK, "dashboard-"+singleton.Conf.Site.DashboardTheme+"/login", mygin.CommonEnvironment(c, gin.H{
		"Title":            singleton.Localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "Login"}),
		"LoginType":        LoginType,
		"RegistrationLink": RegistrationLink,
	}))
}

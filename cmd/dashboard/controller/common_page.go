package controller

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
)

type commonPage struct {
	r *gin.Engine
}

func (cp *commonPage) serve() {
	cr := cp.r.Group("")
	cr.Use(mygin.Authorize(mygin.AuthorizeOption{}))
	cr.POST("/view-password", cp.issueViewPassword)
	cr.Use(cp.checkViewPassword) // 前端查看密码鉴权
	cr.GET("/", cp.home)
	cr.GET("/service", cp.service)
	cr.GET("/ws", cp.ws)
}

type viewPasswordForm struct {
	Password string
}

func (p *commonPage) issueViewPassword(c *gin.Context) {
	var vpf viewPasswordForm
	err := c.ShouldBind(&vpf)
	var hash []byte
	if err == nil && vpf.Password != dao.Conf.Site.ViewPassword {
		err = errors.New("查看密码错误")
	}
	if err == nil {
		hash, err = bcrypt.GenerateFromPassword([]byte(vpf.Password), bcrypt.DefaultCost)
	}
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Title: "出现错误",
			Msg:   fmt.Sprintf("请求错误：%s", err),
		}, true)
		c.Abort()
		return
	}
	c.SetCookie(dao.Conf.Site.CookieName+"-vp", string(hash), 60*60*24, "", "", false, false)
	c.Redirect(http.StatusFound, c.Request.Referer())
}

func (p *commonPage) checkViewPassword(c *gin.Context) {
	if dao.Conf.Site.ViewPassword == "" {
		c.Next()
		return
	}
	if _, authorized := c.Get(model.CtxKeyAuthorizedUser); authorized {
		c.Next()
		return
	}

	// 验证查看密码
	viewPassword, _ := c.Cookie(dao.Conf.Site.CookieName + "-vp")
	if err := bcrypt.CompareHashAndPassword([]byte(viewPassword), []byte(dao.Conf.Site.ViewPassword)); err != nil {
		c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/viewpassword", mygin.CommonEnvironment(c, gin.H{
			"Title":      "验证查看密码",
			"CustomCode": dao.Conf.Site.CustomCode,
		}))
		c.Abort()
		return
	}

	c.Next()
}

func (p *commonPage) service(c *gin.Context) {
	msm := dao.ServiceSentinelShared.LoadStats()
	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/service", mygin.CommonEnvironment(c, gin.H{
		"Title":      "服务状态",
		"Services":   msm,
		"CustomCode": dao.Conf.Site.CustomCode,
	}))
}

func (cp *commonPage) home(c *gin.Context) {
	dao.SortedServerLock.RLock()
	defer dao.SortedServerLock.RUnlock()

	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/home", mygin.CommonEnvironment(c, gin.H{
		"Servers":    dao.SortedServerList,
		"CustomCode": dao.Conf.Site.CustomCode,
	}))
}

var upgrader = websocket.Upgrader{}

func (cp *commonPage) ws(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusInternalServerError,
			Title: "网络错误",
			Msg:   "Websocket协议切换失败",
			Link:  "/",
			Btn:   "返回首页",
		}, true)
		return
	}
	defer conn.Close()
	count := 0
	for {
		dao.SortedServerLock.RLock()
		err = conn.WriteJSON(dao.SortedServerList)
		dao.SortedServerLock.RUnlock()
		if err != nil {
			break
		}
		count += 1
		if count%4 == 0 {
			err = conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				break
			}
		}
		time.Sleep(time.Second * 2)
	}
}

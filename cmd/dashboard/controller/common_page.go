package controller

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/pkg/mygin"
	pb "github.com/p14yground/nezha/proto"
	"github.com/p14yground/nezha/service/dao"
)

type commonPage struct {
	r *gin.Engine
}

func (cp *commonPage) serve() {
	cr := cp.r.Group("")
	cr.Use(mygin.Authorize(mygin.AuthorizeOption{}))
	cr.GET("/", cp.home)
	cr.GET("/ws", cp.ws)
}

func (cp *commonPage) home(c *gin.Context) {
	var admin *model.User
	isLogin, ok := c.Get(model.CtxKeyIsUserLogin)
	if ok && isLogin.(bool) {
		admin = dao.Admin
	}
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	c.HTML(http.StatusOK, "page/home", mygin.CommonEnvironment(c, gin.H{
		"Admin":   admin,
		"Domain":  dao.Conf.Site.Domain,
		"Servers": dao.ServerList,
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
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		var mt int
		var message []byte
		for {
			mt, message, err = conn.ReadMessage()
			if err != nil {
				wg.Done()
				break
			}
			if mt == websocket.TextMessage && string(message) == "track" {
				dao.SendCommand(&pb.Command{
					Type: model.MTReportState,
				})
			}
		}
	}()
	go func() {
		for {
			dao.ServerLock.RLock()
			err = conn.WriteJSON(dao.ServerList)
			dao.ServerLock.RUnlock()
			if err != nil {
				wg.Done()
				break
			}
			time.Sleep(time.Second * 2)
		}
	}()
	wg.Wait()
}

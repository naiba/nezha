package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

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
	cr.GET("/", cp.home)
	cr.GET("/ws", cp.ws)
}

func (cp *commonPage) home(c *gin.Context) {
	dao.ServerLock.RLock()
	defer dao.ServerLock.RUnlock()
	data := gin.H{
		"Servers":    dao.SortedServerList,
		"CustomCode": dao.Conf.Site.CustomCode,
	}
	u, ok := c.Get(model.CtxKeyAuthorizedUser)
	if ok {
		data["Admin"] = u
	}
	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/home", mygin.CommonEnvironment(c, data))
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
	for {
		dao.ServerLock.RLock()
		err = conn.WriteJSON(dao.SortedServerList)
		dao.ServerLock.RUnlock()
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
}

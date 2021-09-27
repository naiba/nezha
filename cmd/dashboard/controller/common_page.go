package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/dao"
)

type terminalContext struct {
	agentConn *websocket.Conn
	userConn  *websocket.Conn
	serverID  uint64
	host      string
	useSSL    bool
}

type commonPage struct {
	r             *gin.Engine
	terminals     map[string]*terminalContext
	terminalsLock *sync.Mutex
}

func (cp *commonPage) serve() {
	cr := cp.r.Group("")
	cr.Use(mygin.Authorize(mygin.AuthorizeOption{}))
	cr.GET("/terminal/:id", cp.terminal)
	cr.POST("/view-password", cp.issueViewPassword)
	cr.Use(cp.checkViewPassword) // 前端查看密码鉴权
	cr.GET("/", cp.home)
	cr.GET("/service", cp.service)
	cr.GET("/ws", cp.ws)
	cr.POST("/terminal", cp.createTerminal)
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
			Code:  http.StatusOK,
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
	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/service", mygin.CommonEnvironment(c, gin.H{
		"Title":      "服务状态",
		"Services":   dao.ServiceSentinelShared.LoadStats(),
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Data struct {
	Now     int64           `json:"now,omitempty"`
	Servers []*model.Server `json:"servers,omitempty"`
}

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
		err = conn.WriteJSON(Data{
			Now:     time.Now().Unix() * 1000,
			Servers: dao.SortedServerList,
		})
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

func (cp *commonPage) terminal(c *gin.Context) {
	terminalID := c.Param("id")
	cp.terminalsLock.Lock()
	if terminalID == "" || cp.terminals[terminalID] == nil {
		cp.terminalsLock.Unlock()
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusForbidden,
			Title: "无权访问",
			Msg:   "终端会话不存在",
			Link:  "/",
			Btn:   "返回首页",
		}, true)
		return
	}

	terminal := cp.terminals[terminalID]
	cp.terminalsLock.Unlock()

	defer func() {
		// 清理 context
		cp.terminalsLock.Lock()
		defer cp.terminalsLock.Unlock()
		delete(cp.terminals, terminalID)
	}()

	var isAgent bool

	if _, authorized := c.Get(model.CtxKeyAuthorizedUser); !authorized {
		dao.ServerLock.RLock()
		_, hasID := dao.SecretToID[c.Request.Header.Get("Secret")]
		dao.ServerLock.RUnlock()
		if !hasID {
			mygin.ShowErrorPage(c, mygin.ErrInfo{
				Code:  http.StatusForbidden,
				Title: "无权访问",
				Msg:   "用户未登录或非法终端",
				Link:  "/",
				Btn:   "返回首页",
			}, true)
			return
		}
		if terminal.userConn == nil {
			mygin.ShowErrorPage(c, mygin.ErrInfo{
				Code:  http.StatusForbidden,
				Title: "无权访问",
				Msg:   "用户不在线",
				Link:  "/",
				Btn:   "返回首页",
			}, true)
			return
		}
		if terminal.agentConn != nil {
			mygin.ShowErrorPage(c, mygin.ErrInfo{
				Code:  http.StatusInternalServerError,
				Title: "连接已存在",
				Msg:   "Websocket协议切换失败",
				Link:  "/",
				Btn:   "返回首页",
			}, true)
			return
		}
		isAgent = true
	} else {
		dao.ServerLock.RLock()
		server := dao.ServerList[terminal.serverID]
		dao.ServerLock.RUnlock()
		if server == nil || server.TaskStream == nil {
			mygin.ShowErrorPage(c, mygin.ErrInfo{
				Code:  http.StatusForbidden,
				Title: "请求失败",
				Msg:   "服务器不存在或处于离线状态",
				Link:  "/server",
				Btn:   "返回重试",
			}, true)
			return
		}

		terminalData, _ := json.Marshal(&model.TerminalTask{
			Host:    terminal.host,
			UseSSL:  terminal.useSSL,
			Session: terminalID,
		})

		if err := server.TaskStream.Send(&proto.Task{
			Type: model.TaskTypeTerminal,
			Data: string(terminalData),
		}); err != nil {
			mygin.ShowErrorPage(c, mygin.ErrInfo{
				Code:  http.StatusForbidden,
				Title: "请求失败",
				Msg:   "Agent信令下发失败",
				Link:  "/server",
				Btn:   "返回重试",
			}, true)
			return
		}
	}

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

	log.Println("NEZHA>> terminal connected", isAgent, c.Request.URL)
	defer log.Println("NEZHA>> terminal disconnected", isAgent, c.Request.URL)

	if isAgent {
		terminal.agentConn = conn
		defer func() {
			// Agent断开链接时断开用户连接
			if terminal.userConn != nil {
				terminal.userConn.Close()
			}
		}()
	} else {
		terminal.userConn = conn
		defer func() {
			// 用户断开链接时断开 Agent 连接
			if terminal.agentConn != nil {
				terminal.agentConn.Close()
			}
		}()
	}

	deadlineCh := make(chan interface{})
	go func() {
		// 对方连接超时
		connectDeadline := time.NewTimer(time.Second * 15)
		<-connectDeadline.C
		deadlineCh <- struct{}{}
	}()

	go func() {
		// PING 保活
		for {
			if err = conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
			time.Sleep(time.Second * 10)
		}
	}()

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	go func() {
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				errorCh <- err
				return
			}
			// 将文本消息转换为命令输入
			if msgType == websocket.TextMessage {
				data = append([]byte{0}, data...)
			}
			dataCh <- data
		}
	}()

	var dataBuffer [][]byte
	var distConn *websocket.Conn
	checkDistConn := func() {
		if distConn == nil {
			if isAgent {
				distConn = terminal.userConn
			} else {
				distConn = terminal.agentConn
			}
		}
	}

	for {
		select {
		case <-deadlineCh:
			checkDistConn()
			if distConn == nil {
				return
			}
		case <-errorCh:
			return
		case data := <-dataCh:
			dataBuffer = append(dataBuffer, data)
			checkDistConn()
			if distConn != nil {
				for i := 0; i < len(dataBuffer); i++ {
					err = distConn.WriteMessage(websocket.BinaryMessage, dataBuffer[i])
					if err != nil {
						return
					}
				}
				dataBuffer = dataBuffer[:0]
			}
		}
	}
}

type createTerminalRequest struct {
	Host     string
	Protocol string
	ID       uint64
}

func (cp *commonPage) createTerminal(c *gin.Context) {
	if _, authorized := c.Get(model.CtxKeyAuthorizedUser); !authorized {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusForbidden,
			Title: "无权访问",
			Msg:   "用户未登录",
			Link:  "/login",
			Btn:   "去登录",
		}, true)
		return
	}
	var createTerminalReq createTerminalRequest
	if err := c.ShouldBind(&createTerminalReq); err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusForbidden,
			Title: "请求失败",
			Msg:   "请求参数有误：" + err.Error(),
			Link:  "/server",
			Btn:   "返回重试",
		}, true)
		return
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusInternalServerError,
			Title: "系统错误",
			Msg:   "生成会话ID失败",
			Link:  "/server",
			Btn:   "返回重试",
		}, true)
		return
	}

	dao.ServerLock.RLock()
	server := dao.ServerList[createTerminalReq.ID]
	dao.ServerLock.RUnlock()
	if server == nil || server.TaskStream == nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusForbidden,
			Title: "请求失败",
			Msg:   "服务器不存在或处于离线状态",
			Link:  "/server",
			Btn:   "返回重试",
		}, true)
		return
	}

	cp.terminalsLock.Lock()
	defer cp.terminalsLock.Unlock()

	cp.terminals[id] = &terminalContext{
		serverID: createTerminalReq.ID,
		host:     createTerminalReq.Host,
		useSSL:   createTerminalReq.Protocol == "https:",
	}

	c.HTML(http.StatusOK, "dashboard/terminal", mygin.CommonEnvironment(c, gin.H{
		"SessionID":  id,
		"ServerName": server.Name,
	}))
}

package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-uuid"
	"github.com/jinzhu/copier"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/singleton"
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
	requestGroup  singleflight.Group
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
	if err == nil && vpf.Password != singleton.Conf.Site.ViewPassword {
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
	c.SetCookie(singleton.Conf.Site.CookieName+"-vp", string(hash), 60*60*24, "", "", false, false)
	c.Redirect(http.StatusFound, c.Request.Referer())
}

func (p *commonPage) checkViewPassword(c *gin.Context) {
	if singleton.Conf.Site.ViewPassword == "" {
		c.Next()
		return
	}
	if _, authorized := c.Get(model.CtxKeyAuthorizedUser); authorized {
		c.Next()
		return
	}

	// 验证查看密码
	viewPassword, _ := c.Cookie(singleton.Conf.Site.CookieName + "-vp")
	if err := bcrypt.CompareHashAndPassword([]byte(viewPassword), []byte(singleton.Conf.Site.ViewPassword)); err != nil {
		c.HTML(http.StatusOK, "theme-"+singleton.Conf.Site.Theme+"/viewpassword", mygin.CommonEnvironment(c, gin.H{
			"Title":      "验证查看密码",
			"CustomCode": singleton.Conf.Site.CustomCode,
		}))
		c.Abort()
		return
	}

	c.Next()
}

func (p *commonPage) service(c *gin.Context) {
	res, _, _ := p.requestGroup.Do("servicePage", func() (interface{}, error) {
		singleton.AlertsLock.RLock()
		defer singleton.AlertsLock.RUnlock()
		var stats map[uint64]model.ServiceItemResponse
		var statsStore map[uint64]model.CycleTransferStats
		copier.Copy(&stats, singleton.ServiceSentinelShared.LoadStats())
		copier.Copy(&statsStore, singleton.AlertsCycleTransferStatsStore)
		return []interface {
		}{
			stats, statsStore,
		}, nil
	})
	c.HTML(http.StatusOK, "theme-"+singleton.Conf.Site.Theme+"/service", mygin.CommonEnvironment(c, gin.H{
		"Title":              "服务状态",
		"Services":           res.([]interface{})[0],
		"CycleTransferStats": res.([]interface{})[1],
		"CustomCode":         singleton.Conf.Site.CustomCode,
	}))
}

func (cp *commonPage) getServerStat() ([]byte, error) {
	v, err, _ := cp.requestGroup.Do("serverStats", func() (any, error) {
		singleton.SortedServerLock.RLock()
		defer singleton.SortedServerLock.RUnlock()
		return utils.Json.Marshal(Data{
			Now:     time.Now().Unix() * 1000,
			Servers: singleton.SortedServerList,
		})
	})
	return v.([]byte), err
}

func (cp *commonPage) home(c *gin.Context) {
	stat, err := cp.getServerStat()
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusInternalServerError,
			Title: "系统错误",
			Msg:   "服务器状态获取失败",
			Link:  "/",
			Btn:   "返回首页",
		}, true)
		return
	}
	c.HTML(http.StatusOK, "theme-"+singleton.Conf.Site.Theme+"/home", mygin.CommonEnvironment(c, gin.H{
		"Servers":    string(stat),
		"CustomCode": singleton.Conf.Site.CustomCode,
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
		stat, err := cp.getServerStat()
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, stat); err != nil {
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
		singleton.ServerLock.RLock()
		_, hasID := singleton.SecretToID[c.Request.Header.Get("Secret")]
		singleton.ServerLock.RUnlock()
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
		singleton.ServerLock.RLock()
		server := singleton.ServerList[terminal.serverID]
		singleton.ServerLock.RUnlock()
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
		cloudflareCookies, _ := c.Cookie("CF_Authorization")
		// CloudflareCookies合法性验证
		// 其应该包含.分隔的三组BASE64-URL编码
		if cloudflareCookies != "" {
			encodedCookies := strings.Split(cloudflareCookies, ".")
			if len(encodedCookies) == 3 {
				for i := 0; i < 3; i++ {
					if valid, _ := regexp.MatchString("^[A-Za-z0-9-_]+$", encodedCookies[i]); !valid {
						cloudflareCookies = ""
						break
					}
				}
			} else {
				cloudflareCookies = ""
			}
		}
		terminalData, _ := utils.Json.Marshal(&model.TerminalTask{
			Host:    terminal.host,
			UseSSL:  terminal.useSSL,
			Session: terminalID,
			Cookie:  cloudflareCookies,
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

	log.Printf("NEZHA>> terminal connected %t %q", isAgent, c.Request.URL)
	defer log.Printf("NEZHA>> terminal disconnected %t %q", isAgent, c.Request.URL)

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

	singleton.ServerLock.RLock()
	server := singleton.ServerList[createTerminalReq.ID]
	singleton.ServerLock.RUnlock()
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

package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/service/singleton"
)

type memberAPI struct {
	r gin.IRouter
}

func (ma *memberAPI) serve() {
	mr := ma.r.Group("")
	mr.Use(mygin.Authorize(mygin.AuthorizeOption{
		Member:   true,
		IsPage:   false,
		Msg:      "访问此接口需要登录",
		Btn:      "点此登录",
		Redirect: "/login",
	}))

	mr.GET("/search-server", ma.searchServer)
	mr.POST("/server", ma.addOrEditServer)
	mr.POST("/monitor", ma.addOrEditMonitor)
	mr.POST("/cron", ma.addOrEditCron)
	mr.GET("/cron/:id/manual", ma.manualTrigger)
	mr.POST("/force-update", ma.forceUpdate)
	mr.POST("/notification", ma.addOrEditNotification)
	mr.POST("/alert-rule", ma.addOrEditAlertRule)
	mr.POST("/setting", ma.updateSetting)
	mr.DELETE("/:model/:id", ma.delete)
	mr.POST("/logout", ma.logout)
	mr.GET("/token", ma.getToken)
	mr.POST("/token", ma.issueNewToken)
	mr.DELETE("/token/:token", ma.deleteToken)

	// API
	v1 := ma.r.Group("v1")
	{
		apiv1 := &apiV1{v1}
		apiv1.serve()
	}
}

type apiResult struct {
	Token string `json:"token"`
	Note  string `json:"note"`
}

// getToken 获取 Token
func (ma *memberAPI) getToken(c *gin.Context) {
	u := c.MustGet(model.CtxKeyAuthorizedUser).(*model.User)
	singleton.ApiLock.RLock()
	defer singleton.ApiLock.RUnlock()

	tokenList := singleton.UserIDToApiTokenList[u.ID]
	res := make([]*apiResult, len(tokenList))
	for i, token := range tokenList {
		res[i] = &apiResult{
			Token: token,
			Note:  singleton.ApiTokenList[token].Note,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"result":  res,
	})
}

type TokenForm struct {
	Note string
}

// issueNewToken 生成新的 token
func (ma *memberAPI) issueNewToken(c *gin.Context) {
	u := c.MustGet(model.CtxKeyAuthorizedUser).(*model.User)
	tf := &TokenForm{}
	err := c.ShouldBindJSON(tf)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	token := &model.ApiToken{
		UserID: u.ID,
		Token:  utils.MD5(fmt.Sprintf("%d%d%s", time.Now().UnixNano(), u.ID, u.Login)),
		Note:   tf.Note,
	}
	singleton.DB.Create(token)

	singleton.ApiLock.Lock()
	singleton.ApiTokenList[token.Token] = token
	singleton.UserIDToApiTokenList[u.ID] = append(singleton.UserIDToApiTokenList[u.ID], token.Token)
	singleton.ApiLock.Unlock()

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
		Result: map[string]string{
			"token": token.Token,
			"note":  token.Note,
		},
	})
}

// deleteToken 删除 token
func (ma *memberAPI) deleteToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: "token 不能为空",
		})
		return
	}
	singleton.ApiLock.Lock()
	defer singleton.ApiLock.Unlock()
	if _, ok := singleton.ApiTokenList[token]; !ok {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: "token 不存在",
		})
		return
	}
	// 在数据库中删除该Token
	singleton.DB.Unscoped().Delete(&model.ApiToken{}, "token = ?", token)

	// 在UserIDToApiTokenList中删除该Token
	for i, t := range singleton.UserIDToApiTokenList[singleton.ApiTokenList[token].UserID] {
		if t == token {
			singleton.UserIDToApiTokenList[singleton.ApiTokenList[token].UserID] = append(singleton.UserIDToApiTokenList[singleton.ApiTokenList[token].UserID][:i], singleton.UserIDToApiTokenList[singleton.ApiTokenList[token].UserID][i+1:]...)
			break
		}
	}
	if len(singleton.UserIDToApiTokenList[singleton.ApiTokenList[token].UserID]) == 0 {
		delete(singleton.UserIDToApiTokenList, singleton.ApiTokenList[token].UserID)
	}
	// 在ApiTokenList中删除该Token
	delete(singleton.ApiTokenList, token)
	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (ma *memberAPI) delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id < 1 {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: "错误的 Server ID",
		})
		return
	}

	var err error
	switch c.Param("model") {
	case "server":
		err = singleton.DB.Unscoped().Delete(&model.Server{}, "id = ?", id).Error
		if err == nil {
			// 删除服务器
			singleton.ServerLock.Lock()
			tag := singleton.ServerList[id].Tag
			delete(singleton.SecretToID, singleton.ServerList[id].Secret)
			delete(singleton.ServerList, id)
			index := 0
			for index < len(singleton.ServerTagToIDList[tag]) {
				if singleton.ServerTagToIDList[tag][index] == id {
					break
				}
				index++
			}
			// 删除旧 Tag-ID 绑定关系
			singleton.ServerTagToIDList[tag] = append(singleton.ServerTagToIDList[tag][:index], singleton.ServerTagToIDList[tag][index+1:]...)
			if len(singleton.ServerTagToIDList[tag]) == 0 {
				delete(singleton.ServerTagToIDList, tag)
			}
			singleton.ServerLock.Unlock()
			singleton.ReSortServer()
			// 删除循环流量状态中的此服务器相关的记录
			singleton.AlertsLock.Lock()
			for i := 0; i < len(singleton.Alerts); i++ {
				if singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID] != nil {
					delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].ServerName, id)
					delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].Transfer, id)
					delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].NextUpdate, id)
				}
			}
			singleton.AlertsLock.Unlock()
			// 删除服务器相关循环流量记录
			singleton.DB.Unscoped().Delete(&model.Transfer{}, "server_id = ?", id)
		}
	case "notification":
		err = singleton.DB.Unscoped().Delete(&model.Notification{}, "id = ?", id).Error
		if err == nil {
			singleton.OnDeleteNotification(id)
		}
	case "monitor":
		err = singleton.DB.Unscoped().Delete(&model.Monitor{}, "id = ?", id).Error
		if err == nil {
			singleton.ServiceSentinelShared.OnMonitorDelete(id)
			err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ?", id).Error
		}
	case "cron":
		err = singleton.DB.Unscoped().Delete(&model.Cron{}, "id = ?", id).Error
		if err == nil {
			singleton.CronLock.RLock()
			defer singleton.CronLock.RUnlock()
			cr := singleton.Crons[id]
			if cr != nil && cr.CronJobID != 0 {
				singleton.Cron.Remove(cr.CronJobID)
			}
			delete(singleton.Crons, id)
		}
	case "alert-rule":
		err = singleton.DB.Unscoped().Delete(&model.AlertRule{}, "id = ?", id).Error
		if err == nil {
			singleton.OnDeleteAlert(id)
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("数据库错误：%s", err),
		})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type searchResult struct {
	Name  string `json:"name,omitempty"`
	Value uint64 `json:"value,omitempty"`
	Text  string `json:"text,omitempty"`
}

func (ma *memberAPI) searchServer(c *gin.Context) {
	var servers []model.Server
	likeWord := "%" + c.Query("word") + "%"
	singleton.DB.Select("id,name").Where("id = ? OR name LIKE ? OR tag LIKE ? OR note LIKE ?",
		c.Query("word"), likeWord, likeWord, likeWord).Find(&servers)

	var resp []searchResult
	for i := 0; i < len(servers); i++ {
		resp = append(resp, searchResult{
			Value: servers[i].ID,
			Name:  servers[i].Name,
			Text:  servers[i].Name,
		})
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"results": resp,
	})
}

type serverForm struct {
	ID           uint64
	Name         string `binding:"required"`
	DisplayIndex int
	Secret       string
	Tag          string
	Note         string
}

func (ma *memberAPI) addOrEditServer(c *gin.Context) {
	admin := c.MustGet(model.CtxKeyAuthorizedUser).(*model.User)
	var sf serverForm
	var s model.Server
	var isEdit bool
	err := c.ShouldBindJSON(&sf)
	if err == nil {
		s.Name = sf.Name
		s.Secret = sf.Secret
		s.DisplayIndex = sf.DisplayIndex
		s.ID = sf.ID
		s.Tag = sf.Tag
		s.Note = sf.Note
		if s.ID == 0 {
			s.Secret = utils.MD5(fmt.Sprintf("%s%s%d", time.Now(), sf.Name, admin.ID))
			s.Secret = s.Secret[:18]
			err = singleton.DB.Create(&s).Error
		} else {
			isEdit = true
			err = singleton.DB.Save(&s).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	if isEdit {
		singleton.ServerLock.Lock()
		s.CopyFromRunningServer(singleton.ServerList[s.ID])
		// 如果修改了 Secret
		if s.Secret != singleton.ServerList[s.ID].Secret {
			// 删除旧 Secret-ID 绑定关系
			singleton.SecretToID[s.Secret] = s.ID
			// 设置新的 Secret-ID 绑定关系
			delete(singleton.SecretToID, singleton.ServerList[s.ID].Secret)
		}
		// 如果修改了Tag
		if s.Tag != singleton.ServerList[s.ID].Tag {
			index := 0
			for index < len(singleton.ServerTagToIDList[s.Tag]) {
				if singleton.ServerTagToIDList[s.Tag][index] == s.ID {
					break
				}
				index++
			}
			// 删除旧 Tag-ID 绑定关系
			singleton.ServerTagToIDList[singleton.ServerList[s.ID].Tag] = append(singleton.ServerTagToIDList[singleton.ServerList[s.ID].Tag][:index], singleton.ServerTagToIDList[singleton.ServerList[s.ID].Tag][index+1:]...)
			// 设置新的 Tag-ID 绑定关系
			singleton.ServerTagToIDList[s.Tag] = append(singleton.ServerTagToIDList[s.Tag], s.ID)
			if len(singleton.ServerTagToIDList[s.Tag]) == 0 {
				delete(singleton.ServerTagToIDList, s.Tag)
			}
		}
		singleton.ServerList[s.ID] = &s
		singleton.ServerLock.Unlock()
	} else {
		s.Host = &model.Host{}
		s.State = &model.HostState{}
		singleton.ServerLock.Lock()
		singleton.SecretToID[s.Secret] = s.ID
		singleton.ServerList[s.ID] = &s
		singleton.ServerTagToIDList[s.Tag] = append(singleton.ServerTagToIDList[s.Tag], s.ID)
		singleton.ServerLock.Unlock()
	}
	singleton.ReSortServer()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type monitorForm struct {
	ID              uint64
	Name            string
	Target          string
	Type            uint8
	Cover           uint8
	Notify          string
	NotificationTag string
	SkipServersRaw  string
	Duration        uint64
}

func (ma *memberAPI) addOrEditMonitor(c *gin.Context) {
	var mf monitorForm
	var m model.Monitor
	err := c.ShouldBindJSON(&mf)
	if err == nil {
		m.Name = mf.Name
		m.Target = strings.TrimSpace(mf.Target)
		m.Type = mf.Type
		m.ID = mf.ID
		m.SkipServersRaw = mf.SkipServersRaw
		m.Cover = mf.Cover
		m.Notify = mf.Notify == "on"
		m.NotificationTag = mf.NotificationTag
		m.Duration = mf.Duration
		err = m.InitSkipServers()
	}
	if err == nil {
		// 保证NotificationTag不为空
		if m.NotificationTag == "" {
			m.NotificationTag = "default"
		}
		if m.ID == 0 {
			err = singleton.DB.Create(&m).Error
		} else {
			err = singleton.DB.Save(&m).Error
		}
	}
	if err == nil {
		err = singleton.ServiceSentinelShared.OnMonitorUpdate(m)
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type cronForm struct {
	ID              uint64
	Name            string
	Scheduler       string
	Command         string
	ServersRaw      string
	Cover           uint8
	PushSuccessful  string
	NotificationTag string
}

func (ma *memberAPI) addOrEditCron(c *gin.Context) {
	var cf cronForm
	var cr model.Cron
	err := c.ShouldBindJSON(&cf)
	if err == nil {
		cr.Name = cf.Name
		cr.Scheduler = cf.Scheduler
		cr.Command = cf.Command
		cr.ServersRaw = cf.ServersRaw
		cr.PushSuccessful = cf.PushSuccessful == "on"
		cr.NotificationTag = cf.NotificationTag
		cr.ID = cf.ID
		cr.Cover = cf.Cover
		err = utils.Json.Unmarshal([]byte(cf.ServersRaw), &cr.Servers)
	}
	tx := singleton.DB.Begin()
	if err == nil {
		// 保证NotificationTag不为空
		if cr.NotificationTag == "" {
			cr.NotificationTag = "default"
		}
		if cf.ID == 0 {
			err = tx.Create(&cr).Error
		} else {
			err = tx.Save(&cr).Error
		}
	}
	if err == nil {
		cr.CronJobID, err = singleton.Cron.AddFunc(cr.Scheduler, singleton.CronTrigger(cr))
	}
	if err == nil {
		err = tx.Commit().Error
	} else {
		tx.Rollback()
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}

	singleton.CronLock.Lock()
	defer singleton.CronLock.Unlock()
	crOld := singleton.Crons[cr.ID]
	if crOld != nil && crOld.CronJobID != 0 {
		singleton.Cron.Remove(crOld.CronJobID)
	}

	delete(singleton.Crons, cr.ID)
	singleton.Crons[cr.ID] = &cr

	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

func (ma *memberAPI) manualTrigger(c *gin.Context) {
	var cr model.Cron
	if err := singleton.DB.First(&cr, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	singleton.ManualTrigger(cr)

	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

func (ma *memberAPI) forceUpdate(c *gin.Context) {
	var forceUpdateServers []uint64
	if err := c.ShouldBindJSON(&forceUpdateServers); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	var executeResult bytes.Buffer

	for i := 0; i < len(forceUpdateServers); i++ {
		singleton.ServerLock.RLock()
		server := singleton.ServerList[forceUpdateServers[i]]
		singleton.ServerLock.RUnlock()
		if server != nil && server.TaskStream != nil {
			if err := server.TaskStream.Send(&proto.Task{
				Type: model.TaskTypeUpgrade,
			}); err != nil {
				executeResult.WriteString(fmt.Sprintf("%d 下发指令失败 %+v<br/>", forceUpdateServers[i], err))
			} else {
				executeResult.WriteString(fmt.Sprintf("%d 下发指令成功<br/>", forceUpdateServers[i]))
			}
		} else {
			executeResult.WriteString(fmt.Sprintf("%d 离线<br/>", forceUpdateServers[i]))
		}
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: executeResult.String(),
	})
}

type notificationForm struct {
	ID            uint64
	Name          string
	Tag           string // 分组名
	URL           string
	RequestMethod int
	RequestType   int
	RequestHeader string
	RequestBody   string
	VerifySSL     string
	SkipCheck     string
}

func (ma *memberAPI) addOrEditNotification(c *gin.Context) {
	var nf notificationForm
	var n model.Notification
	err := c.ShouldBindJSON(&nf)
	if err == nil {
		n.Name = nf.Name
		n.Tag = nf.Tag
		n.RequestMethod = nf.RequestMethod
		n.RequestType = nf.RequestType
		n.RequestHeader = nf.RequestHeader
		n.RequestBody = nf.RequestBody
		n.URL = nf.URL
		verifySSL := nf.VerifySSL == "on"
		n.VerifySSL = &verifySSL
		n.ID = nf.ID
		ns := model.NotificationServerBundle{
			Notification: &n,
			Server:       nil,
		}
		// 勾选了跳过检查
		if nf.SkipCheck != "on" {
			err = ns.Send("这是测试消息")
		}
	}
	if err == nil {
		// 保证Tag不为空
		if n.Tag == "" {
			n.Tag = "default"
		}
		if n.ID == 0 {
			err = singleton.DB.Create(&n).Error
		} else {
			err = singleton.DB.Save(&n).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	singleton.OnRefreshOrAddNotification(&n)
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type alertRuleForm struct {
	ID              uint64
	Name            string
	RulesRaw        string
	NotificationTag string
	Enable          string
}

func (ma *memberAPI) addOrEditAlertRule(c *gin.Context) {
	var arf alertRuleForm
	var r model.AlertRule
	err := c.ShouldBindJSON(&arf)
	if err == nil {
		err = utils.Json.Unmarshal([]byte(arf.RulesRaw), &r.Rules)
	}
	if err == nil {
		if len(r.Rules) == 0 {
			err = errors.New("至少定义一条规则")
		} else {
			for i := 0; i < len(r.Rules); i++ {
				if !r.Rules[i].IsTransferDurationRule() {
					if r.Rules[i].Duration < 3 {
						err = errors.New("错误：Duration 至少为 3")
						break
					}
				} else {
					if r.Rules[i].CycleInterval < 1 {
						err = errors.New("错误: cycle_interval 至少为 1")
						break
					}
					if r.Rules[i].CycleStart == nil {
						err = errors.New("错误: cycle_start 未设置")
						break
					}
					if r.Rules[i].CycleStart.After(time.Now()) {
						err = errors.New("错误: cycle_start 是个未来值")
						break
					}
				}
			}
		}
	}
	if err == nil {
		r.Name = arf.Name
		r.RulesRaw = arf.RulesRaw
		r.NotificationTag = arf.NotificationTag
		enable := arf.Enable == "on"
		r.Enable = &enable
		r.ID = arf.ID
		//保证NotificationTag不为空
		if r.NotificationTag == "" {
			r.NotificationTag = "default"
		}
		if r.ID == 0 {
			err = singleton.DB.Create(&r).Error
		} else {
			err = singleton.DB.Save(&r).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	singleton.OnRefreshOrAddAlert(r)
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type logoutForm struct {
	ID uint64
}

func (ma *memberAPI) logout(c *gin.Context) {
	admin := c.MustGet(model.CtxKeyAuthorizedUser).(*model.User)
	var lf logoutForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	if lf.ID != admin.ID {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", "用户ID不匹配"),
		})
		return
	}
	singleton.DB.Model(admin).UpdateColumns(model.User{
		Token:        "",
		TokenExpired: time.Now(),
	})
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type settingForm struct {
	Title                   string
	Admin                   string
	Language                string
	Theme                   string
	CustomCode              string
	ViewPassword            string
	IgnoredIPNotification   string
	IPChangeNotificationTag string // IP变更提醒的通知组
	GRPCHost                string
	Cover                   uint8

	EnableIPChangeNotification  string
	EnablePlainIPInNotification string
}

func (ma *memberAPI) updateSetting(c *gin.Context) {
	var sf settingForm
	if err := c.ShouldBind(&sf); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	singleton.Conf.Language = sf.Language
	singleton.Conf.EnableIPChangeNotification = sf.EnableIPChangeNotification == "on"
	singleton.Conf.EnablePlainIPInNotification = sf.EnablePlainIPInNotification == "on"
	singleton.Conf.Cover = sf.Cover
	singleton.Conf.GRPCHost = sf.GRPCHost
	singleton.Conf.IgnoredIPNotification = sf.IgnoredIPNotification
	singleton.Conf.IPChangeNotificationTag = sf.IPChangeNotificationTag
	singleton.Conf.Site.Brand = sf.Title
	singleton.Conf.Site.Theme = sf.Theme
	singleton.Conf.Site.CustomCode = sf.CustomCode
	singleton.Conf.Site.ViewPassword = sf.ViewPassword
	singleton.Conf.Oauth2.Admin = sf.Admin
	// 保证NotificationTag不为空
	if singleton.Conf.IPChangeNotificationTag == "" {
		singleton.Conf.IPChangeNotificationTag = "default"
	}
	if err := singleton.Conf.Save(); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	// 更新系统语言
	singleton.InitLocalizer()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

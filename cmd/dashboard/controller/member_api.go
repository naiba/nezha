package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"golang.org/x/net/idna"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/proto"
	"github.com/naiba/nezha/resource"
	"github.com/naiba/nezha/service/singleton"
)

type memberAPI struct {
	r gin.IRouter
}

func (ma *memberAPI) serve() {
	mr := ma.r.Group("")
	mr.Use(mygin.Authorize(mygin.AuthorizeOption{
		MemberOnly: true,
		IsPage:     false,
		Msg:        "访问此接口需要登录",
		Btn:        "点此登录",
		Redirect:   "/login",
	}))

	mr.GET("/search-server", ma.searchServer)
	mr.GET("/search-tasks", ma.searchTask)
	mr.GET("/search-ddns", ma.searchDDNS)
	mr.POST("/server", ma.addOrEditServer)
	mr.POST("/monitor", ma.addOrEditMonitor)
	mr.POST("/cron", ma.addOrEditCron)
	mr.GET("/cron/:id/manual", ma.manualTrigger)
	mr.POST("/force-update", ma.forceUpdate)
	mr.POST("/batch-update-server-group", ma.batchUpdateServerGroup)
	mr.POST("/batch-delete-server", ma.batchDeleteServer)
	mr.POST("/notification", ma.addOrEditNotification)
	mr.POST("/ddns", ma.addOrEditDDNS)
	mr.POST("/nat", ma.addOrEditNAT)
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
	secureToken, err := utils.GenerateRandomString(32)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	token := &model.ApiToken{
		UserID: u.ID,
		Token:  secureToken,
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
		err := singleton.DB.Transaction(func(tx *gorm.DB) error {
			err = singleton.DB.Unscoped().Delete(&model.Server{}, "id = ?", id).Error
			if err != nil {
				return err
			}
			err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "server_id = ?", id).Error
			if err != nil {
				return err
			}
			return nil
		})
		if err == nil {
			// 删除服务器
			singleton.ServerLock.Lock()
			onServerDelete(id)
			singleton.ServerLock.Unlock()
			singleton.ReSortServer()
		}
	case "notification":
		err = singleton.DB.Unscoped().Delete(&model.Notification{}, "id = ?", id).Error
		if err == nil {
			singleton.OnDeleteNotification(id)
		}
	case "ddns":
		err = singleton.DB.Unscoped().Delete(&model.DDNSProfile{}, "id = ?", id).Error
		if err == nil {
			singleton.OnDDNSUpdate()
		}
	case "nat":
		err = singleton.DB.Unscoped().Delete(&model.NAT{}, "id = ?", id).Error
		if err == nil {
			singleton.OnNATUpdate()
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

func (ma *memberAPI) searchTask(c *gin.Context) {
	var tasks []model.Cron
	likeWord := "%" + c.Query("word") + "%"
	singleton.DB.Select("id,name").Where("id = ? OR name LIKE ?",
		c.Query("word"), likeWord).Find(&tasks)

	var resp []searchResult
	for i := 0; i < len(tasks); i++ {
		resp = append(resp, searchResult{
			Value: tasks[i].ID,
			Name:  tasks[i].Name,
			Text:  tasks[i].Name,
		})
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"results": resp,
	})
}

func (ma *memberAPI) searchDDNS(c *gin.Context) {
	var ddns []model.DDNSProfile
	likeWord := "%" + c.Query("word") + "%"
	singleton.DB.Select("id,name").Where("id = ? OR name LIKE ?",
		c.Query("word"), likeWord).Find(&ddns)

	var resp []searchResult
	for i := 0; i < len(ddns); i++ {
		resp = append(resp, searchResult{
			Value: ddns[i].ID,
			Name:  ddns[i].Name,
			Text:  ddns[i].Name,
		})
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"results": resp,
	})
}

type serverForm struct {
	ID              uint64
	Name            string `binding:"required"`
	DisplayIndex    int
	Secret          string
	Tag             string
	Note            string
	PublicNote      string
	HideForGuest    string
	EnableDDNS      string
	DDNSProfilesRaw string
}

func (ma *memberAPI) addOrEditServer(c *gin.Context) {
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
		s.PublicNote = sf.PublicNote
		s.HideForGuest = sf.HideForGuest == "on"
		s.EnableDDNS = sf.EnableDDNS == "on"
		s.DDNSProfilesRaw = sf.DDNSProfilesRaw
		err = utils.Json.Unmarshal([]byte(sf.DDNSProfilesRaw), &s.DDNSProfiles)
		if err == nil {
			if s.ID == 0 {
				s.Secret, err = utils.GenerateRandomString(18)
				if err == nil {
					err = singleton.DB.Create(&s).Error
				}
			} else {
				isEdit = true
				err = singleton.DB.Save(&s).Error
			}
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
		oldTag := singleton.ServerList[s.ID].Tag
		newTag := s.Tag
		if newTag != oldTag {
			index := -1
			for i := 0; i < len(singleton.ServerTagToIDList[oldTag]); i++ {
				if singleton.ServerTagToIDList[oldTag][i] == s.ID {
					index = i
					break
				}
			}
			if index > -1 {
				// 删除旧 Tag-ID 绑定关系
				singleton.ServerTagToIDList[oldTag] = append(singleton.ServerTagToIDList[oldTag][:index], singleton.ServerTagToIDList[oldTag][index+1:]...)
				if len(singleton.ServerTagToIDList[oldTag]) == 0 {
					delete(singleton.ServerTagToIDList, oldTag)
				}
			}
			// 设置新的 Tag-ID 绑定关系
			singleton.ServerTagToIDList[newTag] = append(singleton.ServerTagToIDList[newTag], s.ID)
		}
		singleton.ServerList[s.ID] = &s
		singleton.ServerLock.Unlock()
	} else {
		s.Host = &model.Host{}
		s.State = &model.HostState{}
		s.TaskCloseLock = new(sync.Mutex)
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
	ID                     uint64
	Name                   string
	Target                 string
	Type                   uint8
	Cover                  uint8
	Notify                 string
	NotificationTag        string
	SkipServersRaw         string
	Duration               uint64
	MinLatency             float32
	MaxLatency             float32
	LatencyNotify          string
	EnableTriggerTask      string
	EnableShowInService    string
	FailTriggerTasksRaw    string
	RecoverTriggerTasksRaw string
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
		m.LatencyNotify = mf.LatencyNotify == "on"
		m.MinLatency = mf.MinLatency
		m.MaxLatency = mf.MaxLatency
		m.EnableShowInService = mf.EnableShowInService == "on"
		m.EnableTriggerTask = mf.EnableTriggerTask == "on"
		m.RecoverTriggerTasksRaw = mf.RecoverTriggerTasksRaw
		m.FailTriggerTasksRaw = mf.FailTriggerTasksRaw
		err = m.InitSkipServers()
	}
	if err == nil {
		// 保证NotificationTag不为空
		if m.NotificationTag == "" {
			m.NotificationTag = "default"
		}
		err = utils.Json.Unmarshal([]byte(mf.FailTriggerTasksRaw), &m.FailTriggerTasks)
	}
	if err == nil {
		err = utils.Json.Unmarshal([]byte(mf.RecoverTriggerTasksRaw), &m.RecoverTriggerTasks)
	}
	if err == nil {
		if m.ID == 0 {
			err = singleton.DB.Create(&m).Error
		} else {
			err = singleton.DB.Save(&m).Error
		}
	}
	if err == nil {
		if m.Cover == 0 {
			err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id in (?)", m.ID, strings.Split(m.SkipServersRaw[1:len(m.SkipServersRaw)-1], ",")).Error
		} else {
			err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id not in (?)", m.ID, strings.Split(m.SkipServersRaw[1:len(m.SkipServersRaw)-1], ",")).Error
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
	TaskType        uint8 // 0:计划任务 1:触发任务
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
		cr.TaskType = cf.TaskType
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

	// 计划任务类型不得使用触发服务器执行方式
	if cr.TaskType == model.CronTypeCronTask && cr.Cover == model.CronCoverAlertTrigger {
		err = errors.New("计划任务类型不得使用触发服务器执行方式")
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
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
		// 对于计划任务类型，需要更新CronJob
		if cf.TaskType == model.CronTypeCronTask {
			cr.CronJobID, err = singleton.Cron.AddFunc(cr.Scheduler, singleton.CronTrigger(cr))
		}
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

type BatchUpdateServerGroupRequest struct {
	Servers []uint64
	Group   string
}

func (ma *memberAPI) batchUpdateServerGroup(c *gin.Context) {
	var req BatchUpdateServerGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if err := singleton.DB.Model(&model.Server{}).Where("id in (?)", req.Servers).Update("tag", req.Group).Error; err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	singleton.ServerLock.Lock()

	for i := 0; i < len(req.Servers); i++ {
		serverId := req.Servers[i]
		var s model.Server
		copier.Copy(&s, singleton.ServerList[serverId])
		s.Tag = req.Group
		// 如果修改了Ta
		oldTag := singleton.ServerList[serverId].Tag
		newTag := s.Tag
		if newTag != oldTag {
			index := -1
			for i := 0; i < len(singleton.ServerTagToIDList[oldTag]); i++ {
				if singleton.ServerTagToIDList[oldTag][i] == s.ID {
					index = i
					break
				}
			}
			if index > -1 {
				// 删除旧 Tag-ID 绑定关系
				singleton.ServerTagToIDList[oldTag] = append(singleton.ServerTagToIDList[oldTag][:index], singleton.ServerTagToIDList[oldTag][index+1:]...)
				if len(singleton.ServerTagToIDList[oldTag]) == 0 {
					delete(singleton.ServerTagToIDList, oldTag)
				}
			}
			// 设置新的 Tag-ID 绑定关系
			singleton.ServerTagToIDList[newTag] = append(singleton.ServerTagToIDList[newTag], s.ID)
		}
		singleton.ServerList[s.ID] = &s
	}

	singleton.ServerLock.Unlock()

	singleton.ReSortServer()

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
			Loc:          singleton.Loc,
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

type ddnsForm struct {
	ID                 uint64
	MaxRetries         uint64
	EnableIPv4         string
	EnableIPv6         string
	Name               string
	Provider           uint8
	DomainsRaw         string
	AccessID           string
	AccessSecret       string
	WebhookURL         string
	WebhookMethod      uint8
	WebhookRequestType uint8
	WebhookRequestBody string
	WebhookHeaders     string
}

func (ma *memberAPI) addOrEditDDNS(c *gin.Context) {
	var df ddnsForm
	var p model.DDNSProfile
	err := c.ShouldBindJSON(&df)
	if err == nil {
		if df.MaxRetries < 1 || df.MaxRetries > 10 {
			err = errors.New("重试次数必须为大于 1 且不超过 10 的整数")
		}
	}
	if err == nil {
		p.Name = df.Name
		p.ID = df.ID
		enableIPv4 := df.EnableIPv4 == "on"
		enableIPv6 := df.EnableIPv6 == "on"
		p.EnableIPv4 = &enableIPv4
		p.EnableIPv6 = &enableIPv6
		p.MaxRetries = df.MaxRetries
		p.Provider = df.Provider
		p.DomainsRaw = df.DomainsRaw
		p.Domains = strings.Split(p.DomainsRaw, ",")
		p.AccessID = df.AccessID
		p.AccessSecret = df.AccessSecret
		p.WebhookURL = df.WebhookURL
		p.WebhookMethod = df.WebhookMethod
		p.WebhookRequestType = df.WebhookRequestType
		p.WebhookRequestBody = df.WebhookRequestBody
		p.WebhookHeaders = df.WebhookHeaders

		for n, domain := range p.Domains {
			// IDN to ASCII
			domainValid, domainErr := idna.Lookup.ToASCII(domain)
			if domainErr != nil {
				err = fmt.Errorf("域名 %s 解析错误: %v", domain, domainErr)
				break
			}
			p.Domains[n] = domainValid
		}
	}
	if err == nil {
		if p.ID == 0 {
			err = singleton.DB.Create(&p).Error
		} else {
			err = singleton.DB.Save(&p).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type natForm struct {
	ID       uint64
	Name     string
	ServerID uint64
	Host     string
	Domain   string
}

func (ma *memberAPI) addOrEditNAT(c *gin.Context) {
	var nf natForm
	var n model.NAT
	err := c.ShouldBindJSON(&nf)
	if err == nil {
		n.Name = nf.Name
		n.ID = nf.ID
		n.Domain = nf.Domain
		n.Host = nf.Host
		n.ServerID = nf.ServerID
	}
	if err == nil {
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
	singleton.OnNATUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type alertRuleForm struct {
	ID                     uint64
	Name                   string
	RulesRaw               string
	FailTriggerTasksRaw    string // 失败时触发的任务id
	RecoverTriggerTasksRaw string // 恢复时触发的任务id
	NotificationTag        string
	TriggerMode            int
	Enable                 string
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
		r.FailTriggerTasksRaw = arf.FailTriggerTasksRaw
		r.RecoverTriggerTasksRaw = arf.RecoverTriggerTasksRaw
		r.NotificationTag = arf.NotificationTag
		enable := arf.Enable == "on"
		r.TriggerMode = arf.TriggerMode
		r.Enable = &enable
		r.ID = arf.ID
	}
	if err == nil {
		err = utils.Json.Unmarshal([]byte(arf.FailTriggerTasksRaw), &r.FailTriggerTasks)
	}
	if err == nil {
		err = utils.Json.Unmarshal([]byte(arf.RecoverTriggerTasksRaw), &r.RecoverTriggerTasks)
	}
	//保证NotificationTag不为空
	if err == nil {
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

	if oidcLogoutUrl := singleton.Conf.Oauth2.OidcLogoutURL; oidcLogoutUrl != "" {
		// 重定向到 OIDC 退出登录地址。不知道为什么，这里的重定向不生效
		c.Redirect(http.StatusOK, oidcLogoutUrl)
	}
}

type settingForm struct {
	Title                   string
	Admin                   string
	Language                string
	Theme                   string
	DashboardTheme          string
	CustomCode              string
	CustomCodeDashboard     string
	CustomNameservers       string
	ViewPassword            string
	IgnoredIPNotification   string
	IPChangeNotificationTag string // IP变更提醒的通知组
	GRPCHost                string
	Cover                   uint8

	EnableIPChangeNotification      string
	EnablePlainIPInNotification     string
	DisableSwitchTemplateInFrontend string
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

	if _, yes := model.Themes[sf.Theme]; !yes {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("前台主题不存在：%s", sf.Theme),
		})
		return
	}

	if _, yes := model.DashboardThemes[sf.DashboardTheme]; !yes {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("后台主题不存在：%s", sf.DashboardTheme),
		})
		return
	}

	if !utils.IsFileExists("resource/template/theme-"+sf.Theme+"/home.html") && !resource.IsTemplateFileExist("template/theme-"+sf.Theme+"/home.html") {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("前台主题文件异常：%s", sf.Theme),
		})
		return
	}

	if !utils.IsFileExists("resource/template/dashboard-"+sf.DashboardTheme+"/setting.html") && !resource.IsTemplateFileExist("template/dashboard-"+sf.DashboardTheme+"/setting.html") {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("后台主题文件异常：%s", sf.DashboardTheme),
		})
		return
	}

	singleton.Conf.Language = sf.Language
	singleton.Conf.EnableIPChangeNotification = sf.EnableIPChangeNotification == "on"
	singleton.Conf.EnablePlainIPInNotification = sf.EnablePlainIPInNotification == "on"
	singleton.Conf.DisableSwitchTemplateInFrontend = sf.DisableSwitchTemplateInFrontend == "on"
	singleton.Conf.Cover = sf.Cover
	singleton.Conf.GRPCHost = sf.GRPCHost
	singleton.Conf.IgnoredIPNotification = sf.IgnoredIPNotification
	singleton.Conf.IPChangeNotificationTag = sf.IPChangeNotificationTag
	singleton.Conf.Site.Brand = sf.Title
	singleton.Conf.Site.Theme = sf.Theme
	singleton.Conf.Site.DashboardTheme = sf.DashboardTheme
	singleton.Conf.Site.CustomCode = sf.CustomCode
	singleton.Conf.Site.CustomCodeDashboard = sf.CustomCodeDashboard
	singleton.Conf.DNSServers = sf.CustomNameservers
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
	// 更新DNS服务器
	singleton.OnNameserverUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

func (ma *memberAPI) batchDeleteServer(c *gin.Context) {
	var servers []uint64
	if err := c.ShouldBindJSON(&servers); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	if err := singleton.DB.Unscoped().Delete(&model.Server{}, "id in (?)", servers).Error; err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	singleton.ServerLock.Lock()
	for i := 0; i < len(servers); i++ {
		id := servers[i]
		onServerDelete(id)
	}
	singleton.ServerLock.Unlock()
	singleton.ReSortServer()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

func onServerDelete(id uint64) {
	tag := singleton.ServerList[id].Tag
	delete(singleton.SecretToID, singleton.ServerList[id].Secret)
	delete(singleton.ServerList, id)
	index := -1
	for i := 0; i < len(singleton.ServerTagToIDList[tag]); i++ {
		if singleton.ServerTagToIDList[tag][i] == id {
			index = i
			break
		}
	}
	if index > -1 {

		singleton.ServerTagToIDList[tag] = append(singleton.ServerTagToIDList[tag][:index], singleton.ServerTagToIDList[tag][index+1:]...)
		if len(singleton.ServerTagToIDList[tag]) == 0 {
			delete(singleton.ServerTagToIDList, tag)
		}
	}

	singleton.AlertsLock.Lock()
	for i := 0; i < len(singleton.Alerts); i++ {
		if singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID] != nil {
			delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].ServerName, id)
			delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].Transfer, id)
			delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].NextUpdate, id)
		}
	}
	singleton.AlertsLock.Unlock()

	singleton.DB.Unscoped().Delete(&model.Transfer{}, "server_id = ?", id)
}

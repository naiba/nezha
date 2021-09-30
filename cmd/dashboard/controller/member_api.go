package controller

import (
	"encoding/json"
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
	"github.com/naiba/nezha/service/dao"
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
	mr.POST("/notification", ma.addOrEditNotification)
	mr.POST("/alert-rule", ma.addOrEditAlertRule)
	mr.POST("/setting", ma.updateSetting)
	mr.DELETE("/:model/:id", ma.delete)
	mr.POST("/logout", ma.logout)
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
		err = dao.DB.Unscoped().Delete(&model.Server{}, "id = ?", id).Error
		if err == nil {
			dao.ServerLock.Lock()
			delete(dao.SecretToID, dao.ServerList[id].Secret)
			delete(dao.ServerList, id)
			dao.ServerLock.Unlock()
			dao.ReSortServer()
		}
	case "notification":
		err = dao.DB.Unscoped().Delete(&model.Notification{}, "id = ?", id).Error
		if err == nil {
			dao.OnDeleteNotification(id)
		}
	case "monitor":
		err = dao.DB.Unscoped().Delete(&model.Monitor{}, "id = ?", id).Error
		if err == nil {
			dao.ServiceSentinelShared.OnMonitorDelete(id)
			err = dao.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ?", id).Error
		}
	case "cron":
		err = dao.DB.Unscoped().Delete(&model.Cron{}, "id = ?", id).Error
		if err == nil {
			dao.CronLock.RLock()
			defer dao.CronLock.RUnlock()
			cr := dao.Crons[id]
			if cr != nil && cr.CronJobID != 0 {
				dao.Cron.Remove(cr.CronJobID)
			}
			delete(dao.Crons, id)
		}
	case "alert-rule":
		err = dao.DB.Unscoped().Delete(&model.AlertRule{}, "id = ?", id).Error
		if err == nil {
			dao.OnDeleteAlert(id)
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
	dao.DB.Select("id,name").Where("id = ? OR name LIKE ? OR tag LIKE ? OR note LIKE ?",
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
			err = dao.DB.Create(&s).Error
		} else {
			isEdit = true
			err = dao.DB.Save(&s).Error
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
		dao.ServerLock.Lock()
		s.CopyFromRunningServer(dao.ServerList[s.ID])
		// 如果修改了 Secret
		if s.Secret != dao.ServerList[s.ID].Secret {
			// 删除旧 Secret-ID 绑定关系
			dao.SecretToID[s.Secret] = s.ID
			// 设置新的 Secret-ID 绑定关系
			delete(dao.SecretToID, dao.ServerList[s.ID].Secret)
		}
		dao.ServerList[s.ID] = &s
		dao.ServerLock.Unlock()
	} else {
		s.Host = &model.Host{}
		s.State = &model.HostState{}
		dao.ServerLock.Lock()
		dao.SecretToID[s.Secret] = s.ID
		dao.ServerList[s.ID] = &s
		dao.ServerLock.Unlock()
	}
	dao.ReSortServer()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type monitorForm struct {
	ID             uint64
	Name           string
	Target         string
	Type           uint8
	Cover          uint8
	Notify         string
	SkipServersRaw string
	Duration       uint64
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
		m.Duration = mf.Duration
	}
	if err == nil {
		if m.ID == 0 {
			err = dao.DB.Create(&m).Error
		} else {
			err = dao.DB.Save(&m).Error
		}
	}
	if err == nil {
		err = dao.ServiceSentinelShared.OnMonitorUpdate(m)
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
	ID             uint64
	Name           string
	Scheduler      string
	Command        string
	ServersRaw     string
	Cover          uint8
	PushSuccessful string
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
		cr.ID = cf.ID
		cr.Cover = cf.Cover
		err = json.Unmarshal([]byte(cf.ServersRaw), &cr.Servers)
	}
	tx := dao.DB.Begin()
	if err == nil {
		if cf.ID == 0 {
			err = tx.Create(&cr).Error
		} else {
			err = tx.Save(&cr).Error
		}
	}
	if err == nil {
		cr.CronJobID, err = dao.Cron.AddFunc(cr.Scheduler, dao.CronTrigger(cr))
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

	dao.CronLock.Lock()
	defer dao.CronLock.Unlock()
	crOld := dao.Crons[cr.ID]
	if crOld != nil && crOld.CronJobID != 0 {
		dao.Cron.Remove(crOld.CronJobID)
	}

	delete(dao.Crons, cr.ID)
	dao.Crons[cr.ID] = &cr

	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

func (ma *memberAPI) manualTrigger(c *gin.Context) {
	var cr model.Cron
	if err := dao.DB.First(&cr, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	dao.ManualTrigger(&cr)

	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type notificationForm struct {
	ID            uint64
	Name          string
	URL           string
	RequestMethod int
	RequestType   int
	RequestBody   string
	VerifySSL     string
}

func (ma *memberAPI) addOrEditNotification(c *gin.Context) {
	var nf notificationForm
	var n model.Notification
	err := c.ShouldBindJSON(&nf)
	if err == nil {
		n.Name = nf.Name
		n.RequestMethod = nf.RequestMethod
		n.RequestType = nf.RequestType
		n.RequestBody = nf.RequestBody
		n.URL = nf.URL
		verifySSL := nf.VerifySSL == "on"
		n.VerifySSL = &verifySSL
		n.ID = nf.ID
		err = n.Send("这是测试消息")
	}
	if err == nil {
		if n.ID == 0 {
			err = dao.DB.Create(&n).Error
		} else {
			err = dao.DB.Save(&n).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	dao.OnRefreshOrAddNotification(n)
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type alertRuleForm struct {
	ID       uint64
	Name     string
	RulesRaw string
	Enable   string
}

func (ma *memberAPI) addOrEditAlertRule(c *gin.Context) {
	var arf alertRuleForm
	var r model.AlertRule
	err := c.ShouldBindJSON(&arf)
	if err == nil {
		err = json.Unmarshal([]byte(arf.RulesRaw), &r.Rules)
	}
	if err == nil {
		if len(r.Rules) == 0 {
			err = errors.New("至少定义一条规则")
		} else {
			for i := 0; i < len(r.Rules); i++ {
				if !r.Rules[i].IsTransferDurationRule() && r.Rules[i].Duration < 3 {
					err = errors.New("错误：Duration 至少为 3")
					break
				}
			}
		}
	}
	if err == nil {
		r.Name = arf.Name
		r.RulesRaw = arf.RulesRaw
		enable := arf.Enable == "on"
		r.Enable = &enable
		r.ID = arf.ID
		if r.ID == 0 {
			err = dao.DB.Create(&r).Error
		} else {
			err = dao.DB.Save(&r).Error
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	dao.OnRefreshOrAddAlert(r)
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
	dao.DB.Model(admin).UpdateColumns(model.User{
		Token:        "",
		TokenExpired: time.Now(),
	})
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type settingForm struct {
	Title                      string
	Admin                      string
	Theme                      string
	CustomCode                 string
	ViewPassword               string
	EnableIPChangeNotification string
	IgnoredIPNotification      string
	GRPCHost                   string
	Cover                      uint8
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
	dao.Conf.EnableIPChangeNotification = sf.EnableIPChangeNotification == "on"
	dao.Conf.Cover = sf.Cover
	dao.Conf.GRPCHost = sf.GRPCHost
	dao.Conf.IgnoredIPNotification = sf.IgnoredIPNotification
	dao.Conf.Site.Brand = sf.Title
	dao.Conf.Site.Theme = sf.Theme
	dao.Conf.Site.CustomCode = sf.CustomCode
	dao.Conf.Site.ViewPassword = sf.ViewPassword
	dao.Conf.Oauth2.Admin = sf.Admin
	if err := dao.Conf.Save(); err != nil {
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

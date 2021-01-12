package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naiba/com"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/alertmanager"
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

	mr.POST("/logout", ma.logout)
	mr.POST("/server", ma.addOrEditServer)
	mr.POST("/notification", ma.addOrEditNotification)
	mr.POST("/alert-rule", ma.addOrEditAlertRule)
	mr.POST("/setting", ma.updateSetting)
	mr.DELETE("/:model/:id", ma.delete)
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
		dao.ServerLock.Lock()
		defer dao.ServerLock.Unlock()
		err = dao.DB.Delete(&model.Server{}, "id = ?", id).Error
		if err == nil {
			delete(dao.ServerList, id)
			dao.ReSortServer()
		}
	case "notification":
		err = dao.DB.Delete(&model.Notification{}, "id = ?", id).Error
		if err == nil {
			alertmanager.OnDeleteNotification(id)
		}
	case "alert-rule":
		err = dao.DB.Delete(&model.AlertRule{}, "id = ?", id).Error
		if err == nil {
			alertmanager.OnDeleteAlert(id)
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

type serverForm struct {
	ID           uint64
	Name         string `binding:"required"`
	DisplayIndex int
	Secret       string
	Tag          string
}

func (ma *memberAPI) addOrEditServer(c *gin.Context) {
	admin := c.MustGet(model.CtxKeyAuthorizedUser).(*model.User)
	var sf serverForm
	var s model.Server
	var isEdit bool
	err := c.ShouldBindJSON(&sf)
	if err == nil {
		dao.ServerLock.Lock()
		defer dao.ServerLock.Unlock()
		s.Name = sf.Name
		s.Secret = sf.Secret
		s.DisplayIndex = sf.DisplayIndex
		s.ID = sf.ID
		s.Tag = sf.Tag
		if sf.ID == 0 {
			s.Secret = com.MD5(fmt.Sprintf("%s%s%d", time.Now(), sf.Name, admin.ID))
			s.Secret = s.Secret[:10]
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
		s.Host = dao.ServerList[s.ID].Host
		s.State = dao.ServerList[s.ID].State
	} else {
		s.Host = &model.Host{}
		s.State = &model.State{}
	}
	dao.ServerList[s.ID] = &s
	dao.ReSortServer()
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
	alertmanager.OnRefreshOrAddNotification(n)
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
				if r.Rules[i].Duration < 3 {
					err = errors.New("Duration 至少为 3")
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
	alertmanager.OnRefreshOrAddAlert(r)
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
	Title      string
	Admin      string
	Theme      string
	CustomCode string
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
	dao.Conf.Site.Brand = sf.Title
	dao.Conf.Site.Theme = sf.Theme
	dao.Conf.Site.CustomCode = sf.CustomCode
	dao.Conf.GitHub.Admin = sf.Admin
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

package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naiba/com"

	"github.com/p14yground/nezha/model"
	"github.com/p14yground/nezha/pkg/mygin"
	"github.com/p14yground/nezha/service/dao"
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
	mr.POST("/server", ma.addServer)
}

type serverForm struct {
	Name string `binding:"required"`
}

func (ma *memberAPI) addServer(c *gin.Context) {
	var sf serverForm
	var s model.Server
	err := c.ShouldBindJSON(&sf)
	if err == nil {
		dao.ServerLock.Lock()
		defer dao.ServerLock.Unlock()
		s.Name = sf.Name
		s.Secret = com.MD5(fmt.Sprintf("%s%s%d", time.Now(), sf.Name, dao.Admin.ID))
		s.Secret = s.Secret[:10]
		err = dao.DB.Create(&s).Error
	}
	if err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	dao.ServerList[fmt.Sprintf("%d", s.ID)] = &s
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

type logoutForm struct {
	ID uint64
}

func (ma *memberAPI) logout(c *gin.Context) {
	var lf logoutForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", err),
		})
		return
	}
	if lf.ID != dao.Admin.ID {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("请求错误：%s", "用户ID不匹配"),
		})
		return
	}
	dao.Admin.Token = ""
	dao.Admin.TokenExpired = time.Now()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

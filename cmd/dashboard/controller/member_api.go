package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

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

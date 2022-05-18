package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/singleton"
	"strconv"
	"strings"
)

type apiV1 struct {
	r gin.IRouter
}

func (v *apiV1) serve() {
	r := v.r.Group("")
	// API
	r.Use(mygin.Authorize(mygin.AuthorizeOption{
		Member:   true,
		IsPage:   false,
		AllowAPI: true,
		Msg:      "访问此接口需要认证",
		Btn:      "点此登录",
		Redirect: "/login",
	}))
	r.GET("/server/list", v.serverList)
	r.GET("/server/details", v.serverDetails)

}

// serverList 获取服务器列表 不传入Query参数则获取全部
// header: Authorization: Token
// query: tag (服务器分组)
func (v *apiV1) serverList(c *gin.Context) {
	token, _ := c.Cookie("Authorization")
	tag := c.Query("tag")
	serverAPI := &singleton.ServerAPI{
		Token: token,
		Tag:   tag,
	}
	if tag != "" {
		c.JSON(200, serverAPI.GetListByTag())
		return
	}
	c.JSON(200, serverAPI.GetAllList())
}

// serverDetails 获取服务器信息 不传入Query参数则获取全部
// header: Authorization: Token
// query: id (服务器ID，逗号分隔，优先级高于tag查询)
// query: tag (服务器分组)
func (v *apiV1) serverDetails(c *gin.Context) {
	token, _ := c.Cookie("Authorization")
	var idList []uint64
	idListStr := strings.Split(c.Query("id"), ",")
	if c.Query("id") != "" {
		idList = make([]uint64, len(idListStr))
		for i, v := range idListStr {
			id, _ := strconv.ParseUint(v, 10, 64)
			idList[i] = id
		}
	}
	tag := c.Query("tag")
	serverAPI := &singleton.ServerAPI{
		Token:  token,
		IDList: idList,
		Tag:    tag,
	}
	if tag != "" {
		c.JSON(200, serverAPI.GetStatusByTag())
		return
	}
	if len(idList) != 0 {
		c.JSON(200, serverAPI.GetStatusByIDList())
		return
	}
	c.JSON(200, serverAPI.GetAllStatus())
}

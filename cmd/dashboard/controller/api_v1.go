package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/singleton"
)

type apiV1 struct {
	r gin.IRouter
}

func (v *apiV1) serve() {
	r := v.r.Group("")
	// 强制认证的 API
	r.Use(mygin.Authorize(mygin.AuthorizeOption{
		MemberOnly: true,
		AllowAPI:   true,
		IsPage:     false,
		Msg:        "访问此接口需要认证",
		Btn:        "点此登录",
		Redirect:   "/login",
	}))
	r.GET("/server/list", v.serverList)
	r.GET("/server/details", v.serverDetails)
	r.POST("/server/register", v.RegisterServer)
	// 不强制认证的 API
	mr := v.r.Group("monitor")
	mr.Use(mygin.Authorize(mygin.AuthorizeOption{
		MemberOnly: false,
		IsPage:     false,
		AllowAPI:   true,
		Msg:        "访问此接口需要认证",
		Btn:        "点此登录",
		Redirect:   "/login",
	}))
	mr.Use(mygin.ValidateViewPassword(mygin.ValidateViewPasswordOption{
		IsPage:        false,
		AbortWhenFail: true,
	}))
	mr.GET("/:id", v.monitorHistoriesById)
}

// serverList 获取服务器列表 不传入Query参数则获取全部
// header: Authorization: Token
// query: tag (服务器分组)
func (v *apiV1) serverList(c *gin.Context) {
	tag := c.Query("tag")
	if tag != "" {
		c.JSON(200, singleton.ServerAPI.GetListByTag(tag))
		return
	}
	c.JSON(200, singleton.ServerAPI.GetAllList())
}

// serverDetails 获取服务器信息 不传入Query参数则获取全部
// header: Authorization: Token
// query: id (服务器ID，逗号分隔，优先级高于tag查询)
// query: tag (服务器分组)
func (v *apiV1) serverDetails(c *gin.Context) {
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
	if tag != "" {
		c.JSON(200, singleton.ServerAPI.GetStatusByTag(tag))
		return
	}
	if len(idList) != 0 {
		c.JSON(200, singleton.ServerAPI.GetStatusByIDList(idList))
		return
	}
	c.JSON(200, singleton.ServerAPI.GetAllStatus())
}

// RegisterServer adds a server and responds with the full ServerRegisterResponse
// header: Authorization: Token
// body: RegisterServer
// response: ServerRegisterResponse or Secret string
func (v *apiV1) RegisterServer(c *gin.Context) {
	var rs singleton.RegisterServer
	// Attempt to bind JSON to RegisterServer struct
	if err := c.ShouldBindJSON(&rs); err != nil {
		c.JSON(400, singleton.ServerRegisterResponse{
			CommonResponse: singleton.CommonResponse{
				Code:    400,
				Message: "Parse JSON failed",
			},
		})
		return
	}
	// Check if simple mode is requested
	simple := c.Query("simple") == "true" || c.Query("simple") == "1"
	// Set defaults if fields are empty
	if rs.Name == "" {
		rs.Name = c.ClientIP()
	}
	if rs.Tag == "" {
		rs.Tag = "AutoRegister"
	}
	if rs.HideForGuest == "" {
		rs.HideForGuest = "on"
	}
	// Call the Register function and get the response
	response := singleton.ServerAPI.Register(&rs)
	// Respond with Secret only if in simple mode, otherwise full response
	if simple {
		c.JSON(response.Code, response.Secret)
	} else {
		c.JSON(response.Code, response)
	}
}


func (v *apiV1) monitorHistoriesById(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"code": 400, "message": "id参数错误"})
		return
	}
	server, ok := singleton.ServerList[id]
	if !ok {
		c.AbortWithStatusJSON(404, gin.H{
			"code":    404,
			"message": "id不存在",
		})
		return
	}

	_, isMember := c.Get(model.CtxKeyAuthorizedUser)
	_, isViewPasswordVerfied := c.Get(model.CtxKeyViewPasswordVerified)
	authorized := isMember || isViewPasswordVerfied

	if server.HideForGuest && !authorized {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "需要认证"})
		return
	}

	c.JSON(200, singleton.MonitorAPI.GetMonitorHistories(map[string]any{"server_id": server.ID}))
}

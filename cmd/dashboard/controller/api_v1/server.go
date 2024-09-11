package api_v1

import (
	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"net/http"
	"strconv"
	"strings"
)

// getServerDetails
// @Summary 获取服务器信息
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param id query string false "服务器ID，逗号分隔，优先级高于tag查询"
// @Param tag query string false "服务器分组"
// @Produce json
// @Success 200 {object} singleton.ServerStatusResponse
// @Router /api/v1/server/details [get]
func (v *ApiV1) getServerDetails(c *gin.Context) {
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

// Get getServerList
// @Summary 获取服务器列表
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param tag query string false "服务器分组"
// @Produce json
// @Success 200 {object} singleton.ServerInfoResponse
// @Router /api/v1/server/list [get]
func (v *ApiV1) getServerList(c *gin.Context) {
	tag := c.Query("tag")
	if tag != "" {
		c.JSON(200, singleton.ServerAPI.GetListByTag(tag))
		return
	}
	c.JSON(200, singleton.ServerAPI.GetAllList())
}

// addServer
// @Summary 添加新服务器
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param Payload body singleton.ServerConfigData true "服务器信息"
// @Produce json
// @Success 200 {object} singleton.ServerConfigResponse
// @Router /api/v1/server [post]
func (v *ApiV1) addServer(c *gin.Context) {
	var sf singleton.ServerConfigData
	var err error
	var res *singleton.ServerConfigResponse
	if err = c.ShouldBindJSON(&sf); err == nil {
		res, err = singleton.ServerAPI.AddServer(sf)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// editServer
// @Summary 编辑服务器
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param Payload body singleton.ServerConfigData true "服务器信息"
// @Produce json
// @Success 200 {object} singleton.ServerConfigResponse
// @Router /api/v1/server [put]
func (v *ApiV1) editServer(c *gin.Context) {
	var sf singleton.ServerConfigData
	var err error
	var res *singleton.ServerConfigResponse
	if err = c.ShouldBindJSON(&sf); err == nil {
		res, err = singleton.ServerAPI.EditServer(sf)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// deleteServer
// @Summary 删除服务器
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param Payload body singleton.ServerDeleteRequest true "服务器ID列表"
// @Produce json
// @Success 200 {object} singleton.ServerDeleteResponse
// @Router /api/v1/server [delete]
func (v *ApiV1) deleteServer(c *gin.Context) {
	var err error
	var sf singleton.ServerDeleteRequest
	if err = c.ShouldBindJSON(&sf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	res := singleton.ServerAPI.DeleteServer(sf)
	c.JSON(http.StatusOK, res)
	return
}

// batchEditServerGroup
// @Summary 批量更新服务器分组
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param Payload body singleton.BatchUpdateServerGroupRequest true "更新信息"
// @Produce json
// @Success 200 {object} singleton.BatchUpdateServerGroupResponse
// @Router /api/v1/server/groups [put]
func (v *ApiV1) batchEditServerGroup(c *gin.Context) {
	var req singleton.BatchUpdateServerGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	service := singleton.ServerAPIService{}
	res := service.BatchUpdateGroup(req)
	c.JSON(http.StatusOK, res)
}

// batchUpgradeServerAgent
// @Summary 强制更新agent
// @tags server
// @Accept json
// @Param Authorization header string false "API Token"
// @Param Payload body singleton.ForceUpdateAgentRequest true "需要强制更新的服务器列表"
// @Produce json
// @Success 200 {object} singleton.ForceUpdateAgentResponse
// @Router /api/v1/server/upgrade [post]
func (v *ApiV1) batchUpgradeServerAgent(c *gin.Context) {
	var req singleton.ForceUpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	service := singleton.ServerAPIService{}
	res := service.ForceUpdateAgent(req)
	c.JSON(http.StatusOK, res)
}

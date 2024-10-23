package controller

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

// List notification
// @Summary List notification
// @Security BearerAuth
// @Schemes
// @Description List notification
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /notification [get]
func listNotification(c *gin.Context) ([]model.Notification, error) {
	var notifications []model.Notification
	if err := singleton.DB.Find(&notifications).Error; err != nil {
		return nil, newGormError("%v", err)
	}
	return notifications, nil
}

// Add notification
// @Summary Add notification
// @Security BearerAuth
// @Schemes
// @Description Add notification
// @Tags auth required
// @Accept json
// @param request body model.NotificationForm true "NotificationForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /notification [post]
func createNotification(c *gin.Context) (uint64, error) {
	var nf model.NotificationForm
	if err := c.ShouldBindJSON(&nf); err != nil {
		return 0, err
	}

	var n model.Notification
	n.Name = nf.Name
	n.RequestMethod = nf.RequestMethod
	n.RequestType = nf.RequestType
	n.RequestHeader = nf.RequestHeader
	n.RequestBody = nf.RequestBody
	n.URL = nf.URL
	verifySSL := nf.VerifySSL
	n.VerifySSL = &verifySSL
	n.ID = nf.ID
	ns := model.NotificationServerBundle{
		Notification: &n,
		Server:       nil,
		Loc:          singleton.Loc,
	}
	// 未勾选跳过检查
	if !nf.SkipCheck {
		if err := ns.Send("这是测试消息"); err != nil {
			return 0, err
		}
	}

	if err := singleton.DB.Create(&n).Error; err != nil {
		return 0, newGormError("%v", err)
	}

	singleton.OnRefreshOrAddNotification(&n)
	return n.ID, nil
}

// Edit notification
// @Summary Edit notification
// @Security BearerAuth
// @Schemes
// @Description Edit notification
// @Tags auth required
// @Accept json
// @Param id path uint true "Notification ID"
// @Param body body model.NotificationForm true "NotificationForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /notification/{id} [patch]
func updateNotification(c *gin.Context) (any, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}
	var nf model.NotificationForm
	if err := c.ShouldBindJSON(&nf); err != nil {
		return nil, err
	}

	var n model.Notification
	if err := singleton.DB.First(&n, id).Error; err != nil {
		return nil, fmt.Errorf("notification id %d does not exist", id)
	}

	n.Name = nf.Name
	n.RequestMethod = nf.RequestMethod
	n.RequestType = nf.RequestType
	n.RequestHeader = nf.RequestHeader
	n.RequestBody = nf.RequestBody
	n.URL = nf.URL
	verifySSL := nf.VerifySSL
	n.VerifySSL = &verifySSL
	n.ID = nf.ID
	ns := model.NotificationServerBundle{
		Notification: &n,
		Server:       nil,
		Loc:          singleton.Loc,
	}
	// 未勾选跳过检查
	if !nf.SkipCheck {
		if err := ns.Send("这是测试消息"); err != nil {
			return nil, err
		}
	}

	if err := singleton.DB.Save(&n).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.OnRefreshOrAddNotification(&n)
	return nil, nil
}

// Batch delete notifications
// @Summary Batch delete notifications
// @Security BearerAuth
// @Schemes
// @Description Batch delete notifications
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/notification [post]
func batchDeleteNotification(c *gin.Context) (any, error) {
	var n []uint64

	if err := c.ShouldBindJSON(&n); err != nil {
		return nil, err
	}

	if err := singleton.DB.Unscoped().Delete(&model.Notification{}, "id in (?)", n).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.OnDeleteNotification(n)
	return nil, nil
}

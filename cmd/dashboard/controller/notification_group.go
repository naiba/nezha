package controller

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

// List notification group
// @Summary List notification group
// @Schemes
// @Description List notification group
// @Security BearerAuth
// @Tags common
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.NotificationGroupResponseItem]
// @Router /notification-group [get]
func listNotificationGroup(c *gin.Context) error {
	var ng []model.NotificationGroup
	if err := singleton.DB.Find(&ng).Error; err != nil {
		return err
	}

	var ngn []model.NotificationGroupNotification
	if err := singleton.DB.Find(&ngn).Error; err != nil {
		return err
	}

	groupNotifications := make(map[uint64][]uint64, len(ng))
	for _, n := range ngn {
		if _, ok := groupNotifications[n.NotificationGroupID]; !ok {
			groupNotifications[n.NotificationGroupID] = make([]uint64, 0)
		}
		groupNotifications[n.NotificationGroupID] = append(groupNotifications[n.NotificationGroupID], n.NotificationID)
	}

	ngRes := make([]model.NotificationGroupResponseItem, 0, len(ng))
	for _, n := range ng {
		ngRes = append(ngRes, model.NotificationGroupResponseItem{
			Group:         n,
			Notifications: groupNotifications[n.ID],
		})
	}

	c.JSON(http.StatusOK, model.CommonResponse[[]model.NotificationGroupResponseItem]{
		Success: true,
		Data:    ngRes,
	})
	return nil
}

// New notification group
// @Summary New notification group
// @Schemes
// @Description New notification group
// @Security BearerAuth
// @Tags auth required
// @Accept json
// @Param body body model.NotificationGroupForm true "NotificationGroupForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /notification-group [post]
func newNotificationGroup(c *gin.Context) error {
	var ngf model.NotificationGroupForm
	if err := c.ShouldBindJSON(&ngf); err != nil {
		return err
	}
	ngf.Notifications = slices.Compact(ngf.Notifications)

	var ng model.NotificationGroup
	ng.Name = ngf.Name

	var count int64
	if err := singleton.DB.Model(&model.Notification{}).Where("id in (?)", ngf.Notifications).Count(&count).Error; err != nil {
		return err
	}

	if count != int64(len(ngf.Notifications)) {
		return fmt.Errorf("have invalid notification id")
	}

	singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ng).Error; err != nil {
			return err
		}
		for _, n := range ngf.Notifications {
			if err := tx.Create(&model.NotificationGroupNotification{
				NotificationGroupID: ng.ID,
				NotificationID:      n,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	singleton.OnRefreshOrAddNotificationGroup(&ng, ngf.Notifications)
	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

// Edit notification group
// @Summary Edit notification group
// @Schemes
// @Description Edit notification group
// @Security BearerAuth
// @Tags auth required
// @Accept json
// @Param id path string true "ID"
// @Param body body model.NotificationGroupForm true "NotificationGroupForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /notification-group/{id} [patch]
func editNotificationGroup(c *gin.Context) error {
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	var ngf model.NotificationGroupForm
	if err := c.ShouldBindJSON(&ngf); err != nil {
		return err
	}
	var ngDB model.NotificationGroup
	if err := singleton.DB.First(&ngDB, id).Error; err != nil {
		return fmt.Errorf("group id %d does not exist", id)
	}

	ngDB.Name = ngf.Name
	ngf.Notifications = slices.Compact(ngf.Notifications)

	var count int64
	if err := singleton.DB.Model(&model.Server{}).Where("id in (?)", ngf.Notifications).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ngf.Notifications)) {
		return fmt.Errorf("have invalid notification id")
	}

	err = singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ngDB).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.NotificationGroupNotification{}, "notification_group_id = ?", id).Error; err != nil {
			return err
		}

		for _, n := range ngf.Notifications {
			if err := tx.Create(&model.NotificationGroupNotification{
				NotificationGroupID: ngDB.ID,
				NotificationID:      n,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return newGormError("%v", err)
	}

	singleton.OnRefreshOrAddNotificationGroup(&ngDB, ngf.Notifications)
	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

// Batch delete notification group
// @Summary Batch delete notification group
// @Security BearerAuth
// @Schemes
// @Description Batch delete notification group
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/notification-group [post]
func batchDeleteNotificationGroup(c *gin.Context) error {
	var ngn []uint64
	if err := c.ShouldBindJSON(&ngn); err != nil {
		return err
	}

	err := singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&model.NotificationGroup{}, "id in (?)", ngn).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&model.NotificationGroupNotification{}, "notification_group_id in (?)", ngn).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return newGormError("%v", err)
	}

	singleton.OnDeleteNotificationGroup(ngn)
	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

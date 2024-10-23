package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"gorm.io/gorm"
)

// List monitor
// @Summary List monitor
// @Security BearerAuth
// @Schemes
// @Description List monitor
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.Monitor]
// @Router /monitor [get]
func listMonitor(c *gin.Context) ([]*model.Monitor, error) {
	return singleton.ServiceSentinelShared.Monitors(), nil
}

// Create monitor
// @Summary Create monitor
// @Security BearerAuth
// @Schemes
// @Description Create monitor
// @Tags auth required
// @Accept json
// @param request body model.MonitorForm true "Monitor Request"
// @Produce json
// @Success 200 {object} model.CommonResponse[uint64]
// @Router /monitor [post]
func createMonitor(c *gin.Context) (uint64, error) {
	var mf model.MonitorForm
	if err := c.ShouldBindJSON(&mf); err != nil {
		return 0, err
	}

	var m model.Monitor
	m.Name = mf.Name
	m.Target = strings.TrimSpace(mf.Target)
	m.Type = mf.Type
	m.SkipServers = mf.SkipServers
	m.Cover = mf.Cover
	m.Notify = mf.Notify
	m.NotificationGroupID = mf.NotificationGroupID
	m.Duration = mf.Duration
	m.LatencyNotify = mf.LatencyNotify
	m.MinLatency = mf.MinLatency
	m.MaxLatency = mf.MaxLatency
	m.EnableShowInService = mf.EnableShowInService
	m.EnableTriggerTask = mf.EnableTriggerTask
	m.RecoverTriggerTasks = mf.RecoverTriggerTasks
	m.FailTriggerTasks = mf.FailTriggerTasks

	if err := singleton.DB.Create(&m).Error; err != nil {
		return 0, err
	}

	var skipServers []uint64
	for k := range m.SkipServers {
		skipServers = append(skipServers, k)
	}

	var err error
	if m.Cover == 0 {
		err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id in (?)", m.ID, skipServers).Error
	} else {
		err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id not in (?)", m.ID, skipServers).Error
	}
	if err != nil {
		return 0, err
	}

	return m.ID, singleton.ServiceSentinelShared.OnMonitorUpdate(m)
}

// Update monitor
// @Summary Update monitor
// @Security BearerAuth
// @Schemes
// @Description Update monitor
// @Tags auth required
// @Accept json
// @param id path uint true "Monitor ID"
// @param request body model.MonitorForm true "Monitor Request"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /monitor/{id} [patch]
func updateMonitor(c *gin.Context) (any, error) {
	strID := c.Param("id")
	id, err := strconv.ParseUint(strID, 10, 64)
	if err != nil {
		return nil, err
	}
	var mf model.MonitorForm
	if err := c.ShouldBindJSON(&mf); err != nil {
		return nil, err
	}
	var m model.Monitor
	if err := singleton.DB.First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("monitor id %d does not exist", id)
	}
	m.Name = mf.Name
	m.Target = strings.TrimSpace(mf.Target)
	m.Type = mf.Type
	m.SkipServers = mf.SkipServers
	m.Cover = mf.Cover
	m.Notify = mf.Notify
	m.NotificationGroupID = mf.NotificationGroupID
	m.Duration = mf.Duration
	m.LatencyNotify = mf.LatencyNotify
	m.MinLatency = mf.MinLatency
	m.MaxLatency = mf.MaxLatency
	m.EnableShowInService = mf.EnableShowInService
	m.EnableTriggerTask = mf.EnableTriggerTask
	m.RecoverTriggerTasks = mf.RecoverTriggerTasks
	m.FailTriggerTasks = mf.FailTriggerTasks

	if err := singleton.DB.Save(&m).Error; err != nil {
		return nil, err
	}

	var skipServers []uint64
	for k := range m.SkipServers {
		skipServers = append(skipServers, k)
	}

	if m.Cover == 0 {
		err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id in (?)", m.ID, skipServers).Error
	} else {
		err = singleton.DB.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id = ? and server_id not in (?)", m.ID, skipServers).Error
	}
	if err != nil {
		return nil, err
	}

	return nil, singleton.ServiceSentinelShared.OnMonitorUpdate(m)
}

// Batch delete monitor
// @Summary Batch delete monitor
// @Security BearerAuth
// @Schemes
// @Description Batch delete monitor
// @Tags auth required
// @Accept json
// @param request body []uint true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/monitor [post]
func batchDeleteMonitor(c *gin.Context) (any, error) {
	var ids []uint64
	if err := c.ShouldBindJSON(&ids); err != nil {
		return nil, err
	}
	err := singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&model.Monitor{}, "id in (?)", ids).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&model.MonitorHistory{}, "monitor_id in (?)", ids).Error
	})
	if err != nil {
		return nil, err
	}
	singleton.ServiceSentinelShared.OnMonitorDelete(ids)
	return nil, nil
}

package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/service/singleton"
)

// List server
// @Summary List server
// @Security BearerAuth
// @Schemes
// @Description List server
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.Server]
// @Router /server [get]
func listServer(c *gin.Context) ([]*model.Server, error) {
	singleton.SortedServerLock.RLock()
	defer singleton.SortedServerLock.RUnlock()

	var ssl []*model.Server
	if err := copier.Copy(&ssl, &singleton.SortedServerList); err != nil {
		return nil, err
	}
	return ssl, nil
}

// Edit server
// @Summary Edit server
// @Security BearerAuth
// @Schemes
// @Description Edit server
// @Tags auth required
// @Accept json
// @Param id path uint true "Server ID"
// @Param body body model.ServerForm true "ServerForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /server/{id} [patch]
func updateServer(c *gin.Context) (any, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}
	var sf model.ServerForm
	if err := c.ShouldBindJSON(&sf); err != nil {
		return nil, err
	}

	var s model.Server
	if err := singleton.DB.First(&s, id).Error; err != nil {
		return nil, singleton.Localizer.ErrorT("server id %d does not exist", id)
	}

	s.Name = sf.Name
	s.DisplayIndex = sf.DisplayIndex
	s.Note = sf.Note
	s.PublicNote = sf.PublicNote
	s.HideForGuest = sf.HideForGuest
	s.EnableDDNS = sf.EnableDDNS
	s.DDNSProfiles = sf.DDNSProfiles
	ddnsProfilesRaw, err := utils.Json.Marshal(s.DDNSProfiles)
	if err != nil {
		return nil, err
	}
	s.DDNSProfilesRaw = string(ddnsProfilesRaw)

	if err := singleton.DB.Save(&s).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.ServerLock.Lock()
	s.CopyFromRunningServer(singleton.ServerList[s.ID])
	singleton.ServerList[s.ID] = &s
	singleton.ServerLock.Unlock()
	singleton.ReSortServer()

	return nil, nil
}

// Batch delete server
// @Summary Batch delete server
// @Security BearerAuth
// @Schemes
// @Description Batch delete server
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/server [post]
func batchDeleteServer(c *gin.Context) (any, error) {
	var servers []uint64
	if err := c.ShouldBindJSON(&servers); err != nil {
		return nil, err
	}

	if err := singleton.DB.Unscoped().Delete(&model.Server{}, "id in (?)", servers).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.ServerLock.Lock()
	for i := 0; i < len(servers); i++ {
		id := servers[i]
		delete(singleton.ServerList, id)

		singleton.AlertsLock.Lock()
		for i := 0; i < len(singleton.Alerts); i++ {
			if singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID] != nil {
				delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].ServerName, id)
				delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].Transfer, id)
				delete(singleton.AlertsCycleTransferStatsStore[singleton.Alerts[i].ID].NextUpdate, id)
			}
		}
		singleton.AlertsLock.Unlock()

		singleton.DB.Unscoped().Delete(&model.Transfer{}, "server_id = ?", id)
	}
	singleton.ServerLock.Unlock()

	singleton.ReSortServer()

	return nil, nil
}

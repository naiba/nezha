package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/service/singleton"
)

// Edit server
// @Summary Edit server
// @Security BearerAuth
// @Schemes
// @Description Edit server
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /server/{id} [patch]
func editServer(c *gin.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}
	var sf model.EditServer
	var s model.Server
	if err := c.ShouldBindJSON(&sf); err != nil {
		return err
	}
	s.Name = sf.Name
	s.DisplayIndex = sf.DisplayIndex
	s.ID = id
	s.Note = sf.Note
	s.PublicNote = sf.PublicNote
	s.HideForGuest = sf.HideForGuest
	s.EnableDDNS = sf.EnableDDNS
	s.DDNSProfiles = sf.DDNSProfiles
	ddnsProfilesRaw, err := utils.Json.Marshal(s.DDNSProfiles)
	if err != nil {
		return err
	}
	s.DDNSProfilesRaw = string(ddnsProfilesRaw)

	if err := singleton.DB.Save(&s).Error; err != nil {
		return newGormError("%v", err)
	}

	singleton.ServerLock.Lock()
	s.CopyFromRunningServer(singleton.ServerList[s.ID])
	singleton.ServerList[s.ID] = &s
	singleton.ServerLock.Unlock()
	singleton.ReSortServer()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
	return nil
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
func batchDeleteServer(c *gin.Context) error {
	var servers []uint64
	if err := c.ShouldBindJSON(&servers); err != nil {
		return err
	}

	if err := singleton.DB.Unscoped().Delete(&model.Server{}, "id in (?)", servers).Error; err != nil {
		return newGormError("%v", err)
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

	c.JSON(http.StatusOK, model.CommonResponse[interface{}]{
		Success: true,
	})
	return nil
}

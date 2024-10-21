package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

// List server group
// @Summary List server group
// @Schemes
// @Description List server group
// @Security BearerAuth
// @Tags common
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.ServerGroupResponseItem]
// @Router /server-group [get]
func listServerGroup(c *gin.Context) error {
	var sg []model.ServerGroup
	if err := singleton.DB.Find(&sg).Error; err != nil {
		return err
	}

	groupServers := make(map[uint64][]uint64, 0)
	var sgs []model.ServerGroupServer
	if err := singleton.DB.Find(&sgs).Error; err != nil {
		return err
	}
	for _, s := range sgs {
		if _, ok := groupServers[s.ServerGroupId]; !ok {
			groupServers[s.ServerGroupId] = make([]uint64, 0)
		}
		groupServers[s.ServerGroupId] = append(groupServers[s.ServerGroupId], s.ServerId)
	}

	var sgRes []model.ServerGroupResponseItem
	for _, s := range sg {
		sgRes = append(sgRes, model.ServerGroupResponseItem{
			Group:   s,
			Servers: groupServers[s.ID],
		})
	}

	c.JSON(http.StatusOK, model.CommonResponse[[]model.ServerGroupResponseItem]{
		Success: true,
		Data:    sgRes,
	})
	return nil
}

// New server group
// @Summary New server group
// @Schemes
// @Description New server group
// @Security BearerAuth
// @Tags auth required
// @Accept json
// @Param body body model.ServerGroupForm true "ServerGroupForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /server-group [post]
func newServerGroup(c *gin.Context) error {
	var sgf model.ServerGroupForm
	if err := c.ShouldBindJSON(&sgf); err != nil {
		return err
	}

	var sg model.ServerGroup
	sg.Name = sgf.Name

	var count int64
	if err := singleton.DB.Model(&model.Server{}).Where("id = ?", sgf.Servers).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(sgf.Servers)) {
		return fmt.Errorf("have invalid server id")
	}

	singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&sg).Error; err != nil {
			return err
		}
		for _, s := range sgf.Servers {
			if err := tx.Create(&model.ServerGroupServer{
				ServerGroupId: sg.ID,
				ServerId:      s,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

// Edit server group
// @Summary Edit server group
// @Schemes
// @Description Edit server group
// @Security BearerAuth
// @Tags auth required
// @Accept json
// @Param id path string true "ID"
// @Param body body model.ServerGroupForm true "ServerGroupForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /server-group/{id} [put]
func editServerGroup(c *gin.Context) error {
	id := c.Param("id")
	var sg model.ServerGroupForm
	if err := c.ShouldBindJSON(&sg); err != nil {
		return err
	}
	var sgDB model.ServerGroup
	if err := singleton.DB.First(&sgDB, id).Error; err != nil {
		return newGormError("%v", err)
	}
	sgDB.Name = sg.Name

	var count int64
	if err := singleton.DB.Model(&model.Server{}).Where("id = ?", sg.Servers).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(sg.Servers)) {
		return fmt.Errorf("have invalid server id")
	}

	err := singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&sgDB).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.ServerGroupServer{}, "server_group_id = ?", id).Error; err != nil {
			return err
		}

		for _, s := range sg.Servers {
			if err := tx.Create(&model.ServerGroupServer{
				ServerGroupId: sgDB.ID,
				ServerId:      s,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return newGormError("%v", err)
	}

	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

// Batch delete server group
// @Summary Batch delete server group
// @Security BearerAuth
// @Schemes
// @Description Batch delete server group
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/server-group [post]
func batchDeleteServerGroup(c *gin.Context) error {
	var sgs []uint64
	if err := c.ShouldBindJSON(&sgs); err != nil {
		return err
	}

	err := singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&model.ServerGroup{}, "id in (?)", sgs).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&model.ServerGroupServer{}, "server_group_id in (?)", sgs).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return newGormError("%v", err)
	}

	c.JSON(http.StatusOK, model.CommonResponse[any]{
		Success: true,
	})
	return nil
}

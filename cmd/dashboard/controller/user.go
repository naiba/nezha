package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"golang.org/x/crypto/bcrypt"
)

// List user
// @Summary List user
// @Security BearerAuth
// @Schemes
// @Description List user
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.User]
// @Router /user [get]
func listUser(c *gin.Context) ([]model.User, error) {
	var users []model.User
	if err := singleton.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create user
// @Summary Create user
// @Security BearerAuth
// @Schemes
// @Description Create user
// @Tags auth required
// @Accept json
// @param request body model.UserForm true "User Request"
// @Produce json
// @Success 200 {object} model.CommonResponse[uint64]
// @Router /user [post]
func createUser(c *gin.Context) (uint64, error) {
	var uf model.UserForm
	if err := c.ShouldBindJSON(&uf); err != nil {
		return 0, err
	}

	if len(uf.Password) < 6 {
		return 0, errors.New("password length must be greater than 6")
	}
	if uf.Username == "" {
		return 0, errors.New("username can't be empty")
	}

	var u model.User
	u.Username = uf.Username

	hash, err := bcrypt.GenerateFromPassword([]byte(uf.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u.Password = string(hash)

	if err := singleton.DB.Create(&u).Error; err != nil {
		return 0, err
	}

	return u.ID, nil
}

// Batch delete users
// @Summary Batch delete users
// @Security BearerAuth
// @Schemes
// @Description Batch delete users
// @Tags auth required
// @Accept json
// @param request body []uint true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/user [post]
func batchDeleteUser(c *gin.Context) (any, error) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		return nil, err
	}
	return nil, singleton.DB.Where("id IN (?)", ids).Delete(&model.User{}).Error
}

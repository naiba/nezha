package general

import (
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

type UserInfo struct {
	Sub      string   `json:"sub"`
	Username string   `json:"preferred_username"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Groups   []string `json:"groups,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

func (u UserInfo) MapToNezhaUser(loginClaim string, groupClaim string, adminGroups []string, autoCreate bool) model.User {
	var user model.User
	var login string
	var groups []string
	var isAdmin bool
	if loginClaim == "email" {
		login = u.Email
	} else if loginClaim == "preferred_username" {
		login = u.Username
	} else {
		login = u.Sub
	}
	if groupClaim == "roles" {
		groups = u.Roles
	} else {
		groups = u.Groups
	}
	// Check if user is admin
	adminGroupSet := make(map[string]struct{}, len(adminGroups))
	for _, adminGroup := range adminGroups {
		adminGroupSet[adminGroup] = struct{}{}
	}
	for _, group := range groups {
		if _, found := adminGroupSet[group]; found {
			isAdmin = true
			break
		}
	}
	result := singleton.DB.Where("login = ?", login).First(&user)
	user.Login = login
	user.Email = u.Email
	user.Name = u.Name
	user.SuperAdmin = isAdmin
	if result.Error != nil && autoCreate {
		singleton.DB.Create(&user)
	} else if result.Error != nil {
		return model.User{}
	}
	return user
}

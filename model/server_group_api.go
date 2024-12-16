package model

import "github.com/gin-gonic/gin"

type ServerGroupForm struct {
	Name    string   `json:"name" minLength:"1"`
	Servers []uint64 `json:"servers"`
}

type ServerGroupResponseItem struct {
	Group   ServerGroup `json:"group"`
	Servers []uint64    `json:"servers"`
}

func (sg *ServerGroupResponseItem) HasPermission(c *gin.Context) bool {
	return sg.Group.HasPermission(c)
}

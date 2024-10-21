package model

type ServerGroupForm struct {
	Name    string   `json:"name"`
	Servers []uint64 `json:"servers"`
}

type ServerGroupResponseItem struct {
	Group   ServerGroup `json:"group"`
	Servers []uint64    `json:"servers"`
}

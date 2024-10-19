package model

type ServerGroupServer struct {
	Common
	ServerGroupId uint64 `json:"server_group_id"`
	ServerId      uint64 `json:"server_id"`
}

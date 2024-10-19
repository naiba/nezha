package model

type UserGroupUser struct {
	Common
	UserGroupId uint64 `json:"user_group_id"`
	UserId      uint64 `json:"user_id"`
}

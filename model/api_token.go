package model

import "time"

type ApiToken struct {
	Common
	UserId       uint64    `json:"user_id"`
	Token        string    `json:"token"`
	TokenExpired time.Time `json:"token_expired"`
}

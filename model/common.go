package model

import (
	"time"
)

const CtxKeyAuthorizedUser = "ckau"

const CacheKeyOauth2State = "p:a:state"

type Common struct {
	ID        uint64    `gorm:"primaryKey"`
	CreatedAt time.Time `sql:"index"`
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Response struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

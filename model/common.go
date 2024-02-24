package model

import (
	"time"

	"gorm.io/gorm"
)

const CtxKeyAuthorizedUser = "ckau"
const CtxKeyViewPasswordVerified = "ckvpv"
const CtxKeyPreferredTheme = "ckpt"
const CacheKeyOauth2State = "p:a:state"

type Common struct {
	ID        uint64         `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"index;<-:create"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Response struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

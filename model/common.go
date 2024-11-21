package model

import (
	"time"
)

const (
	CtxKeyAuthorizedUser = "ckau"
	CtxKeyRealIPStr      = "ckri"
)

type CtxKeyRealIP struct{}

type Common struct {
	ID        uint64    `gorm:"primaryKey" json:"id,omitempty"`
	CreatedAt time.Time `gorm:"index;<-:create" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	// Do not use soft deletion
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Response struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

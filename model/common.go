package model

import "time"

// CtxKeyAuthorizedUser ..
const CtxKeyAuthorizedUser = "ckau"

// CtxKeyOauth2State ..
const CtxKeyOauth2State = "cko2s"

// Common ..
type Common struct {
	ID        uint64 `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// Response ..
type Response struct {
	Code    uint64      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

package model

import "time"

const CtxKeyAuthorizedUser = "ckau"

const CacheKeyOauth2State = "p:a:state"
const CacheKeyServicePage = "p:c:service"

type Common struct {
	ID        uint64 `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Response struct {
	Code    uint64      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

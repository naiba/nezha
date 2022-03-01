package model

import "time"

const CtxKeyAuthorizedUser = "ckau"

const CacheKeyOauth2State = "p:a:state"

var Loc *time.Location

func init() {
	var err error
	Loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
}

type Common struct {
	ID        uint64    `gorm:"primary_key"`
	CreatedAt time.Time `sql:"index"`
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Response struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

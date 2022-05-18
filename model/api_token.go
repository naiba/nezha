package model

type ApiToken struct {
	Common
	UserID uint64 `json:"user_id"`
	Token  string `json:"token"`
	Note   string `json:"note"`
}

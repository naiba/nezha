package model

type User struct {
	Common
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty" gorm:"type:char(72)"`
}

package model

const (
	RoleAdmin uint8 = iota
	RoleMember
)

type User struct {
	Common
	Username string `json:"username,omitempty" gorm:"uniqueIndex"`
	Password string `json:"password,omitempty" gorm:"type:char(72)"`
	Role     uint8  `json:"role,omitempty"`
}

type Profile struct {
	User
	LoginIP string `json:"login_ip,omitempty"`
}

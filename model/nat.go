package model

type NAT struct {
	Common
	Name     string
	ServerID uint64
	Host     string
	Domain   string `gorm:"unique"`
}

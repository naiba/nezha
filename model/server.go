package model

// Server ..
type Server struct {
	Common
	Name   string
	Secret string

	Host  Host
	State State
}

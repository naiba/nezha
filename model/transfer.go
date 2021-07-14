package model

type Transfer struct {
	Common
	ServerID uint64
	In       uint64
	Out      uint64
}

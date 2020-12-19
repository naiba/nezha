package model

type Rule struct {
	Common
	Name     string
	Type     string // 指标类型，cpu、memory、swap、disk、net_in、net_out、net_all、transfer_in、transfer_out、transfer_all、offline
	Min      uint64 // 最小阈值
	Max      uint64 // 最大阈值
	Duration uint64 // 持续时间
}

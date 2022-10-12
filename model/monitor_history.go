package model

// MonitorHistory 历史监控记录
type MonitorHistory struct {
	Common
	MonitorID uint64
	AvgDelay  float32 // 平均延迟，毫秒
	Up        uint64  // 检查状态良好计数
	Down      uint64  // 检查状态异常计数
	Data      string
}

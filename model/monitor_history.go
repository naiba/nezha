package model

import (
	"time"

	"gorm.io/gorm"
)

// MonitorHistory 历史监控记录
type MonitorHistory struct {
	ID        uint64         `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"index;<-:create;index:idx_server_id_created_at_monitor_id_avg_delay"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	MonitorID uint64         `gorm:"index:idx_server_id_created_at_monitor_id_avg_delay"`
	ServerID  uint64         `gorm:"index:idx_server_id_created_at_monitor_id_avg_delay"`
	AvgDelay  float32        `gorm:"index:idx_server_id_created_at_monitor_id_avg_delay"` // 平均延迟，毫秒
	Up        uint64         // 检查状态良好计数
	Down      uint64         // 检查状态异常计数
	Data      string
}

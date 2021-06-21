package model

import "time"

const (
	RuleCoverAll = iota
	RuleCoverIgnoreAll
)

type Rule struct {
	// 指标类型，cpu、memory、swap、disk、net_in_speed、net_out_speed
	// net_all_speed、transfer_in、transfer_out、transfer_all、offline
	Type     string          `json:"type,omitempty"`
	Min      uint64          `json:"min,omitempty"`      // 最小阈值 (百分比、字节 kb ÷ 1024)
	Max      uint64          `json:"max,omitempty"`      // 最大阈值 (百分比、字节 kb ÷ 1024)
	Duration uint64          `json:"duration,omitempty"` // 持续时间 (秒)
	Cover    uint64          `json:"cover,omitempty"`    // 覆盖范围 RuleCoverAll/IgnoreAll
	Ignore   map[uint64]bool `json:"ignore,omitempty"`   // 覆盖范围的排除
}

func percentage(used, total uint64) uint64 {
	if total == 0 {
		return 0
	}
	return used * 100 / total
}

// Snapshot 未通过规则返回 struct{}{}, 通过返回 nil
func (u *Rule) Snapshot(server *Server) interface{} {
	// 监控全部但是排除了此服务器
	if u.Cover == RuleCoverAll && u.Ignore[server.ID] {
		return nil
	}
	// 忽略全部但是指定监控了此服务器
	if u.Cover == RuleCoverIgnoreAll && !u.Ignore[server.ID] {
		return nil
	}

	var src uint64

	switch u.Type {
	case "cpu":
		src = uint64(server.State.CPU)
	case "memory":
		src = percentage(server.State.MemUsed, server.Host.MemTotal)
	case "swap":
		src = percentage(server.State.SwapUsed, server.Host.SwapTotal)
	case "disk":
		src = percentage(server.State.DiskUsed, server.Host.DiskTotal)
	case "net_in_speed":
		src = server.State.NetInSpeed
	case "net_out_speed":
		src = server.State.NetOutSpeed
	case "net_all_speed":
		src = server.State.NetOutSpeed + server.State.NetOutSpeed
	case "transfer_in":
		src = server.State.NetInTransfer
	case "transfer_out":
		src = server.State.NetOutTransfer
	case "transfer_all":
		src = server.State.NetOutTransfer + server.State.NetInTransfer
	case "offline":
		if server.LastActive.IsZero() {
			src = 0
		} else {
			src = uint64(server.LastActive.Unix())
		}
	}

	if u.Type == "offline" && uint64(time.Now().Unix())-src > 6 {
		return struct{}{}
	} else if (u.Max > 0 && src > u.Max) || (u.Min > 0 && src < u.Min) {
		return struct{}{}
	}

	return nil
}

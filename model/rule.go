package model

import (
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/naiba/nezha/pkg/utils"
)

const (
	RuleCoverAll = iota
	RuleCoverIgnoreAll
)

type NResult struct {
	N uint64
}

type Rule struct {
	// 指标类型，cpu、memory、swap、disk、net_in_speed、net_out_speed
	// net_all_speed、transfer_in、transfer_out、transfer_all、offline
	// transfer_in_cycle、transfer_out_cycle、transfer_all_cycle
	Type          string          `json:"type,omitempty"`
	Min           float64         `json:"min,omitempty"`            // 最小阈值 (百分比、字节 kb ÷ 1024)
	Max           float64         `json:"max,omitempty"`            // 最大阈值 (百分比、字节 kb ÷ 1024)
	CycleStart    *time.Time      `json:"cycle_start,omitempty"`    // 流量统计的开始时间
	CycleInterval uint64          `json:"cycle_interval,omitempty"` // 流量统计周期
	CycleUnit     string          `json:"cycle_unit,omitempty"`     // 流量统计周期单位，默认hour,可选(hour, day, week, month, year)
	Duration      uint64          `json:"duration,omitempty"`       // 持续时间 (秒)
	Cover         uint64          `json:"cover,omitempty"`          // 覆盖范围 RuleCoverAll/IgnoreAll
	Ignore        map[uint64]bool `json:"ignore,omitempty"`         // 覆盖范围的排除

	// 只作为缓存使用，记录下次该检测的时间
	NextTransferAt  map[uint64]time.Time   `json:"-"`
	LastCycleStatus map[uint64]interface{} `json:"-"`
}

func percentage(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) * 100 / float64(total)
}

// Snapshot 未通过规则返回 struct{}{}, 通过返回 nil
func (u *Rule) Snapshot(cycleTransferStats *CycleTransferStats, server *Server, db *gorm.DB) interface{} {
	// 监控全部但是排除了此服务器
	if u.Cover == RuleCoverAll && u.Ignore[server.ID] {
		return nil
	}
	// 忽略全部但是指定监控了此服务器
	if u.Cover == RuleCoverIgnoreAll && !u.Ignore[server.ID] {
		return nil
	}

	// 循环区间流量检测 · 短期无需重复检测
	if u.IsTransferDurationRule() && u.NextTransferAt[server.ID].After(time.Now()) {
		return u.LastCycleStatus[server.ID]
	}

	var src float64

	switch u.Type {
	case "cpu":
		src = float64(server.State.CPU)
	case "gpu":
		src = float64(server.State.GPU)
	case "memory":
		src = percentage(server.State.MemUsed, server.Host.MemTotal)
	case "swap":
		src = percentage(server.State.SwapUsed, server.Host.SwapTotal)
	case "disk":
		src = percentage(server.State.DiskUsed, server.Host.DiskTotal)
	case "net_in_speed":
		src = float64(server.State.NetInSpeed)
	case "net_out_speed":
		src = float64(server.State.NetOutSpeed)
	case "net_all_speed":
		src = float64(server.State.NetOutSpeed + server.State.NetOutSpeed)
	case "transfer_in":
		src = float64(server.State.NetInTransfer)
	case "transfer_out":
		src = float64(server.State.NetOutTransfer)
	case "transfer_all":
		src = float64(server.State.NetOutTransfer + server.State.NetInTransfer)
	case "offline":
		if server.LastActive.IsZero() {
			src = 0
		} else {
			src = float64(server.LastActive.Unix())
		}
	case "transfer_in_cycle":
		src = float64(utils.Uint64SubInt64(server.State.NetInTransfer, server.PrevTransferInSnapshot))
		if u.CycleInterval != 0 {
			var res NResult
			db.Model(&Transfer{}).Select("SUM(`in`) AS n").Where("datetime(`created_at`) >= datetime(?) AND server_id = ?", u.GetTransferDurationStart().UTC(), server.ID).Scan(&res)
			src += float64(res.N)
		}
	case "transfer_out_cycle":
		src = float64(utils.Uint64SubInt64(server.State.NetOutTransfer, server.PrevTransferOutSnapshot))
		if u.CycleInterval != 0 {
			var res NResult
			db.Model(&Transfer{}).Select("SUM(`out`) AS n").Where("datetime(`created_at`) >= datetime(?) AND server_id = ?", u.GetTransferDurationStart().UTC(), server.ID).Scan(&res)
			src += float64(res.N)
		}
	case "transfer_all_cycle":
		src = float64(utils.Uint64SubInt64(server.State.NetOutTransfer, server.PrevTransferOutSnapshot) + utils.Uint64SubInt64(server.State.NetInTransfer, server.PrevTransferInSnapshot))
		if u.CycleInterval != 0 {
			var res NResult
			db.Model(&Transfer{}).Select("SUM(`in`+`out`) AS n").Where("datetime(`created_at`) >= datetime(?) AND server_id = ?", u.GetTransferDurationStart().UTC(), server.ID).Scan(&res)
			src += float64(res.N)
		}
	case "load1":
		src = server.State.Load1
	case "load5":
		src = server.State.Load5
	case "load15":
		src = server.State.Load15
	case "tcp_conn_count":
		src = float64(server.State.TcpConnCount)
	case "udp_conn_count":
		src = float64(server.State.UdpConnCount)
	case "process_count":
		src = float64(server.State.ProcessCount)
	case "temperature_max":
		var temp []float64
		if server.State.Temperatures != nil {
			for _, tempStat := range server.State.Temperatures {
				if tempStat.Temperature != 0 {
					temp = append(temp, tempStat.Temperature)
				}
			}
			src = slices.Max(temp)
		}
	}

	// 循环区间流量检测 · 更新下次需要检测时间
	if u.IsTransferDurationRule() {
		seconds := 1800 * ((u.Max - src) / u.Max)
		if seconds < 180 {
			seconds = 180
		}
		if u.NextTransferAt == nil {
			u.NextTransferAt = make(map[uint64]time.Time)
		}
		if u.LastCycleStatus == nil {
			u.LastCycleStatus = make(map[uint64]interface{})
		}
		u.NextTransferAt[server.ID] = time.Now().Add(time.Second * time.Duration(seconds))
		if (u.Max > 0 && src > u.Max) || (u.Min > 0 && src < u.Min) {
			u.LastCycleStatus[server.ID] = struct{}{}
		} else {
			u.LastCycleStatus[server.ID] = nil
		}
		if cycleTransferStats.ServerName[server.ID] != server.Name {
			cycleTransferStats.ServerName[server.ID] = server.Name
		}
		cycleTransferStats.Transfer[server.ID] = uint64(src)
		cycleTransferStats.NextUpdate[server.ID] = u.NextTransferAt[server.ID]
		// 自动更新周期流量展示起止时间
		cycleTransferStats.From = u.GetTransferDurationStart()
		cycleTransferStats.To = u.GetTransferDurationEnd()
	}

	if u.Type == "offline" && float64(time.Now().Unix())-src > 6 {
		return struct{}{}
	} else if (u.Max > 0 && src > u.Max) || (u.Min > 0 && src < u.Min) {
		return struct{}{}
	}

	return nil
}

// IsTransferDurationRule 判断该规则是否属于周期流量规则 属于则返回true
func (rule Rule) IsTransferDurationRule() bool {
	return strings.HasSuffix(rule.Type, "_cycle")
}

// GetTransferDurationStart 获取周期流量的起始时间
func (rule Rule) GetTransferDurationStart() time.Time {
	// Accept uppercase and lowercase
	unit := strings.ToLower(rule.CycleUnit)
	startTime := *rule.CycleStart
	var nextTime time.Time
	switch unit {
	case "year":
		nextTime = startTime.AddDate(int(rule.CycleInterval), 0, 0)
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(int(rule.CycleInterval), 0, 0)
		}
	case "month":
		nextTime = startTime.AddDate(0, int(rule.CycleInterval), 0)
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, int(rule.CycleInterval), 0)
		}
	case "week":
		nextTime = startTime.AddDate(0, 0, 7*int(rule.CycleInterval))
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, 0, 7*int(rule.CycleInterval))
		}
	case "day":
		nextTime = startTime.AddDate(0, 0, int(rule.CycleInterval))
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, 0, int(rule.CycleInterval))
		}
	default:
		// For hour unit or not set.
		interval := 3600 * int64(rule.CycleInterval)
		startTime = time.Unix(rule.CycleStart.Unix()+(time.Now().Unix()-rule.CycleStart.Unix())/interval*interval, 0)
	}

	return startTime
}

// GetTransferDurationEnd 获取周期流量结束时间
func (rule Rule) GetTransferDurationEnd() time.Time {
	// Accept uppercase and lowercase
	unit := strings.ToLower(rule.CycleUnit)
	startTime := *rule.CycleStart
	var nextTime time.Time
	switch unit {
	case "year":
		nextTime = startTime.AddDate(int(rule.CycleInterval), 0, 0)
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(int(rule.CycleInterval), 0, 0)
		}
	case "month":
		nextTime = startTime.AddDate(0, int(rule.CycleInterval), 0)
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, int(rule.CycleInterval), 0)
		}
	case "week":
		nextTime = startTime.AddDate(0, 0, 7*int(rule.CycleInterval))
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, 0, 7*int(rule.CycleInterval))
		}
	case "day":
		nextTime = startTime.AddDate(0, 0, int(rule.CycleInterval))
		for time.Now().After(nextTime) {
			startTime = nextTime
			nextTime = nextTime.AddDate(0, 0, int(rule.CycleInterval))
		}
	default:
		// For hour unit or not set.
		interval := 3600 * int64(rule.CycleInterval)
		startTime = time.Unix(rule.CycleStart.Unix()+(time.Now().Unix()-rule.CycleStart.Unix())/interval*interval, 0)
		nextTime = time.Unix(startTime.Unix()+interval, 0)
	}

	return nextTime
}

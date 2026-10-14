package model

import (
	"time"
)

// 在线统计窗口的取值边界（天）。后台管理端可设置统计范围，查询窗口不允许
// 超过该设置，历史数据按该设置做保留清理。
const (
	DefaultOnlineStatsDays = 7
	MinOnlineStatsDays     = 7
	MaxOnlineStatsDays     = 3650
)

// OnlineActiveWindow 服务器状态上报距该窗口内视为在线。与 MCP server.list
// 的 onlineWindow 口径保持一致；超过该窗口未上报的进行中会话由定时清扫关闭。
const OnlineActiveWindow = 30 * time.Second

// OnlineSession 记录一台服务器的一段连续在线时间。Ongoing=true 表示会话
// 仍在进行（此时 EndedAt 为零值）；会话结束时写入 EndedAt 并置 Ongoing=false。
// LastActiveAt 由上报侧节流持久化，用于 dashboard 异常停机后把进行中的会话
// 结转到最后一次已知活跃时刻。
type OnlineSession struct {
	ID        uint64    `gorm:"primaryKey" json:"id,omitempty"`
	CreatedAt time.Time `gorm:"index;<-:create" json:"created_at,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
	ServerID  uint64    `gorm:"index:idx_online_sessions_server_started" json:"server_id,omitempty"`
	StartedAt time.Time `gorm:"index:idx_online_sessions_server_started" json:"started_at,omitempty"`
	EndedAt   time.Time `gorm:"index" json:"ended_at,omitempty"` // 会话结束时刻，进行中为零值
	// 上报侧节流持久化的最后活跃时刻（≥OnlineActiveWindow 间隔写一次）
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
	Ongoing      bool      `json:"ongoing"`
}

// OnlinePeriod 是裁剪到查询窗口后的在线时间段，供前端时间轴直接渲染。
type OnlinePeriod struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Ongoing bool      `json:"ongoing"`
}

// DailyOnlineStat 是单个自然日（面板时区）在查询窗口内的在线秒数。
// 窗口首尾为非整日时，只统计与窗口重叠的部分。
type DailyOnlineStat struct {
	Date          string `json:"date"`
	OnlineSeconds uint64 `json:"online_seconds"`
}

// OnlineHistoryResponse 是单台服务器在线时间段统计的聚合结果。
type OnlineHistoryResponse struct {
	ServerID           uint64    `json:"server_id"`
	ServerName         string    `json:"server_name"`
	Days               int       `json:"days"`
	MaxDays            int       `json:"max_days"`
	WindowStart        time.Time `json:"window_start"`
	WindowEnd          time.Time `json:"window_end"`
	TotalOnlineSeconds uint64    `json:"total_online_seconds"`
	OnlineRate         float64   `json:"online_rate"`
	CurrentlyOnline    bool      `json:"currently_online"`
	// 从未上报过的服务器该值为 nil；指针避免把 Go 零值时间序列化成 0001-01-01
	LastActive *time.Time        `json:"last_active,omitempty"`
	Periods    []OnlinePeriod    `json:"periods"`
	Daily      []DailyOnlineStat `json:"daily"`
}

// ClipOnlineSessions 把与窗口 [windowStart, windowEnd) 相交的会话裁剪成
// OnlinePeriod 列表。进行中的会话以 windowEnd 为右边界（保留 Ongoing 标记）。
// 输入需按 StartedAt 升序；输出与输入同序。
func ClipOnlineSessions(sessions []OnlineSession, windowStart, windowEnd time.Time) []OnlinePeriod {
	periods := make([]OnlinePeriod, 0, len(sessions))
	for _, s := range sessions {
		start := s.StartedAt
		if start.Before(windowStart) {
			start = windowStart
		}
		end := s.EndedAt
		ongoing := s.Ongoing || end.IsZero()
		if ongoing || end.After(windowEnd) {
			end = windowEnd
		}
		if !start.Before(end) {
			// 与窗口零交集（理论上不会由查询条件选出）或空会话
			continue
		}
		periods = append(periods, OnlinePeriod{Start: start, End: end, Ongoing: ongoing})
	}
	return periods
}

// SumOnlinePeriods 返回在线时间段的总时长（秒），向下取整。
func SumOnlinePeriods(periods []OnlinePeriod) uint64 {
	var total time.Duration
	for _, p := range periods {
		total += p.End.Sub(p.Start)
	}
	return uint64(total.Seconds())
}

// DailyOnlineBreakdown 把在线时间段按面板时区的自然日拆分，返回窗口覆盖的
// 每一天的在线秒数（含在线为 0 秒的日期），按日期升序。
func DailyOnlineBreakdown(periods []OnlinePeriod, windowStart, windowEnd time.Time, loc *time.Location) []DailyOnlineStat {
	first := windowStart.In(loc)
	baseDay := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
	last := windowEnd.In(loc)

	var days []DailyOnlineStat
	for d := baseDay; d.Before(last); d = d.AddDate(0, 0, 1) {
		days = append(days, DailyOnlineStat{Date: d.Format("2006-01-02")})
	}
	if len(days) == 0 {
		return days
	}

	for _, p := range periods {
		// 逐日推进，累加该会话与每一天 [dayStart, nextDay) 的交集
		for i := range days {
			dayStart := dayStartAt(baseDay, i)
			nextDay := dayStartAt(baseDay, i+1)
			overlapStart := maxTime(p.Start, dayStart)
			overlapEnd := minTime(p.End, nextDay)
			if overlapStart.Before(overlapEnd) {
				days[i].OnlineSeconds += uint64(overlapEnd.Sub(overlapStart).Seconds())
			}
		}
	}
	return days
}

func dayStartAt(baseDay time.Time, offset int) time.Time {
	y, m, d := baseDay.AddDate(0, 0, offset).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, baseDay.Location())
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

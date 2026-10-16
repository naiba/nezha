package model

import (
	"gorm.io/gorm"
	"sort"
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
	ServerID  uint64    `gorm:"index:idx_online_sessions_server_started;index:idx_online_sessions_server_closed,priority:1;index:idx_online_sessions_server_open,priority:1" json:"server_id,omitempty"`
	StartedAt time.Time `gorm:"index:idx_online_sessions_server_started;index:idx_online_sessions_server_open,priority:3" json:"started_at,omitempty"`
	EndedAt   time.Time `gorm:"index;index:idx_online_sessions_server_closed,priority:3" json:"ended_at,omitempty"` // 会话结束时刻，进行中为零值
	// 上报侧节流持久化的最后活跃时刻（≥OnlineActiveWindow 间隔写一次）
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
	Ongoing      bool      `gorm:"index:idx_online_sessions_server_closed,priority:2;index:idx_online_sessions_server_open,priority:2" json:"ongoing"`
}

// BeforeSave keeps SQLite's lexicographic timestamp comparisons chronological.
// Query parameters and map-based updates must use UTC as well.
func (s *OnlineSession) BeforeSave(_ *gorm.DB) error {
	s.StartedAt = s.StartedAt.UTC()
	s.EndedAt = s.EndedAt.UTC()
	s.LastActiveAt = s.LastActiveAt.UTC()
	return nil
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
	ServerID   uint64 `json:"server_id"`
	ServerName string `json:"server_name"`
	Days       int    `json:"days"`
	MaxDays    int    `json:"max_days"`
	// Daily 拆日所用面板时区：IANA 名与窗口末端相对 UTC 的偏移秒数，
	// 前端按同一时区渲染，保证跨时区访客的图形与数字一致
	Timezone           string    `json:"timezone"`
	TzOffset           int       `json:"tz_offset"`
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

// DailyOnlineBreakdown 把在线时间段按面板时区的自然日拆分，返回窗口覆盖的
// 每一天的在线秒数（含在线为 0 秒的日期），按日期升序。
// 取整口径：每天先累计原始时长（日界切分不损失精度），仅在每日汇总时向下
// 取整一次；调用方以各日之和作为总时长，保证响应内部 total 与 daily[] 严格
// 对账，不与"逐段截断再累加"的口径混用（亚秒会话会让后者系统性偏小）。
// 时段二分定位起始日界后逐日推进，复杂度 O(P·log D + D)；单服务器的会话
// 互不重叠，内层推进总量受窗口天数约束，长窗口（如 days=3650）下不会出现
// 时段数×天数的双重循环。
func DailyOnlineBreakdown(periods []OnlinePeriod, windowStart, windowEnd time.Time, loc *time.Location) []DailyOnlineStat {
	first := windowStart.In(loc)
	// Advance a timezone-independent calendar, not a normalized local midnight:
	// midnight DST gaps can normalize into the previous day and repeat dates.
	date := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, time.UTC)
	var days []DailyOnlineStat
	var dayStarts, dayEnds []time.Time
	for {
		start := onlineCalendarBoundary(date, loc)
		if !start.Before(windowEnd) {
			break
		}
		nextDate := date.AddDate(0, 0, 1)
		end := onlineCalendarBoundary(nextDate, loc)
		if start.Before(end) {
			dayStarts = append(dayStarts, start)
			dayEnds = append(dayEnds, end)
			days = append(days, DailyOnlineStat{Date: date.Format("2006-01-02")})
		}
		date = nextDate
	}

	if len(days) == 0 {
		return days
	}

	dayDurations := make([]time.Duration, len(days))
	for _, p := range periods {
		// 该时段覆盖的第一天：不晚于时段开始的最后一个日界
		i := sort.Search(len(dayStarts), func(i int) bool {
			return dayStarts[i].After(p.Start)
		}) - 1
		if i < 0 {
			i = 0
		}
		// 逐日推进，累加该时段与每一天 [dayStart, nextDay) 的交集
		for ; i < len(dayStarts) && dayStarts[i].Before(p.End); i++ {
			overlapStart := maxTime(p.Start, dayStarts[i])
			overlapEnd := minTime(p.End, dayEnds[i])
			if overlapStart.Before(overlapEnd) {
				dayDurations[i] += overlapEnd.Sub(overlapStart)
			}
		}
	}
	for i := range days {
		days[i].OnlineSeconds = uint64(dayDurations[i].Seconds())
	}
	return days
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

// A missing midnight (or an entirely skipped date) starts at the timezone
// transition, not at the previous date returned by time.Date normalization.
func onlineCalendarBoundary(date time.Time, loc *time.Location) time.Time {
	boundary := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	if boundary.Format("2006-01-02") < date.Format("2006-01-02") {
		_, end := boundary.ZoneBounds()
		if !end.IsZero() {
			return end
		}
	}
	// time.Date may choose the second midnight in a rollback. Recover the
	// earlier occurrence using the offset on the preceding side of the transition.
	zoneStart, _ := boundary.ZoneBounds()
	if !zoneStart.IsZero() {
		_, currentOffset := boundary.Zone()
		_, previousOffset := zoneStart.Add(-time.Nanosecond).Zone()
		if previousOffset > currentOffset {
			earlier := boundary.Add(-time.Duration(previousOffset-currentOffset) * time.Second)
			if earlier.Format("2006-01-02 15:04:05") == boundary.Format("2006-01-02 15:04:05") {
				boundary = earlier
			}
		}
	}
	return boundary
}

// OnlineHistoryWindowStart defines the shared query and retention calendar window.
func OnlineHistoryWindowStart(now time.Time, days int, loc *time.Location) time.Time {
	// Calendar days follow the dashboard timezone, not the host timezone;
	// DST days can contain 23 or 25 hours. Queries and pruning must agree.
	if loc == nil {
		loc = time.UTC
	}
	return now.In(loc).AddDate(0, 0, -days).UTC()
}

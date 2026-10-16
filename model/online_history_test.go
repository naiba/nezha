package model

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testZone = time.FixedZone("TEST", 8*3600)

func ts(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, testZone)
}

func TestClipOnlineSessions(t *testing.T) {
	windowStart := ts(2026, 10, 1, 0, 0, 0)
	windowEnd := ts(2026, 10, 8, 0, 0, 0)

	t.Run("session fully inside window", func(t *testing.T) {
		sessions := []OnlineSession{
			{ServerID: 1, StartedAt: ts(2026, 10, 2, 8, 0, 0), EndedAt: ts(2026, 10, 2, 9, 30, 0)},
		}
		periods := ClipOnlineSessions(sessions, windowStart, windowEnd)
		require.Len(t, periods, 1)
		require.Equal(t, ts(2026, 10, 2, 8, 0, 0), periods[0].Start)
		require.Equal(t, ts(2026, 10, 2, 9, 30, 0), periods[0].End)
		require.False(t, periods[0].Ongoing)
	})

	t.Run("session started before window is clipped", func(t *testing.T) {
		sessions := []OnlineSession{
			{ServerID: 1, StartedAt: ts(2026, 9, 28, 12, 0, 0), EndedAt: ts(2026, 10, 1, 5, 0, 0)},
		}
		periods := ClipOnlineSessions(sessions, windowStart, windowEnd)
		require.Len(t, periods, 1)
		require.Equal(t, windowStart, periods[0].Start)
		require.Equal(t, ts(2026, 10, 1, 5, 0, 0), periods[0].End)
	})

	t.Run("ongoing session ends at window end", func(t *testing.T) {
		sessions := []OnlineSession{
			{ServerID: 1, StartedAt: ts(2026, 10, 7, 10, 0, 0), Ongoing: true},
		}
		periods := ClipOnlineSessions(sessions, windowStart, windowEnd)
		require.Len(t, periods, 1)
		require.Equal(t, windowEnd, periods[0].End)
		require.True(t, periods[0].Ongoing)
	})

	t.Run("zero ended_at treated as ongoing", func(t *testing.T) {
		sessions := []OnlineSession{
			{ServerID: 1, StartedAt: ts(2026, 10, 7, 10, 0, 0)},
		}
		periods := ClipOnlineSessions(sessions, windowStart, windowEnd)
		require.Len(t, periods, 1)
		require.True(t, periods[0].Ongoing)
		require.Equal(t, windowEnd, periods[0].End)
	})

	t.Run("session outside window yields no period", func(t *testing.T) {
		sessions := []OnlineSession{
			{ServerID: 1, StartedAt: ts(2026, 9, 1, 0, 0, 0), EndedAt: ts(2026, 9, 2, 0, 0, 0)},
		}
		require.Empty(t, ClipOnlineSessions(sessions, windowStart, windowEnd))
	})
}

func TestDailyOnlineBreakdown(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	windowStart := time.Date(2026, 10, 1, 10, 0, 0, 0, loc)
	windowEnd := time.Date(2026, 10, 3, 14, 0, 0, 0, loc)

	periods := []OnlinePeriod{
		// 跨午夜：2 日 23:00 ~ 3 日 01:00
		{Start: time.Date(2026, 10, 2, 23, 0, 0, 0, loc), End: time.Date(2026, 10, 3, 1, 0, 0, 0, loc)},
		// 窗口前就开始的会话，裁剪后从窗口起点开始
		{Start: windowStart, End: time.Date(2026, 10, 1, 12, 0, 0, 0, loc)},
	}

	daily := DailyOnlineBreakdown(periods, windowStart, windowEnd, loc)
	require.Len(t, daily, 3)
	require.Equal(t, "2026-10-01", daily[0].Date)
	require.Equal(t, "2026-10-02", daily[1].Date)
	require.Equal(t, "2026-10-03", daily[2].Date)

	// 1 日：12:00-10:00 = 2h；2 日：23:00-24:00 = 1h；3 日：0:00-1:00 + 窗口外无 = 1h
	require.Equal(t, uint64(7200), daily[0].OnlineSeconds)
	require.Equal(t, uint64(3600), daily[1].OnlineSeconds)
	require.Equal(t, uint64(3600), daily[2].OnlineSeconds)

	// 每日合计即总时长（口径：每日累计原始时长后统一取整）
	var sum uint64
	for _, d := range daily {
		sum += d.OnlineSeconds
	}
	require.Equal(t, uint64(7200+3600+3600), sum)
}

// 评审回归：取整必须按日累计原始时长后统一转换。逐段截断再累加会让亚秒
// 会话系统性偏小——同一天两个 1.5s 的会话应得 3s，而不是 1+1=2s
func TestDailyOnlineBreakdownSubSecond(t *testing.T) {
	loc := testZone
	windowStart := ts(2026, 10, 1, 0, 0, 0)
	windowEnd := ts(2026, 10, 2, 0, 0, 0)
	periods := []OnlinePeriod{
		{Start: windowStart.Add(100 * time.Millisecond), End: windowStart.Add(1600 * time.Millisecond)},
		{Start: windowStart.Add(2 * time.Second), End: windowStart.Add(3500 * time.Millisecond)},
	}
	daily := DailyOnlineBreakdown(periods, windowStart, windowEnd, loc)
	require.Len(t, daily, 1)
	require.Equal(t, uint64(3), daily[0].OnlineSeconds)
}

// 跨日时段按日界切分不损失精度：各日分别累计原始时长后各自取整
func TestDailyOnlineBreakdownCrossDayFraction(t *testing.T) {
	loc := testZone
	windowStart := ts(2026, 10, 1, 0, 0, 0)
	windowEnd := ts(2026, 10, 3, 0, 0, 0)
	periods := []OnlinePeriod{{
		Start: time.Date(2026, 10, 1, 23, 59, 59, 250000000, loc),
		End:   time.Date(2026, 10, 2, 0, 0, 2, 750000000, loc),
	}}
	daily := DailyOnlineBreakdown(periods, windowStart, windowEnd, loc)
	require.Len(t, daily, 2)
	require.Equal(t, uint64(0), daily[0].OnlineSeconds) // 0.75s → 0
	require.Equal(t, uint64(2), daily[1].OnlineSeconds) // 2.75s → 2
}

func TestDailyOnlineBreakdownZeroOnline(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	windowStart := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	windowEnd := time.Date(2026, 10, 4, 0, 0, 0, 0, loc)
	daily := DailyOnlineBreakdown(nil, windowStart, windowEnd, loc)
	require.Len(t, daily, 3)
	for _, d := range daily {
		require.Zero(t, d.OnlineSeconds)
	}
}

func TestOnlineStatsDaysBounds(t *testing.T) {
	require.Equal(t, 7, DefaultOnlineStatsDays)
	require.Equal(t, 7, MinOnlineStatsDays)
	require.Equal(t, 3650, MaxOnlineStatsDays)
}

// F1 回归保护：指针推进实现必须与逐日朴素实现对任意不相交时段集合等价
// （单服务器会话互不重叠的约束一致），长窗口大量时段下不再随时段数平方放大。
func TestDailyOnlineBreakdownEquivalence(t *testing.T) {
	loc := testZone
	windowStart := ts(2026, 1, 1, 0, 0, 0)
	windowEnd := windowStart.AddDate(0, 0, 90)

	rng := rand.New(rand.NewSource(42))
	var periods []OnlinePeriod
	cursor := windowStart
	for cursor.Before(windowEnd) {
		// 起止带亚秒抖动：让逐段截断与按日累计两种口径可区分
		start := cursor.Add(time.Duration(rng.Intn(4*3600))*time.Second + time.Duration(rng.Int63n(int64(time.Second))))
		end := start.Add(time.Duration(1+rng.Intn(6*3600))*time.Second + time.Duration(rng.Int63n(int64(time.Second))))
		if end.After(windowEnd) {
			end = windowEnd
		}
		periods = append(periods, OnlinePeriod{Start: start, End: end})
		cursor = end.Add(time.Duration(1+rng.Intn(3600)) * time.Second)
	}

	days := DailyOnlineBreakdown(periods, windowStart, windowEnd, loc)

	// 参考实现与生产同一口径：每日累计原始时长、汇总时向下取整一次
	reference := map[string]time.Duration{}
	for _, p := range periods {
		for d := windowStart; d.Before(windowEnd); d = d.AddDate(0, 0, 1) {
			overlapStart := maxTime(p.Start, d)
			overlapEnd := minTime(p.End, d.AddDate(0, 0, 1))
			if overlapStart.Before(overlapEnd) {
				reference[d.Format("2006-01-02")] += overlapEnd.Sub(overlapStart)
			}
		}
	}
	require.Len(t, days, 90)
	for _, day := range days {
		require.Equal(t, uint64(reference[day.Date].Seconds()), day.OnlineSeconds, "date %s", day.Date)
	}
}

func TestDailyOnlineBreakdownMidnightDSTTransition(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	start := time.Date(2018, 11, 3, 0, 0, 0, 0, loc)
	end := time.Date(2018, 11, 6, 0, 0, 0, 0, loc)
	days := DailyOnlineBreakdown([]OnlinePeriod{{Start: start, End: end}}, start, end, loc)
	require.Equal(t, []DailyOnlineStat{{Date: "2018-11-03", OnlineSeconds: 86400}, {Date: "2018-11-04", OnlineSeconds: 82800}, {Date: "2018-11-05", OnlineSeconds: 86400}}, days)
}

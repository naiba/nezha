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

func TestSumOnlinePeriods(t *testing.T) {
	periods := []OnlinePeriod{
		{Start: ts(2026, 10, 1, 8, 0, 0), End: ts(2026, 10, 1, 9, 0, 0)},                 // 1h
		{Start: ts(2026, 10, 2, 0, 0, 0), End: ts(2026, 10, 2, 0, 30, 30)},               // 30m30s
		{Start: ts(2026, 10, 3, 23, 0, 0), End: ts(2026, 10, 4, 0, 0, 0), Ongoing: true}, // 1h
	}
	require.Equal(t, uint64(3600+1830+3600), SumOnlinePeriods(periods))
	require.Equal(t, uint64(0), SumOnlinePeriods(nil))
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

	// 每日合计与总时长一致
	var sum uint64
	for _, d := range daily {
		sum += d.OnlineSeconds
	}
	require.Equal(t, SumOnlinePeriods(periods), sum)
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
		start := cursor.Add(time.Duration(rng.Intn(4*3600)) * time.Second)
		end := start.Add(time.Duration(1+rng.Intn(6*3600)) * time.Second)
		if end.After(windowEnd) {
			end = windowEnd
		}
		periods = append(periods, OnlinePeriod{Start: start, End: end})
		cursor = end.Add(time.Duration(1+rng.Intn(3600)) * time.Second)
	}

	days := DailyOnlineBreakdown(periods, windowStart, windowEnd, loc)

	reference := map[string]uint64{}
	for _, p := range periods {
		for d := windowStart; d.Before(windowEnd); d = d.AddDate(0, 0, 1) {
			overlapStart := maxTime(p.Start, d)
			overlapEnd := minTime(p.End, d.AddDate(0, 0, 1))
			if overlapStart.Before(overlapEnd) {
				reference[d.Format("2006-01-02")] += uint64(overlapEnd.Sub(overlapStart).Seconds())
			}
		}
	}
	require.Len(t, days, 90)
	for _, day := range days {
		require.Equal(t, reference[day.Date], day.OnlineSeconds, "date %s", day.Date)
	}
}

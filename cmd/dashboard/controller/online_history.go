package controller

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/service/singleton"
)

// Get server online history
// @Summary Get server online history
// @Security BearerAuth
// @Schemes
// @Description Get online periods and aggregated uptime statistics of a specific server
// @Tags common
// @Param id path uint true "Server ID"
// @Param days query int false "Query window in days (default: configured online_stats_days, clamped to it)"
// @Produce json
// @Success 200 {object} model.CommonResponse[model.OnlineHistoryResponse]
// @Router /server/{id}/online-history [get]
func getServerOnlineHistory(c *gin.Context) (*model.OnlineHistoryResponse, error) {
	idStr := c.Param("id")
	serverID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	server, ok := singleton.ServerShared.Get(serverID)
	if !ok || server == nil {
		return nil, singleton.Localizer.ErrorT("server not found")
	}
	if !userCanViewServer(c, server) {
		return nil, singleton.Localizer.ErrorT("unauthorized")
	}
	_, isMember := c.Get(model.CtxKeyAuthorizedUser)

	maxDays := singleton.Conf.OnlineStatsDays
	if maxDays < model.MinOnlineStatsDays || maxDays > model.MaxOnlineStatsDays {
		maxDays = model.DefaultOnlineStatsDays
	}
	days := maxDays
	if daysStr := c.Query("days"); daysStr != "" {
		d, err := strconv.Atoi(daysStr)
		if err != nil || d < 1 {
			return nil, singleton.Localizer.ErrorT("invalid days parameter")
		}
		// 查询窗口收敛到后台设置的统计范围内
		if d > maxDays {
			d = maxDays
		}
		days = d
	}

	if !isMember && days > 1 {
		return nil, singleton.Localizer.ErrorT("unauthorized: only 1d data available for guests")
	}

	now := time.Now()
	windowEnd := now
	windowStart := now.AddDate(0, 0, -days)

	var sessions []model.OnlineSession
	// 时间列直接比较（与写入时的绑定格式同构），让 (server_id, started_at)
	// 索引可以完成范围扫描，days=1 也不会退化为该设备全部历史行的扫描
	if err := singleton.DB.
		Where("server_id = ? AND started_at < ? AND (ongoing = ? OR ended_at > ?)",
			serverID, windowEnd, true, windowStart).
		Order("started_at ASC").
		Find(&sessions).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	periods := model.ClipOnlineSessions(sessions, windowStart, windowEnd)
	totalSeconds := model.SumOnlinePeriods(periods)
	windowSeconds := uint64(windowEnd.Sub(windowStart).Seconds())
	onlineRate := 0.0
	if windowSeconds > 0 {
		onlineRate = float64(totalSeconds) / float64(windowSeconds) * 100
	}

	runtime := server.RuntimeSnapshot()
	currentlyOnline := !runtime.LastActive.IsZero() && now.Sub(runtime.LastActive) <= model.OnlineActiveWindow
	var lastActive *time.Time
	if !runtime.LastActive.IsZero() {
		lastActiveUTC := runtime.LastActive.UTC()
		lastActive = &lastActiveUTC
	}

	// 响应统一以 UTC 序列化，避免窗口时间与库中读回的会话时间混用
	// 本地偏移与 Z 两种写法（时段为值拷贝，不影响调用方）
	utcPeriods := make([]model.OnlinePeriod, len(periods))
	for i, p := range periods {
		utcPeriods[i] = model.OnlinePeriod{Start: p.Start.UTC(), End: p.End.UTC(), Ongoing: p.Ongoing}
	}

	return &model.OnlineHistoryResponse{
		ServerID:           serverID,
		ServerName:         server.Name,
		Days:               days,
		MaxDays:            maxDays,
		Timezone:           singleton.Loc.String(),
		TzOffset:           tzOffsetSeconds(now, singleton.Loc),
		WindowStart:        windowStart.UTC(),
		WindowEnd:          windowEnd.UTC(),
		TotalOnlineSeconds: totalSeconds,
		OnlineRate:         onlineRate,
		CurrentlyOnline:    currentlyOnline,
		LastActive:         lastActive,
		Periods:            utcPeriods,
		Daily:              model.DailyOnlineBreakdown(periods, windowStart, windowEnd, singleton.Loc),
	}, nil
}

// tzOffsetSeconds 返回指定时区在给定时刻相对 UTC 的偏移秒数。
func tzOffsetSeconds(at time.Time, loc *time.Location) int {
	_, offset := at.In(loc).Zone()
	return offset
}

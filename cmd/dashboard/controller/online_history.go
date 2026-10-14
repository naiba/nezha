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
	if err := singleton.DB.
		Where("server_id = ? AND datetime(started_at) < datetime(?) AND (ongoing = ? OR datetime(ended_at) > datetime(?))",
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
		lastActive = &runtime.LastActive
	}

	return &model.OnlineHistoryResponse{
		ServerID:           serverID,
		ServerName:         server.Name,
		Days:               days,
		MaxDays:            maxDays,
		WindowStart:        windowStart,
		WindowEnd:          windowEnd,
		TotalOnlineSeconds: totalSeconds,
		OnlineRate:         onlineRate,
		CurrentlyOnline:    currentlyOnline,
		LastActive:         lastActive,
		Periods:            periods,
		Daily:              model.DailyOnlineBreakdown(periods, windowStart, windowEnd, singleton.Loc),
	}, nil
}

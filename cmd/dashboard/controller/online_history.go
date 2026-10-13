package controller

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/service/singleton"
)

// Get server online history
// @Summary Get server online history
// @Security BearerAuth
// @Security APITokenAuth
// @Schemes
// @Description Get online periods and aggregated uptime statistics of a server. PAT scope required: nezha:server:read.
// @Tags common
// @Param id path uint true "Server ID"
// @Param days query int false "Query window in days; defaults to the configured statistics range and is clamped to it"
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

	maxDays := singleton.Conf.OnlineStatsDays
	if maxDays < model.MinOnlineStatsDays || maxDays > model.MaxOnlineStatsDays {
		maxDays = model.DefaultOnlineStatsDays
	}
	days := maxDays
	if daysStr := c.Query("days"); daysStr != "" {
		d, err := strconv.Atoi(daysStr)
		if err != nil || d < 1 {
			return nil, errors.New("invalid days parameter")
		}
		// 查询窗口收敛到后台设置的统计范围内
		if d > maxDays {
			d = maxDays
		}
		days = d
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
		LastActive:         runtime.LastActive,
		Periods:            periods,
		Daily:              model.DailyOnlineBreakdown(periods, windowStart, windowEnd, singleton.Loc),
	}, nil
}

// serveOnlineStatsPage 提供内置的设备在线统计页面（/online-stats）。页面为
// 自包含静态 HTML，数据全部来自 /api/v1 既有鉴权接口，与主题前端解耦。
func serveOnlineStatsPage(frontendDist fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !serveOnlineStatsAsset(c, frontendDist, "index.html") {
			c.JSON(http.StatusNotFound, newErrorResponse(errors.New("online stats page not found")))
		}
	}
}

// serveOnlineStatsAsset 从嵌入的 online-stats-dist 中读取静态资源，供页面
// 及其同目录引用（css/js）使用。
func serveOnlineStatsAsset(c *gin.Context, frontendDist fs.FS, name string) bool {
	if !fs.ValidPath(name) {
		return false
	}
	assetFS, err := fs.Sub(frontendDist, "online-stats-dist")
	if err != nil {
		return false
	}
	file, err := assetFS.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil || fileStat.IsDir() {
		return false
	}
	readSeeker, ok := file.(io.ReadSeeker)
	if !ok {
		return false
	}
	c.Header("Cache-Control", "no-store")
	http.ServeContent(c.Writer, c.Request, name, fileStat.ModTime(), readSeeker)
	return true
}

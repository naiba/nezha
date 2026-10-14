package singleton

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/nezhahq/nezha/model"
)

func setupOnlineHistoryTestDB(t *testing.T) {
	t.Helper()

	previousDB := DB
	var err error
	DB, err = gorm.Open(openSQLiteDialector(filepath.Join(t.TempDir(), "dashboard.sqlite")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := DB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB = previousDB
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close online history test database: %v", err)
		}
	})

	require.NoError(t, DB.AutoMigrate(&model.Server{}, &model.OnlineSession{}))
	require.NoError(t, DB.Exec("INSERT INTO servers (id, name, uuid) VALUES (1, 'server-1', 'online-history-test-1')").Error)
}

func getOnlineSession(t *testing.T, id uint64) model.OnlineSession {
	t.Helper()
	var s model.OnlineSession
	require.NoError(t, DB.First(&s, id).Error)
	return s
}

func countOnlineSessions(t *testing.T, cond string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&model.OnlineSession{}).Where(cond, args...).Count(&count).Error)
	return count
}

func TestRecordOnlineOpensSingleSession(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	r := NewOnlineHistoryRecorder()

	now := time.Now()
	r.RecordOnline(1, now)
	r.RecordOnline(1, now.Add(time.Second))

	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ?", 1))
	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 1, true))
}

func TestRecordOnlinePersistsLastActiveThrottled(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	r := NewOnlineHistoryRecorder()

	now := time.Now()
	r.RecordOnline(1, now)
	// 窗口内的重复上报不应触发落库
	r.RecordOnline(1, now.Add(5*time.Second))
	s := r.open[1]
	stored := getOnlineSession(t, s.id)
	require.WithinDuration(t, now, stored.LastActiveAt, time.Second)

	// 超过窗口后落库
	r.RecordOnline(1, now.Add(model.OnlineActiveWindow+time.Second))
	stored = getOnlineSession(t, s.id)
	require.WithinDuration(t, now.Add(model.OnlineActiveWindow+time.Second), stored.LastActiveAt, time.Second)
}

func TestRecordOfflineClosesSession(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	r := NewOnlineHistoryRecorder()

	start := time.Now()
	r.RecordOnline(1, start)
	r.RecordOffline(1, start.Add(10*time.Minute))
	// 重复关闭应为无操作
	r.RecordOffline(1, start.Add(11*time.Minute))

	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ?", 1))
	require.Equal(t, int64(0), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 1, true))
	var s model.OnlineSession
	require.NoError(t, DB.First(&s).Error)
	require.WithinDuration(t, start.Add(10*time.Minute), s.EndedAt, time.Second)
	require.WithinDuration(t, start, s.StartedAt, time.Second)
}

func TestRecordOnlineContinuesSessionAfterRestart(t *testing.T) {
	setupOnlineHistoryTestDB(t)

	// 模拟重启前遗留的进行中会话（最后活跃时间在在线窗口内：dashboard 只是短暂重启）
	now := time.Now()
	legacy := model.OnlineSession{ServerID: 1, StartedAt: now.Add(-time.Minute), LastActiveAt: now.Add(-5 * time.Second), Ongoing: true}
	require.NoError(t, DB.Create(&legacy).Error)

	// 新 recorder（构造时从库恢复）：新近会话应保留并续接，不产生新会话
	r := NewOnlineHistoryRecorder()
	r.RecordOnline(1, now)

	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ?", 1))
	continued := getOnlineSession(t, legacy.ID)
	require.True(t, continued.Ongoing)
	require.WithinDuration(t, now.Add(-time.Minute), continued.StartedAt, time.Second)
}

func TestRestoreOngoingSessionsClosesStaleSessions(t *testing.T) {
	setupOnlineHistoryTestDB(t)

	// 上次活跃时间远早于在线窗口：重启恢复时按最后已知活跃时刻结转
	staleStart := time.Now().Add(-time.Hour)
	stale := model.OnlineSession{ServerID: 1, StartedAt: staleStart, LastActiveAt: staleStart.Add(30 * time.Minute), Ongoing: true}
	require.NoError(t, DB.Create(&stale).Error)

	r := NewOnlineHistoryRecorder()
	r.RecordOnline(1, time.Now())

	closed := getOnlineSession(t, stale.ID)
	require.False(t, closed.Ongoing)
	require.WithinDuration(t, staleStart.Add(30*time.Minute), closed.EndedAt, time.Second)
	// 结转后新的上报开启新会话
	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 1, true))
}

func TestSweepStaleClosesOnlyStaleSessions(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	r := NewOnlineHistoryRecorder()

	now := time.Now()
	r.RecordOnline(1, now.Add(-time.Hour))
	// 第二台服务器：会话在 DB 但没有 servers 记录（服务器已被删除），手工放进内存 map
	require.NoError(t, DB.Exec("INSERT INTO online_sessions (server_id, started_at, last_active_at, ongoing, created_at, updated_at) VALUES (2, ?, ?, 1, ?, ?)", now.Add(-time.Hour), now.Add(-time.Hour), now, now).Error)
	var second model.OnlineSession
	require.NoError(t, DB.Where("server_id = ?", 2).First(&second).Error)
	r.mu.Lock()
	r.open[2] = &onlineSessionRuntime{id: second.ID, lastPersist: now.Add(-time.Hour)}
	r.mu.Unlock()

	// 服务器 1 活跃，服务器 2 不存在 → 仅关闭服务器 2 的会话
	lastActive := func(serverID uint64) (time.Time, bool) {
		if serverID == 1 {
			return now, true
		}
		return time.Time{}, false
	}
	r.SweepStale(now, lastActive)

	require.Equal(t, int64(1), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 1, true))
	require.Equal(t, int64(0), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 2, true))

	// 服务器 1 失联后也会被关闭，结束时刻为最后已知活跃时间
	r.SweepStale(now.Add(model.OnlineActiveWindow*2), func(uint64) (time.Time, bool) {
		return now, true
	})
	require.Equal(t, int64(0), countOnlineSessions(t, "server_id = ? AND ongoing = ?", 1, true))
	var closedFirst = getOnlineSession(t, func() uint64 {
		var s model.OnlineSession
		require.NoError(t, DB.Where("server_id = ?", 1).First(&s).Error)
		return s.ID
	}())
	require.WithinDuration(t, now, closedFirst.EndedAt, time.Second)
}

func TestPruneOnlineSessions(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	previousConf := Conf
	Conf = &ConfigClass{Config: &model.Config{}}
	Conf.OnlineStatsDays = 7
	t.Cleanup(func() { Conf = previousConf })

	now := time.Now()
	old := now.AddDate(0, 0, -30)
	rows := []model.OnlineSession{
		{ServerID: 1, StartedAt: old, EndedAt: old.Add(time.Hour), LastActiveAt: old.Add(time.Hour)},   // 超窗口 → 清理
		{ServerID: 1, StartedAt: old, EndedAt: now.Add(-time.Hour), LastActiveAt: now.Add(-time.Hour)}, // 跨窗口边界 → 保留
		{ServerID: 1, StartedAt: now.Add(-time.Hour), EndedAt: now, LastActiveAt: now},                 // 窗口内 → 保留
		{ServerID: 1, StartedAt: now, Ongoing: true, LastActiveAt: now},                                // 进行中 → 保留
		{ServerID: 99, StartedAt: now.Add(-time.Hour), EndedAt: now, LastActiveAt: now},                // 服务器已删除 → 清理
	}
	for i := range rows {
		require.NoError(t, DB.Create(&rows[i]).Error)
	}

	PruneOnlineSessions()

	require.Equal(t, int64(3), countOnlineSessions(t, "1 = 1"))
	require.Equal(t, int64(1), countOnlineSessions(t, "ongoing = ?", true))
}

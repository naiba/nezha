package singleton

import (
	"log"
	"sync"
	"time"

	"github.com/nezhahq/nezha/model"
)

// OnlineHistoryRecorder 维护每台服务器当前进行中的在线会话。会话的打开与
// 关闭由 agent 状态流的建立/断开驱动（service/rpc），并由定时清扫兜底关闭
// 失联（未正常断流）的会话；dashboard 重启时通过数据库中 Ongoing 会话恢复。
type OnlineHistoryRecorder struct {
	mu   sync.Mutex
	open map[uint64]*onlineSessionRuntime
}

type onlineSessionRuntime struct {
	id uint64
	// lastPersist 是 LastActiveAt 最近一次落库时间，用于节流写库。
	lastPersist time.Time
}

var OnlineHistoryShared *OnlineHistoryRecorder

func NewOnlineHistoryRecorder() *OnlineHistoryRecorder {
	r := &OnlineHistoryRecorder{open: make(map[uint64]*onlineSessionRuntime)}
	r.restoreOngoingSessions()
	return r
}

// restoreOngoingSessions 在启动时恢复进行中的会话：LastActiveAt 距今超过在线窗口的，
// 按最后一次已知活跃时刻结转关闭；仍在窗口内的保持打开，agent 重连后无缝
// 续接（RecordOnline 走数据库查找命中现有会话）。
func (r *OnlineHistoryRecorder) restoreOngoingSessions() {
	if DB == nil {
		return
	}
	var ongoing []model.OnlineSession
	if err := DB.Where("ongoing = ?", true).Find(&ongoing).Error; err != nil {
		log.Printf("NEZHA>> Failed to load ongoing online sessions: %v", err)
		return
	}
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range ongoing {
		if s.LastActiveAt.Before(now.Add(-model.OnlineActiveWindow)) {
			end := s.LastActiveAt
			if end.Before(s.StartedAt) {
				end = s.StartedAt
			}
			if err := closeOnlineSessionRow(s.ID, end); err != nil {
				log.Printf("NEZHA>> Failed to close stale online session %d: %v", s.ID, err)
				continue
			}
			continue
		}
		r.open[s.ServerID] = &onlineSessionRuntime{id: s.ID, lastPersist: s.LastActiveAt}
	}
}

// RecordOnline 记录一次状态上报：无进行中会话时开启新会话（重启遗留的
// Ongoing 会话直接续接）；已有会话时节流落库 LastActiveAt。
func (r *OnlineHistoryRecorder) RecordOnline(serverID uint64, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt, ok := r.open[serverID]; ok {
		if at.Sub(rt.lastPersist) >= model.OnlineActiveWindow {
			rt.lastPersist = at
			if err := DB.Model(&model.OnlineSession{}).Where("id = ?", rt.id).Update("last_active_at", at).Error; err != nil {
				log.Printf("NEZHA>> Failed to persist online session last_active_at: %v", err)
			}
		}
		return
	}

	var existing model.OnlineSession
	// Select("id")+Limit(1)+Find 查无行时不返回 ErrRecordNotFound：gorm 默认
	// logger 会把每次设备首报的"查无会话"打成 Warn+完整 SQL，噪音与设备
	// 重连次数成正比
	if err := DB.Model(&model.OnlineSession{}).
		Where("server_id = ? AND ongoing = ?", serverID, true).
		Select("id").Limit(1).Find(&existing).Error; err != nil {
		log.Printf("NEZHA>> Failed to look up ongoing online session: %v", err)
		return
	}
	if existing.ID > 0 {
		r.open[serverID] = &onlineSessionRuntime{id: existing.ID, lastPersist: at}
		return
	}

	s := model.OnlineSession{ServerID: serverID, StartedAt: at, LastActiveAt: at, Ongoing: true}
	if err := DB.Create(&s).Error; err != nil {
		log.Printf("NEZHA>> Failed to create online session: %v", err)
		return
	}
	r.open[serverID] = &onlineSessionRuntime{id: s.ID, lastPersist: at}
}

// RecordOffline 关闭该服务器当前进行中的会话。at 是会话结束时刻：正常断流
// 为当前时间，清扫/结转为最后一次已知活跃时间。
func (r *OnlineHistoryRecorder) RecordOffline(serverID uint64, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rt, ok := r.open[serverID]
	if !ok {
		return
	}
	delete(r.open, serverID)
	end := at
	if end.Before(rt.lastPersist) {
		end = rt.lastPersist
	}
	if err := closeOnlineSessionRow(rt.id, end); err != nil {
		log.Printf("NEZHA>> Failed to close online session %d: %v", rt.id, err)
	}
}

// SweepStale 关闭超过在线窗口仍无状态上报的进行中会话。lastActiveFn 返回
// 服务器的运行时最后活跃时间与是否存在。
func (r *OnlineHistoryRecorder) SweepStale(now time.Time, lastActiveFn func(serverID uint64) (time.Time, bool)) {
	type staleSession struct {
		serverID uint64
		at       time.Time
	}
	var stale []staleSession

	r.mu.Lock()
	for serverID, rt := range r.open {
		lastActive, ok := lastActiveFn(serverID)
		if ok && !lastActive.Before(now.Add(-model.OnlineActiveWindow)) {
			continue
		}
		at := lastActive
		if !ok || at.Before(rt.lastPersist) {
			at = rt.lastPersist
		}
		stale = append(stale, staleSession{serverID: serverID, at: at})
	}
	r.mu.Unlock()

	for _, s := range stale {
		r.RecordOffline(s.serverID, s.at)
	}
}

// SweepOnlineSessions 是定时清扫入口，接上服务器运行时的最后活跃时间。
func SweepOnlineSessions() {
	if OnlineHistoryShared == nil || ServerShared == nil || DB == nil {
		return
	}
	OnlineHistoryShared.SweepStale(time.Now(), onlineServerLastActive)
}

func onlineServerLastActive(serverID uint64) (time.Time, bool) {
	server, ok := ServerShared.Get(serverID)
	if !ok || server == nil {
		return time.Time{}, false
	}
	lastActive := server.RuntimeSnapshot().LastActive
	return lastActive, !lastActive.IsZero()
}

// PruneOnlineSessions 清理超出统计窗口的已结束会话与已删除服务器的残留记录。
// 时间列直接比较（与写入时的绑定格式同构）以走 ended_at 索引，并用分批
// DELETE 替换单条巨型删除，避免大表上长时间占用 sqlite 写锁、阻塞上报路径。
func PruneOnlineSessions() {
	if DB == nil {
		return
	}
	keepDays := Conf.OnlineStatsDays
	if keepDays < model.MinOnlineStatsDays || keepDays > model.MaxOnlineStatsDays {
		keepDays = model.DefaultOnlineStatsDays
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	pruneOnlineSessionsBatch("ongoing = ? AND ended_at < ?", false, cutoff)
	pruneOnlineSessionsBatch("server_id NOT IN (SELECT `id` FROM servers)")
}

// pruneOnlineSessionsBatch 按主键分批删除匹配的行，每批之间释放写锁。
func pruneOnlineSessionsBatch(where string, args ...any) {
	const batchSize = 10000
	for {
		var ids []uint64
		if err := DB.Model(&model.OnlineSession{}).Unscoped().
			Where(where, args...).Limit(batchSize).Pluck("id", &ids).Error; err != nil {
			log.Printf("NEZHA>> Failed to select online sessions for pruning: %v", err)
			return
		}
		if len(ids) == 0 {
			return
		}
		if err := DB.Unscoped().Delete(&model.OnlineSession{}, ids).Error; err != nil {
			log.Printf("NEZHA>> Failed to prune online sessions: %v", err)
			return
		}
		if len(ids) < batchSize {
			return
		}
	}
}

func closeOnlineSessionRow(id uint64, end time.Time) error {
	return DB.Model(&model.OnlineSession{}).Where("id = ?", id).Updates(map[string]any{
		"ended_at": end,
		"ongoing":  false,
	}).Error
}

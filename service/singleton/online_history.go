package singleton

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/nezhahq/nezha/model"
)

// OnlineHistoryRecorder 维护每台服务器当前进行中的在线会话。会话的打开与
// 关闭由 agent 状态流的建立/断开驱动（service/rpc），并由定时清扫兜底关闭
// 失联（未正常断流）的会话；dashboard 重启时通过数据库中 Ongoing 会话恢复。
type OnlineHistoryRecorder struct {
	mu       sync.Mutex
	restored bool
	open     map[uint64]*onlineSessionRuntime
	// closing 是关库失败后的待关闭状态：断线边界已被观察到，会话 ID 与
	// 结束时间在此固定、不可被后续上报覆盖；重试只为旧会话完成持久化，
	// 重连不得把它续接成活跃会话（同服务器可能同时有多条待关闭会话）。
	closing map[uint64][]*onlineSessionClosing
}

type onlineSessionRuntime struct {
	id           uint64
	lastObserved time.Time
	revision     uint64
	// lastPersist 是 LastActiveAt 最近一次落库时间，用于节流写库。
	lastPersist time.Time
}

type onlineSessionClosing struct {
	id uint64
	// end 是已观察到的会话结束时刻，重试持久化时原样使用。
	end time.Time
}

var OnlineHistoryShared *OnlineHistoryRecorder

func NewOnlineHistoryRecorder() *OnlineHistoryRecorder {
	r := &OnlineHistoryRecorder{
		open:    make(map[uint64]*onlineSessionRuntime),
		closing: make(map[uint64][]*onlineSessionClosing),
	}
	r.mu.Lock()
	r.restoreOngoingSessions()
	r.mu.Unlock()
	return r
}

// restoreOngoingSessions 每台服务器仅允许最新且仍在在线窗口内的会话续接，
// 其余会话按最后持久化的活跃时间结转。崩溃会丢失内存中的待关闭边界，
// 恢复时采用 LastActiveAt 保守结转，不推测未持久化的断线时间。
func (r *OnlineHistoryRecorder) restoreOngoingSessions() bool {
	if DB == nil {
		return false
	}
	var ongoing []model.OnlineSession
	if err := DB.Where("ongoing = ?", true).
		Order("server_id ASC").Order("started_at DESC").Order("id DESC").Find(&ongoing).Error; err != nil {
		log.Printf("NEZHA>> Failed to load ongoing online sessions: %v", err)
		return false
	}
	now := time.Now()
	seenServers := make(map[uint64]bool)
	for _, s := range ongoing {
		// 关闭失败后重连可留下同服务器多条 Ongoing 行；直接赋值 open 会覆盖
		// 旧 ID，导致其永久脱管。按开始时间（同时间按 ID）选最新行，其余
		// 必须关闭或进入 closing 重试，即使最新行本身已经过期也不能复活旧行。
		duplicate := seenServers[s.ServerID]
		seenServers[s.ServerID] = true
		if duplicate || s.LastActiveAt.Before(now.Add(-model.OnlineActiveWindow)) {
			end := s.LastActiveAt
			if end.Before(s.StartedAt) {
				end = s.StartedAt
			}
			if err := closeOnlineSessionRow(s.ID, end); err != nil {
				log.Printf("NEZHA>> Failed to close recovered online session %d: %v", s.ID, err)
				// 关库失败不能跳过，否则会留下永不闭合、也无人重试的
				// Ongoing 行；进入待关闭状态交由清扫周期重试。过期或
				// 被更新会话替代的行均不得被重连续接为活跃会话
				r.closing[s.ServerID] = append(r.closing[s.ServerID],
					&onlineSessionClosing{id: s.ID, end: end})
			}
			continue
		}
		r.open[s.ServerID] = &onlineSessionRuntime{id: s.ID, lastPersist: s.LastActiveAt, lastObserved: s.LastActiveAt}
	}
	r.restored = true
	return true
}

// flushClosing 尽力重试关闭该服务器的待关闭会话：按入列时固定的结束时间
// 持久化，成功才移除；失败的条目原样保留，结束时间不得被后续上报改写。
func (r *OnlineHistoryRecorder) flushClosing(serverID uint64) {
	pendings, ok := r.closing[serverID]
	if !ok {
		return
	}
	kept := pendings[:0]
	for _, pc := range pendings {
		if err := closeOnlineSessionRow(pc.id, pc.end); err != nil {
			log.Printf("NEZHA>> Failed to retry closing online session %d: %v", pc.id, err)
			kept = append(kept, pc)
		}
	}
	if len(kept) == 0 {
		delete(r.closing, serverID)
	} else {
		r.closing[serverID] = kept
	}
}

// RecordOnline 记录一次状态上报：无进行中会话时开启新会话（重启遗留且仍在
// 窗口内的 Ongoing 会话直接续接）；已有会话时节流落库 LastActiveAt。
func (r *OnlineHistoryRecorder) RecordOnline(serverID uint64, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// A failed startup read is not an empty history: retry before accepting reports.
	if !r.restored && !r.restoreOngoingSessions() {
		return
	}
	at = at.UTC()
	r.flushClosing(serverID)

	if rt, ok := r.open[serverID]; ok {
		rt.lastObserved = at
		rt.revision++
		if at.Sub(rt.lastPersist) >= model.OnlineActiveWindow {
			if err := DB.Model(&model.OnlineSession{}).Where("id = ?", rt.id).Update("last_active_at", at).Error; err != nil {
				log.Printf("NEZHA>> Failed to persist online session last_active_at: %v", err)
			} else {
				// Only committed writes advance the throttle; failures retry on the next report.
				rt.lastPersist = at
			}
		}
		return
	}

	// All ongoing rows are owned by open or closing after successful recovery.

	s := model.OnlineSession{ServerID: serverID, StartedAt: at, LastActiveAt: at, Ongoing: true}
	if err := DB.Create(&s).Error; err != nil {
		log.Printf("NEZHA>> Failed to create online session: %v", err)
		return
	}
	r.open[serverID] = &onlineSessionRuntime{id: s.ID, lastPersist: at, lastObserved: at}
}

// RecordOffline 关闭该服务器当前进行中的会话。at 是会话结束时刻：正常断流
// 为当前时间，清扫/结转为最后一次已知活跃时间。
// 关库失败时把会话转入待关闭状态而非保留为活跃条目：会话 ID 与已观察到的
// 结束时刻（at 与 lastPersist 的较大者）就此固定，后续上报不得覆盖；重试
// 只为旧会话完成持久化，重连则开启新会话——断线后的离线间隔不得计作在线。
func (r *OnlineHistoryRecorder) RecordOffline(serverID uint64, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Disconnect may be the first event after startup storage recovers.
	if !r.restored && !r.restoreOngoingSessions() {
		return
	}
	r.recordOfflineLocked(serverID, at)
}

func (r *OnlineHistoryRecorder) recordOfflineLocked(serverID uint64, at time.Time) {
	rt, ok := r.open[serverID]
	if !ok {
		return
	}
	end := at
	if end.Before(rt.lastPersist) {
		end = rt.lastPersist
	}
	if err := closeOnlineSessionRow(rt.id, end); err != nil {
		log.Printf("NEZHA>> Failed to close online session %d: %v", rt.id, err)
		// 结束时刻已观察到，不能因写库失败丢失，也不能把行继续当活跃
		// 会话：转待关闭状态，由 RecordOnline 冲洗与清扫周期重试
		r.closing[serverID] = append(r.closing[serverID], &onlineSessionClosing{id: rt.id, end: end})
		delete(r.open, serverID)
		return
	}
	delete(r.open, serverID)
}

// SweepStale 关闭超过在线窗口仍无状态上报的进行中会话。lastActiveFn 返回
// 服务器的运行时最后活跃时间与是否存在。
func (r *OnlineHistoryRecorder) SweepStale(now time.Time, lastActiveFn func(serverID uint64) (time.Time, bool)) {
	// Never call the runtime observer while holding mu: reports acquire the
	// runtime lease before this mutex. Snapshot identities, then revalidate them.
	r.mu.Lock()
	if !r.restored && !r.restoreOngoingSessions() {
		r.mu.Unlock()
		return
	}
	candidates := make(map[uint64]onlineSessionRuntime, len(r.open))
	for id, rt := range r.open {
		candidates[id] = *rt
	}
	for id := range r.closing {
		r.flushClosing(id)
	}
	r.mu.Unlock()
	for id, candidate := range candidates {
		lastActive, ok := lastActiveFn(id)
		r.mu.Lock()
		rt := r.open[id]
		if rt != nil && rt.id == candidate.id && rt.revision == candidate.revision {
			if lastActive.Before(rt.lastObserved) {
				lastActive = rt.lastObserved
			}
			if (!ok || lastActive.Before(now.Add(-model.OnlineActiveWindow))) && rt.lastObserved.Before(now.Add(-model.OnlineActiveWindow)) {
				r.recordOfflineLocked(id, lastActive)
			}
		}
		r.mu.Unlock()
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
	cutoff := time.Now().AddDate(0, 0, -keepDays).UTC()
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
		"ended_at": end.UTC(),
		"ongoing":  false,
	}).Error
}

// Sessions snapshots rows and volatile close boundaries under the same lock.
// Otherwise a failed close can count confirmed offline time as ongoing, or a
// successful retry can remove the boundary after the reader fetched a stale row.
func (r *OnlineHistoryRecorder) Sessions(serverID uint64, start, end time.Time) ([]model.OnlineSession, error) {
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
		if !r.restored && !r.restoreOngoingSessions() {
			return nil, fmt.Errorf("online history recovery unavailable")
		}
	}
	var sessions []model.OnlineSession
	// Separate branches use (server_id, ongoing, ended_at) and
	// (server_id, ongoing, started_at), avoiding a scan of retained closed rows.
	for _, ongoing := range []bool{false, true} {
		var rows []model.OnlineSession
		q := DB.Where("server_id = ? AND ongoing = ?", serverID, ongoing)
		if ongoing {
			q = q.Where("started_at < ?", end.UTC())
		} else {
			q = q.Where("ended_at > ?", start.UTC())
		}
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		sessions = append(sessions, rows...)
	}
	if r != nil {
		endings := make(map[uint64]time.Time)
		for _, pending := range r.closing[serverID] {
			endings[pending.id] = pending.end
		}
		for i := range sessions {
			if end, ok := endings[sessions[i].ID]; ok {
				sessions[i].Ongoing = false
				sessions[i].EndedAt = end
			}
		}
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].StartedAt.Before(sessions[j].StartedAt) })
	return sessions, nil
}

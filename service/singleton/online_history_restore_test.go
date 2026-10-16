package singleton

import (
	"testing"
	"time"

	"github.com/nezhahq/nezha/model"
	"github.com/stretchr/testify/require"
)

func TestRestoreOngoingSessionsReconcilesRecentDuplicates(t *testing.T) {
	for _, failClose := range []bool{false, true} {
		name := "successful_close"
		if failClose {
			name = "retry_failed_close"
		}
		t.Run(name, func(t *testing.T) {
			setupOnlineHistoryTestDB(t)
			base := time.Now().Add(-10 * time.Second).Truncate(time.Second)
			// Insert out of chronological order: choosing by ID alone would keep the wrong session.
			newest := model.OnlineSession{ServerID: 1, StartedAt: base.Add(4 * time.Second), LastActiveAt: base.Add(5 * time.Second), Ongoing: true}
			oldest := model.OnlineSession{ServerID: 1, StartedAt: base, LastActiveAt: base.Add(time.Second), Ongoing: true}
			middle := model.OnlineSession{ServerID: 1, StartedAt: base.Add(2 * time.Second), LastActiveAt: base.Add(3 * time.Second), Ongoing: true}
			for _, session := range []*model.OnlineSession{&newest, &oldest, &middle} {
				require.NoError(t, DB.Create(session).Error)
			}
			if failClose {
				injectCloseUpdateFailure(t)
			}
			r := NewOnlineHistoryRecorder()
			require.Len(t, r.open, 1)
			require.Equal(t, newest.ID, r.open[1].id)
			if failClose {
				require.Len(t, r.closing[1], 2)
				require.Equal(t, int64(3), countOnlineSessions(t, "ongoing = ?", true))
				dropCloseUpdateFailure(t)
			}
			r.SweepStale(base.Add(6*time.Second), func(uint64) (time.Time, bool) { return newest.LastActiveAt, true })
			require.Empty(t, r.closing)
			require.True(t, getOnlineSession(t, newest.ID).Ongoing)
			for _, previous := range []model.OnlineSession{oldest, middle} {
				stored := getOnlineSession(t, previous.ID)
				require.False(t, stored.Ongoing)
				require.True(t, stored.EndedAt.Equal(previous.LastActiveAt))
			}
			r.RecordOnline(1, base.Add(7*time.Second))
			require.Equal(t, newest.ID, r.open[1].id)
			require.Equal(t, int64(3), countOnlineSessions(t, "server_id = ?", 1))
			r.SweepStale(base.Add(2*model.OnlineActiveWindow), func(uint64) (time.Time, bool) { return time.Time{}, false })
			require.Empty(t, r.open)
			require.Empty(t, r.closing)
			require.Equal(t, int64(0), countOnlineSessions(t, "ongoing = ?", true))
		})
	}
}

func TestRestoreOngoingSessionsUsesIDToBreakStartTimeTies(t *testing.T) {
	setupOnlineHistoryTestDB(t)
	start := time.Now().Add(-5 * time.Second)
	first := model.OnlineSession{ServerID: 1, StartedAt: start, LastActiveAt: start, Ongoing: true}
	second := first
	other := model.OnlineSession{ServerID: 2, StartedAt: start, LastActiveAt: start, Ongoing: true}
	for _, session := range []*model.OnlineSession{&first, &second, &other} {
		require.NoError(t, DB.Create(session).Error)
	}
	r := NewOnlineHistoryRecorder()
	require.Len(t, r.open, 2)
	require.Equal(t, second.ID, r.open[1].id)
	require.Equal(t, other.ID, r.open[2].id)
	require.False(t, getOnlineSession(t, first.ID).Ongoing)
	require.True(t, getOnlineSession(t, first.ID).EndedAt.Equal(start))
	require.Equal(t, int64(2), countOnlineSessions(t, "ongoing = ?", true))
}

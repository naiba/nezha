package singleton

import (
	"errors"
	"testing"
	"time"

	"github.com/nezhahq/nezha/model"
	"github.com/stretchr/testify/require"
)

func TestOnlineHistoryPruneWaitsForRetentionCommit(t *testing.T) {
	for _, saveFails := range []bool{true, false} {
		t.Run(map[bool]string{true: "failed_save", false: "successful_save"}[saveFails], func(t *testing.T) {
			setupOnlineHistoryTestDB(t)
			previous := Conf
			Conf = &ConfigClass{Config: &model.Config{}}
			Conf.OnlineStatsDays = 30
			t.Cleanup(func() { Conf = previous })
			now := time.Now().UTC()
			row := model.OnlineSession{ServerID: 1, StartedAt: now.AddDate(0, 0, -15), EndedAt: now.AddDate(0, 0, -14)}
			require.NoError(t, DB.Create(&row).Error)
			saveEntered := make(chan struct{})
			releaseSave := make(chan struct{})
			saved := make(chan error, 1)
			days := 7
			go func() {
				saved <- Conf.CommitOnlineHistoryRetention(&days, func() error {
					close(saveEntered)
					<-releaseSave
					if saveFails {
						return errors.New("retention save unavailable")
					}
					return nil
				})
			}()
			<-saveEntered
			pruneStarted := make(chan struct{})
			pruned := make(chan struct{})
			go func() { close(pruneStarted); PruneOnlineSessions(); close(pruned) }()
			<-pruneStarted
			premature := false
			select {
			case <-pruned:
				premature = true
			case <-time.After(100 * time.Millisecond):
			}
			close(releaseSave)
			err := <-saved
			<-pruned
			require.False(t, premature, "pruning must not read an uncommitted retention policy")
			if saveFails {
				require.ErrorContains(t, err, "retention save unavailable")
				require.Equal(t, 30, Conf.OnlineHistoryRetentionDays())
				require.Equal(t, int64(1), countOnlineSessions(t, "id = ?", row.ID))
			} else {
				require.NoError(t, err)
				require.Equal(t, 7, Conf.OnlineHistoryRetentionDays())
				require.Equal(t, int64(0), countOnlineSessions(t, "id = ?", row.ID))
			}
		})
	}
}

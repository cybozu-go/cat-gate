package runners

import (
	"context"
	"time"

	"github.com/cybozu-go/cat-gate/internal/constants"
	"github.com/cybozu-go/cat-gate/internal/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const gcIntervalHours = 24

var historyDeletionDuration = 24 * 60 * 60 // 1 day

type GarbageCollector struct {
}

func (gc GarbageCollector) NeedLeaderElection() bool {
	return true
}

func (gc GarbageCollector) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Hour * gcIntervalHours)
	defer ticker.Stop()
	logger := log.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			controller.GateRemovalHistories.Range(func(imageHash, value interface{}) bool {
				lastGateRemovalTime := value.(time.Time)
				// Delete history that has not been updated for a long time to prevent memory leaks.
				if time.Since(lastGateRemovalTime) > time.Duration(historyDeletionDuration)*time.Second {
					logger.V(constants.LevelDebug).Info("delete old history", "image hash", imageHash, "lastGateRemovalTime", lastGateRemovalTime)
					controller.GateRemovalHistories.Delete(imageHash)
				}
				return true
			})
		}
	}
}

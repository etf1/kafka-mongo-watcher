package service

import (
	"fmt"

	"github.com/etf1/kafka-mongo-watcher/internal/checkpoint"
)

// GetCheckpointTracker returns the resume token checkpoint tracker, or nil when
// checkpoint is disabled (always disabled in replay mode)
func (container *Container) GetCheckpointTracker() *checkpoint.Tracker {
	if container.Cfg.Replay || !container.Cfg.MongoDB.Options.CheckpointEnabled {
		return nil
	}
	if container.checkpointTracker == nil {
		container.checkpointTracker = checkpoint.NewTracker(container.getCheckpointStore(), container.GetLogger())
	}
	return container.checkpointTracker
}

func (container *Container) getCheckpointStore() checkpoint.Store {
	cfg := container.Cfg
	return checkpoint.NewMongoStore(
		container.GetMongoConnection().Collection(cfg.MongoDB.Options.CheckpointCollection),
		fmt.Sprintf("%s/%s/%s/%s", cfg.AppName, cfg.MongoDB.DatabaseName, cfg.MongoDB.CollectionName, cfg.Kafka.Topic),
	)
}

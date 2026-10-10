package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
)

var cfg = &Base{
	AppName:         AppName,
	PrintConfig:     false,
	LogCliVerbose:   true,
	LogLevel:        logger.LevelString(logger.InfoLevel.String()),
	Replay:          false,
	OtelSampleRatio: 1,
	PprofEnabled:    true,
	HttpServer: HttpServer{
		HTTPAddr:     ":8001",
		DebugEnabled: false,

		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	},
	MongoDB: MongoDB{
		URI:                    "mongodb://root:toor@127.0.0.1:27011,127.0.0.1:27012,127.0.0.1:27013/watcher?replicaSet=replicaset&authSource=admin",
		DatabaseName:           "watcher",
		CollectionName:         "items",
		ServerSelectionTimeout: 2 * time.Second,
		Options: MongoDBOptions{
			FullDocument:         false,
			WatchMaxRetries:      3,
			WatchRetryDelay:      500 * time.Millisecond,
			CheckpointEnabled:    true,
			CheckpointCollection: "kafka_mongo_watcher_checkpoints",
			CheckpointInterval:   1 * time.Second,
			ResumeOnHistoryLost:  ResumeOnHistoryLostFail,
		},
	},
	Kafka: Kafka{
		BootstrapServers:   "127.0.0.1:9092",
		Topic:              "kafka-mongo-watcher",
		ProduceChannelSize: 10000,
		WithDecorators:     true,
		MessageMaxBytes:    1024 * 1024,
	},
}

// NewBase returns a new base configuration
func TestNewBase(t *testing.T) {
	os.Setenv("PRINT_CONFIG", "false")

	ctx := context.Background()
	base := NewBase(ctx, "")

	assert.IsType(t, new(Base), base)
	assert.Equal(t, cfg, base)
}

func TestValidate(t *testing.T) {
	valid := func() *Base {
		return &Base{MongoDB: MongoDB{Options: MongoDBOptions{
			CheckpointEnabled:   true,
			CheckpointInterval:  time.Second,
			ResumeOnHistoryLost: ResumeOnHistoryLostFail,
		}}}
	}

	assert.NoError(t, valid().Validate())

	zeroInterval := valid()
	zeroInterval.MongoDB.Options.CheckpointInterval = 0
	assert.Error(t, zeroInterval.Validate())

	checkpointDisabled := valid()
	checkpointDisabled.MongoDB.Options.CheckpointEnabled = false
	checkpointDisabled.MongoDB.Options.CheckpointInterval = 0
	assert.NoError(t, checkpointDisabled.Validate())

	unknownPolicy := valid()
	unknownPolicy.MongoDB.Options.ResumeOnHistoryLost = "later"
	assert.Error(t, unknownPolicy.Validate())
}

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
	PrintConfig:   false,
	LogCliVerbose: true,
	LogLevel:      logger.LevelString(logger.InfoLevel.String()),
	Replay:        false,
	TechServer: TechServer{
		PprofEnabled: true,
		HTTPAddr:     ":8001",

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
			FullDocument:    false,
			WatchMaxRetries: 3,
			WatchRetryDelay: 500 * time.Millisecond,
		},
	},
	Kafka: Kafka{
		BootstrapServers:   "127.0.0.1:9092",
		Topic:              "kafka-mongo-watcher",
		ProduceChannelSize: 10000,
	},
}

// NewBase returns a new base configuration
func TestNewBase(t *testing.T) {
	os.Setenv("KAFKA_MONGO_WATCHER_PRINT_CONFIG", "false")

	ctx := context.Background()
	base := NewBase(ctx)

	assert.IsType(t, new(Base), base)
	assert.Equal(t, cfg, base)
}

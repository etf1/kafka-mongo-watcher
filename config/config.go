package config

import (
	"context"
	"fmt"
	"time"

	"github.com/gol4ng/logger"

	"github.com/etf1/go-config"
	"github.com/etf1/go-config/env"
	"github.com/etf1/go-config/prefix"
)

const AppName = "kafka-mongo-watcher"

var AppVersion = "wip"

// Base is the base configuration provider
type Base struct {
	AppName               string             `config:"APP_NAME"`
	PrintConfig           bool               `config:"PRINT_CONFIG"`
	LogLevel              logger.LevelString `config:"LOG_LEVEL"`
	LogCliVerbose         bool               `config:"LOG_CLI_VERBOSE"`
	GraylogEndpoint       string             `config:"GRAYLOG_ENDPOINT"`
	Replay                bool               `config:"REPLAY"`
	CustomPipeline        string             `config:"CUSTOM_PIPELINE"`
	OtelCollectorEndpoint string             `config:"OPEN_TELEMETRY_COLLECTOR_ENDPOINT"`
	OtelSampleRatio       float64            `config:"OPEN_TELEMETRY_SAMPLE_RATIO"`
	PprofEnabled          bool               `config:"PPROF_ENABLED"`

	HttpServer
	MongoDB
	Kafka
}

// HttpServer is the configuration provider for monitoring and debug HTTP server
type HttpServer struct {
	HTTPAddr     string `config:"HTTP_TECH_ADDR"`
	DebugEnabled bool   `config:"HTTP_DEBUG_ENABLED"`

	ReadHeaderTimeout time.Duration `config:"HTTP_READ_HEADER_TIMEOUT"`
	WriteTimeout      time.Duration `config:"HTTP_WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `config:"HTTP_IDLE_TIMEOUT"`
}

// MongoDB is the configuration provider for MongoDB
type MongoDB struct {
	URI                    string        `config:"MONGODB_URI"`
	DatabaseName           string        `config:"MONGODB_DATABASE_NAME"`
	CollectionName         string        `config:"MONGODB_COLLECTION_NAME"`
	ServerSelectionTimeout time.Duration `config:"MONGODB_SERVER_SELECTION_TIMEOUT"`
	Options                MongoDBOptions
}

type MongoDBOptions struct {
	BatchSize               int32         `config:"MONGODB_OPTION_BATCH_SIZE"`
	FullDocument            bool          `config:"MONGODB_OPTION_FULL_DOCUMENT"`
	IgnoreUpdateDescription bool          `config:"MONGODB_OPTION_IGNORE_UPDATE_DESCRIPTION"`
	MaxAwaitTime            time.Duration `config:"MONGODB_OPTION_MAX_AWAIT_TIME"`
	ResumeAfter             string        `config:"MONGODB_OPTION_RESUME_AFTER"`
	StartAtOperationTimeI   uint32        `config:"MONGODB_OPTION_START_AT_OPERATION_TIME_I"`
	StartAtOperationTimeT   uint32        `config:"MONGODB_OPTION_START_AT_OPERATION_TIME_T"`
	WatchRetryDelay         time.Duration `config:"MONGODB_OPTION_WATCH_RETRY_DELAY"`
	WatchMaxRetries         int32         `config:"MONGODB_OPTION_WATCH_MAX_RETRIES"`
}

// Kafka is the configuration provider for Kafka
type Kafka struct {
	BootstrapServers   string `config:"KAFKA_BOOTSTRAP_SERVERS"`
	Topic              string `config:"KAFKA_TOPIC"`
	ProduceChannelSize int    `config:"KAFKA_PRODUCE_CHANNEL_SIZE"`
	WithDecorators     bool   `config:"KAFKA_WITH_DECORATORS"`
	MessageMaxBytes    int    `config:"KAFKA_MESSAGE_MAX_BYTES"`
}

// NewBase returns a new base configuration
func NewBase(ctx context.Context, configPrefix string) *Base {
	cfg := &Base{
		AppName:         AppName,
		PrintConfig:     true,
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
				FullDocument:    false,
				WatchMaxRetries: 3,
				WatchRetryDelay: 500 * time.Millisecond,
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

	loader := config.NewDefaultConfigLoader().PrependBackends(
		prefix.NewBackend(configPrefix, env.NewBackend()),
	)

	loader.LoadOrFatal(ctx, cfg)

	if cfg.PrintConfig {
		fmt.Println(config.TableString(cfg))
	}

	return cfg
}

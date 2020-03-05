package config

import (
	"context"
	"fmt"
	"time"

	"github.com/gol4ng/logger"

	"go.etf1.tf1.fr/etf1-platform/pkg/config"
)

const AppName = "kafka-mongo-watcher"

var AppVersion = "wip"

// Base is the base configuration provider
type Base struct {
	LogLevel        logger.LevelString `config:"KAFKA_MONGO_WATCHER_LOG_LEVEL"`
	LogCliVerbose   bool               `config:"KAFKA_MONGO_WATCHER_LOG_CLI_VERBOSE"`
	GraylogEndpoint string             `config:"KAFKA_MONGO_WATCHER_GRAYLOG_ENDPOINT"`
	Replay          bool               `config:"KAFKA_MONGO_WATCHER_REPLAY"`
	WorkerNumber    int                `config:"KAFKA_MONGO_WATCHER_WORKER_NUMBER"`
	WorkerTimeout   time.Duration      `config:"KAFKA_MONGO_WATCHER_WORKER_TIMEOUT"`

	TechServer
	MongoDB
	Kafka
}

// TechServer is the configuration provider for monitoring HTTP server
type TechServer struct {
	PprofEnabled bool   `config:"KAFKA_MONGO_WATCHER_PPROF_ENABLED"`
	HTTPAddr     string `config:"KAFKA_MONGO_WATCHER_HTTP_TECH_ADDR"`

	ReadHeaderTimeout time.Duration `config:"KAFKA_MONGO_WATCHER_HTTP_READ_HEADER_TIMEOUT"`
	WriteTimeout      time.Duration `config:"KAFKA_MONGO_WATCHER_HTTP_WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `config:"KAFKA_MONGO_WATCHER_HTTP_IDLE_TIMEOUT"`
}

// MongoDB is the configuration provider for MongoDB
type MongoDB struct {
	URI            string `config:"KAFKA_MONGO_WATCHER_MONGODB_URI"`
	DatabaseName   string `config:"KAFKA_MONGO_WATCHER_MONGODB_DATABASE_NAME"`
	CollectionName string `config:"KAFKA_MONGO_WATCHER_MONGODB_COLLECTION_NAME"`
}

// Kafka is the configuration provider for Kafka
type Kafka struct {
	BootstrapServers string `config:"KAFKA_MONGO_WATCHER_KAFKA_BOOTSTRAP_SERVERS"`
	Topic            string `config:"KAFKA_MONGO_WATCHER_KAFKA_TOPIC"`
}

// NewBase returns a new base configuration
func NewBase() *Base {
	cfg := &Base{
		LogCliVerbose: true,
		LogLevel:      logger.LevelString(logger.InfoLevel.String()),
		Replay:        false,
		WorkerNumber:  5,
		WorkerTimeout: 10 * time.Second,
		TechServer: TechServer{
			PprofEnabled: true,
			HTTPAddr:     ":8001",

			ReadHeaderTimeout: 1 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       90 * time.Second,
		},
		MongoDB: MongoDB{
			URI:            "mongodb://root:toor@127.0.0.1:27011,127.0.0.1:27012,127.0.0.1:27013/local?replicaSet=replicaset",
			DatabaseName:   "local",
			CollectionName: "items",
		},
		Kafka: Kafka{
			BootstrapServers: "localhost",
			Topic:            "kafka-mongo-watcher",
		},
	}

	config.LoadOrFatal(context.Background(), cfg)
	fmt.Println(config.TableString(cfg))

	return cfg
}

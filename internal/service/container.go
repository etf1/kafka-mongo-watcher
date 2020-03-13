package service

import (
	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/server"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// Container stores all the application services references
type Container struct {
	Cfg *config.Base

	logger logger.LoggerInterface

	techServer *server.TechServer

	mongoDB         *mongodriver.Database
	mongoCollection mongo.CollectionAdapter

	kafkaProducer *kafkaconfluent.Producer
	kafkaRecorder metrics.KafkaRecorder

	kafkaClient       kafka.Client
	kafkaProducerPool kafka.ProducerPool

	mongoReplayerClient mongo.Client
	mongoWatcherClient  mongo.Client
}

// NewContainer returns a dependency injection container that allows
// to retrieve services
func NewContainer(cfg *config.Base) *Container {
	return &Container{
		Cfg: cfg,
	}
}

package service

import (
	"context"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/server"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

// Container stores all the application services references
type Container struct {
	Cfg         *config.Base
	baseContext context.Context

	logger logger.LoggerInterface

	techServer *server.TechServer

	mongoDB         *mongodriver.Database
	mongoCollection mongo.CollectionAdapter

	replayProducer                       *mongo.ReplayProducer
	watchProducer                        *mongo.WatchProducer
	changeEventTransformerToKafkaMessage *mongo.ChangeEventKafkaMessageTransformer

	kafkaProducer *kafkaconfluent.Producer
	kafkaRecorder metrics.KafkaRecorder

	kafkaClient kafka.Client
}

// NewContainer returns a dependency injection container that allows
// to retrieve services
func NewContainer(cfg *config.Base, ctx context.Context) *Container {
	return &Container{
		Cfg:         cfg,
		baseContext: ctx,
	}
}

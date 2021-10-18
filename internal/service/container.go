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
	"go.opentelemetry.io/otel/trace"
)

// Container stores all the application services references
type Container struct {
	baseContext context.Context
	Cfg         *config.Base

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

	tracerProvider trace.TracerProvider
}

// NewContainer returns a dependency injection container that allows
// to retrieve services
func NewContainer(ctx context.Context, cfg *config.Base) *Container {
	return &Container{
		Cfg:         cfg,
		baseContext: ctx,
	}
}

package service

import (
	"context"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/worker"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Container struct {
	Ctx context.Context
	Cfg *config.Base

	logger logger.LoggerInterface

	mongoDB         *mongodriver.Database
	mongoCollection *mongodriver.Collection

	kafkaProducer *kafkaconfluent.Producer

	kafkaClient kafka.Client
	mongoClient mongo.Client

	worker worker.Worker
}

func NewContainer(ctx context.Context, cfg *config.Base) *Container {
	return &Container{
		Ctx: ctx,
		Cfg: cfg,
	}
}

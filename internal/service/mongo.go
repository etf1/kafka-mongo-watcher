package service

import (
	"context"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (container *Container) GetChangeEventProducer() mongo.ChangeEventProducer {
	if container.Cfg.Replay {
		return container.getReplayProducer().Produce
	} else {
		return container.getWatchProducer().GetProducer(container.getWatchOptions()...)
	}
}

func (container *Container) GetChangeEventKafkaMessageTransformer() *mongo.ChangeEventKafkaMessageTransformer {
	if container.changeEventTransformerToKafkaMessage == nil {
		container.changeEventTransformerToKafkaMessage = mongo.NewChangeEventKafkaMessageTransformer(
			container.Cfg.Topic,
			container.GetLogger(),
		)
	}
	return container.changeEventTransformerToKafkaMessage
}

func (container *Container) getReplayProducer() *mongo.ReplayProducer {
	if container.replayProducer == nil {
		container.replayProducer = mongo.NewReplayProducer(
			container.GetMongoCollection(),
			container.GetLogger(),
			container.Cfg.CustomPipeline,
		)
	}
	return container.replayProducer
}

func (container *Container) getWatchProducer() *mongo.WatchProducer {
	if container.watchProducer == nil {
		container.watchProducer = mongo.NewWatchProducer(
			container.GetMongoCollection(),
			container.GetLogger(),
			container.Cfg.CustomPipeline,
		)
	}
	return container.watchProducer
}

func (container *Container) getWatchOptions() []mongo.WatchOption {
	configOptions := container.Cfg.MongoDB.Options
	options := []mongo.WatchOption{
		mongo.WithBatchSize(configOptions.BatchSize),
		mongo.WithFullDocument(configOptions.FullDocument),
		mongo.WithMaxAwaitTime(configOptions.MaxAwaitTime),
		mongo.WithResumeAfter([]byte(configOptions.ResumeAfter)),
		mongo.WithMaxRetries(configOptions.WatchMaxRetries),
		mongo.WithRetryDelay(configOptions.WatchRetryDelay),
		mongo.WithIgnoreUpdateDescription(configOptions.IgnoreUpdateDescription),
	}

	switch {
	case configOptions.StartAtOperationTimeT > 0:
		startAt := primitive.Timestamp{
			T: configOptions.StartAtOperationTimeT,
			I: configOptions.StartAtOperationTimeI,
		}
		options = append(options, mongo.WithStartAtOperationTime(startAt))
	case configOptions.StartAtDelay > 0:
		from := time.Now().Add(-1 * configOptions.StartAtDelay)
		startAt := primitive.Timestamp{
			T: uint32(from.Unix()),
			I: 0,
		}
		options = append(options, mongo.WithStartAtOperationTime(startAt))
	}

	return options
}

func (container *Container) GetMongoCollection() mongo.CollectionAdapter {
	if container.mongoCollection == nil {
		container.mongoCollection = mongo.NewCollectionAdapter(
			container.GetMongoConnection().Collection(container.Cfg.MongoDB.CollectionName),
		)
	}
	return container.mongoCollection
}

func (container *Container) GetMongoConnection() *mongodriver.Database {
	if container.mongoDB == nil {
		mongoCfg := container.Cfg.MongoDB
		if db, err := newMongoClient(container.baseContext, container.GetLogger(), mongoCfg.URI, mongoCfg.DatabaseName, mongoCfg.ServerSelectionTimeout); err != nil {
			panic(err)
		} else {
			container.mongoDB = db
		}
	}
	return container.mongoDB
}

func newMongoClient(ctx context.Context, log logger.LoggerInterface, uri, database string, serverSelectionTimeout time.Duration) (*mongodriver.Database, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetReadPreference(readpref.Primary()).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetAppName(config.AppName)
	mongoClient, err := mongodriver.NewClient(opts)
	if err != nil {
		log.Error("Failed to create mongodb client", logger.String("uri", uri), logger.Error("error", err))
		return nil, err
	}

	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Error("Failed to connect to mongodb database", logger.String("uri", uri), logger.Error("error", err))
		return nil, err
	}

	log.Info("Connected to mongodb database", logger.String("uri", uri))

	db := mongoClient.Database(database)
	return db, nil
}

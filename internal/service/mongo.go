package service

import (
	"context"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (container *Container) GetChangeEvent(ctx context.Context) (changeEventChan chan *mongo.ChangeEvent, err error) {
	l := container.GetLogger()
	if container.Cfg.Replay {
		changeEventChan, err = container.getReplayProducer().Produce(ctx)
		if err != nil {
			l.Error("Mongo produce replay error", logger.Error("error", err))
		}
	} else {
		changeEventChan, err = container.getWatchProducer().GetProducer(container.getWatchOptions()...)(ctx)
		if err != nil {
			l.Error("Mongo produce watch error", logger.Error("error", err))
		}
	}
	return changeEventChan, err
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
		)
	}
	return container.replayProducer
}

func (container *Container) getWatchProducer() *mongo.WatchProducer {
	if container.watchProducer == nil {
		container.watchProducer = mongo.NewWatchProducer(
			container.GetMongoCollection(),
			container.GetLogger(),
		)
	}
	return container.watchProducer
}

func (container *Container) getWatchOptions() []mongo.WatchOption {
	configOptions := container.Cfg.MongoDB.Options
	return []mongo.WatchOption{
		mongo.WithBatchSize(configOptions.BatchSize),
		mongo.WithFullDocument(configOptions.FullDocument),
		mongo.WithMaxAwaitTime(configOptions.MaxAwaitTime),
	}
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
		if db, err := newMongoClient(container.baseContext, container.GetLogger(), mongoCfg.URI, mongoCfg.DatabaseName); err != nil {
			panic(err)
		} else {
			container.mongoDB = db
		}
	}
	return container.mongoDB
}

func newMongoClient(ctx context.Context, log logger.LoggerInterface, uri, database string) (*mongodriver.Database, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetReadPreference(readpref.Primary()).
		SetServerSelectionTimeout(2 * time.Second).
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

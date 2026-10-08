package service

import (
	"context"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func (container *Container) GetChangeEventProducer() mongo.ChangeEventProducer {
	if container.Cfg.Replay {
		return container.getReplayProducer().Produce
	}
	return func(ctx context.Context) (chan *mongo.ChangeEvent, error) {
		options, err := container.getWatchOptions(ctx)
		if err != nil {
			return nil, err
		}
		return container.getWatchProducer().GetProducer(options...)(ctx)
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

func (container *Container) getWatchOptions(ctx context.Context) ([]mongo.WatchOption, error) {
	configOptions := container.Cfg.MongoDB.Options
	options := []mongo.WatchOption{
		mongo.WithBatchSize(configOptions.BatchSize),
		mongo.WithFullDocument(configOptions.FullDocument),
		mongo.WithMaxAwaitTime(configOptions.MaxAwaitTime),
		mongo.WithMaxRetries(configOptions.WatchMaxRetries),
		mongo.WithRetryDelay(configOptions.WatchRetryDelay),
		mongo.WithIgnoreUpdateDescription(configOptions.IgnoreUpdateDescription),
		mongo.WithStartFromNowOnHistoryLost(configOptions.ResumeOnHistoryLost == config.ResumeOnHistoryLostNow),
	}

	// The stored checkpoint takes precedence over the configured starting point
	if tracker := container.GetCheckpointTracker(); tracker != nil {
		resumeToken, err := container.getCheckpointStore().Load(ctx)
		if err != nil {
			container.GetLogger().Error("Unable to load checkpoint", logger.Error("error", err))
			return nil, err
		}
		if len(resumeToken) > 0 {
			container.GetLogger().Info("Resuming change stream from checkpoint", logger.String("resume_token", resumeToken.String()))
			return append(options, mongo.WithStartAfter(resumeToken)), nil
		}
		container.GetLogger().Info("No checkpoint found, using configured starting point")
	}

	options = append(options, mongo.WithResumeAfter([]byte(configOptions.ResumeAfter)))

	switch {
	case configOptions.StartAtOperationTimeT > 0:
		startAt := bson.Timestamp{
			T: configOptions.StartAtOperationTimeT,
			I: configOptions.StartAtOperationTimeI,
		}
		options = append(options, mongo.WithStartAtOperationTime(startAt))
	case configOptions.StartAtDelay > 0:
		from := time.Now().Add(-1 * configOptions.StartAtDelay)
		startAt := bson.Timestamp{
			T: uint32(from.Unix()),
			I: 0,
		}
		options = append(options, mongo.WithStartAtOperationTime(startAt))
	}

	return options, nil
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
	mongoClient, err := mongodriver.Connect(opts)
	if err != nil {
		log.Error("Failed to create mongodb client", logger.String("uri", uri), logger.Error("error", err))
		return nil, err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Error("Failed to connect to mongodb database", logger.String("uri", uri), logger.Error("error", err))
		_ = mongoClient.Disconnect(ctx)
		return nil, err
	}

	log.Info("Connected to mongodb database", logger.String("uri", uri))

	db := mongoClient.Database(database)
	return db, nil
}

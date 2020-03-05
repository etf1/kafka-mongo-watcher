package service

import (
	"context"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (container *Container) GetMongoClient() mongo.Client {
	if container.mongoClient == nil {
		container.mongoClient = mongo.NewClient(
			container.Ctx,
			container.GetLogger(),
		)
	}

	return container.mongoClient
}

func (container *Container) GetMongoConnection() *mongodriver.Database {
	if container.mongoDB == nil {
		mongoCfg := container.Cfg.MongoDB

		if db, err := newMongoClient(container.Ctx, container.GetLogger(), mongoCfg.URI, mongoCfg.DatabaseName); err != nil {
			panic(err)
		} else {
			container.mongoDB = db
		}
	}

	return container.mongoDB
}

func (container *Container) GetMongoCollection() *mongodriver.Collection {
	if container.mongoCollection == nil {
		container.mongoCollection = container.GetMongoConnection().Collection(container.Cfg.MongoDB.CollectionName)
	}

	return container.mongoCollection
}

func newMongoClient(ctx context.Context, log logger.LoggerInterface, uri, database string) (*mongodriver.Database, error) {
	opts := options.Client().ApplyURI(uri).SetReadPreference(readpref.Primary()).SetServerSelectionTimeout(2 * time.Second)
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

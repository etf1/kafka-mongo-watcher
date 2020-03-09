package mongo

import (
	"context"
	"time"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDriverCursor interface {
	Decode(val interface{}) error
	Next(ctx context.Context) bool
	Close(ctx context.Context) error
}

type Option func(*client)

type Client interface {
	Replay(collection CollectionAdapter, itemsChan chan *WatchItem) error
	Watch(collection CollectionAdapter, itemsChan chan *WatchItem) error
}

type client struct {
	ctx                 context.Context
	logger              logger.LoggerInterface
	fullDocumentEnabled bool
	batchSize           int32
	maxAwaitTime        time.Duration
}

func NewClient(ctx context.Context, logger logger.LoggerInterface, options ...Option) *client {
	client := &client{
		ctx:                 ctx,
		logger:              logger,
		fullDocumentEnabled: false,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

func WithBatchSize(batchSize int32) Option {
	return func(c *client) {
		c.batchSize = batchSize
	}
}

func WithFullDocument(enabled bool) Option {
	return func(c *client) {
		c.fullDocumentEnabled = enabled
	}
}

func WithMaxAwaitTime(maxAwaitTime time.Duration) Option {
	return func(c *client) {
		c.maxAwaitTime = maxAwaitTime
	}
}

func (c *client) Replay(collection CollectionAdapter, itemsChan chan *WatchItem) error {
	pipeline := bson.A{
		bson.D{{Key: "$replaceRoot", Value: bson.D{
			{
				Key: "newRoot",
				Value: bson.M{
					"_id": bson.M{
						"_id":         "$_id",
						"copyingData": true,
					},
					"operationType": "insert",
					"ns": bson.M{
						"db":   "intref001",
						"coll": "video",
					},
					"documentKey": bson.M{
						"_id": "$_id",
					},
					"fullDocument": "$$ROOT",
				},
			},
		}}},
	}

	cursor, err := collection.Aggregate(c.ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(c.ctx)

	c.watchCursor(cursor, itemsChan)
	return nil
}

func (c *client) Watch(collection CollectionAdapter, itemsChan chan *WatchItem) error {
	var emptyPipeline = []bson.M{}

	println(c.maxAwaitTime)
	opts := &options.ChangeStreamOptions{
		BatchSize:    &c.batchSize,
		MaxAwaitTime: &c.maxAwaitTime,
	}
	if c.fullDocumentEnabled {
		opts.SetFullDocument(options.UpdateLookup)
	}

	cursor, err := collection.Watch(c.ctx, emptyPipeline, opts)
	if err != nil {
		c.logger.Error("Mongo client: An error has occured while watching collection", logger.String("collection", collection.Name()), logger.Error("error", err))
		return err
	}
	defer cursor.Close(c.ctx)

	for {
		c.watchCursor(cursor, itemsChan)
	}

	return nil
}

func (c *client) watchCursor(cursor MongoDriverCursor, itemsChan chan *WatchItem) {
	for cursor.Next(c.ctx) {
		var event changeEvent
		if err := cursor.Decode(&event); err != nil {
			c.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
		}

		if err := c.sendIntoChannel(event, itemsChan); err != nil {
			c.logger.Error("Mongo client: Unable to send document", logger.Error("error", err))
		}
	}
}

func (c *client) sendIntoChannel(event changeEvent, itemsChan chan *WatchItem) error {
	docID, err := event.documentID()
	if err != nil {
		c.logger.Error("Mongo client: Unable to extract document id from event", logger.Error("error", err))
		return err
	}
	jsonBytes, err := event.marshal()
	if err != nil {
		c.logger.Error("Mongo client: Unable to unmarshal change event to json", logger.Error("error", err))
		return err
	}

	itemsChan <- &WatchItem{
		Key:   []byte(docID),
		Value: jsonBytes,
	}

	return nil
}

package mongo

import (
	"context"
	"encoding/json"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDriverCursor interface {
	Decode(val interface{}) error
	Next(ctx context.Context) bool
	Close(ctx context.Context) error
}

type MongoDriverCollection interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (MongoDriverCursor, error)
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongodriver.ChangeStream, error)
	Name() string
}

type Client interface {
	Replay(collection MongoDriverCollection, itemsChan chan *WatchItem) error
	Watch(collection MongoDriverCollection, itemsChan chan *WatchItem) error
}

type client struct {
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewClient(ctx context.Context, logger logger.LoggerInterface) *client {
	return &client{
		ctx:    ctx,
		logger: logger,
	}
}

func (c *client) Replay(collection MongoDriverCollection, itemsChan chan *WatchItem) error {
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

func (c *client) Watch(collection MongoDriverCollection, itemsChan chan *WatchItem) error {
	var emptyPipeline = []bson.M{}

	opts := &options.ChangeStreamOptions{}
	opts.SetFullDocument(options.UpdateLookup) // TODO dans config

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
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			c.logger.Error("Mongo client: Unable to decode bson value from cursor", logger.Error("error", err))
		}

		if err := c.sendIntoChannel(result, itemsChan); err != nil {
			c.logger.Error("Mongo client: Unable to send document", logger.Error("error", err))
		}
	}
}

func (c *client) sendIntoChannel(result bson.M, itemsChan chan *WatchItem) error {
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		c.logger.Error("Mongo client: Unable to unmarshal bson to json", logger.Error("error", err))
		return err
	}

	var oplog OperationLog
	json.Unmarshal(jsonBytes, &oplog)

	itemsChan <- &WatchItem{
		Key:   []byte(oplog.DocumentKey.ID),
		Value: jsonBytes,
	}

	return nil
}

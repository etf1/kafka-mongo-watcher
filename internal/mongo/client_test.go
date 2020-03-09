package mongo

import (
	"context"
	"os"
	"testing"

	"github.com/gol4ng/logger"
	"github.com/gol4ng/logger/formatter"
	"github.com/gol4ng/logger/handler"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

var mongoCollection *mongodriver.Collection

func TestNewClient(t *testing.T) {
	context := context.Background()
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))
	clientTest := NewClient(context, logger)

	assert.Equal(t, logger, clientTest.logger)
	assert.Equal(t, context, clientTest.ctx)
}

func TestReplay(t *testing.T) {
	mongoClient := newClient()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mongoCollection := NewMockMongoDriverCollection(ctrl)
	mongoCursor := NewMockMongoDriverCursor(ctrl)

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

	// expected
	mongoCollection.EXPECT().Aggregate(mongoClient.ctx, pipeline).Return(mongoCursor, nil)
	mongoCursor.EXPECT().Next(mongoClient.ctx).Return(false)
	mongoCursor.EXPECT().Close(mongoClient.ctx).Return(nil)
	// test
	mongoClient.Replay(mongoCollection, nil)
}

func newClient() *client {
	context := context.Background()
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))
	return NewClient(context, logger)
}

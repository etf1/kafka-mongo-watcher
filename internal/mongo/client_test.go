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
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))
	clientTest := NewClient(logger)

	assert.Equal(t, logger, clientTest.logger)
}

func TestReplay(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoClient := NewClient(logger.NewNopLogger())
	mongoCollection := NewMockCollectionAdapter(ctrl)

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

	mongoCursor := NewMockMongoDriverCursor(ctrl)
	mongoCursor.EXPECT().Next(ctx)
	mongoCursor.EXPECT().Close(ctx)

	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	itemsChan := make(chan *WatchItem)

	// When - Then
	mongoClient.Replay(ctx, mongoCollection, itemsChan)
}

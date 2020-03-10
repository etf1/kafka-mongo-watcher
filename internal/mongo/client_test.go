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

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

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
						"db":   "test-db",
						"coll": "test-collection",
					},
					"documentKey": bson.M{
						"_id": "$_id",
					},
					"fullDocument": "$$ROOT",
				},
			},
		}}},
	}

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCursor.EXPECT().Next(ctx).Return(false)

	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	// When
	itemsChan, err := mongoClient.Replay(ctx, mongoCollection)

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

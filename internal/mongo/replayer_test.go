package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

var pipeline = bson.A{
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

func TestNewReplayer(t *testing.T) {
	logger := logger.NewNopLogger()
	replayer := NewReplayer(WithLogger(logger))

	assert.Equal(t, logger, replayer.logger)
}

func TestReplayerOplogs(t *testing.T) {
	ctx := context.Background()
	replayer := NewReplayer()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)
	mongoCursor.EXPECT().Next(ctx).Return(false)

	// When
	itemsChan, err := replayer.Oplogs(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestReplayerOplogsWithAggregateError(t *testing.T) {
	ctx := context.Background()
	replayer := NewReplayer()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, errors.New("aggregate error"))

	// When
	itemsChan, err := replayer.Oplogs(ctx, mongoCollection)

	// Then
	assert := assert.New(t)
	assert.NotNil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)

}

func TestReplayerOplogsWithResults(t *testing.T) {
	ctx := context.Background()
	replayer := NewReplayer()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	firstCall := mongoCursor.EXPECT().Next(ctx).Return(true)
	var event ChangeEvent
	mongoCursor.EXPECT().Decode(&event).Return(nil)
	mongoCursor.EXPECT().Next(ctx).Return(false).After(firstCall)

	itemsChan := make(chan *WatchItem)

	// When
	itemsChan, err := replayer.Oplogs(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestReplayerOplogsWithResultsWithDecodeError(t *testing.T) {
	ctx := context.Background()
	replayer := NewReplayer()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	firstCall := mongoCursor.EXPECT().Next(ctx).Return(true)
	var event ChangeEvent
	mongoCursor.EXPECT().Decode(&event).Return(errors.New("decode error"))
	mongoCursor.EXPECT().Next(ctx).Return(false).After(firstCall)

	itemsChan := make(chan *WatchItem)

	// When
	itemsChan, err := replayer.Oplogs(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

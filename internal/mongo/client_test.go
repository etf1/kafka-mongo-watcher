package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func TestNewClient(t *testing.T) {
	logger := logger.NewNopLogger()
	clientTest := NewClient(logger)
	assert.Equal(t, logger, clientTest.logger)
}

func TestReplay(t *testing.T) {
	ctx := context.Background()
	mongoClient := NewClient(logger.NewNopLogger())

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
	itemsChan, err := mongoClient.Replay(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestReplayWithResults(t *testing.T) {
	ctx := context.Background()
	mongoClient := NewClient(logger.NewNopLogger())

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
	itemsChan, err := mongoClient.Replay(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestWatch(t *testing.T) {
	ctx := context.Background()
	logger := logger.NewNopLogger()
	mongoClient := NewClient(logger, WithBatchSize(int32(10)), WithFullDocument(true), WithMaxAwaitTime(time.Duration(10)))
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)
	opts := &options.ChangeStreamOptions{
		BatchSize:    &batchSize,
		MaxAwaitTime: &maxAwaitTime,
	}
	opts.SetFullDocument(options.UpdateLookup)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockDriverCursor(ctrl)

	var emptyPipeline = []bson.M{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, nil)
	mongoCursor.EXPECT().Next(ctx).Return(false).AnyTimes()

	// When
	itemsChan, err := mongoClient.Watch(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestWatchWithResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := logger.NewNopLogger()
	mongoClient := NewClient(logger, WithBatchSize(int32(10)), WithFullDocument(true), WithMaxAwaitTime(time.Duration(10)))
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)
	opts := &options.ChangeStreamOptions{
		BatchSize:    &batchSize,
		MaxAwaitTime: &maxAwaitTime,
	}
	opts.SetFullDocument(options.UpdateLookup)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockDriverCursor(ctrl)

	var emptyPipeline = []bson.M{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, nil)

	mongoCursor.EXPECT().Next(ctx).Return(true).AnyTimes()
	var event ChangeEvent
	mongoCursor.EXPECT().Decode(&event).Return(nil).AnyTimes()

	// When
	itemsChan, err := mongoClient.Watch(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

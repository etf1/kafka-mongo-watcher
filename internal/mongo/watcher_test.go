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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewWatcher(t *testing.T) {
	logger := logger.NewNopLogger()
	watcher := NewWatcher(WithLogger(logger))

	assert.Equal(t, logger, watcher.logger)
}

func TestWatcherOplogs(t *testing.T) {
	ctx := context.Background()
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	watcher := NewWatcher().
		WithOptions(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))

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
	itemsChan, err := watcher.Oplogs(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestWatcherOplogsWithWatchError(t *testing.T) {
	ctx := context.Background()
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	watcher := NewWatcher().
		WithOptions(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))

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
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, errors.New("aggregate error"))
	mongoCollection.EXPECT().Name()

	// When
	itemsChan, err := watcher.Oplogs(ctx, mongoCollection)

	// Then
	assert := assert.New(t)

	assert.NotNil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

func TestWatcherOplogsWithResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	ctx := context.Background()
	watcher := NewWatcher().
		WithOptions(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))

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
	objID, _ := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	mongoCursor.EXPECT().Decode(&event).Return(nil).SetArg(0, ChangeEvent{
		Operation:   "insert",
		DocumentKey: documentKey{ID: objID},
	}).AnyTimes()

	// When
	itemsChan, err := watcher.Oplogs(ctx, mongoCollection)
	time.Sleep(100 * time.Millisecond) // Need to wait for the watchCursor goroutine to be executed

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(itemsChan), 0)
	assert.Equal(len(itemsChan), 0)
}

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
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewWatcher(t *testing.T) {
	logger := logger.NewNopLogger()
	watcher := NewWatcher(WithLogger(logger))

	assert.Equal(t, logger, watcher.logger)
}

func TestWatcherOplogsWhenNoResults(t *testing.T) {
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
	events, err := watcher.Oplogs(ctx, mongoCollection)

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatcherOplogsWhenWatchError(t *testing.T) {
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

	var expectedErr = errors.New("aggregate error")
	mongoCollection.EXPECT().Watch(ctx, []bson.M{}, opts).Return(mongoCursor, expectedErr)
	mongoCollection.EXPECT().Name()

	// When
	events, err := watcher.Oplogs(ctx, mongoCollection)

	// Then
	assert := assert.New(t)

	assert.Equal(expectedErr, err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatcherOplogsWhenHaveResults(t *testing.T) {
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
	var e ChangeEvent
	mongoCursor.EXPECT().Decode(&e).Return(nil).AnyTimes()

	// When
	events, err := watcher.Oplogs(ctx, mongoCollection)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.IsType(new(ChangeEvent), event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

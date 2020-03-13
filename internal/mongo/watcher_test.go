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

func TestWatchProduceWhenNoResults(t *testing.T) {
	ctx := context.Background()
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

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))(ctx)

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatchProduceWhenWatchError(t *testing.T) {
	ctx := context.Background()
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

	var expectedErr = errors.New("aggregate error")
	mongoCollection.EXPECT().Watch(ctx, []bson.M{}, opts).Return(mongoCursor, expectedErr)
	mongoCollection.EXPECT().Name()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))(ctx)

	// Then
	assert := assert.New(t)

	assert.Equal(expectedErr, err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatchProduceWhenHaveResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	ctx := context.Background()

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

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime))(ctx)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.IsType(new(ChangeEvent), event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

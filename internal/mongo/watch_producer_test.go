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
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, nil)
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	mongoCursor.EXPECT().Next(ctx).Return(false).AnyTimes()
	mongoCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()
	mongoCursor.EXPECT().ResumeToken().Return(bson.Raw{}).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime), WithMaxRetries(0))(ctx)

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
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}

	var expectedErr = errors.New("aggregate error")
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, expectedErr)
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	mongoCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime), WithMaxRetries(0))(ctx)

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
	resumeAfter := []byte(`{"_data":"1234567890987654321"}`)
	startAtOperationTime := primitive.Timestamp{
		I: uint32(10),
		T: uint32(10),
	}

	ctx := context.Background()

	opts := &options.ChangeStreamOptions{
		BatchSize:            &batchSize,
		MaxAwaitTime:         &maxAwaitTime,
		StartAtOperationTime: &startAtOperationTime,
	}
	opts.SetFullDocument(options.UpdateLookup)
	opts.SetResumeAfter(bson.M{"_data": "1234567890987654321"})

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, opts).Return(mongoCursor, nil).AnyTimes()
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()

	mongoCursor.EXPECT().ID().Return(int64(1234)).AnyTimes()
	mongoCursor.EXPECT().Err().Return(nil).AnyTimes()
	mongoCursor.EXPECT().ResumeToken().Return(bson.Raw{}).AnyTimes()
	mongoCursor.EXPECT().Next(ctx).Return(true).AnyTimes()
	var e ChangeEvent
	mongoCursor.EXPECT().Decode(&e).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	// When
	events, err := watcher.GetProducer(
		WithBatchSize(batchSize),
		WithFullDocument(true),
		WithMaxAwaitTime(maxAwaitTime),
		WithResumeAfter(resumeAfter),
		WithStartAtOperationTime(startAtOperationTime),
		WithMaxRetries(0),
	)(ctx)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.IsType(new(ChangeEvent), event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatchProduceWhenCustomPipeline(t *testing.T) {
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
	mongoCursor := NewMockStreamCursor(ctrl)

	var pipeline = bson.A{}

	customPipeline := "[ { \"$match\": {\"fullDocument.active\": true} } ]"
	pipeline = append(bson.A{
		bson.D{
			{
				Key: "$match",
				Value: bson.D{
					{
						Key:   "fullDocument.active",
						Value: true,
					},
				},
			},
		},
	}, pipeline...)

	mongoCollection.EXPECT().Watch(ctx, pipeline, opts).Return(mongoCursor, nil)
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	mongoCursor.EXPECT().Next(ctx).Return(false).AnyTimes()
	mongoCursor.EXPECT().ID().Return(int64(1234)).AnyTimes()
	mongoCursor.EXPECT().Err().Return(nil).AnyTimes()
	mongoCursor.EXPECT().ResumeToken().Return(bson.Raw{}).AnyTimes()
	mongoCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), customPipeline)

	// When
	events, err := watcher.GetProducer(WithBatchSize(batchSize), WithFullDocument(true), WithMaxAwaitTime(maxAwaitTime), WithMaxRetries(0))(ctx)

	// Then
	assert := assert.New(t)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestWatchProduceWhenCtxCanceledDuringSend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := &options.ChangeStreamOptions{
		BatchSize:    &batchSize,
		MaxAwaitTime: &maxAwaitTime,
	}
	opts.SetFullDocument(options.UpdateLookup)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(gomock.Any(), emptyPipeline, opts).Return(mongoCursor, nil)
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()

	mongoCursor.EXPECT().ID().Return(int64(1234)).AnyTimes()
	mongoCursor.EXPECT().Err().Return(nil).AnyTimes()
	mongoCursor.EXPECT().ResumeToken().Return(bson.Raw{}).AnyTimes()
	mongoCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()
	// Next always returns true: the goroutine will loop and try to send events.
	mongoCursor.EXPECT().Next(gomock.Any()).Return(true).AnyTimes()
	var e ChangeEvent
	mongoCursor.EXPECT().Decode(&e).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	events, err := watcher.GetProducer(
		WithBatchSize(batchSize),
		WithFullDocument(true),
		WithMaxAwaitTime(maxAwaitTime),
		WithMaxRetries(0),
	)(ctx)

	assert := assert.New(t)
	assert.Nil(err)

	// Cancel the context while nobody reads from events.
	// The goroutine must exit via the ctx.Done() branch of the select in sendEvents.
	cancel()

	// Drain the events channel: it must close within a reasonable time.
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-events:
			if !ok {
				return // channel closed — goroutine exited correctly
			}
		case <-timeout:
			t.Fatal("events channel was not closed after context cancellation")
		}
	}
}

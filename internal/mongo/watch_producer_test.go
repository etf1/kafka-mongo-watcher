package mongo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type changeStreamOptionsMatcher struct {
	expected options.ChangeStreamOptions
}

func (m changeStreamOptionsMatcher) Matches(value interface{}) bool {
	actual, ok := value.(options.Lister[options.ChangeStreamOptions])
	if !ok {
		return false
	}

	var actualOptions options.ChangeStreamOptions
	for _, setter := range actual.List() {
		if err := setter(&actualOptions); err != nil {
			return false
		}
	}

	return reflect.DeepEqual(m.expected, actualOptions)
}

func (m changeStreamOptionsMatcher) String() string {
	return "matches change stream options"
}

func matchesChangeStreamOptions(builder options.Lister[options.ChangeStreamOptions]) gomock.Matcher {
	var expected options.ChangeStreamOptions
	for _, setter := range builder.List() {
		if err := setter(&expected); err != nil {
			panic(err)
		}
	}
	return changeStreamOptionsMatcher{expected: expected}
}

func TestWatchProduceWhenNoResults(t *testing.T) {
	ctx := context.Background()
	batchSize := int32(10)
	maxAwaitTime := time.Duration(10)

	opts := options.ChangeStream().
		SetBatchSize(batchSize).
		SetMaxAwaitTime(maxAwaitTime).
		SetFullDocument(options.UpdateLookup)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, matchesChangeStreamOptions(opts)).Return(mongoCursor, nil)
	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	mongoCursor.EXPECT().Next(ctx).Return(false).AnyTimes()
	mongoCursor.EXPECT().Err().Return(nil).AnyTimes()
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

	opts := options.ChangeStream().
		SetBatchSize(batchSize).
		SetMaxAwaitTime(maxAwaitTime).
		SetFullDocument(options.UpdateLookup)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}

	var expectedErr = errors.New("aggregate error")
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, matchesChangeStreamOptions(opts)).Return(mongoCursor, expectedErr)
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
	startAtOperationTime := bson.Timestamp{
		I: uint32(10),
		T: uint32(10),
	}

	ctx := context.Background()

	opts := options.ChangeStream().
		SetBatchSize(batchSize).
		SetMaxAwaitTime(maxAwaitTime).
		SetFullDocument(options.UpdateLookup).
		// resumeAfter takes precedence: MongoDB only accepts one starting point
		SetResumeAfter(bson.M{"_data": "1234567890987654321"})

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(ctx, emptyPipeline, matchesChangeStreamOptions(opts)).Return(mongoCursor, nil).AnyTimes()
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

	opts := options.ChangeStream().
		SetBatchSize(batchSize).
		SetMaxAwaitTime(maxAwaitTime).
		SetFullDocument(options.UpdateLookup)

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

	mongoCollection.EXPECT().Watch(ctx, pipeline, matchesChangeStreamOptions(opts)).Return(mongoCursor, nil)
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

	opts := options.ChangeStream().
		SetBatchSize(batchSize).
		SetMaxAwaitTime(maxAwaitTime).
		SetFullDocument(options.UpdateLookup)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCursor := NewMockStreamCursor(ctrl)

	var emptyPipeline = bson.A{}
	mongoCollection.EXPECT().Watch(gomock.Any(), emptyPipeline, matchesChangeStreamOptions(opts)).Return(mongoCursor, nil)
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

func TestWatchProduceReconnectUsesLastEventClusterTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAtOperationTime := bson.Timestamp{T: 10, I: 1}
	lastEventClusterTime := bson.Timestamp{T: 20, I: 3}

	initialOpts := options.ChangeStream().
		SetBatchSize(0).
		SetMaxAwaitTime(0).
		SetStartAtOperationTime(&startAtOperationTime)
	// On reconnection, the stream restarts at the last sent event
	reconnectOpts := options.ChangeStream().
		SetBatchSize(0).
		SetMaxAwaitTime(0).
		SetStartAtOperationTime(&lastEventClusterTime)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	firstCursor := NewMockStreamCursor(ctrl)
	secondCursor := NewMockStreamCursor(ctrl)

	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	gomock.InOrder(
		mongoCollection.EXPECT().Watch(ctx, bson.A{}, matchesChangeStreamOptions(initialOpts)).Return(firstCursor, nil),
		mongoCollection.EXPECT().Watch(ctx, bson.A{}, matchesChangeStreamOptions(reconnectOpts)).Return(secondCursor, nil),
	)

	// first cursor sends one event then breaks
	gomock.InOrder(
		firstCursor.EXPECT().Next(ctx).Return(true),
		firstCursor.EXPECT().Next(ctx).Return(false),
	)
	firstCursor.EXPECT().Decode(gomock.Any()).DoAndReturn(func(val interface{}) error {
		val.(*ChangeEvent).clusterTimestamp = lastEventClusterTime
		return nil
	})
	firstCursor.EXPECT().ID().Return(int64(1))
	firstCursor.EXPECT().Err().Return(errors.New("connection lost"))
	firstCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	// second cursor sends events until the context is canceled
	secondCursor.EXPECT().Next(ctx).Return(true).AnyTimes()
	secondCursor.EXPECT().Decode(gomock.Any()).Return(nil).AnyTimes()
	secondCursor.EXPECT().ID().Return(int64(1)).AnyTimes()
	secondCursor.EXPECT().Err().Return(nil).AnyTimes()
	secondCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	events, err := watcher.GetProducer(
		WithStartAtOperationTime(startAtOperationTime),
		WithMaxRetries(1),
		WithRetryDelay(time.Millisecond),
	)(ctx)
	assert.Nil(t, err)

	<-events // from first cursor
	<-events // from second cursor
	cancel()
	for range events {
	}
}

func TestWatchProduceReconnectKeepsPreviousPositionWhenNoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checkpoint := bson.Timestamp{T: 10, I: 1}
	opts := options.ChangeStream().
		SetBatchSize(0).
		SetMaxAwaitTime(0).
		SetStartAtOperationTime(&checkpoint)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	firstCursor := NewMockStreamCursor(ctrl)
	secondCursor := NewMockStreamCursor(ctrl)

	mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
	gomock.InOrder(
		mongoCollection.EXPECT().Watch(ctx, bson.A{}, matchesChangeStreamOptions(opts)).Return(firstCursor, nil),
		mongoCollection.EXPECT().Watch(ctx, bson.A{}, matchesChangeStreamOptions(opts)).Return(secondCursor, nil),
	)

	// first cursor breaks before sending any event
	firstCursor.EXPECT().Next(ctx).Return(false)
	firstCursor.EXPECT().Err().Return(errors.New("connection lost"))
	firstCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	secondCursor.EXPECT().Next(ctx).Return(true).AnyTimes()
	secondCursor.EXPECT().Decode(gomock.Any()).Return(nil).AnyTimes()
	secondCursor.EXPECT().ID().Return(int64(1)).AnyTimes()
	secondCursor.EXPECT().Err().Return(nil).AnyTimes()
	secondCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

	watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")

	events, err := watcher.GetProducer(WithStartAtOperationTime(checkpoint), WithMaxRetries(1))(ctx)
	assert.Nil(t, err)

	<-events
	cancel()
	for range events {
	}
}

func TestWatchProduceWhenResumePointLost(t *testing.T) {
	historyLost := mongodriver.CommandError{Code: errorCodeChangeStreamHistoryLost, Message: "history lost"}
	checkpoint := bson.Timestamp{T: 10, I: 1}

	t.Run("fail without retrying", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ctx := context.Background()

		mongoCollection := NewMockCollectionAdapter(ctrl)
		mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
		mongoCollection.EXPECT().Watch(ctx, bson.A{}, gomock.Any()).Return(nil, historyLost).Times(1)

		watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")
		_, err := watcher.GetProducer(WithStartAtOperationTime(checkpoint), WithMaxRetries(3))(ctx)

		assert.True(t, IsResumePointLost(err))
	})

	t.Run("restart from now", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		fromNow := options.ChangeStream().SetBatchSize(0).SetMaxAwaitTime(0)

		mongoCollection := NewMockCollectionAdapter(ctrl)
		mongoCursor := NewMockStreamCursor(ctrl)
		mongoCollection.EXPECT().Name().Return("coll").AnyTimes()
		gomock.InOrder(
			mongoCollection.EXPECT().Watch(ctx, bson.A{}, gomock.Any()).Return(nil, historyLost),
			mongoCollection.EXPECT().Watch(ctx, bson.A{}, matchesChangeStreamOptions(fromNow)).Return(mongoCursor, nil),
		)
		mongoCursor.EXPECT().Next(ctx).Return(false).AnyTimes()
		mongoCursor.EXPECT().Err().Return(nil).AnyTimes()
		mongoCursor.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

		watcher := NewWatchProducer(mongoCollection, logger.NewNopLogger(), "")
		events, err := watcher.GetProducer(WithStartAtOperationTime(checkpoint), WithStartFromNowOnHistoryLost(true), WithMaxRetries(0))(ctx)

		assert.Nil(t, err)
		for range events {
		}
	})
}

func TestIsResumePointLost(t *testing.T) {
	assert.True(t, IsResumePointLost(ErrResumePointTooOld))
	assert.True(t, IsResumePointLost(fmt.Errorf("wrapped: %w", ErrResumePointTooOld)))
	assert.True(t, IsResumePointLost(mongodriver.CommandError{Code: errorCodeChangeStreamHistoryLost}))
	assert.False(t, IsResumePointLost(mongodriver.CommandError{Code: 11600}))
	assert.False(t, IsResumePointLost(errors.New("connection lost")))
}

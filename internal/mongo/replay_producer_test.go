package mongo

import (
	"context"
	"errors"
	"testing"

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

func TestReplayProduceWhenNoResults(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCursor.EXPECT().Next(ctx).Return(false)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	replayer := NewReplayProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := replayer.Produce(ctx)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.Nil(event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestReplayProduceWhenAggregateError(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")

	mongoCursor := NewMockDriverCursor(ctrl)
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, errors.New("aggregate error"))

	replayer := NewReplayProducer(mongoCollection, logger.NewNopLogger())
	// When
	events, err := replayer.Produce(ctx)

	// Then
	assert := assert.New(t)
	assert.NotNil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)

}

func TestReplayProduceWhenHaveResults(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCursor := NewMockDriverCursor(ctrl)
	firstCall := mongoCursor.EXPECT().Next(ctx).Return(true)
	mongoCursor.EXPECT().Next(ctx).Return(false).After(firstCall)

	var e ChangeEvent
	mongoCursor.EXPECT().Decode(&e).Return(nil)

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	replayer := NewReplayProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := replayer.Produce(ctx)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.IsType(new(ChangeEvent), event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

func TestReplayProduceWhenResultsWithDecodeError(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mongoDatabase := NewMockDriverDatabase(ctrl)
	mongoDatabase.EXPECT().Name().Return("test-db")

	mongoCursor := NewMockDriverCursor(ctrl)
	firstCall := mongoCursor.EXPECT().Next(ctx).Return(true)
	mongoCursor.EXPECT().Next(ctx).Return(false).After(firstCall)

	var e ChangeEvent
	mongoCursor.EXPECT().Decode(&e).Return(errors.New("decode error"))

	mongoCollection := NewMockCollectionAdapter(ctrl)
	mongoCollection.EXPECT().Database().Return(mongoDatabase)
	mongoCollection.EXPECT().Name().Return("test-collection")
	mongoCollection.EXPECT().Aggregate(ctx, pipeline).Return(mongoCursor, nil)

	replayer := NewReplayProducer(mongoCollection, logger.NewNopLogger())

	// When
	events, err := replayer.Produce(ctx)

	// Then
	assert := assert.New(t)

	event := <-events
	assert.Nil(event)

	assert.Nil(err)
	assert.Equal(cap(events), 0)
	assert.Equal(len(events), 0)
}

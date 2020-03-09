package worker

import (
	"context"
	"testing"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5
	timeout := 5 * time.Second

	logger := logger.NewNopLogger()

	mongoClient := mongo.NewMockClient(ctrl)
	kafkaClient := kafka.NewMockClient(ctrl)

	// When
	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)

	// Then
	assert := assert.New(t)
	assert.IsType(new(worker), workerInstance)

	assert.Equal(ctx, workerInstance.ctx)
	assert.Equal(mongoClient, workerInstance.mongoClient)
	assert.Equal(mongoClient, workerInstance.mongoClient)
	assert.Equal(number, workerInstance.number)
	assert.Equal(timeout, workerInstance.timeout)
	assert.Equal(0, workerInstance.numberRunning)
}

func TestClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5
	timeout := 5 * time.Second

	logger := logger.NewNopLogger()

	mongoClient := mongo.NewMockClient(ctrl)
	kafkaClient := kafka.NewMockClient(ctrl)

	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)

	// When
	workerInstance.Close()

	// Then
	assert := assert.New(t)
	assert.Equal(workerInstance.numberRunning, 0)
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

// func TestReplay(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	// Given
// 	ctx := context.Background()
// 	number := 5
// 	timeout := 1 * time.Millisecond

// 	logger := logger.NewNopLogger()

// 	mongoClient := mongo.NewMockClient(ctrl)
// 	kafkaClient := kafka.NewMockClient(ctrl)

// 	// collection := mongo.NewC

// 	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
// 	mongoClient.EXPECT().Replay(collection, workerInstance.itemsChan)

// 	// When - Then
// 	workerInstance.Replay(collection)

// 	// Then
// 	assert := assert.New(t)
// 	assert.Equal(workerInstance.numberRunning, 0)
// 	assert.Equal(cap(workerInstance.itemsChan), 0)
// 	assert.Equal(len(workerInstance.itemsChan), 0)
// }

// func TestReplayWhenNoDocument(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	// Given
// 	ctx := context.Background()
// 	number := 5
// 	timeout := 1 * time.Millisecond

// 	logger := logger.NewNopLogger()

// 	mongoClient := mongo.NewMockClient(ctrl)
// 	kafkaClient := kafka.NewMockClient(ctrl)

// 	// collection := mongo.NewC

// 	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
// 	mongoClient.EXPECT().Replay(collection, workerInstance.itemsChan)

// 	// When - Then
// 	workerInstance.Replay(collection)

// 	// Then
// 	assert := assert.New(t)
// 	assert.Equal(workerInstance.numberRunning, 0)
// 	assert.Equal(cap(workerInstance.itemsChan), 0)
// 	assert.Equal(len(workerInstance.itemsChan), 0)
// }

// func TestWatchAndProduce(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	// Given
// 	ctx := context.Background()
// 	number := 5
// 	timeout := 1 * time.Millisecond

// 	logger := logger.NewNopLogger()

// 	mongoClient := mongo.NewMockClient(ctrl)
// 	kafkaClient := kafka.NewMockClient(ctrl)

// 	// collection := mongo.NewC

// 	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
// 	mongoClient.EXPECT().Watch(collection, workerInstance.itemsChan)

// 	// When - Then
// 	workerInstance.WatchAndProduce(collection)

// 	// Then
// 	assert := assert.New(t)
// 	assert.Equal(workerInstance.numberRunning, 0)
// 	assert.Equal(cap(workerInstance.itemsChan), 0)
// 	assert.Equal(len(workerInstance.itemsChan), 0)
// }

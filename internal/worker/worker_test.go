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
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
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
	assert.Equal(int32(0), workerInstance.numberRunning)
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
	assert.Equal(workerInstance.numberRunning, int32(0))
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

func TestReplayWhenMongoEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5
	timeout := 300 * time.Millisecond

	logger := logger.NewNopLogger()

	collection := mongo.NewMockCollectionAdapter(ctrl)

	mongoClient := mongo.NewMockClient(ctrl)

	topic := "my-test-topic"
	kafkaClient := kafka.NewMockClient(ctrl)
	kafkaClient.EXPECT().Produce(&kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
		Key:            []byte(`1`),
		Value:          []byte(`A test value to be sent`),
	})

	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
	mongoClient.EXPECT().Replay(collection, workerInstance.itemsChan)

	// Run this asynchronously to simulate MongoDB sending events
	go func() {
		workerInstance.itemsChan <- &mongo.WatchItem{
			Key:   []byte(`1`),
			Value: []byte(`A test value to be sent`),
		}
	}()

	// When - Then
	workerInstance.Replay(collection, topic)

	// Then
	assert := assert.New(t)
	assert.Equal(workerInstance.numberRunning, int32(0))
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

func TestReplayWhenNoMongoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5
	timeout := 1 * time.Millisecond

	logger := logger.NewNopLogger()

	mongoClient := mongo.NewMockClient(ctrl)
	kafkaClient := kafka.NewMockClient(ctrl)

	collection := mongo.NewMockCollectionAdapter(ctrl)

	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
	mongoClient.EXPECT().Replay(collection, workerInstance.itemsChan)

	// When - Then
	workerInstance.Replay(collection, "my-test-topic")

	// Then
	assert := assert.New(t)
	assert.Equal(workerInstance.numberRunning, int32(0))
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

func TestWatchAndProduceWhenMongoEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx, cancel := context.WithCancel(context.Background())
	number := 5
	timeout := 300 * time.Millisecond

	logger := logger.NewNopLogger()

	collection := mongo.NewMockCollectionAdapter(ctrl)

	mongoClient := mongo.NewMockClient(ctrl)

	topic := "my-test-topic"
	kafkaClient := kafka.NewMockClient(ctrl)
	kafkaClient.EXPECT().Produce(&kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
		Key:            []byte(`1`),
		Value:          []byte(`A test value to be sent`),
	})

	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
	mongoClient.EXPECT().Watch(collection, workerInstance.itemsChan)

	// Run this asynchronously to simulate MongoDB sending events
	go func() {
		workerInstance.itemsChan <- &mongo.WatchItem{
			Key:   []byte(`1`),
			Value: []byte(`A test value to be sent`),
		}

		// Cancel context because there is no timeout when using WatchAndProduce()
		cancel()
	}()

	// When - Then
	workerInstance.WatchAndProduce(collection, topic)

	// Then
	assert := assert.New(t)
	assert.Equal(workerInstance.numberRunning, int32(0))
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

func TestWatchAndProduceWhenNoMongoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx, cancel := context.WithCancel(context.Background())
	number := 5
	timeout := 1 * time.Millisecond

	logger := logger.NewNopLogger()

	mongoClient := mongo.NewMockClient(ctrl)
	kafkaClient := kafka.NewMockClient(ctrl)

	collection := mongo.NewMockCollectionAdapter(ctrl)

	workerInstance := New(ctx, logger, mongoClient, kafkaClient, number, timeout)
	mongoClient.EXPECT().Watch(collection, workerInstance.itemsChan)

	// Run this asynchronously
	go func() {
		// Cancel context because there is no timeout when using WatchAndProduce()
		cancel()
	}()

	// When - Then
	workerInstance.WatchAndProduce(collection, "my-test-topic")

	// Then
	assert := assert.New(t)
	assert.Equal(workerInstance.numberRunning, int32(0))
	assert.Equal(cap(workerInstance.itemsChan), 0)
	assert.Equal(len(workerInstance.itemsChan), 0)
}

package kafka

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewProducerPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	number := 5

	logger := logger.NewNopLogger()

	kafkaClient := NewMockClient(ctrl)

	// When
	producer := NewProducerPool(logger, kafkaClient, number)

	// Then
	assert := assert.New(t)
	assert.IsType(new(producerPool), producer)

	assert.Equal(number, producer.number)
	assert.Equal(int32(0), producer.numberRunning)
}

func TestClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	number := 5

	logger := logger.NewNopLogger()

	kafkaClient := NewMockClient(ctrl)

	producer := NewProducerPool(logger, kafkaClient, number)
	atomic.AddInt32(&producer.numberRunning, 1)
	producer.waitGroup.Add(1)

	// When
	time.Sleep(10 * time.Millisecond)
	producer.Close()

	// Then
	assert := assert.New(t)
	assert.Equal(producer.numberRunning, int32(0))
}

func TestProduceWhenMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5

	logger := logger.NewNopLogger()

	topic := "my-test-topic"
	messages := make(chan *Message)

	go func() {
		messages <- &Message{
			Topic: topic,
			Key:   []byte(`1`),
			Value: []byte(`A test value to be sent`),
		}
		close(messages)
	}()

	kafkaClient := NewMockClient(ctrl)
	kafkaClient.EXPECT().Produce(&kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
		Key:            []byte(`1`),
		Value:          []byte(`A test value to be sent`),
	})

	producer := NewProducerPool(logger, kafkaClient, number)

	// When - Then
	producer.Produce(ctx, messages)

	// Then
	assert := assert.New(t)
	assert.Equal(producer.numberRunning, int32(0))
	assert.Equal(cap(messages), 0)
	assert.Equal(len(messages), 0)
}

func TestProduceWhenNoMongoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5

	logger := logger.NewNopLogger()

	messages := make(chan *Message)
	close(messages)

	kafkaClient := NewMockClient(ctrl)

	producer := NewProducerPool(logger, kafkaClient, number)

	// When - Then
	producer.Produce(ctx, messages)

	// Then
	assert := assert.New(t)
	assert.Equal(producer.numberRunning, int32(0))
	assert.Equal(cap(messages), 0)
	assert.Equal(len(messages), 0)
}

func TestProduceWhenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	ctx := context.Background()
	number := 5

	logger := logger.NewNopLogger()

	messages := make(chan *Message)
	close(messages)

	kafkaClient := NewMockClient(ctrl)

	producer := NewProducerPool(logger, kafkaClient, number)

	// When - Then
	producer.Produce(ctx, messages)

	// Then
	assert := assert.New(t)
	assert.Equal(producer.numberRunning, int32(0))
	assert.Equal(cap(messages), 0)
	assert.Equal(len(messages), 0)
}

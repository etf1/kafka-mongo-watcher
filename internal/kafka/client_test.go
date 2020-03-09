package kafka

import (
	"os"
	"testing"

	"github.com/gol4ng/logger"
	"github.com/gol4ng/logger/formatter"
	"github.com/gol4ng/logger/handler"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewClient(t *testing.T) {
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))
	producer, err := kafkaconfluent.NewProducer(&kafkaconfluent.ConfigMap{"bootstrap.servers": "test"})
	clientTest := NewClient(logger, producer)

	assert.Nil(t, err)
	assert.IsType(t, new(client), clientTest)

	assert.Equal(t, logger, clientTest.logger)
	assert.Equal(t, producer, clientTest.producer)
}

func TestProduce(t *testing.T) {
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))

	// message
	topic := "topic"
	message := &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
		Key:            []byte("key"),
		Value:          []byte("value"),
	}

	// kafka producer mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kafkaProducer := NewMockKafkaProducer(ctrl)

	clientTest := NewClient(logger, kafkaProducer)

	// expected call
	kafkaProducer.EXPECT().Produce(message, nil).Return(nil).Times(1)

	// test
	err := clientTest.Produce(message)
	assert.Nil(t, err)
}

func TestClose(t *testing.T) {
	logger := logger.NewLogger(handler.Stream(os.Stdout, formatter.NewDefaultFormatter()))

	// kafka producer mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kafkaProducer := NewMockKafkaProducer(ctrl)

	clientTest := NewClient(logger, kafkaProducer)

	// expected call
	kafkaProducer.EXPECT().Close().Return().Times(1)

	// test
	clientTest.Close()
}

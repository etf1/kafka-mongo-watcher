package kafka

import (
	"errors"
	"testing"

	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewClientLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)

	logger := logger.NewNopLogger()

	// When
	cli := NewClientLogger(client, logger)

	assert.IsType(t, new(clientLogger), cli)

	assert.Equal(t, client, cli.client)
	assert.Equal(t, logger, cli.logger)
}

func TestClientLoggerProduceWhenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	var topicName = "test-topic"
	message := &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{
			Topic: &topicName,
		},
	}

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(message)

	logger := logger.NewNopLogger()
	cli := NewClientLogger(client, logger)

	// When
	err := cli.Produce(message)

	// Then
	assert.Nil(t, err)
}

func TestClientLoggerProduceWhenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	expectedErr := errors.New("an unexpected error occurred")

	var topicName = "test-topic"
	message := &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{
			Topic: &topicName,
		},
	}

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(message).Return(expectedErr)

	logger := logger.NewNopLogger()
	cli := NewClientLogger(client, logger)

	// When
	err := cli.Produce(message)

	// Then
	assert.Equal(t, err, expectedErr)
}

func TestClientLoggerClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)
	client.EXPECT().Close()

	logger := logger.NewNopLogger()
	cli := NewClientLogger(client, logger)

	// When - Then
	cli.Close()
}

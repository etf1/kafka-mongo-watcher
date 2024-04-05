package kafka

import (
	"testing"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gol4ng/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

func TestClientLoggerProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	messages := make(chan *Message)
	go func() {
		messages <- &Message{
			Topic: "test-topic",
		}
	}()

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(gomock.AssignableToTypeOf(messages))

	logger := logger.NewNopLogger()
	cli := NewClientLogger(client, logger)

	// When - Then
	cli.Produce(messages)
}

func TestClientLoggerEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	logger := logger.NewNopLogger()

	cli := NewClientLogger(client, logger)

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
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

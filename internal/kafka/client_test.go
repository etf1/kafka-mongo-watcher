package kafka

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	producer := NewMockKafkaProducer(ctrl)

	// When
	cli := NewClient(producer)

	// Then
	assert.IsType(t, new(client), cli)
	assert.Equal(t, producer, cli.producer)
}

func TestClientProduceWhenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	message := &kafkaconfluent.Message{}
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().Produce(message, nil).Return(nil)

	cli := NewClient(producer)

	// When
	err := cli.Produce(message)

	// Then
	assert.Nil(t, err)
}

func TestClientProduceWhenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	expectedErr := errors.New("an unexpected error occured")

	message := &kafkaconfluent.Message{}
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().Produce(message, nil).Return(expectedErr)

	cli := NewClient(producer)

	// When
	err := cli.Produce(message)

	// Then
	assert.Equal(t, err, expectedErr)
}

func TestClientClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().Close()

	cli := NewClient(producer)

	// When - Then
	cli.Close()
}

package kafka

import (
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

func TestClientProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	produceChannel := make(chan *kafkaconfluent.Message, 1)

	message := &kafkaconfluent.Message{}
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().ProduceChannel().Return(produceChannel)

	cli := NewClient(producer)

	// When
	cli.Produce(message)

	// Then
	inserted := <-produceChannel

	assert := assert.New(t)
	assert.IsType(new(kafkaconfluent.Message), inserted)
}

func TestClientEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().Events().Return(events)

	cli := NewClient(producer)

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
}

func TestClientClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().Len().Return(0)
	producer.EXPECT().Close()

	cli := NewClient(producer)

	// When - Then
	cli.Close()
}

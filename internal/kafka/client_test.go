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

	messages := make(chan *Message)
	go func() {
		defer close(messages)
		messages <- &Message{
			Topic: "test-topic",
			Key:   []byte(`my-key`),
			Value: []byte(`my-value`),
			Headers: []Header{
				Header{Key: "x-test-header", Value: []byte("test")},
			},
		}
	}()

	producer := NewMockKafkaProducer(ctrl)
	producer.EXPECT().ProduceChannel().Return(produceChannel)
	producer.EXPECT().Len().Return(0)
	producer.EXPECT().Close()

	cli := NewClient(producer)

	// When
	cli.Produce(messages)

	// Then
	inserted := <-produceChannel

	assert := assert.New(t)
	assert.IsType(new(kafkaconfluent.Message), inserted)

	assert.Equal("test-topic", *inserted.TopicPartition.Topic)
	assert.Equal([]byte("my-key"), inserted.Key)
	assert.Equal([]byte("my-value"), inserted.Value)

	assert.Equal("x-test-header", inserted.Headers[0].Key)
	assert.Equal([]byte("test"), inserted.Headers[0].Value)
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

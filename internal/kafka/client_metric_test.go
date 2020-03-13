package kafka

import (
	"errors"
	"testing"

	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewClientMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)

	recorder := metrics.NewMockKafkaRecorder(ctrl)

	// When
	cli := NewClientMetric(client, recorder)

	assert.IsType(t, new(clientMetric), cli)

	assert.Equal(t, client, cli.client)
	assert.Equal(t, recorder, cli.recorder)
}

func TestClientMetricProduceWhenSuccess(t *testing.T) {
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
	client.EXPECT().Produce(message).Return(nil)

	recorder := metrics.NewMockKafkaRecorder(ctrl)
	recorder.EXPECT().IncKafkaClientProduceSuccessCounter("test-topic")

	cli := NewClientMetric(client, recorder)

	// When
	err := cli.Produce(message)

	// Then
	assert.Nil(t, err)
}

func TestClientMetricProduceWhenError(t *testing.T) {
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

	recorder := metrics.NewMockKafkaRecorder(ctrl)
	recorder.EXPECT().IncKafkaClientProduceErrorCounter("test-topic")

	cli := NewClientMetric(client, recorder)

	// When
	err := cli.Produce(message)

	// Then
	assert.Equal(t, err, expectedErr)
}

func TestClientMetricEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	recorder := metrics.NewMockKafkaRecorder(ctrl)

	cli := NewClientMetric(client, recorder)

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
}

func TestClientMetricClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)
	client.EXPECT().Close()

	recorder := metrics.NewMockKafkaRecorder(ctrl)
	cli := NewClientMetric(client, recorder)

	// When - Then
	cli.Close()
}

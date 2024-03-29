package kafka

import (
	"testing"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

func TestClientMetricProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	messages := make(chan *Message)
	go func() {
		defer close(messages)
		messages <- &Message{
			Topic: "test-topic",
		}
	}()

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(gomock.AssignableToTypeOf(messages))

	recorder := metrics.NewMockKafkaRecorder(ctrl)
	recorder.EXPECT().IncKafkaClientProduceCounter("test-topic")

	cli := NewClientMetric(client, recorder)

	// When - Then
	cli.Produce(messages)
	time.Sleep(100 * time.Millisecond) // Wait for incoming channel to be read
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

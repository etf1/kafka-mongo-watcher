package kafka

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestNewClientTracer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)

	addedHeader := kafkaconfluent.Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	addHeaderFn := func(message *kafkaconfluent.Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	// When
	cli := NewClientTracer(client, addHeaderFn)

	// Then
	assert.IsType(t, new(clientTracer), cli)
	assert.Equal(t, client, cli.client)
	assert.IsType(t, tracerFunc(nil), cli.fn)
}

func TestClientTracerProduceWhenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	message := &kafkaconfluent.Message{}

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(message).Return(nil)

	addedHeader := kafkaconfluent.Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	tracerFn := func(message *kafkaconfluent.Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	cli := NewClientTracer(client, tracerFn)

	// When
	err := cli.Produce(message)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, addedHeader, message.Headers[0])
}

func TestClientTracerProduceWhenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	expectedErr := errors.New("an unexpected error occurred")

	message := &kafkaconfluent.Message{}

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(message).Return(expectedErr)

	addedHeader := kafkaconfluent.Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	tracerFn := func(message *kafkaconfluent.Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	cli := NewClientTracer(client, tracerFn)

	// When
	err := cli.Produce(message)

	// Then
	assert.Equal(t, err, expectedErr)
	assert.Equal(t, addedHeader, message.Headers[0])
}

func TestClientTracerEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	tracerFn := func(message *kafkaconfluent.Message) {}

	cli := NewClientTracer(client, tracerFn)

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
}

func TestClientTracerClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)
	client.EXPECT().Close()

	addedHeader := kafkaconfluent.Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	tracerFn := func(message *kafkaconfluent.Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	cli := NewClientTracer(client, tracerFn)

	// When - Then
	cli.Close()
}

func TestAddTracingHeader(t *testing.T) {
	messageTest := &kafkaconfluent.Message{
		Headers: []kafkaconfluent.Header{
			kafkaconfluent.Header{Key: "test-key1", Value: []byte(`my-test-value1`)},
			kafkaconfluent.Header{Key: "test-key2", Value: []byte(`my-test-value2`)},
		},
	}
	AddTracingHeader(messageTest)
	assert.Equal(t, "x-tracing", messageTest.Headers[2].Key)
	assert.Regexp(t, `kafka-mongo-watcher,\d*`, string(messageTest.Headers[2].Value))
}

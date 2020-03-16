package kafka

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type TrackerMock struct {
	mock.Mock
}

func (t TrackerMock) Function(msg *Message) {
	t.Called(msg)
}

func TestNewClientTracer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)

	addedHeader := Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	addHeaderFn := func(message *Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	// When
	cli := NewClientTracer(client, addHeaderFn)

	// Then
	assert.IsType(t, new(clientTracer), cli)
	assert.Equal(t, client, cli.client)
	assert.IsType(t, tracerFunc(nil), cli.fn)
}

func TestClientTracerProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	messages := make(chan *Message)
	message := &Message{
		Topic: "test-topic",
	}

	go func() {
		defer close(messages)
		messages <- message
	}()

	client := NewMockClient(ctrl)
	client.EXPECT().Produce(gomock.AssignableToTypeOf(messages))

	var tracerMock TrackerMock
	tracerMock.On("Function", message)

	cli := NewClientTracer(client, tracerMock.Function)

	// When - Then
	cli.Produce(messages)
	time.Sleep(100 * time.Millisecond) // Wait for incoming channel to be read
}

func TestClientTracerEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	tracerFn := func(message *Message) {}

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

	addedHeader := Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	tracerFn := func(message *Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	cli := NewClientTracer(client, tracerFn)

	// When - Then
	cli.Close()
}

func TestAddTracingHeader(t *testing.T) {
	// Given
	messageTest := &Message{
		Headers: []Header{
			Header{Key: "test-key1", Value: []byte(`my-test-value1`)},
			Header{Key: "test-key2", Value: []byte(`my-test-value2`)},
		},
	}

	// Then
	AddTracingHeader(messageTest)

	// Then
	assert.Equal(t, "x-tracing", messageTest.Headers[2].Key)
	assert.Regexp(t, `kafka-mongo-watcher,\d*`, string(messageTest.Headers[2].Value))
}

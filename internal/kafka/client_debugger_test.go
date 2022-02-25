package kafka

import (
	"testing"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DebuggerMock struct {
	mock.Mock
}

func (t DebuggerMock) Function(msg *Message) {
	t.Called(msg)
}

func TestNewClientDebugger(t *testing.T) {
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
	cli := NewClientDebugger(client, addHeaderFn)

	// Then
	assert.IsType(t, new(clientDebugger), cli)
	assert.Equal(t, client, cli.client)
	assert.IsType(t, debugFunc(nil), cli.fn)
}

func TestClientDebuggerProduce(t *testing.T) {
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

	var debuggerMock DebuggerMock
	debuggerMock.On("Function", message)

	cli := NewClientDebugger(client, debuggerMock.Function)

	// When - Then
	cli.Produce(messages)
	time.Sleep(100 * time.Millisecond) // Wait for incoming channel to be read
}

func TestClientDebuggerEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	debuggerFn := func(message *Message) {}

	cli := NewClientDebugger(client, debuggerFn)

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
}

func TestClientDebuggerClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)
	client.EXPECT().Close()

	addedHeader := Header{
		Key:   "my-test-key",
		Value: []byte(`my-test-value`),
	}

	debuggerFn := func(message *Message) {
		message.Headers = append(message.Headers, addedHeader)
	}

	cli := NewClientDebugger(client, debuggerFn)

	// When - Then
	cli.Close()
}

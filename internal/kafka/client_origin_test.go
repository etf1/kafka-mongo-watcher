package kafka

import (
	"testing"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type OriginMock struct {
	mock.Mock
}

func (o OriginMock) Function(msg *Message, origin string) {
	o.Called(msg, origin)
}

func TestNewClientOrigin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)

	addOriginFn := func(message *Message, origin string) {
		message.Headers = append(message.Headers, Header{
			Key:   "my-test-key",
			Value: []byte(origin),
		})
	}

	// When
	cli := NewClientOrigin(client, "my-origin", addOriginFn)

	// Then
	assert.IsType(t, new(clientOrigin), cli)
	assert.Equal(t, client, cli.client)
	assert.Equal(t, "my-origin", cli.origin)
	assert.IsType(t, originFunc(nil), cli.fn)
}

func TestClientOriginProduce(t *testing.T) {
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

	var originMock OriginMock
	originMock.On("Function", message, "my-origin")

	cli := NewClientOrigin(client, "my-origin", originMock.Function)

	// When - Then
	cli.Produce(messages)
	time.Sleep(100 * time.Millisecond) // Wait for incoming channel to be read
}

func TestClientOriginEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	events := make(chan kafkaconfluent.Event)
	client := NewMockClient(ctrl)
	client.EXPECT().Events().Return(events)

	cli := NewClientOrigin(client, "my-origin", func(message *Message, origin string) {})

	// When
	result := cli.Events()

	// Then
	assert.Equal(t, events, result)
}

func TestClientOriginClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	client := NewMockClient(ctrl)
	client.EXPECT().Close()

	cli := NewClientOrigin(client, "my-origin", func(message *Message, origin string) {
		message.Headers = append(message.Headers, Header{
			Key:   "my-test-key",
			Value: []byte(origin),
		})
	})

	// When - Then
	cli.Close()
}

func TestAddOriginHeader(t *testing.T) {
	// Given
	messageTest := &Message{
		Headers: []Header{
			{Key: "test-key1", Value: []byte(`my-test-value1`)},
			{Key: "test-key2", Value: []byte(`my-test-value2`)},
		},
	}

	// When
	AddOriginHeader(messageTest, "my-service")

	// Then
	assert.Equal(t, XOriginHeaderName, messageTest.Headers[2].Key)
	assert.Equal(t, "my-service", string(messageTest.Headers[2].Value))
}

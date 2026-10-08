package kafka

import (
	"testing"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestClientCheckpointProduce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	messages := make(chan *Message, 2)
	messages <- &Message{Topic: "test-topic", ResumeToken: []byte(`{"_data":"1"}`)}
	messages <- &Message{Topic: "test-topic"}
	close(messages)

	var produced []*Message
	client := NewMockClient(ctrl)
	client.EXPECT().Produce(gomock.Any()).Do(func(next chan *Message) {
		for message := range next {
			produced = append(produced, message)
		}
	})

	var tracked [][]byte
	cli := NewClientCheckpoint(client, func(resumeToken []byte) {
		tracked = append(tracked, resumeToken)
	})

	cli.Produce(messages)

	assert.Equal(t, [][]byte{[]byte(`{"_data":"1"}`)}, tracked)
	assert.Len(t, produced, 2)
	assert.Equal(t, []Header{{Key: XResumeTokenHeaderName, Value: []byte(`{"_data":"1"}`)}}, produced[0].Headers)
	assert.Empty(t, produced[1].Headers)
}

func TestResumeTokenFromHeaders(t *testing.T) {
	assert.Nil(t, ResumeTokenFromHeaders(nil))
	assert.Equal(t, []byte("token"), ResumeTokenFromHeaders([]kafkaconfluent.Header{
		{Key: "other", Value: []byte("x")},
		{Key: XResumeTokenHeaderName, Value: []byte("token")},
	}))
}

func TestDeliveryDispatcher(t *testing.T) {
	events := make(chan kafkaconfluent.Event, 2)
	events <- &kafkaconfluent.Message{}
	events <- kafkaconfluent.NewError(kafkaconfluent.ErrAllBrokersDown, "down", false)
	close(events)

	var first, second int
	dispatcher := NewDeliveryDispatcher()
	dispatcher.Register(func(*kafkaconfluent.Message) { first++ })
	dispatcher.Register(func(*kafkaconfluent.Message) { second++ })

	dispatcher.Run(events)

	<-dispatcher.Done()
	assert.Equal(t, 1, first)
	assert.Equal(t, 1, second)
}

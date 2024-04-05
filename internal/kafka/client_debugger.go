package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type debugFunc func(message *Message)

type clientDebugger struct {
	client Client
	fn     debugFunc
}

// NewClientDebugger returns a kafka client that allows retrieve debug information from an original client
func NewClientDebugger(cli Client, fn debugFunc) *clientDebugger {
	return &clientDebugger{
		client: cli,
		fn:     fn,
	}
}

// Produce sends the message to a debug function and then produces it
func (c *clientDebugger) Produce(messages chan *Message) {
	var next = make(chan *Message, len(messages))
	go func() {
		defer close(next)
		for message := range messages {
			c.fn(message)
			next <- message
		}
	}()

	c.client.Produce(next)
}

// Events returns the kafka producer events
func (c *clientDebugger) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientDebugger) Close() {
	c.client.Close()
}

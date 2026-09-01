package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// XOriginHeaderName corresponds to the X-Origin header to is sent in Kafka messages
// with some origin information
const XOriginHeaderName = "x-origin"

// AddOriginHeader simply adds an origin header with the provided origin information
// to enable simple debugging
func AddOriginHeader(message *Message, origin string) {
	message.Headers = append(message.Headers, Header{
		Key:   XOriginHeaderName,
		Value: []byte(origin),
	})
}

type originFunc func(message *Message, origin string)

type clientOrigin struct {
	client Client
	fn     originFunc
	origin string
}

// NewClientOrigin returns a kafka client that allows adding origin information from an original client
func NewClientOrigin(cli Client, origin string, fn originFunc) *clientOrigin {
	return &clientOrigin{
		client: cli,
		fn:     fn,
		origin: origin,
	}
}

// Produce adds origin information on the message and then produces it
func (c *clientOrigin) Produce(messages chan *Message) {
	var next = make(chan *Message, len(messages))
	go func() {
		defer close(next)
		for message := range messages {
			c.fn(message, c.origin)
			next <- message
		}
	}()

	c.client.Produce(next)
}

// Events returns the kafka producer events
func (c *clientOrigin) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientOrigin) Close() {
	c.client.Close()
}

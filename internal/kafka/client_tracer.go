package kafka

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/config"
)

// XTracingHeaderName corresponds to the X-Tracing header to is sent in Kafka messages
// with some tracing information
const XTracingHeaderName = "x-tracing"

// AddTracingHeader simply adds a tracing header with application name and a timestamp
// to enable simple debugging
func AddTracingHeader(message *Message) {
	now := time.Now()

	message.Headers = append(message.Headers, Header{
		Key:   XTracingHeaderName,
		Value: []byte(fmt.Sprintf(`%s,%d`, config.AppName, now.Unix())),
	})
}

type tracerFunc func(message *Message)

type clientTracer struct {
	client Client
	fn     tracerFunc
}

// NewClientTracer returns a kafka client that allows adding trace information from an original client
func NewClientTracer(cli Client, fn tracerFunc) *clientTracer {
	return &clientTracer{
		client: cli,
		fn:     fn,
	}
}

// Produce adds tracing information on the message and then produces it
func (c *clientTracer) Produce(messages chan *Message) {
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
func (c *clientTracer) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientTracer) Close() {
	c.client.Close()
}

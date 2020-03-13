package kafka

import (
	"fmt"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// XTracingHeaderName corresponds to the X-Tracing header to is sent in Kafka messages
// with some tracing information
const XTracingHeaderName = "x-tracing"

// AddTracingHeader simply adds a tracing header with application name and a timestamp
// to enable simple debugging
func AddTracingHeader(message *kafka.Message) {
	now := time.Now()

	message.Headers = append(message.Headers, kafka.Header{
		Key:   XTracingHeaderName,
		Value: []byte(fmt.Sprintf(`%s,%d`, config.AppName, now.Unix())),
	})
}

type tracerFunc func(message *kafka.Message)

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
func (c *clientTracer) Produce(message *kafka.Message) {
	c.fn(message)
	c.client.Produce(message)
}

// Events returns the kafka producer events
func (c *clientTracer) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientTracer) Close() {
	c.client.Close()
}

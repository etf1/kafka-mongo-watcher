package kafka

import (
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Client interface {
	Produce(messages chan *Message)
	Events() chan kafka.Event
	Close()
}

type client struct {
	producer KafkaProducer
}

// NewClient returns a basic kafka client
func NewClient(producer KafkaProducer) *client {
	return &client{
		producer: producer,
	}
}

// queueFullRetryDelay is the delay before producing again a message rejected because
// the producer queue is full
var queueFullRetryDelay = 100 * time.Millisecond

// Produce sends the messages using the producer, in order
func (c *client) Produce(messages chan *Message) {
	defer c.Close()

	// Produce() instead of the deprecated ProduceChannel(): with otelconfluent, the
	// produce channel is forwarded by a goroutine that could still hold the last
	// message when the producer is closed ("send on closed channel" panic).
	for message := range messages {
		c.produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &message.Topic, Partition: kafka.PartitionAny},
			Key:            message.Key,
			Value:          message.Value,
			Headers:        buildHeaders(message.Headers),
		})
	}
}

// produce enqueues the message, waiting while the producer queue is full.
// Like the channel producer, an enqueue failure is reported as a failed delivery.
func (c *client) produce(message *kafka.Message) {
	for {
		err := c.producer.Produce(message, nil)
		var kafkaErr kafka.Error
		if errors.As(err, &kafkaErr) && kafkaErr.Code() == kafka.ErrQueueFull {
			time.Sleep(queueFullRetryDelay)
			continue
		}
		if err != nil {
			message.TopicPartition.Error = err
			c.producer.Events() <- message
		}
		return
	}
}

func buildHeaders(headers []Header) []kafka.Header {
	var kafkaHeaders = make([]kafka.Header, 0)

	for _, header := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   header.Key,
			Value: header.Value,
		})
	}

	return kafkaHeaders
}

// Events returns the kafka producer events
func (c *client) Events() chan kafka.Event {
	return c.producer.Events()
}

// Close allows to close/disconnect the kafka client
func (c *client) Close() {
	// Wait for all messages to be delivered and their reports to be retrieved
	for c.producer.Flush(1000) > 0 {
	}

	c.producer.Close()
}

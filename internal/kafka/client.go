package kafka

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Client interface {
	Produce(message *kafka.Message) error
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

// Produce sends a message using the producer
func (c *client) Produce(message *kafka.Message) error {
	return c.producer.Produce(message, nil)
}

// Close allows to close/disconnect the kafka client
func (c *client) Close() {
	c.producer.Close()
}

package kafka

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Client interface {
	Produce(message *kafka.Message)
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

// Produce sends a message using the producer
func (c *client) Produce(message *kafka.Message) {
	c.producer.ProduceChannel() <- message
}

// Events returns the kafka producer events
func (c *client) Events() chan kafka.Event {
	return c.producer.Events()
}

// Close allows to close/disconnect the kafka client
func (c *client) Close() {
	for wait := true; wait; wait = c.producer.Len() > 0 {
		// Wait for all events to be retrieved from Kafka library
	}

	c.producer.Close()
}

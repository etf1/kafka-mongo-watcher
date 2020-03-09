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

func NewClient(producer KafkaProducer) *client {
	return &client{
		producer: producer,
	}
}

func (c *client) Produce(message *kafka.Message) error {
	return c.producer.Produce(message, nil)
}

func (c *client) Close() {
	c.producer.Close()
}

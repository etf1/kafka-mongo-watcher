package kafka

import (
	"github.com/gol4ng/logger"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type clientLogger struct {
	client Client
	logger logger.LoggerInterface
}

// NewClientLogger returns a kafka client that allows adding log information from an original client
func NewClientLogger(cli Client, logger logger.LoggerInterface) *clientLogger {
	return &clientLogger{
		client: cli,
		logger: logger,
	}
}

// Produce logs the message production information and then produces it
func (c *clientLogger) Produce(message *kafka.Message) {
	c.logger.Info("Kafka client: Producing message", logger.String("topic", *message.TopicPartition.Topic), logger.ByteString("key", message.Key), logger.ByteString("value", message.Value))
	c.client.Produce(message)
}

// Events returns the kafka producer events
func (c *clientLogger) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientLogger) Close() {
	c.client.Close()
}

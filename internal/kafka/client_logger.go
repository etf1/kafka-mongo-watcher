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
func (c *clientLogger) Produce(message *kafka.Message) error {
	c.logger.Info("Kafka client: Producing message", logger.String("topic", *message.TopicPartition.Topic), logger.ByteString("key", message.Key), logger.ByteString("value", message.Value))
	return c.client.Produce(message)
}

func (c *clientLogger) Close() {
	c.client.Close()
}
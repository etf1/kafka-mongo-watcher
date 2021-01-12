package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gol4ng/logger"
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
func (c *clientLogger) Produce(messages chan *Message) {
	var next = make(chan *Message, len(messages))
	go func() {
		defer close(next)
		for message := range messages {
			c.logger.Info("Kafka client: Producing message", logger.String("topic", message.Topic), logger.ByteString("key", message.Key), logger.ByteString("value", message.Value))
			next <- message
		}
	}()

	c.client.Produce(next)
}

// Events returns the kafka producer events
func (c *clientLogger) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientLogger) Close() {
	c.client.Close()
}

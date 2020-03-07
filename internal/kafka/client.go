package kafka

import (
	"fmt"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/gol4ng/logger"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// xTracingHeaderName corresponds to the X-Tracing header to is sent in Kafka messages
// with some tracing information
const xTracingHeaderName = "x-tracing"

type Client interface {
	Produce(message *kafka.Message) error
	Close()
}

type client struct {
	logger   logger.LoggerInterface
	producer *kafka.Producer
}

func NewClient(logger logger.LoggerInterface, producer *kafka.Producer) Client {
	return &client{
		logger:   logger,
		producer: producer,
	}
}

func (c *client) Produce(message *kafka.Message) error {
	c.logger.Info("Kafka client: Producing message", logger.ByteString("key", message.Key), logger.ByteString("value", message.Value))

	addTracingHeader(message)
	return c.producer.Produce(message, nil)
}

func (c *client) Close() {
	c.producer.Close()
}

func addTracingHeader(message *kafka.Message) {
	now := time.Now()

	message.Headers = append(message.Headers, kafka.Header{
		Key:   xTracingHeaderName,
		Value: []byte(fmt.Sprintf(`%s,%d`, config.AppName, now.Unix())),
	})
}

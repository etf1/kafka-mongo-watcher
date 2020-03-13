package kafka

import (
	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type clientMetric struct {
	client   Client
	recorder metrics.KafkaRecorder
}

// NewClientMetric returns a kafka client that allows adding trace information from an original client
func NewClientMetric(cli Client, recorder metrics.KafkaRecorder) *clientMetric {
	return &clientMetric{
		client:   cli,
		recorder: recorder,
	}
}

func (c *clientMetric) Record() {
	for e := range c.Events() {
		switch ev := e.(type) {
		case *kafkaconfluent.Message:
			if ev.TopicPartition.Error != nil {
				c.recorder.IncKafkaProducerErrorCounter(*ev.TopicPartition.Topic)
			} else {
				c.recorder.IncKafkaProducerSuccessCounter(*ev.TopicPartition.Topic)
			}
		}
	}
}

// Produce adds tracing information on the message and then produces it
func (c *clientMetric) Produce(message *kafka.Message) error {
	err := c.client.Produce(message)
	if err != nil {
		c.recorder.IncKafkaClientProduceErrorCounter(*message.TopicPartition.Topic)
	} else {
		c.recorder.IncKafkaClientProduceSuccessCounter(*message.TopicPartition.Topic)
	}

	return err
}

// Events returns the kafka producer events
func (c *clientMetric) Events() chan kafka.Event {
	return c.client.Events()
}

func (c *clientMetric) Close() {
	c.client.Close()
}

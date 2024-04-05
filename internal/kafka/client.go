package kafka

import (
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

// Produce sends a message using the producer
func (c *client) Produce(messages chan *Message) {
	defer c.Close()

	for message := range messages {
		c.producer.ProduceChannel() <- &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &message.Topic, Partition: kafka.PartitionAny},
			Key:            message.Key,
			Value:          message.Value,
			Headers:        buildHeaders(message.Headers),
		}
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
	for wait := true; wait; wait = c.producer.Len() > 0 {
		// Wait for all events to be retrieved from Kafka library
	}

	c.producer.Close()
}

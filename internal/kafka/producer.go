package kafka

import "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

type KafkaProducer interface {
	Close()
	Events() chan kafka.Event
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
}

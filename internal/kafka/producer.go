package kafka

import "github.com/confluentinc/confluent-kafka-go/kafka"

type KafkaProducer interface {
	Close()
	Events() chan kafka.Event
	Len() int
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	ProduceChannel() chan *kafka.Message
}

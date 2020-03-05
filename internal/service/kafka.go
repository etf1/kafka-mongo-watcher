package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func (container *Container) GetKafkaProducer() *kafkaconfluent.Producer {
	if container.kafkaProducer == nil {
		producer, err := kafkaconfluent.NewProducer(&kafkaconfluent.ConfigMap{
			"bootstrap.servers": container.Cfg.Kafka.BootstrapServers,
		})
		if err != nil {
			panic(err)
		}

		log := container.GetLogger()
		log.Info("Connected to kafka producer", logger.String("bootstrao-servers", container.Cfg.Kafka.BootstrapServers))

		container.kafkaProducer = producer
	}

	return container.kafkaProducer
}

func (container *Container) GetKafkaClient() kafka.Client {
	if container.kafkaClient == nil {
		container.kafkaClient = kafka.NewClient(
			container.GetLogger(),
			container.GetKafkaProducer(),
		)
	}

	return container.kafkaClient
}

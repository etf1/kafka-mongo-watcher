package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func (container *Container) GetKafkaProducer() kafka.KafkaProducer {
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
		container.kafkaClient = container.decorateKafkaClientWithMetrics(
			container.decorateKafkaClientWithLogger(
				container.decorateKafkaClientWithTracer(
					container.getKafkaBaseClient(),
				),
			),
		)
	}

	return container.kafkaClient
}

func (container *Container) getKafkaBaseClient() kafka.Client {
	return kafka.NewClient(
		container.GetKafkaProducer(),
	)
}

func (container *Container) decorateKafkaClientWithLogger(client kafka.Client) kafka.Client {
	return kafka.NewClientLogger(client, container.GetLogger())
}

func (container *Container) decorateKafkaClientWithTracer(client kafka.Client) kafka.Client {
	return kafka.NewClientTracer(client, kafka.AddTracingHeader)
}

func (container *Container) decorateKafkaClientWithMetrics(client kafka.Client) kafka.Client {
	clientMetric := kafka.NewClientMetric(client, container.GetKafkaRecorder())
	go clientMetric.Record()

	return clientMetric
}

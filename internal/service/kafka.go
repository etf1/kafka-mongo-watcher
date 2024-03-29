package service

import (
	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/opentelemetry-go-contrib/instrumentation/github.com/confluentinc/confluent-kafka-go/otelconfluent"
	"github.com/gol4ng/logger"
)

func (container *Container) GetKafkaProducer() *kafkaconfluent.Producer {
	if container.kafkaProducer == nil {
		producer, err := kafkaconfluent.NewProducer(&kafkaconfluent.ConfigMap{
			"bootstrap.servers":       container.Cfg.Kafka.BootstrapServers,
			"go.produce.channel.size": container.Cfg.Kafka.ProduceChannelSize,
		})
		if err != nil {
			panic(err)
		}

		log := container.GetLogger()
		log.Info("Connected to kafka producer", logger.String("bootstrap-servers", container.Cfg.Kafka.BootstrapServers))

		container.kafkaProducer = producer
	}

	return container.kafkaProducer
}

func (container *Container) GetKafkaClient() kafka.Client {
	if container.kafkaClient == nil {
		if container.Cfg.Kafka.WithDecorators {
			container.kafkaClient = container.decorateKafkaClientWithMetrics(
				container.decorateKafkaClientWithLogger(
					container.decorateKafkaClientWithTracer(
						container.decorateKafkaClientWithDebugger(
							container.getKafkaBaseClient(),
						),
					),
				),
			)
		} else {
			container.kafkaClient = container.getKafkaBaseClient()
		}
	}

	return container.kafkaClient
}

func (container *Container) getKafkaBaseClient() kafka.Client {
	originalKafkaProducer := container.GetKafkaProducer()
	var kafkaProducer kafka.KafkaProducer = originalKafkaProducer

	if container.Cfg.OtelCollectorEndpoint != "" && container.Cfg.Kafka.WithDecorators {
		// In case OpenTelemetry endpoint is enabled, decorate the Kafka producer.
		kafkaProducer = container.decorateKafkaClientWithOpenTelemetry(originalKafkaProducer)
	}

	return kafka.NewClient(kafkaProducer)
}

func (container *Container) decorateKafkaClientWithOpenTelemetry(producer *kafkaconfluent.Producer) *otelconfluent.Producer {
	return otelconfluent.NewProducerWithTracing(
		producer,
		otelconfluent.WithTracerProvider(container.GetTracerProvider()),
		otelconfluent.WithTracerName("etf1/kafka-mongo-watcher"),
	)
}

func (container *Container) decorateKafkaClientWithDebugger(client kafka.Client) kafka.Client {
	if container.Cfg.HttpServer.DebugEnabled {
		client = kafka.NewClientDebugger(client, container.GetDebugger().Add)
	}

	return client
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

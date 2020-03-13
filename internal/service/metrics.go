package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func (container *Container) GetMetricsRegistry() prometheus.Registerer {
	return prometheus.DefaultRegisterer
}

func (container *Container) GetKafkaRecorder() metrics.KafkaRecorder {
	if container.kafkaRecorder == nil {
		container.kafkaRecorder = metrics.NewKafkaRecorder().RegisterOn(container.GetMetricsRegistry())
	}

	return container.kafkaRecorder
}

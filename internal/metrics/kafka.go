package metrics

import (
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/prometheus/client_golang/prometheus"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type KafkaRecorder struct {
	producerSuccessCounter *prometheus.CounterVec
	producerErrorCounter   *prometheus.CounterVec
}

// NewKafkaRecorder returns a kafka recorder that is used to send metrics
func NewKafkaRecorder() *KafkaRecorder {
	return &KafkaRecorder{
		// Kafka producer metrics
		producerSuccessCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_success_counter_total",
				Help:      "This represent the number of successful messages pushed into Kafka",
			},
			[]string{"topic"},
		),
		producerErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_error_counter_total",
				Help:      "This represent the number of error messages handled by Kafka producer",
			},
			[]string{"topic"},
		),
	}
}

// RegisterOn allows to specify a specific Prometheus registry
func (r *KafkaRecorder) RegisterOn(registry prometheus.Registerer) *KafkaRecorder {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	registry.MustRegister(
		r.producerSuccessCounter,
		r.producerErrorCounter,
	)

	return r
}

// Unregister allows to unregister kafka metrics from current Prometheus register
func (r *KafkaRecorder) Unregister(registry prometheus.Registerer) *KafkaRecorder {
	registry.Unregister(r.producerSuccessCounter)
	registry.Unregister(r.producerErrorCounter)

	return r
}

// RecordProducer listens for events from the kafka producer and sends metrics
// into Prometheus registry
func (r *KafkaRecorder) RecordProducer(producer kafka.KafkaProducer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafkaconfluent.Message:
			if ev.TopicPartition.Error != nil {
				r.IncKafkaProducerErrorCounter(*ev.TopicPartition.Topic)
			} else {
				r.IncKafkaProducerSuccessCounter(*ev.TopicPartition.Topic)
			}
		}
	}
}

// IncKafkaProducerSuccessCounter increments the producer success counter
func (r *KafkaRecorder) IncKafkaProducerSuccessCounter(topic string) {
	r.producerSuccessCounter.WithLabelValues(topic).Inc()
}

// IncKafkaProducerErrorCounter increments the producer error counter
func (r *KafkaRecorder) IncKafkaProducerErrorCounter(topic string) {
	r.producerErrorCounter.WithLabelValues(topic).Inc()
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// KafkaRecorder allows to record metrics about Kafka
type KafkaRecorder interface {
	IncKafkaClientProduceCounter(topic string)
	IncKafkaProducerSuccessCounter(topic string)
	IncKafkaProducerErrorCounter(topic string)
	RegisterOn(registry prometheus.Registerer) KafkaRecorder
	Unregister(registry prometheus.Registerer) KafkaRecorder
}

type kafkaRecorder struct {
	clientProduceCounter   *prometheus.CounterVec
	producerSuccessCounter *prometheus.CounterVec
	producerErrorCounter   *prometheus.CounterVec
}

// NewKafkaRecorder returns a kafka recorder that is used to send metrics
func NewKafkaRecorder() *kafkaRecorder {
	return &kafkaRecorder{
		// Kafka producer metrics
		clientProduceCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "client_produce_counter_total",
				Help:      "This represent the number of messages pushed by Kafka client",
			},
			[]string{"topic"},
		),
		producerSuccessCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_event_success_counter_total",
				Help:      "This represent the number of successful messages pushed into Kafka",
			},
			[]string{"topic"},
		),
		producerErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_event_error_counter_total",
				Help:      "This represent the number of error messages handled by Kafka producer",
			},
			[]string{"topic"},
		),
	}
}

// RegisterOn allows to specify a specific Prometheus registry
func (r *kafkaRecorder) RegisterOn(registry prometheus.Registerer) KafkaRecorder {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	registry.MustRegister(
		r.clientProduceCounter,
		r.producerSuccessCounter,
		r.producerErrorCounter,
	)

	return r
}

// Unregister allows to unregister kafka metrics from current Prometheus register
func (r *kafkaRecorder) Unregister(registry prometheus.Registerer) KafkaRecorder {
	registry.Unregister(r.clientProduceCounter)
	registry.Unregister(r.producerSuccessCounter)
	registry.Unregister(r.producerErrorCounter)

	return r
}

// IncKafkaClientProduceCounter increments the client produce counter
func (r *kafkaRecorder) IncKafkaClientProduceCounter(topic string) {
	r.clientProduceCounter.WithLabelValues(topic).Inc()
}

// IncKafkaProducerSuccessCounter increments the producer success counter
func (r *kafkaRecorder) IncKafkaProducerSuccessCounter(topic string) {
	r.producerSuccessCounter.WithLabelValues(topic).Inc()
}

// IncKafkaProducerErrorCounter increments the producer error counter
func (r *kafkaRecorder) IncKafkaProducerErrorCounter(topic string) {
	r.producerErrorCounter.WithLabelValues(topic).Inc()
}

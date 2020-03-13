package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// KafkaRecorder allows to record metrics about Kafka
type KafkaRecorder interface {
	IncKafkaClientProduceSuccessCounter(topic string)
	IncKafkaClientProduceErrorCounter(topic string)
	IncKafkaProducerSuccessCounter(topic string)
	IncKafkaProducerErrorCounter(topic string)
	RegisterOn(registry prometheus.Registerer) KafkaRecorder
}

type kafkaRecorder struct {
	clientProduceSuccessCounter *prometheus.CounterVec
	clientProduceErrorCounter   *prometheus.CounterVec
	producerSuccessCounter      *prometheus.CounterVec
	producerErrorCounter        *prometheus.CounterVec
}

// NewKafkaRecorder returns a kafka recorder that is used to send metrics
func NewKafkaRecorder() *kafkaRecorder {
	return &kafkaRecorder{
		// Kafka producer metrics
		clientProduceSuccessCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "client_produce_success_counter_total",
				Help:      "This represent the number of successful messages pushed by Kafka client",
			},
			[]string{"topic"},
		),
		clientProduceErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "client_produce_error_counter_total",
				Help:      "This represent the number of error messages handled by Kafka client",
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
		r.clientProduceSuccessCounter,
		r.clientProduceErrorCounter,
		r.producerSuccessCounter,
		r.producerErrorCounter,
	)

	return r
}

// Unregister allows to unregister kafka metrics from current Prometheus register
func (r *kafkaRecorder) Unregister(registry prometheus.Registerer) *kafkaRecorder {
	registry.Unregister(r.clientProduceSuccessCounter)
	registry.Unregister(r.clientProduceErrorCounter)
	registry.Unregister(r.producerSuccessCounter)
	registry.Unregister(r.producerErrorCounter)

	return r
}

// IncKafkaClientProduceSuccessCounter increments the client produce success counter
func (r *kafkaRecorder) IncKafkaClientProduceSuccessCounter(topic string) {
	r.clientProduceSuccessCounter.WithLabelValues(topic).Inc()
}

// IncKafkaClientProduceErrorounter increments the client produce error counter
func (r *kafkaRecorder) IncKafkaClientProduceErrorCounter(topic string) {
	r.clientProduceErrorCounter.WithLabelValues(topic).Inc()
}

// IncKafkaProducerSuccessCounter increments the producer success counter
func (r *kafkaRecorder) IncKafkaProducerSuccessCounter(topic string) {
	r.producerSuccessCounter.WithLabelValues(topic).Inc()
}

// IncKafkaProducerErrorCounter increments the producer error counter
func (r *kafkaRecorder) IncKafkaProducerErrorCounter(topic string) {
	r.producerErrorCounter.WithLabelValues(topic).Inc()
}

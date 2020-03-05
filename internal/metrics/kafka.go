package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type KafkaRecorder struct {
	kafkaProducerSuccessCounter *prometheus.CounterVec
	kafkaProducerErrorCounter   *prometheus.CounterVec
}

func (r *KafkaRecorder) RegisterOn(registry prometheus.Registerer) *KafkaRecorder {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	registry.MustRegister(
		r.kafkaProducerSuccessCounter,
		r.kafkaProducerErrorCounter,
	)
	return r
}

func (r *KafkaRecorder) IncKafkaProducerSuccessCounter(topic string) {
	r.kafkaProducerSuccessCounter.WithLabelValues(topic).Inc()
}

func (r *KafkaRecorder) IncKafkaProducerErrorCounter(topic string) {
	r.kafkaProducerErrorCounter.WithLabelValues(topic).Inc()
}

func NewKafkaRecorder() *KafkaRecorder {
	return &Recorder{
		// Kafka producer metrics
		kafkaProducerSuccessCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_success_counter_total",
				Help:      "This represent the number of successful messages pushed into Kafka",
			},
			[]string{"topic"},
		),
		kafkaProducerErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kafka",
				Name:      "producer_error_counter_total",
				Help:      "This represent the number of error messages handled by Kafka producer",
			},
			[]string{"topic"},
		),
	}
}

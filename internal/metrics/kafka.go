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

func (r *KafkaRecorder) Record(producer kafka.KafkaProducer) {
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

func (r *KafkaRecorder) IncKafkaProducerSuccessCounter(topic string) {
	r.producerSuccessCounter.WithLabelValues(topic).Inc()
}

func (r *KafkaRecorder) IncKafkaProducerErrorCounter(topic string) {
	r.producerErrorCounter.WithLabelValues(topic).Inc()
}

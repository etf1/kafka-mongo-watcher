package metrics

import (
	"errors"
	"testing"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type prometheusRegistererMock struct {
	collectors []prometheus.Collector
}

func (r *prometheusRegistererMock) Register(collector prometheus.Collector) error {
	return nil
}

func (r *prometheusRegistererMock) MustRegister(collectors ...prometheus.Collector) {
	r.collectors = append(r.collectors, collectors...)
}

func (r *prometheusRegistererMock) Unregister(prometheus.Collector) bool {
	return true
}

func TestNewKafkaRecorder(t *testing.T) {
	// When
	recorder := NewKafkaRecorder()

	// Then
	assert := assert.New(t)
	assert.IsType(new(KafkaRecorder), recorder)
	assert.IsType(new(prometheus.CounterVec), recorder.producerSuccessCounter)
	assert.IsType(new(prometheus.CounterVec), recorder.producerErrorCounter)
}

func TestRegisterOn(t *testing.T) {
	// Given
	metricsRegistered = false

	testRegistry := &prometheusRegistererMock{}

	recorder := NewKafkaRecorder()

	// When
	recorder.RegisterOn(testRegistry)

	// Then
	assert := assert.New(t)
	assert.Len(testRegistry.collectors, 2)
}

func TestRegisterOnWhenAlreadyRegistered(t *testing.T) {
	// Given
	metricsRegistered = false

	testRegistry := &prometheusRegistererMock{}

	recorder := NewKafkaRecorder()

	// When registering twice...
	recorder.RegisterOn(testRegistry)
	recorder.RegisterOn(testRegistry)

	// Then
	assert := assert.New(t)
	assert.Len(testRegistry.collectors, 2)
}

func TestRecordProducer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Given
	recorder := NewKafkaRecorder()
	recorder.RegisterOn(prometheus.DefaultRegisterer)

	topicName := "my-test-topic"

	eventsChannel := make(chan kafkaconfluent.Event)

	producer := kafka.NewMockKafkaProducer(ctrl)
	producer.EXPECT().Events().Return(eventsChannel)

	// When
	go recorder.RecordProducer(producer)

	eventsChannel <- &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topicName, Error: nil},
	}
	eventsChannel <- &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topicName, Error: nil},
	}
	eventsChannel <- &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topicName, Error: errors.New("unexpected error")},
	}

	close(eventsChannel)

	// Then
	assert := assert.New(t)

	assert.Equal(float64(2), testutil.ToFloat64(recorder.producerSuccessCounter))
	assert.Equal(float64(1), testutil.ToFloat64(recorder.producerErrorCounter))
}

func TestIncKafkaProducerSuccessCounter(t *testing.T) {
	// Given
	recorder := NewKafkaRecorder()
	recorder.RegisterOn(prometheus.DefaultRegisterer)

	// When
	recorder.IncKafkaProducerSuccessCounter("test-topic")
	recorder.IncKafkaProducerSuccessCounter("test-topic")
	recorder.IncKafkaProducerSuccessCounter("test-topic")

	// Then
	assert := assert.New(t)

	assert.Equal(float64(3), testutil.ToFloat64(recorder.producerSuccessCounter))
}

func TestIncKafkaProducerErrorCounter(t *testing.T) {
	// Given
	recorder := NewKafkaRecorder()
	recorder.RegisterOn(prometheus.DefaultRegisterer)

	// When
	recorder.IncKafkaProducerErrorCounter("test-topic")
	recorder.IncKafkaProducerErrorCounter("test-topic")
	recorder.IncKafkaProducerErrorCounter("test-topic")

	// Then
	assert := assert.New(t)

	assert.Equal(float64(3), testutil.ToFloat64(recorder.producerErrorCounter))
}

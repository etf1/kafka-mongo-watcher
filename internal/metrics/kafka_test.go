package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
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

func (r *prometheusRegistererMock) Unregister(collector prometheus.Collector) bool {
	for i, c := range r.collectors {
		if c == collector {
			r.collectors = append(r.collectors[:i], r.collectors[i+1:]...)
		}
	}

	return true
}

func TestNewKafkaRecorder(t *testing.T) {
	// When
	recorder := NewKafkaRecorder()

	// Then
	assert := assert.New(t)
	assert.IsType(new(kafkaRecorder), recorder)
	assert.IsType(new(prometheus.CounterVec), recorder.clientProduceSuccessCounter)
	assert.IsType(new(prometheus.CounterVec), recorder.clientProduceErrorCounter)
	assert.IsType(new(prometheus.CounterVec), recorder.producerSuccessCounter)
	assert.IsType(new(prometheus.CounterVec), recorder.producerErrorCounter)
}

func TestRegisterOn(t *testing.T) {
	// Given
	testRegistry := &prometheusRegistererMock{}

	recorder := NewKafkaRecorder()

	// When
	recorder.RegisterOn(testRegistry)

	// Then
	assert := assert.New(t)
	assert.Len(testRegistry.collectors, 4)
}

func TestUnregister(t *testing.T) {
	// Given
	assert := assert.New(t)

	testRegistry := &prometheusRegistererMock{}

	// When registering metrics
	recorder := NewKafkaRecorder()
	recorder.RegisterOn(testRegistry)

	assert.Len(testRegistry.collectors, 4)

	// And unregistering metrics
	recorder.Unregister(testRegistry)

	// Then
	assert.Len(testRegistry.collectors, 0)
}

func TestIncKafkaClientProduceSuccessCounter(t *testing.T) {
	// Given
	recorder := NewKafkaRecorder()

	testRegistry := &prometheusRegistererMock{}
	recorder.RegisterOn(testRegistry)

	// When
	recorder.IncKafkaClientProduceSuccessCounter("test-topic")
	recorder.IncKafkaClientProduceSuccessCounter("test-topic")
	recorder.IncKafkaClientProduceSuccessCounter("test-topic")

	// Then
	assert := assert.New(t)

	assert.Equal(float64(3), testutil.ToFloat64(recorder.clientProduceSuccessCounter))
}

func TestIncKafkaClientProduceErrorCounter(t *testing.T) {
	// Given
	recorder := NewKafkaRecorder()

	testRegistry := &prometheusRegistererMock{}
	recorder.RegisterOn(testRegistry)

	// When
	recorder.IncKafkaClientProduceErrorCounter("test-topic")
	recorder.IncKafkaClientProduceErrorCounter("test-topic")
	recorder.IncKafkaClientProduceErrorCounter("test-topic")

	// Then
	assert := assert.New(t)

	assert.Equal(float64(3), testutil.ToFloat64(recorder.clientProduceErrorCounter))
}

func TestIncKafkaProducerSuccessCounter(t *testing.T) {
	// Given
	recorder := NewKafkaRecorder()

	testRegistry := &prometheusRegistererMock{}
	recorder.RegisterOn(testRegistry)

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

	testRegistry := &prometheusRegistererMock{}
	recorder.RegisterOn(testRegistry)

	// When
	recorder.IncKafkaProducerErrorCounter("test-topic")
	recorder.IncKafkaProducerErrorCounter("test-topic")
	recorder.IncKafkaProducerErrorCounter("test-topic")

	// Then
	assert := assert.New(t)

	assert.Equal(float64(3), testutil.ToFloat64(recorder.producerErrorCounter))
}

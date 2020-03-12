package kafka

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/gol4ng/logger"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type ProducerPool interface {
	Close()
	Produce(ctx context.Context, messages chan *Message)
}

type producerPool struct {
	logger        logger.LoggerInterface
	client        Client
	number        int
	numberRunning int32
	waitGroup     sync.WaitGroup
}

// NewProducerPool initializes a new kafka producer pool
func NewProducerPool(logger logger.LoggerInterface, client Client, number int) *producerPool {
	return &producerPool{
		logger:        logger,
		client:        client,
		number:        number,
		numberRunning: 0,
		waitGroup:     sync.WaitGroup{},
	}
}

// Close stops the producerPool and its goroutines
func (w *producerPool) Close() {
	numberRunningToStop := w.numberRunning
	for i := int32(0); i < numberRunningToStop; i++ {
		w.waitGroup.Done()
		w.numberRunning--
	}
}

// Produce allows to retrieve kafka messages from channel and produce them using the kafka client
func (w *producerPool) Produce(ctx context.Context, messages chan *Message) {
	for i := 0; i < w.number; i++ {
		atomic.AddInt32(&w.numberRunning, 1)
		w.waitGroup.Add(1)
		go w.produce(ctx, messages)
	}

	w.waitGroup.Wait()
}

func (w *producerPool) produce(ctx context.Context, items chan *Message) {
	defer func() {
		atomic.AddInt32(&w.numberRunning, -1)
		w.waitGroup.Done()
	}()

	for item := range items {
		w.client.Produce(&kafkaconfluent.Message{
			TopicPartition: kafkaconfluent.TopicPartition{Topic: &item.Topic, Partition: kafkaconfluent.PartitionAny},
			Key:            item.Key,
			Value:          item.Value,
		})
	}
}

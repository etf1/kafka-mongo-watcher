package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Worker interface {
	Close()
	Replay(collection mongo.CollectionAdapter, topic string)
	WatchAndProduce(collection mongo.CollectionAdapter, topic string)
}

type worker struct {
	ctx           context.Context
	logger        logger.LoggerInterface
	mongoClient   mongo.Client
	kafkaClient   kafka.Client
	itemsChan     chan *mongo.WatchItem
	number        int
	numberRunning int32
	timeout       time.Duration
	waitGroup     sync.WaitGroup
}

func New(ctx context.Context, logger logger.LoggerInterface, mongoClient mongo.Client, kafkaClient kafka.Client, number int, timeout time.Duration) *worker {
	return &worker{
		ctx:           ctx,
		logger:        logger,
		mongoClient:   mongoClient,
		kafkaClient:   kafkaClient,
		itemsChan:     make(chan *mongo.WatchItem),
		number:        number,
		numberRunning: 0,
		timeout:       timeout,
		waitGroup:     sync.WaitGroup{},
	}
}

func (w *worker) Close() {
	for i := int32(0); i < w.numberRunning; i++ {
		w.waitGroup.Done()
	}

	close(w.itemsChan)
}

func (w *worker) Replay(collection mongo.CollectionAdapter, topic string) {
	go w.mongoClient.Replay(collection, w.itemsChan)
	w.work(topic, true)
}

func (w *worker) WatchAndProduce(collection mongo.CollectionAdapter, topic string) {
	go w.mongoClient.Watch(collection, w.itemsChan)
	w.work(topic, false)
}

func (w *worker) work(topic string, canTimeout bool) {
	for i := 0; i < w.number; i++ {
		w.waitGroup.Add(1)
		atomic.AddInt32(&w.numberRunning, 1)
		go w.produce(topic, canTimeout)
	}

	w.waitGroup.Wait()
}

func (w *worker) produce(topic string, canTimeout bool) {
	defer w.waitGroup.Done()

	for {
		select {
		case <-w.ctx.Done():
			atomic.AddInt32(&w.numberRunning, -1)
			return
		case <-time.After(w.timeout):
			if canTimeout {
				atomic.AddInt32(&w.numberRunning, -1)
				return
			}
		case item := <-w.itemsChan:
			w.kafkaClient.Produce(&kafkaconfluent.Message{
				TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
				Key:            item.Key,
				Value:          item.Value,
			})
		}
	}
}

package worker

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Worker interface {
	Close()
	Replay(ctx context.Context, collection mongo.CollectionAdapter, topic string)
	WatchAndProduce(ctx context.Context, collection mongo.CollectionAdapter, topic string)
}

type worker struct {
	logger        logger.LoggerInterface
	mongoClient   mongo.Client
	kafkaClient   kafka.Client
	number        int
	numberRunning int32
	waitGroup     sync.WaitGroup
}

func New(logger logger.LoggerInterface, mongoClient mongo.Client, kafkaClient kafka.Client, number int) *worker {
	return &worker{
		logger:        logger,
		mongoClient:   mongoClient,
		kafkaClient:   kafkaClient,
		number:        number,
		numberRunning: 0,
		waitGroup:     sync.WaitGroup{},
	}
}

func (w *worker) Close() {
	for i := int32(0); i < w.numberRunning; i++ {
		w.waitGroup.Done()
	}
}

func (w *worker) Replay(ctx context.Context, collection mongo.CollectionAdapter, topic string) {
	itemsChan, err := w.mongoClient.Replay(ctx, collection)
	if err != nil {
		w.logger.Error("An error occured while trying to replay mongodb collection", logger.Error("error", err))
		return
	}

	w.work(ctx, topic, itemsChan)
}

func (w *worker) WatchAndProduce(ctx context.Context, collection mongo.CollectionAdapter, topic string) {
	itemsChan, err := w.mongoClient.Watch(ctx, collection)
	if err != nil {
		w.logger.Error("An error occured while watching mongodb collection", logger.Error("error", err))
		return
	}

	w.work(ctx, topic, itemsChan)
}

func (w *worker) work(ctx context.Context, topic string, itemsChan chan *mongo.WatchItem) {
	for i := 0; i < w.number; i++ {
		atomic.AddInt32(&w.numberRunning, 1)
		w.waitGroup.Add(1)
		go w.produce(ctx, topic, itemsChan)
	}

	w.waitGroup.Wait()
}

func (w *worker) produce(ctx context.Context, topic string, itemsChan chan *mongo.WatchItem) {
	defer func() {
		atomic.AddInt32(&w.numberRunning, -1)
		w.waitGroup.Done()
	}()

	for item := range itemsChan {
		w.kafkaClient.Produce(&kafkaconfluent.Message{
			TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
			Key:            item.Key,
			Value:          item.Value,
		})
	}
}

package worker

import (
	"context"
	"sync"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/gol4ng/logger"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Worker interface {
	Close()
	Replay(collection *mongodriver.Collection, topic string)
	WatchAndProduce(collection *mongodriver.Collection, topic string)
}

type worker struct {
	ctx         context.Context
	logger      logger.LoggerInterface
	mongoClient mongo.Client
	kafkaClient kafka.Client
	itemsChan   chan *mongo.WatchItem
	number      int
	waitGroup   sync.WaitGroup
}

func New(ctx context.Context, logger logger.LoggerInterface, mongoClient mongo.Client, kafkaClient kafka.Client, number int) Worker {
	return &worker{
		ctx:         ctx,
		logger:      logger,
		mongoClient: mongoClient,
		kafkaClient: kafkaClient,
		itemsChan:   make(chan *mongo.WatchItem),
		number:      number,
		waitGroup:   sync.WaitGroup{},
	}
}

func (w *worker) Close() {
	for i := 0; i < w.number; i++ {
		w.waitGroup.Done()
	}

	close(w.itemsChan)
}

func (w *worker) Replay(collection *mongodriver.Collection, topic string) {
	var finished = make(chan bool)
	go w.mongoClient.Replay(collection, w.itemsChan, finished)
	go w.work(topic)

	<-finished
	w.Close()
}

func (w *worker) WatchAndProduce(collection *mongodriver.Collection, topic string) {
	go w.mongoClient.Watch(collection, w.itemsChan)
	w.work(topic)
}

func (w *worker) work(topic string) {
	for i := 0; i < w.number; i++ {
		w.waitGroup.Add(1)
		go w.produce(topic)
	}

	w.waitGroup.Wait()
}

func (w *worker) produce(topic string) {
	defer w.waitGroup.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case item := <-w.itemsChan:
			w.kafkaClient.Produce(&kafkaconfluent.Message{
				TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Partition: kafkaconfluent.PartitionAny},
				Key:            item.Key,
				Value:          item.Value,
			})
		}
	}
}

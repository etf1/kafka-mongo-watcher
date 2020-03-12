package main

import (
	"context"
	"os"
	"syscall"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/gol4ng/logger"
	signal_subscriber "github.com/gol4ng/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewBase(ctx)

	container := service.NewContainer(cfg)
	logger := container.GetLogger()

	go container.GetTechServer().Start(ctx)

	defer handleExitSignal(ctx, cancel, container)()

	events := getMongoEvents(ctx, container)
	messages := make(chan *kafka.Message)
	go transformMongoEventsToKafkaMessages(logger, container.Cfg.Kafka.Topic, events, messages)

	container.GetKafkaProducerPool().Produce(ctx, messages)
}

func getMongoEvents(ctx context.Context, container *service.Container) chan *mongo.ChangeEvent {
	mongoClient := getMongoClient(container)
	collection := container.GetMongoCollection(ctx)

	events, err := mongoClient.Oplogs(ctx, collection)
	if err != nil {
		println(err.Error())
		container.GetLogger().Error("error")
	}

	return events
}

func getMongoClient(container *service.Container) (client mongo.Client) {
	if container.Cfg.Replay {
		client = container.GetMongoReplayerClient()
	} else {
		client = container.GetMongoWatcherClient()
	}

	return
}

func transformMongoEventsToKafkaMessages(logger logger.LoggerInterface, topic string, events chan *mongo.ChangeEvent, messages chan *kafka.Message) {
	defer close(messages)
	mongo.TransformChangeEventToKafkaMessage(logger, topic, events, messages)
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers)
func handleExitSignal(ctx context.Context, cancel context.CancelFunc, container *service.Container) func() {
	return signal_subscriber.SubscribeWithKiller(func(signal os.Signal) {
		log := container.GetLogger()
		log.Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))

		cancel()
		container.GetKafkaProducerPool().Close()
		container.GetMongoConnection(ctx).Client().Disconnect(ctx)
		container.GetKafkaClient().Close()
		container.GetTechServer().Close(ctx)
	}, os.Interrupt, syscall.SIGTERM)
}

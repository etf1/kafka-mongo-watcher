package main

import (
	"context"
	"os"
	"syscall"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/gol4ng/logger"
	signal_subscriber "github.com/gol4ng/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewBase(ctx)

	container := service.NewContainer(cfg)

	go container.GetTechServer().Start(ctx)

	collection := container.GetMongoCollection(ctx)

	mongoClient := getMongoClient(container)
	worker := container.GetWorker(mongoClient)

	defer handleExitSignal(ctx, cancel, container, mongoClient)()
	worker.Work(ctx, collection, container.Cfg.Kafka.Topic)
}

func getMongoClient(container *service.Container) (client mongo.Client) {
	if container.Cfg.Replay {
		client = container.GetMongoReplayerClient()
	} else {
		client = container.GetMongoWatcherClient()
	}

	return
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers)
func handleExitSignal(ctx context.Context, cancel context.CancelFunc, container *service.Container, mongoClient mongo.Client) func() {
	return signal_subscriber.SubscribeWithKiller(func(signal os.Signal) {
		log := container.GetLogger()
		log.Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))

		cancel()
		container.GetWorker(mongoClient).Close()
		container.GetMongoConnection(ctx).Client().Disconnect(ctx)
		container.GetKafkaClient().Close()
		container.GetTechServer().Close(ctx)
	}, os.Interrupt, syscall.SIGTERM)
}

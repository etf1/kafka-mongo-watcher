package main

import (
	"context"
	"os"
	"syscall"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/gol4ng/logger"
	signal_subscriber "github.com/gol4ng/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewBase(ctx)

	// TODO pass base context
	container := service.NewContainer(cfg)
	go container.GetTechServer().Start(ctx)

	defer handleExitSignal(ctx, cancel, container)()

	changeEventChan, err := container.GetChangeEvent(ctx)
	if err != nil {
		panic(err)
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	container.GetKafkaProducerPool().Produce(ctx, kafkaMessageChan)
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

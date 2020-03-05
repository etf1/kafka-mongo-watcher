package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/gol4ng/logger"
	signal_subscriber "github.com/gol4ng/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewBase()

	container := service.NewContainer(ctx, cfg)
	worker := container.GetWorker()

	if container.Cfg.Replay {
		worker.Replay(container.GetMongoCollection(), container.Cfg.Kafka.Topic)
	} else {
		go notifyOnExitSignal(cancel)
		defer handleExitSignal(ctx, container)()

		worker.WatchAndProduce(container.GetMongoCollection(), container.Cfg.Kafka.Topic)
	}
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers)
func handleExitSignal(ctx context.Context, container *service.Container) func() {
	return signal_subscriber.SubscribeWithKiller(func(signal os.Signal) {
		log := container.GetLogger()
		log.Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))

		container.GetWorker().Close()
		container.GetMongoConnection().Client().Disconnect(ctx)
		container.GetKafkaClient().Close()
	}, os.Interrupt, syscall.SIGTERM)
}

// Notifies for an exit signal in order to quit application on a proper way (shutting down connections and servers)
func notifyOnExitSignal(cancel context.CancelFunc) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("Shutting down.")
	cancel()
}

package main

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/gol4ng/logger"
	signal_subscriber "github.com/gol4ng/signal"
)

var (
	configPrefix = "kafka_mongo_watcher"
)

func main() {

	if prefixFromEnv := os.Getenv("KAFKA_MONGO_WATCHER_PREFIX"); prefixFromEnv != "" {
		configPrefix = prefixFromEnv
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewBase(ctx, configPrefix)

	container := service.NewContainer(ctx, cfg)
	go container.GetHttpServer().Start(ctx)

	defer handleExitSignal(cancel, container)()

	changeEventChan, err := container.GetChangeEventProducer()(ctx)
	if err != nil {
		panic(err)
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	container.GetKafkaClient().Produce(kafkaMessageChan)
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers)
func handleExitSignal(cancel context.CancelFunc, container *service.Container) func() {
	return signal_subscriber.SubscribeWithKiller(func(signal os.Signal) {
		log := container.GetLogger()
		log.Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))

		// Cancel the main context immediately so all goroutines start shutting down.
		cancel()
		container.GetKafkaClient().Close()

		// Use independent background contexts so these calls succeed even after cancel().
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		if err := container.GetMongoConnection().Client().Disconnect(disconnectCtx); err != nil {
			log.Error("Failed to disconnect MongoDB client", logger.Error("error", err))
		}

		httpShutdownCtx, httpShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer httpShutdownCancel()
		if err := container.GetHttpServer().Close(httpShutdownCtx); err != nil {
			log.Error("Failed to close HTTP server", logger.Error("error", err))
		}
	}, os.Interrupt, syscall.SIGTERM)
}

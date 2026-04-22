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

	// The signal handler only cancels the context. All cleanup happens after
	// Produce() returns, which guarantees that cursors have been closed by
	// the watch goroutine (via defer) before we disconnect the MongoDB client.
	defer handleExitSignal(cancel, container)()

	changeEventChan, err := container.GetChangeEventProducer()(ctx)
	if err != nil {
		panic(err)
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	container.GetKafkaClient().Produce(kafkaMessageChan)

	// Produce() has returned: all cursors are already closed by watch/replay
	// goroutines. Now it is safe to tear down the underlying connections.
	cleanup(container)
}

// cleanup disconnects MongoDB, then shuts down the HTTP server.
// Kafka client is already closed by Produce()'s own defer.
func cleanup(container *service.Container) {
	log := container.GetLogger()

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
}

// handleExitSignal registers a signal handler that only cancels the main
// context. The returned function is an unsubscriber to be deferred.
func handleExitSignal(cancel context.CancelFunc, container *service.Container) func() {
	return signal_subscriber.SubscribeWithKiller(func(signal os.Signal) {
		container.GetLogger().Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))
		cancel()
	}, os.Interrupt, syscall.SIGTERM)
}

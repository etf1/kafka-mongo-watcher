package main

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
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
	defer cleanup(container)

	const producerStartTimeout = 2 * time.Minute
	changeEventChan, err := startProducerWithRetry(ctx, container, producerStartTimeout)
	if err != nil {
		container.GetLogger().Error("Giving up: unable to start change event producer", logger.Error("error", err))
		return
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	container.GetKafkaClient().Produce(kafkaMessageChan)
}

// startProducerWithRetry retries the change event producer creation with
// exponential backoff until it succeeds or the timeout / context expires.
// This handles transient errors such as "too many cursors" from previous
// pods that left orphaned change streams on the MongoDB server.
func startProducerWithRetry(ctx context.Context, container *service.Container, timeout time.Duration) (chan *mongo.ChangeEvent, error) {
	log := container.GetLogger()
	producer := container.GetChangeEventProducer()

	delay := 1 * time.Second
	const maxDelay = 30 * time.Second
	deadline := time.After(timeout)

	for {
		ch, err := producer(ctx)
		if err == nil {
			return ch, nil
		}
		log.Warning("Failed to start change event producer, retrying…",
			logger.Error("error", err), logger.Duration("retry_in", delay))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, err
		case <-time.After(delay):
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

// cleanup disconnects MongoDB, then shuts down the HTTP server.
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

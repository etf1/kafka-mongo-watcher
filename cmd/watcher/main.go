package main

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/checkpoint"
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
	if err := cfg.Validate(); err != nil {
		container.GetLogger().Error("Invalid configuration", logger.Error("error", err))
		cancel()
		os.Exit(1)
	}
	go container.GetHttpServer().Start(ctx)

	// closed once the change stream cursor is closed, nil while it is not opened
	var watchDone <-chan struct{}

	defer handleExitSignal(cancel, container)()
	// Also run on panic: the change stream cursor is closed before MongoDB is disconnected
	defer func() { cleanup(container, cancel, watchDone) }()

	// Created before the change stream: a kafka producer failure must not leave
	// a cursor opened on the MongoDB server
	kafkaClient := container.GetKafkaClient()

	const producerStartTimeout = 2 * time.Minute
	changeEventChan, err := startProducerWithRetry(ctx, container, producerStartTimeout)
	if err != nil {
		container.GetLogger().Error("Giving up: unable to start change event producer", logger.Error("error", err))
		kafkaClient.Close()
		return
	}
	changeEventChan, watchDone = notifyWhenClosed(ctx, changeEventChan)

	tracker := container.GetCheckpointTracker()
	if tracker != nil {
		go tracker.Run(ctx, cfg.MongoDB.Options.CheckpointInterval)
	}

	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	kafkaClient.Produce(kafkaMessageChan)

	if tracker != nil {
		flushCheckpoint(container, tracker)
	}
}

// flushCheckpoint waits for the last delivery reports and saves the final checkpoint,
// before MongoDB is disconnected.
func flushCheckpoint(container *service.Container, tracker *checkpoint.Tracker) {
	log := container.GetLogger()

	select {
	case <-container.GetKafkaDeliveryDispatcher().Done():
	case <-time.After(10 * time.Second):
		log.Warning("Timeout while waiting for kafka delivery reports")
	}

	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer flushCancel()
	if err := tracker.Flush(flushCtx); err != nil {
		log.Error("Failed to save final checkpoint", logger.Error("error", err))
	}
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
		if mongo.IsResumePointLost(err) {
			// Retrying will always fail, see MONGODB_OPTION_RESUME_ON_HISTORY_LOST
			return nil, err
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

// notifyWhenClosed forwards the change events and returns a channel closed once the
// producer has closed its events channel, i.e. once its cursor is closed.
// After the context is canceled, the remaining events are dropped so the producer
// is never blocked: they will be replayed from the checkpoint.
func notifyWhenClosed(ctx context.Context, events chan *mongo.ChangeEvent) (chan *mongo.ChangeEvent, <-chan struct{}) {
	forwarded := make(chan *mongo.ChangeEvent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(forwarded)
		for event := range events {
			select {
			case forwarded <- event:
			case <-ctx.Done():
			}
		}
	}()
	return forwarded, done
}

// cleanup stops the change stream and waits for its cursor to be closed, then
// disconnects MongoDB and shuts down the HTTP server.
func cleanup(container *service.Container, cancel context.CancelFunc, watchDone <-chan struct{}) {
	log := container.GetLogger()

	cancel()
	if watchDone != nil {
		select {
		case <-watchDone:
		case <-time.After(10 * time.Second):
			log.Warning("Timeout while waiting for the change stream cursor to be closed")
		}
	}

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
// A second signal does not kill the application, so that the change stream cursor
// is always closed: Kubernetes sends SIGKILL after the termination grace period.
func handleExitSignal(cancel context.CancelFunc, container *service.Container) func() {
	return signal_subscriber.Subscribe(func(signal os.Signal) {
		container.GetLogger().Info("Signal received: gracefully stopping application", logger.String("signal", signal.String()))
		cancel()
	}, os.Interrupt, syscall.SIGTERM)
}

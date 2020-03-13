package mongo

import (
	"context"
	"time"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WatchProducer struct {
	collection CollectionAdapter
	logger     logger.LoggerInterface
}

func (w *WatchProducer) GetProducer(o ...WatchOption) ChangeEventProducer {
	return func(ctx context.Context) (chan *ChangeEvent, error) {

		config := NewWatchConfig(o...)
		var emptyPipeline = []bson.M{}

		opts := &options.ChangeStreamOptions{
			BatchSize:    &config.batchSize,
			MaxAwaitTime: &config.maxAwaitTime,
		}
		if config.fullDocumentEnabled {
			opts.SetFullDocument(options.UpdateLookup)
		}

		cursor, err := w.collection.Watch(ctx, emptyPipeline, opts)
		if err != nil {
			w.logger.Error("Mongo client: An error has occured while watching collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			return nil, err
		}

		var events = make(chan *ChangeEvent)

		go func() {
			defer close(events)

			for {
				w.sendEvents(ctx, cursor, events)
			}
		}()

		return events, nil
	}
}

func (w *WatchProducer) sendEvents(ctx context.Context, cursor DriverCursor, events chan *ChangeEvent) {
	for cursor.Next(ctx) {
		event := &ChangeEvent{}
		if err := cursor.Decode(event); err != nil {
			w.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
			continue
		}

		events <- event
	}
}

func NewWatchProducer(adapter CollectionAdapter, logger logger.LoggerInterface)*WatchProducer{
	return &WatchProducer{
		collection: adapter,
		logger:     logger,
	}
}

type WatchOption func(*WatchConfig)

type WatchConfig struct {
	batchSize           int32
	fullDocumentEnabled bool
	maxAwaitTime        time.Duration
}

func (o *WatchConfig) apply(options ...WatchOption) {
	for _, option := range options {
		option(o)
	}
}

func NewWatchConfig(o ...WatchOption) *WatchConfig {
	watchOptions := &WatchConfig{
		batchSize:           0,
		fullDocumentEnabled: false,
		maxAwaitTime:        0,
	}
	watchOptions.apply(o...)
	return watchOptions
}

// WithBatchSize allows to specify a batch size when using changestream event
func WithBatchSize(batchSize int32) WatchOption {
	return func(w *WatchConfig) {
		w.batchSize = batchSize
	}
}

// WithFullDocument allows to returns the full document in oplogs when using
// changestream event
func WithFullDocument(enabled bool) WatchOption {
	return func(w *WatchConfig) {
		w.fullDocumentEnabled = enabled
	}
}

// WithMaxAwaitTime allows to specify the maximum await for new oplogs when using
// changestream event
func WithMaxAwaitTime(maxAwaitTime time.Duration) WatchOption {
	return func(w *WatchConfig) {
		w.maxAwaitTime = maxAwaitTime
	}
}

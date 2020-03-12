package mongo

import (
	"context"
	"time"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WatcherOption func(*watcher)

type watcher struct {
	*client

	fullDocumentEnabled bool
	batchSize           int32
	maxAwaitTime        time.Duration
}

// NewWatcher returns a new mongodb client
func NewWatcher(options ...Option) *watcher {
	return &watcher{
		client:              newClient(options...),
		fullDocumentEnabled: false,
	}
}

func (w *watcher) WithOptions(options ...WatcherOption) *watcher {
	for _, option := range options {
		option(w)
	}

	return w
}

// WithBatchSize allows to specify a batch size when using changestream event
func WithBatchSize(batchSize int32) WatcherOption {
	return func(w *watcher) {
		w.batchSize = batchSize
	}
}

// WithFullDocument allows to returns the full document in oplogs when using
// changestream event
func WithFullDocument(enabled bool) WatcherOption {
	return func(w *watcher) {
		w.fullDocumentEnabled = enabled
	}
}

// WithMaxAwaitTime allows to specify the maximum await for new oplogs when using
// changestream event
func WithMaxAwaitTime(maxAwaitTime time.Duration) WatcherOption {
	return func(w *watcher) {
		w.maxAwaitTime = maxAwaitTime
	}
}

// Do is using the mongodb changestream feature to watch for oplogs of the specified collection
func (w *watcher) Oplogs(ctx context.Context, collection CollectionAdapter) (chan *ChangeEvent, error) {
	var emptyPipeline = []bson.M{}

	opts := &options.ChangeStreamOptions{
		BatchSize:    &w.batchSize,
		MaxAwaitTime: &w.maxAwaitTime,
	}
	if w.fullDocumentEnabled {
		opts.SetFullDocument(options.UpdateLookup)
	}

	cursor, err := collection.Watch(ctx, emptyPipeline, opts)
	if err != nil {
		w.logger.Error("Mongo client: An error has occured while watching collection", logger.String("collection", collection.Name()), logger.Error("error", err))
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

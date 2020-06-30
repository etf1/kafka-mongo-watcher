package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WatchProducer struct {
	collection     CollectionAdapter
	logger         logger.LoggerInterface
	customPipeline string
}

func (w *WatchProducer) GetProducer(o ...WatchOption) ChangeEventProducer {
	return func(ctx context.Context) (chan *ChangeEvent, error) {

		config := NewWatchConfig(o...)
		var pipeline = bson.A{}

		if w.customPipeline != "" {
			var customElements = bson.A{}
			if err := bson.UnmarshalExtJSON([]byte(w.customPipeline), true, &customElements); err != nil {
				return nil, err
			}

			pipeline = append(customElements, pipeline...)
		}

		opts := &options.ChangeStreamOptions{
			BatchSize:            &config.batchSize,
			MaxAwaitTime:         &config.maxAwaitTime,
			StartAtOperationTime: config.startAtOperationTime,
		}
		if config.fullDocumentEnabled {
			opts.SetFullDocument(options.UpdateLookup)
		}
		if len(config.resumeAfter) != 0 {
			opts.SetResumeAfter(config.resumeAfter)
		}

		cursor, err := w.collection.Watch(ctx, pipeline, opts)
		if err != nil {
			w.logger.Error("Mongo client: An error has occured while trying to watch collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			return nil, err
		}

		var events = make(chan *ChangeEvent)

		go func() {
			defer cursor.Close(ctx)
			defer close(events)
			if err := w.sendEvents(ctx, cursor, events); err != nil {
				w.logger.Error("Mongo client : An error has occured while while watching collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			}
		}()

		return events, nil
	}
}

func (w *WatchProducer) sendEvents(ctx context.Context, cursor DriverCursor, events chan *ChangeEvent) error {
	for cursor.Next(ctx) {
		if cursor.ID() == 0 {
			return errors.New("change stream cursor has been closed")
		}
		if err := cursor.Err(); err != nil {
			return cursor.Err()
		}
		event := &ChangeEvent{}
		if err := cursor.Decode(event); err != nil {
			w.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
			continue
		}
		events <- event
	}
	return cursor.Err()
}

func NewWatchProducer(adapter CollectionAdapter, logger logger.LoggerInterface, customPipeline string) *WatchProducer {
	return &WatchProducer{
		collection:     adapter,
		logger:         logger,
		customPipeline: customPipeline,
	}
}

type WatchOption func(*WatchConfig)

type WatchConfig struct {
	batchSize            int32
	fullDocumentEnabled  bool
	maxAwaitTime         time.Duration
	resumeAfter          bson.M
	startAtOperationTime *primitive.Timestamp
}

func (o *WatchConfig) apply(options ...WatchOption) {
	for _, option := range options {
		option(o)
	}
}

func NewWatchConfig(o ...WatchOption) *WatchConfig {
	watchOptions := &WatchConfig{
		batchSize:            0,
		fullDocumentEnabled:  false,
		maxAwaitTime:         0,
		resumeAfter:          bson.M{},
		startAtOperationTime: nil,
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

// WithResumeAfter allows to specify the resume token for the change stream to resume
// notifications after the operation specified in the resume token
func WithResumeAfter(resumeAfter []byte) WatchOption {
	return func(w *WatchConfig) {
		if len(resumeAfter) != 0 {
			err := bson.UnmarshalExtJSON(resumeAfter, false, &w.resumeAfter)
			if err != nil {
				panic(err)
			}
		}
	}
}

// WithStartAtOperationTime allows to specify the timestamp for the change stream to only
// return changes that occurred at or after the given timestamp.
func WithStartAtOperationTime(startAtOperationTime primitive.Timestamp) WatchOption {
	return func(w *WatchConfig) {
		if startAtOperationTime.I != 0 || startAtOperationTime.T != 0 {
			w.startAtOperationTime = &startAtOperationTime
		}
	}
}

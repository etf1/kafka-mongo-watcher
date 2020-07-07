package mongo

import (
	"context"
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

		cursor, err := w.watch(ctx, pipeline, config, config.resumeAfter, nil)

		if err != nil {
			w.logger.Error("Mongo client: An error has occured while trying to watch collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			return nil, err
		}

		var events = make(chan *ChangeEvent)

		go func() {
			defer close(events)
			for {
				select {
				case <-ctx.Done():
					w.logger.Info("Context canceled")
					cursor.Close(ctx)
					return
				case startAfter := <-w.sendEvents(ctx, cursor, events):
					w.logger.Info("Mongo client : Retry to watch collection", logger.String("collection", w.collection.Name()), logger.Any("start_after", startAfter))
					cursor.Close(ctx)
					if config.maxRetries == 0 {
						return
					}
					cursor, err = w.watch(ctx, pipeline, config, nil, startAfter)
					if err != nil {
						w.logger.Error("Mongo client : An error has occured while retrying to watch collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
						return
					}
				}
			}
		}()

		return events, nil
	}
}

func (w *WatchProducer) watch(ctx context.Context, pipeline bson.A, config *WatchConfig, resumeAfter bson.M, startAfter bson.Raw) (cursor StreamCursor, err error) {
	// retries loop
	attempt := int32(0)
	for {
		opts := &options.ChangeStreamOptions{
			BatchSize:            &config.batchSize,
			MaxAwaitTime:         &config.maxAwaitTime,
			StartAtOperationTime: config.startAtOperationTime,
		}
		if config.fullDocumentEnabled {
			opts.SetFullDocument(options.UpdateLookup)
		}

		if startAfter != nil {
			opts.SetStartAfter(startAfter)
		} else if len(resumeAfter) > 0 {
			opts.SetResumeAfter(resumeAfter)
		}

		cursor, err = w.collection.Watch(ctx, pipeline, opts)
		if err == nil {
			break
		}
		if attempt >= config.maxRetries {
			w.logger.Warning("failed to open cursor on collection, reach max retries", logger.String("collection", w.collection.Name()), logger.Int32("max_retries", config.maxRetries), logger.Error("error", err))
			break
		}
		attempt++
		w.logger.Warning("failed to open cursor on collection", logger.String("collection", w.collection.Name()), logger.Int32("attempt", attempt), logger.Duration("retry_delay", config.retryDelay), logger.Error("error", err))
		if config.retryDelay > 0 {
			time.Sleep(config.retryDelay)
		}
	}
	return
}

func (w *WatchProducer) sendEvents(ctx context.Context, cursor StreamCursor, events chan *ChangeEvent) <-chan bson.Raw {
	resumeToken := make(chan bson.Raw, 1)

	go func() {
		defer close(resumeToken)
		for cursor.Next(ctx) {
			if cursor.ID() == 0 {
				w.logger.Error("Mongo client: Cursor has been closed")
				break
			}
			if err := cursor.Err(); err != nil {
				w.logger.Error("Mongo client: Failed to watch collection", logger.Error("error", err))
				break
			}
			event := &ChangeEvent{}
			if err := cursor.Decode(event); err != nil {
				w.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
				continue
			}
			events <- event
		}
		resumeToken <- cursor.ResumeToken()
	}()

	return resumeToken
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
	maxRetries           int32
	retryDelay           time.Duration
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
		maxRetries:           3,
		retryDelay:           250 * time.Millisecond,
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

// WithMaxRetries allows to specify the max retry attempts when watching collection fail
// return changes that occurred at or after the given maxRetries.
func WithMaxRetries(maxRetries int32) WatchOption {
	return func(w *WatchConfig) {
		if maxRetries >= 0 {
			w.maxRetries = maxRetries
		}
	}
}

// WithRetryDelay allows to specify the delay between each retry attempt when watching collection fail
// return changes that occurred at or after the given duration.
func WithRetryDelay(retryDelay time.Duration) WatchOption {
	return func(w *WatchConfig) {
		if retryDelay > 0 {
			w.retryDelay = retryDelay
		}
	}
}

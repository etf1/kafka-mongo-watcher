package mongo

import (
	"context"
	"time"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

		position := config.initialPosition()
		cursor, err := w.watch(ctx, pipeline, config, &position)

		if err != nil {
			w.logger.Error("Mongo client: An error has occured while trying to watch collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			return nil, err
		}

		var events = make(chan *ChangeEvent)

		go func() {
			defer close(events)
			defer func() { closeCursor(cursor) }()
			for {
				resumeToken := <-w.sendEvents(ctx, cursor, events, config.ignoreUpdateDescription)
				// sendEvents exited: either the cursor broke or the context was canceled.
				if ctx.Err() != nil {
					w.logger.Info("Context canceled")
					return
				}
				// Keep the previous position when the cursor did not return any token,
				// otherwise the stream would restart from "now" and lose events.
				if len(resumeToken) > 0 {
					position = streamPosition{resumeToken: resumeToken}
				}
				w.logger.Info("Mongo client : Retry to watch collection", logger.String("collection", w.collection.Name()), logger.Any("resume_after", position.resumeToken))
				closeCursor(cursor)
				cursor = nil
				if config.maxRetries == 0 {
					return
				}
				cursor, err = w.watch(ctx, pipeline, config, &position)
				if err != nil {
					w.logger.Error("Mongo client : An error has occured while retrying to watch collection", logger.String("collection", w.collection.Name()), logger.Error("error", err))
					return
				}
			}
		}()

		return events, nil
	}
}

// streamPosition is the logical starting point of a change stream.
// MongoDB only accepts one of these options at a time.
// Resume tokens are applied with resumeAfter rather than startAfter, which is not
// supported by Amazon DocumentDB.
type streamPosition struct {
	resumeToken          bson.Raw
	resumeAfter          bson.M
	startAtOperationTime *bson.Timestamp
}

func (p streamPosition) isSet() bool {
	return len(p.resumeToken) > 0 || len(p.resumeAfter) > 0 || p.startAtOperationTime != nil
}

func (p streamPosition) apply(opts *options.ChangeStreamOptionsBuilder) {
	switch {
	case len(p.resumeToken) > 0:
		opts.SetResumeAfter(p.resumeToken)
	case len(p.resumeAfter) > 0:
		opts.SetResumeAfter(p.resumeAfter)
	case p.startAtOperationTime != nil:
		opts.SetStartAtOperationTime(p.startAtOperationTime)
	}
}

func (w *WatchProducer) watch(ctx context.Context, pipeline bson.A, config *WatchConfig, position *streamPosition) (cursor StreamCursor, err error) {
	// retries loop
	attempt := int32(0)
	for {
		opts := options.ChangeStream().
			SetBatchSize(config.batchSize).
			SetMaxAwaitTime(config.maxAwaitTime)
		if config.fullDocumentEnabled {
			opts.SetFullDocument(options.UpdateLookup)
		}
		position.apply(opts)

		cursor, err = w.collection.Watch(ctx, pipeline, opts)
		if err == nil {
			break
		}
		if cursor != nil {
			closeCursor(cursor)
			cursor = nil
		}
		if IsResumePointLost(err) {
			// Retrying with the same position will always fail.
			if !config.startFromNowOnHistoryLost || !position.isSet() {
				w.logger.Error("Mongo client: resume point is no longer available in the oplog", logger.String("collection", w.collection.Name()), logger.Error("error", err))
				return
			}
			w.logger.Error("Mongo client: resume point is no longer available in the oplog, restarting from now: some events have been lost", logger.String("collection", w.collection.Name()), logger.Error("error", err))
			*position = streamPosition{}
			continue
		}
		if attempt >= config.maxRetries {
			w.logger.Warning("failed to open cursor on collection, reach max retries", logger.String("collection", w.collection.Name()), logger.Int32("max_retries", config.maxRetries), logger.Error("error", err))
			break
		}
		attempt++
		w.logger.Warning("failed to open cursor on collection", logger.String("collection", w.collection.Name()), logger.Int32("attempt", attempt), logger.Duration("retry_delay", config.retryDelay), logger.Error("error", err))
		if config.retryDelay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(config.retryDelay):
			}
		}
	}
	return
}

func (w *WatchProducer) sendEvents(ctx context.Context, cursor StreamCursor, events chan *ChangeEvent, ignoreUpdateDescription bool) <-chan bson.Raw {
	resumeToken := make(chan bson.Raw, 1)

	go func() {
		defer close(resumeToken)
		for cursor.Next(ctx) {
			event := &ChangeEvent{}
			if err := cursor.Decode(event); err != nil {
				w.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
				continue
			}
			if ignoreUpdateDescription {
				event.Updates = nil
			}
			select {
			case events <- event:
			case <-ctx.Done():
				resumeToken <- cursor.ResumeToken()
				return
			}
			// Checked after sending: the last event returned by a closed cursor
			// (e.g. invalidate) must not be dropped.
			if cursor.ID() == 0 {
				w.logger.Error("Mongo client: Cursor has been closed")
				break
			}
		}
		if err := cursor.Err(); err != nil && ctx.Err() == nil {
			w.logger.Error("Mongo client: Failed to watch collection", logger.Error("error", err))
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
	batchSize                 int32
	fullDocumentEnabled       bool
	ignoreUpdateDescription   bool
	maxAwaitTime              time.Duration
	resumeToken               bson.Raw
	resumeAfter               bson.M
	startAtOperationTime      *bson.Timestamp
	maxRetries                int32
	retryDelay                time.Duration
	startFromNowOnHistoryLost bool
}

// initialPosition returns the configured starting point, by priority:
// resumeToken (checkpoint) > resumeAfter > startAtOperationTime > now.
func (o *WatchConfig) initialPosition() streamPosition {
	switch {
	case len(o.resumeToken) > 0:
		return streamPosition{resumeToken: o.resumeToken}
	case len(o.resumeAfter) > 0:
		return streamPosition{resumeAfter: o.resumeAfter}
	default:
		return streamPosition{startAtOperationTime: o.startAtOperationTime}
	}
}

func (o *WatchConfig) apply(options ...WatchOption) {
	for _, option := range options {
		option(o)
	}
}

func NewWatchConfig(o ...WatchOption) *WatchConfig {
	watchOptions := &WatchConfig{
		batchSize:               0,
		fullDocumentEnabled:     false,
		ignoreUpdateDescription: false,
		maxAwaitTime:            0,
		resumeAfter:             bson.M{},
		startAtOperationTime:    nil,
		maxRetries:              3,
		retryDelay:              250 * time.Millisecond,
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

// WithResumeToken allows to specify the resume token (e.g. a stored checkpoint) after which
// the change stream resumes. It takes precedence over resumeAfter and startAtOperationTime.
func WithResumeToken(resumeToken bson.Raw) WatchOption {
	return func(w *WatchConfig) {
		w.resumeToken = resumeToken
	}
}

// WithStartFromNowOnHistoryLost allows to restart the change stream from now when the
// resume point is no longer available in the oplog, instead of failing.
func WithStartFromNowOnHistoryLost(enabled bool) WatchOption {
	return func(w *WatchConfig) {
		w.startFromNowOnHistoryLost = enabled
	}
}

func WithIgnoreUpdateDescription(ignore bool) WatchOption {
	return func(w *WatchConfig) {
		w.ignoreUpdateDescription = ignore
	}
}

// WithStartAtOperationTime allows to specify the timestamp for the change stream to only
// return changes that occurred at or after the given timestamp.
func WithStartAtOperationTime(startAtOperationTime bson.Timestamp) WatchOption {
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

package mongo

import (
	"context"
	"reflect"
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
			// consecutive reconnections that did not move the stream forward
			stalls := int32(0)
			for {
				openedAt := time.Now()
				end := <-w.sendEvents(ctx, cursor, events, config.ignoreUpdateDescription)
				// sendEvents exited: either the cursor broke or the context was canceled.
				if ctx.Err() != nil {
					w.logger.Info("Context canceled")
					return
				}
				previous := position
				switch {
				case end.invalidated:
					// startAtOperationTime is inclusive: resuming at the invalidate cluster
					// time returns the same invalidate event again, forever. Resume just
					// after it (startAfter is not supported by Amazon DocumentDB).
					next := bson.Timestamp{T: end.lastClusterTime.T, I: end.lastClusterTime.I + 1}
					position = streamPosition{startAtOperationTime: &next}
					w.logger.Warning("Mongo client : Change stream invalidated (collection dropped or renamed), resuming after it", logger.String("collection", w.collection.Name()), logger.Any("cluster_time", end.lastClusterTime))
				case !end.lastClusterTime.IsZero():
					// Keep the previous position when no event was sent, otherwise the stream
					// would restart from "now" and lose events. The last event is sent again
					// (at-least-once).
					position = streamPosition{startAtOperationTime: &end.lastClusterTime}
				}
				if position.equal(previous) && time.Since(openedAt) < stallWindow {
					stalls++
				} else {
					stalls = 0
				}
				w.logger.Info("Mongo client : Retry to watch collection", logger.String("collection", w.collection.Name()), logger.Any("start_at_operation_time", position.startAtOperationTime))
				closeCursor(cursor)
				cursor = nil
				if config.maxRetries == 0 {
					return
				}
				if stalls > config.maxRetries {
					w.logger.Error("Mongo client : Change stream keeps closing without moving forward, reach max retries", logger.String("collection", w.collection.Name()), logger.Int32("max_retries", config.maxRetries), logger.Any("start_at_operation_time", position.startAtOperationTime))
					return
				}
				if stalls > 0 && config.retryDelay > 0 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(config.retryDelay):
					}
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
// Resuming (checkpoint, reconnection) relies on startAtOperationTime: startAfter is
// not supported by Amazon DocumentDB and resumeAfter with an old token did not
// return any event on it.
type streamPosition struct {
	resumeAfter          bson.M
	startAtOperationTime *bson.Timestamp
}

func (p streamPosition) isSet() bool {
	return len(p.resumeAfter) > 0 || p.startAtOperationTime != nil
}

func (p streamPosition) equal(other streamPosition) bool {
	if (p.startAtOperationTime == nil) != (other.startAtOperationTime == nil) {
		return false
	}
	if p.startAtOperationTime != nil && !p.startAtOperationTime.Equal(*other.startAtOperationTime) {
		return false
	}
	return reflect.DeepEqual(p.resumeAfter, other.resumeAfter)
}

func (p streamPosition) apply(opts *options.ChangeStreamOptionsBuilder) {
	switch {
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

// stallWindow is the minimum lifetime of a cursor that closes without moving the
// stream forward for its reconnection not to count as a stall: an idle stream that
// breaks from time to time must not exhaust the retries.
const stallWindow = time.Minute

// streamEnd describes how a cursor ended
type streamEnd struct {
	// lastClusterTime is the cluster time of the last sent event (zero if none)
	lastClusterTime bson.Timestamp
	// invalidated is true when the last sent event closes the stream (drop, rename, invalidate)
	invalidated bool
}

// sendEvents forwards the cursor events until it breaks or the context is canceled
func (w *WatchProducer) sendEvents(ctx context.Context, cursor StreamCursor, events chan *ChangeEvent, ignoreUpdateDescription bool) <-chan streamEnd {
	result := make(chan streamEnd, 1)

	go func() {
		defer close(result)
		var end streamEnd
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
				if clusterTime := event.ClusterTimestamp(); !clusterTime.IsZero() {
					end.lastClusterTime = clusterTime
				}
				end.invalidated = event.invalidatesStream()
			case <-ctx.Done():
				result <- end
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
		result <- end
	}()

	return result
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
	resumeAfter               bson.M
	startAtOperationTime      *bson.Timestamp
	maxRetries                int32
	retryDelay                time.Duration
	startFromNowOnHistoryLost bool
}

// initialPosition returns the configured starting point, by priority:
// resumeAfter > startAtOperationTime (checkpoint or configuration) > now.
func (o *WatchConfig) initialPosition() streamPosition {
	switch {
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

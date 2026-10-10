package checkpoint

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type entry struct {
	resumeToken string
	clusterTime bson.Timestamp
	acked       bool
}

// Tracker follows the produced events in the change stream order and only commits
// the position of the last event of the contiguous acknowledged prefix: delivery
// reports of different partitions may arrive out of order.
// A message that can never be delivered (e.g. too large) is skipped. Any other
// delivery failure blocks the checkpoint and calls the failure handler, so that the
// application restarts and replays the event from the checkpoint.
type Tracker struct {
	mu sync.Mutex
	// flushMu serializes the flushes (periodic and final): an older position saved
	// last would move the stored checkpoint backward
	flushMu   sync.Mutex
	store     Store
	logger    logger.LoggerInterface
	pending   *list.List
	index     map[string]*list.Element
	committed *entry
	saved     *entry
	onFailure func(error)
	failed    bool
}

// NewTracker returns a new checkpoint tracker
func NewTracker(store Store, log logger.LoggerInterface) *Tracker {
	return &Tracker{
		store:   store,
		logger:  log,
		pending: list.New(),
		index:   map[string]*list.Element{},
	}
}

// SetFailureHandler sets the function called once on the first delivery failure
// that blocks the checkpoint
func (t *Tracker) SetFailureHandler(onFailure func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onFailure = onFailure
}

// Track registers an event that is about to be produced, in the change stream order.
// The resume token identifies the event in its delivery report.
func (t *Tracker) Track(resumeToken []byte, clusterTime bson.Timestamp) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := string(resumeToken)
	if _, ok := t.index[key]; ok {
		return
	}
	t.index[key] = t.pending.PushBack(&entry{resumeToken: key, clusterTime: clusterTime})
}

// OnDelivery handles a kafka delivery report
func (t *Tracker) OnDelivery(message *kafkaconfluent.Message) {
	resumeToken := kafka.ResumeTokenFromHeaders(message.Headers)
	if resumeToken == nil {
		return
	}
	if err := message.TopicPartition.Error; err != nil {
		if isPermanentDeliveryError(err) {
			t.logger.Error("Checkpoint: message can never be delivered, it is skipped and lost",
				logger.ByteString("resume_token", resumeToken), logger.Error("error", err))
			t.ack(resumeToken)
			return
		}
		t.logger.Error("Checkpoint: message delivery failed, checkpoint will not move past this event until restart",
			logger.ByteString("resume_token", resumeToken), logger.Error("error", err))
		t.fail(err)
		return
	}
	t.ack(resumeToken)
}

// fail calls the failure handler once: the pending events can no longer be
// committed, they would grow without limit
func (t *Tracker) fail(err error) {
	t.mu.Lock()
	onFailure := t.onFailure
	first := !t.failed
	t.failed = true
	t.mu.Unlock()

	if first && onFailure != nil {
		onFailure(err)
	}
}

// isPermanentDeliveryError returns true when the message itself can not be
// delivered: replaying it would always fail
func isPermanentDeliveryError(err error) bool {
	var kafkaErr kafkaconfluent.Error
	if !errors.As(err, &kafkaErr) {
		return false
	}
	switch kafkaErr.Code() {
	case kafkaconfluent.ErrMsgSizeTooLarge,
		kafkaconfluent.ErrInvalidMsgSize,
		kafkaconfluent.ErrInvalidMsg,
		kafkaconfluent.ErrInvalidRecord,
		kafkaconfluent.ErrRecordListTooLarge,
		kafkaconfluent.ErrBadMsg:
		return true
	}
	return false
}

func (t *Tracker) ack(resumeToken []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	element, ok := t.index[string(resumeToken)]
	if !ok {
		return
	}
	element.Value.(*entry).acked = true

	for front := t.pending.Front(); front != nil && front.Value.(*entry).acked; front = t.pending.Front() {
		committed := front.Value.(*entry)
		delete(t.index, committed.resumeToken)
		t.pending.Remove(front)
		// events without cluster time can not be used to resume
		if !committed.clusterTime.IsZero() {
			t.committed = committed
		}
	}
}

// Flush saves the last committed position if it changed since the last save
func (t *Tracker) Flush(ctx context.Context) error {
	t.flushMu.Lock()
	defer t.flushMu.Unlock()

	t.mu.Lock()
	committed, saved := t.committed, t.saved
	t.mu.Unlock()

	if committed == nil || committed == saved {
		return nil
	}
	if saved != nil && !committed.clusterTime.After(saved.clusterTime) {
		return nil
	}

	checkpoint := Checkpoint{ClusterTime: committed.clusterTime}
	var resumeToken bson.Raw
	if err := bson.UnmarshalExtJSON([]byte(committed.resumeToken), true, &resumeToken); err == nil {
		checkpoint.ResumeToken = resumeToken
	}
	if err := t.store.Save(ctx, checkpoint); err != nil {
		return err
	}

	t.mu.Lock()
	t.saved = committed
	t.mu.Unlock()
	return nil
}

// Run periodically flushes the checkpoint until the context is done
func (t *Tracker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.Flush(ctx); err != nil {
				t.logger.Error("Checkpoint: unable to save checkpoint", logger.Error("error", err))
			}
		}
	}
}

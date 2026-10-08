package checkpoint

import (
	"container/list"
	"context"
	"sync"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type entry struct {
	resumeToken string
	acked       bool
}

// Tracker follows the produced events in the change stream order and only commits
// the resume token of the last event of the contiguous acknowledged prefix: delivery
// reports of different partitions may arrive out of order.
// A failed delivery blocks the checkpoint, the event will be replayed on restart.
type Tracker struct {
	mu        sync.Mutex
	store     Store
	logger    logger.LoggerInterface
	pending   *list.List
	index     map[string]*list.Element
	committed string
	saved     string
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

// Track registers an event that is about to be produced, in the change stream order
func (t *Tracker) Track(resumeToken []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := string(resumeToken)
	if _, ok := t.index[key]; ok {
		return
	}
	t.index[key] = t.pending.PushBack(&entry{resumeToken: key})
}

// OnDelivery handles a kafka delivery report
func (t *Tracker) OnDelivery(message *kafkaconfluent.Message) {
	resumeToken := kafka.ResumeTokenFromHeaders(message.Headers)
	if resumeToken == nil {
		return
	}
	if err := message.TopicPartition.Error; err != nil {
		t.logger.Error("Checkpoint: message delivery failed, checkpoint will not move past this event until restart",
			logger.ByteString("resume_token", resumeToken), logger.Error("error", err))
		return
	}
	t.ack(resumeToken)
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
		t.committed = front.Value.(*entry).resumeToken
		delete(t.index, t.committed)
		t.pending.Remove(front)
	}
}

// Flush saves the last committed resume token if it changed since the last save
func (t *Tracker) Flush(ctx context.Context) error {
	t.mu.Lock()
	committed, saved := t.committed, t.saved
	t.mu.Unlock()

	if committed == "" || committed == saved {
		return nil
	}

	var resumeToken bson.Raw
	if err := bson.UnmarshalExtJSON([]byte(committed), true, &resumeToken); err != nil {
		return err
	}
	if err := t.store.Save(ctx, resumeToken); err != nil {
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
				t.logger.Error("Checkpoint: unable to save resume token", logger.Error("error", err))
			}
		}
	}
}

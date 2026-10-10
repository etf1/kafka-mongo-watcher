package checkpoint

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type memoryStore struct {
	saves []Checkpoint
}

func (s *memoryStore) Load(context.Context) (*Checkpoint, error) {
	if len(s.saves) == 0 {
		return nil, nil
	}
	return &s.saves[len(s.saves)-1], nil
}

func (s *memoryStore) Save(_ context.Context, checkpoint Checkpoint) error {
	s.saves = append(s.saves, checkpoint)
	return nil
}

// lastData returns the resume token _data of the last saved checkpoint, and checks
// that its cluster time matches the tracked one
func (s *memoryStore) lastData(t *testing.T) string {
	if len(s.saves) == 0 {
		return ""
	}
	last := s.saves[len(s.saves)-1]
	data := last.ResumeToken.Lookup("_data").StringValue()
	assert.Equal(t, clusterTime(data), last.ClusterTime)
	return data
}

func token(data string) []byte {
	return []byte(`{"_data":"` + data + `"}`)
}

func clusterTime(data string) bson.Timestamp {
	return bson.Timestamp{T: 100, I: uint32(data[0])}
}

func track(tracker *Tracker, data string) {
	tracker.Track(token(data), clusterTime(data))
}

func delivery(resumeToken []byte, err error) *kafkaconfluent.Message {
	topic := "topic"
	return &kafkaconfluent.Message{
		TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic, Error: err},
		Headers:        []kafkaconfluent.Header{{Key: kafka.XResumeTokenHeaderName, Value: resumeToken}},
	}
}

func TestTrackerCommitsContiguousAckedPrefix(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	track(tracker, "1")
	track(tracker, "2")
	track(tracker, "3")

	// out of order ack: nothing can be committed yet
	tracker.OnDelivery(delivery(token("2"), nil))
	assert.Nil(t, tracker.Flush(ctx))
	assert.Len(t, store.saves, 0)

	tracker.OnDelivery(delivery(token("1"), nil))
	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "2", store.lastData(t))

	tracker.OnDelivery(delivery(token("3"), nil))
	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "3", store.lastData(t))
}

func TestTrackerFlushOnlyWhenChanged(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	track(tracker, "1")
	tracker.OnDelivery(delivery(token("1"), nil))

	assert.Nil(t, tracker.Flush(ctx))
	assert.Nil(t, tracker.Flush(ctx))
	assert.Len(t, store.saves, 1)
}

func TestTrackerFailedDeliveryBlocksCheckpoint(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	track(tracker, "1")
	track(tracker, "2")
	track(tracker, "3")

	tracker.OnDelivery(delivery(token("1"), nil))
	tracker.OnDelivery(delivery(token("2"), errors.New("delivery failed")))
	tracker.OnDelivery(delivery(token("3"), nil))

	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "1", store.lastData(t))
}

func TestTrackerIgnoresMessagesWithoutToken(t *testing.T) {
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	topic := "topic"
	tracker.OnDelivery(&kafkaconfluent.Message{TopicPartition: kafkaconfluent.TopicPartition{Topic: &topic}})
	tracker.OnDelivery(delivery(token("unknown"), nil))

	assert.Nil(t, tracker.Flush(context.Background()))
	assert.Len(t, store.saves, 0)
}

func TestTrackerSkipsEventsWithoutClusterTime(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	track(tracker, "1")
	tracker.Track(token("2"), bson.Timestamp{})
	tracker.OnDelivery(delivery(token("1"), nil))
	tracker.OnDelivery(delivery(token("2"), nil))

	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "1", store.lastData(t))
}

func TestCheckpointAge(t *testing.T) {
	now := time.Unix(1791463102, 0)
	checkpoint := Checkpoint{ClusterTime: bson.Timestamp{T: 1791460981, I: 78}}

	assert.Equal(t, 2121*time.Second, checkpoint.Age(now))
}

// blockingStore blocks the first save until released
type blockingStore struct {
	mu      sync.Mutex
	saves   []Checkpoint
	entered chan struct{}
	release chan struct{}
}

func (s *blockingStore) Load(context.Context) (*Checkpoint, error) { return nil, nil }

func (s *blockingStore) Save(_ context.Context, checkpoint Checkpoint) error {
	s.mu.Lock()
	first := len(s.saves) == 0
	s.saves = append(s.saves, checkpoint)
	s.mu.Unlock()
	if first {
		close(s.entered)
		<-s.release
	}
	return nil
}

func (s *blockingStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.saves)
}

func TestTrackerConcurrentFlushesDoNotRegress(t *testing.T) {
	store := &blockingStore{entered: make(chan struct{}), release: make(chan struct{})}
	tracker := NewTracker(store, logger.NewNopLogger())

	track(tracker, "a")
	tracker.OnDelivery(delivery(token("a"), nil))
	first := make(chan error)
	go func() { first <- tracker.Flush(context.Background()) }()
	<-store.entered

	track(tracker, "b")
	tracker.OnDelivery(delivery(token("b"), nil))
	second := make(chan error)
	go func() { second <- tracker.Flush(context.Background()) }()

	// the second flush waits for the first one
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, store.count())

	close(store.release)
	assert.NoError(t, <-first)
	assert.NoError(t, <-second)

	assert.Equal(t, []bson.Timestamp{clusterTime("a"), clusterTime("b")},
		[]bson.Timestamp{store.saves[0].ClusterTime, store.saves[1].ClusterTime})
}

func TestTrackerFlushSkipsPositionNotAfterSavedOne(t *testing.T) {
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	// two events of the same transaction share their cluster time
	tracker.Track(token("a1"), clusterTime("a"))
	tracker.Track(token("a2"), clusterTime("a"))
	tracker.OnDelivery(delivery(token("a1"), nil))
	assert.NoError(t, tracker.Flush(context.Background()))
	tracker.OnDelivery(delivery(token("a2"), nil))
	assert.NoError(t, tracker.Flush(context.Background()))

	assert.Len(t, store.saves, 1)
}

func TestTrackerSkipsPermanentDeliveryFailure(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())
	tracker.SetFailureHandler(func(error) { t.Fatal("a permanent failure must not stop the application") })

	track(tracker, "1")
	track(tracker, "2")
	track(tracker, "3")

	tracker.OnDelivery(delivery(token("1"), nil))
	tracker.OnDelivery(delivery(token("2"), kafkaconfluent.NewError(kafkaconfluent.ErrMsgSizeTooLarge, "too large", false)))
	tracker.OnDelivery(delivery(token("3"), nil))

	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "3", store.lastData(t))
}

func TestTrackerTransientDeliveryFailureCallsHandlerOnce(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	var failures []error
	tracker.SetFailureHandler(func(err error) { failures = append(failures, err) })

	track(tracker, "1")
	track(tracker, "2")
	track(tracker, "3")

	timedOut := kafkaconfluent.NewError(kafkaconfluent.ErrMsgTimedOut, "timed out", false)
	tracker.OnDelivery(delivery(token("1"), nil))
	tracker.OnDelivery(delivery(token("2"), timedOut))
	tracker.OnDelivery(delivery(token("3"), timedOut))

	assert.Equal(t, []error{timedOut}, failures)
	// the checkpoint stops before the failed event, which is replayed on restart
	assert.Nil(t, tracker.Flush(ctx))
	assert.Equal(t, "1", store.lastData(t))
}

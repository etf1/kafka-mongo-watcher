package checkpoint

import (
	"context"
	"errors"
	"testing"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type memoryStore struct {
	saves []bson.Raw
}

func (s *memoryStore) Load(context.Context) (bson.Raw, error) {
	if len(s.saves) == 0 {
		return nil, nil
	}
	return s.saves[len(s.saves)-1], nil
}

func (s *memoryStore) Save(_ context.Context, resumeToken bson.Raw) error {
	s.saves = append(s.saves, resumeToken)
	return nil
}

func (s *memoryStore) lastData(t *testing.T) string {
	if len(s.saves) == 0 {
		return ""
	}
	return s.saves[len(s.saves)-1].Lookup("_data").StringValue()
}

func token(data string) []byte {
	return []byte(`{"_data":"` + data + `"}`)
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

	tracker.Track(token("1"))
	tracker.Track(token("2"))
	tracker.Track(token("3"))

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

	tracker.Track(token("1"))
	tracker.OnDelivery(delivery(token("1"), nil))

	assert.Nil(t, tracker.Flush(ctx))
	assert.Nil(t, tracker.Flush(ctx))
	assert.Len(t, store.saves, 1)
}

func TestTrackerFailedDeliveryBlocksCheckpoint(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	tracker := NewTracker(store, logger.NewNopLogger())

	tracker.Track(token("1"))
	tracker.Track(token("2"))
	tracker.Track(token("3"))

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

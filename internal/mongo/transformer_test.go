package mongo

import (
	"testing"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTransformChangeEventToKafkaMessageWhenHaveEvents(t *testing.T) {
	// Given
	logger := logger.NewNopLogger()
	topic := "my-test-topic"
	messages := make(chan *kafka.Message)

	events := make(chan *ChangeEvent)
	go func() {
		objectID, _ := primitive.ObjectIDFromHex("5ccfdbb519580ee49d50803c")
		events <- &ChangeEvent{
			DocumentKey: documentKey{ID: objectID},
			Document:    bson.M{"hello": "this-is-my-test"},
		}

		objectID, _ = primitive.ObjectIDFromHex("5ccfdbb519580ee49d50803d")
		events <- &ChangeEvent{
			DocumentKey: documentKey{ID: objectID},
			Document:    bson.M{"hello": "this-is-my-second-test-event"},
		}
	}()

	// When
	go TransformChangeEventToKafkaMessage(logger, topic, events, messages)

	// Then
	assert := assert.New(t)

	// First message
	message := <-messages
	assert.Equal("my-test-topic", message.Topic)
	expectedKey := []byte(`5ccfdbb519580ee49d50803c`)
	assert.Equal(expectedKey, message.Key)
	expectedValue := []byte(`{"_id":null,"operationType":"","fullDocument":{"hello":"this-is-my-test"},"ns":null,"documentKey":{"_id":{"$oid":"5ccfdbb519580ee49d50803c"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`)
	assert.Equal(expectedValue, message.Value)

	// Second message
	message = <-messages
	assert.Equal("my-test-topic", message.Topic)
	expectedKey = []byte(`5ccfdbb519580ee49d50803d`)
	assert.Equal(expectedKey, message.Key)
	expectedValue = []byte(`{"_id":null,"operationType":"","fullDocument":{"hello":"this-is-my-second-test-event"},"ns":null,"documentKey":{"_id":{"$oid":"5ccfdbb519580ee49d50803d"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`)
	assert.Equal(expectedValue, message.Value)
}

func TestTransformChangeEventToKafkaMessageWhenDocumentIDError(t *testing.T) {
	// Given
	logger := logger.NewNopLogger()
	topic := "my-test-topic"
	messages := make(chan *kafka.Message)

	events := make(chan *ChangeEvent)
	go func() {
		objectID, _ := primitive.ObjectIDFromHex("incorrect-document-id")
		events <- &ChangeEvent{
			DocumentKey: documentKey{ID: objectID},
			Document:    bson.M{"hello": "this-is-my-test"},
		}

		objectID, _ = primitive.ObjectIDFromHex("5ccfdbb519580ee49d50803d")
		events <- &ChangeEvent{
			DocumentKey: documentKey{ID: objectID},
			Document:    bson.M{"hello": "this-is-my-second-test-event"},
		}
	}()

	// When
	go TransformChangeEventToKafkaMessage(logger, topic, events, messages)

	// Then
	assert := assert.New(t)

	// Second message is retrieved, not the first one
	message := <-messages
	assert.Equal("my-test-topic", message.Topic)
	expectedKey := []byte(`5ccfdbb519580ee49d50803d`)
	assert.Equal(expectedKey, message.Key)
	expectedValue := []byte(`{"_id":null,"operationType":"","fullDocument":{"hello":"this-is-my-second-test-event"},"ns":null,"documentKey":{"_id":{"$oid":"5ccfdbb519580ee49d50803d"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`)
	assert.Equal(expectedValue, message.Value)
}

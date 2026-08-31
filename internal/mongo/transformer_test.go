package mongo

import (
	"testing"

	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
)

func TestTransformChangeEventToKafkaMessageWhenHaveEvents(t *testing.T) {
	// Given
	topic := "my-test-topic"

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

	transformer := NewChangeEventKafkaMessageTransformer(topic, nil, logger.NewNopLogger())

	// When
	messages := transformer.Transform(events)

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
	topic := "my-test-topic"

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

	transformer := NewChangeEventKafkaMessageTransformer(topic, nil, logger.NewNopLogger())

	// When
	messages := transformer.Transform(events)

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

func TestParseDocumentHeaders(t *testing.T) {
	tests := []struct {
		name     string
		mapping  string
		expected []DocumentHeader
		wantErr  bool
	}{
		{name: "empty", mapping: "", expected: nil},
		{name: "blank", mapping: " , ", expected: nil},
		{name: "single", mapping: "x-update-source=last_update_source", expected: []DocumentHeader{{Key: "x-update-source", Path: []string{"last_update_source"}}}},
		{name: "several with spaces and nested path", mapping: " x-a = a , x-b=props.b ", expected: []DocumentHeader{{Key: "x-a", Path: []string{"a"}}, {Key: "x-b", Path: []string{"props", "b"}}}},
		{name: "missing separator", mapping: "x-a", wantErr: true},
		{name: "empty field segment", mapping: "x-a=a..b", wantErr: true},
		{name: "trailing dot", mapping: "x-a=a.", wantErr: true},
		{name: "leading dot", mapping: "x-a=.a", wantErr: true},
		{name: "missing header", mapping: "=a", wantErr: true},
		{name: "missing field", mapping: "x-a=", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := ParseDocumentHeaders(tt.mapping)

			assert.Equal(t, tt.wantErr, err != nil, "error: %v", err)
			assert.Equal(t, tt.expected, headers)
		})
	}
}

func TestTransformChangeEventToKafkaMessageWithDocumentHeaders(t *testing.T) {
	headers := []DocumentHeader{
		{Key: "x-update-source", Path: []string{"last_update_source"}},
		{Key: "x-nested", Path: []string{"props", "nested"}},
	}
	objectID, _ := primitive.ObjectIDFromHex("5ccfdbb519580ee49d50803c")

	tests := []struct {
		name     string
		document bson.M
		expected []kafka.Header
	}{
		{
			name:     "fields present",
			document: bson.M{"last_update_source": "blackmirror", "props": bson.M{"nested": "x"}},
			expected: []kafka.Header{{Key: "x-update-source", Value: []byte("blackmirror")}, {Key: "x-nested", Value: []byte("x")}},
		},
		{
			name:     "fields missing",
			document: bson.M{"title": "Lucy", "props": bson.M{"other": "x"}},
			expected: nil,
		},
		{
			name:     "field empty",
			document: bson.M{"last_update_source": ""},
			expected: nil,
		},
		{
			name:     "fields not strings",
			document: bson.M{"last_update_source": 42, "props": "not-a-document"},
			expected: nil,
		},
		{
			name:     "no full document",
			document: nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			events := make(chan *ChangeEvent, 1)
			events <- &ChangeEvent{DocumentKey: documentKey{ID: objectID}, Document: tt.document}
			close(events)

			transformer := NewChangeEventKafkaMessageTransformer("my-test-topic", headers, logger.NewNopLogger())

			// When
			message := <-transformer.Transform(events)

			// Then
			assert.Equal(t, tt.expected, message.Headers)
		})
	}
}

func TestDocumentHeaderValue(t *testing.T) {
	tests := []struct {
		name     string
		document bson.M
		path     []string
		expected string
		found    bool
	}{
		{name: "empty path", document: bson.M{"a": "x"}, path: nil, expected: "", found: false},
		{name: "nil document", document: nil, path: []string{"a"}, expected: "", found: false},
		{name: "empty document", document: bson.M{}, path: []string{"a"}, expected: "", found: false},
		{name: "top-level string", document: bson.M{"a": "x"}, path: []string{"a"}, expected: "x", found: true},
		{name: "empty string is found", document: bson.M{"a": ""}, path: []string{"a"}, expected: "", found: true},
		{name: "nil value", document: bson.M{"a": nil}, path: []string{"a"}, expected: "", found: false},
		{name: "nil value in the path", document: bson.M{"a": nil}, path: []string{"a", "b"}, expected: "", found: false},
		{name: "deeply nested string", document: bson.M{"a": bson.M{"b": bson.M{"c": "x"}}}, path: []string{"a", "b", "c"}, expected: "x", found: true},
		{name: "path through a string", document: bson.M{"a": "x"}, path: []string{"a", "b"}, expected: "", found: false},
		{name: "path ending on a document", document: bson.M{"a": bson.M{"b": "x"}}, path: []string{"a"}, expected: "", found: false},
		{name: "array in the path", document: bson.M{"a": primitive.A{bson.M{"b": "x"}}}, path: []string{"a", "0", "b"}, expected: "", found: false},
		{name: "nested bson.D", document: bson.M{"a": bson.D{{Key: "b", Value: "x"}}}, path: []string{"a", "b"}, expected: "x", found: true},
		{name: "bson.D nested in bson.D", document: bson.M{"a": bson.D{{Key: "b", Value: bson.D{{Key: "c", Value: "x"}}}}}, path: []string{"a", "b", "c"}, expected: "x", found: true},
		{name: "bson.M nested in bson.D", document: bson.M{"a": bson.D{{Key: "b", Value: bson.M{"c": "x"}}}}, path: []string{"a", "b", "c"}, expected: "x", found: true},
		{name: "key missing in bson.D", document: bson.M{"a": bson.D{{Key: "b", Value: "x"}}}, path: []string{"a", "c"}, expected: "", found: false},
		{name: "duplicate key in bson.D takes the first", document: bson.M{"a": bson.D{{Key: "b", Value: "x"}, {Key: "b", Value: "y"}}}, path: []string{"a", "b"}, expected: "x", found: true},
		{name: "nested plain map", document: bson.M{"a": map[string]any{"b": "x"}}, path: []string{"a", "b"}, expected: "x", found: true},
		{name: "int32", document: bson.M{"a": int32(42)}, path: []string{"a"}, expected: "", found: false},
		{name: "int64", document: bson.M{"a": int64(42)}, path: []string{"a"}, expected: "", found: false},
		{name: "float64", document: bson.M{"a": 4.2}, path: []string{"a"}, expected: "", found: false},
		{name: "bool", document: bson.M{"a": true}, path: []string{"a"}, expected: "", found: false},
		{name: "object id", document: bson.M{"a": primitive.NewObjectID()}, path: []string{"a"}, expected: "", found: false},
		{name: "date time", document: bson.M{"a": primitive.DateTime(0)}, path: []string{"a"}, expected: "", found: false},
		{name: "binary", document: bson.M{"a": primitive.Binary{Data: []byte("x")}}, path: []string{"a"}, expected: "", found: false},
		{name: "symbol", document: bson.M{"a": primitive.Symbol("x")}, path: []string{"a"}, expected: "", found: false},
		{name: "key containing a dot is not split", document: bson.M{"a.b": "x"}, path: []string{"a", "b"}, expected: "", found: false},
		{name: "key containing a dot as a single segment", document: bson.M{"a.b": "x"}, path: []string{"a.b"}, expected: "x", found: true},
		{name: "case sensitive", document: bson.M{"Last_Update_Source": "x"}, path: []string{"last_update_source"}, expected: "", found: false},
		{name: "empty segment", document: bson.M{"a": bson.M{"b": "x"}}, path: []string{"a", ""}, expected: "", found: false},
		{name: "unicode key", document: bson.M{"clé": "x"}, path: []string{"clé"}, expected: "x", found: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := DocumentHeader{Key: "x", Path: tt.path}.value(tt.document)

			assert.Equal(t, tt.expected, value)
			assert.Equal(t, tt.found, found)
		})
	}
}

// Exercises value against a change event decoded by the driver, whose nested documents are bson.M.
func TestDocumentHeaderValueFromDecodedEvent(t *testing.T) {
	// Given
	raw := `{"_id":{"_data":"8265"},"operationType":"update","fullDocument":{"_id":{"$oid":"5ccfdbb519580ee49d50803c"},"last_update_source":"blackmirror","version":{"$numberLong":"41"},"props":{"nested":"x","deeper":{"leaf":"y"},"list":[{"item":"z"}]}},"clusterTime":{"$timestamp":{"t":1,"i":1}}}`

	var event ChangeEvent
	if err := bson.UnmarshalExtJSON([]byte(raw), true, &event); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     []string
		expected string
		found    bool
	}{
		{name: "top-level string", path: []string{"last_update_source"}, expected: "blackmirror", found: true},
		{name: "nested string", path: []string{"props", "nested"}, expected: "x", found: true},
		{name: "deeply nested string", path: []string{"props", "deeper", "leaf"}, expected: "y", found: true},
		{name: "object id", path: []string{"_id"}, expected: "", found: false},
		{name: "number", path: []string{"version"}, expected: "", found: false},
		{name: "array item", path: []string{"props", "list", "0", "item"}, expected: "", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := DocumentHeader{Key: "x", Path: tt.path}.value(event.Document)

			assert.Equal(t, tt.expected, value)
			assert.Equal(t, tt.found, found)
		})
	}
}

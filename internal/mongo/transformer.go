package mongo

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
)

// DocumentHeader maps a message header to the field of the change event full document carried as its value
type DocumentHeader struct {
	Key  string
	Path []string
}

// ParseDocumentHeaders parses a comma-separated list of "header=field" pairs, a field being a
// dot-separated path inside the change event full document
func ParseDocumentHeaders(mapping string) ([]DocumentHeader, error) {
	var headers []DocumentHeader

	for _, pair := range strings.Split(mapping, ",") {
		if pair = strings.TrimSpace(pair); pair == "" {
			continue
		}

		key, field, found := strings.Cut(pair, "=")
		key, field = strings.TrimSpace(key), strings.TrimSpace(field)
		if !found || key == "" || field == "" {
			return nil, fmt.Errorf("invalid document header %q: expected header=field", pair)
		}

		path := strings.Split(field, ".")
		if slices.Contains(path, "") {
			return nil, fmt.Errorf("invalid document header %q: empty field segment", pair)
		}

		headers = append(headers, DocumentHeader{Key: key, Path: path})
	}

	return headers, nil
}

// value returns the string value of the field in the document, if any, walking the path
// through nested documents of any form: bson.M, bson.D or a plain map
func (h DocumentHeader) value(document bson.M) (string, bool) {
	if len(h.Path) == 0 {
		return "", false
	}

	var value any = document

	for _, key := range h.Path {
		found := false
		switch nested := value.(type) {
		case bson.M:
			value, found = nested[key]
		case map[string]any:
			value, found = nested[key]
		case bson.D:
			for _, element := range nested {
				if element.Key == key {
					value, found = element.Value, true
					break
				}
			}
		}
		if !found {
			return "", false
		}
	}

	str, ok := value.(string)
	return str, ok
}

// ChangeEventKafkaMessageTransformer transforms mongodb change events into a format that will be used by the kafka client
type ChangeEventKafkaMessageTransformer struct {
	topic   string
	headers []DocumentHeader
	logger  logger.LoggerInterface
}

func (t *ChangeEventKafkaMessageTransformer) Transform(changeEvents chan *ChangeEvent) chan *kafka.Message {
	var messageChan = make(chan *kafka.Message, len(changeEvents))
	go func() {
		defer close(messageChan)
		for event := range changeEvents {
			documentID, err := event.documentID()
			if err != nil {
				t.logger.Error("Mongo transformer: Unable to extract document id from event", logger.Error("error", err))
				continue
			}

			jsonBytes, err := event.marshal()
			if err != nil {
				t.logger.Error("Mongo transformer: Unable to unmarshal change event to json", logger.Error("error", err))
				continue
			}

			t.logger.Info("Mongo transformer: Retrieve event", logger.String("document_id", documentID), logger.ByteString("event", jsonBytes))

			messageChan <- &kafka.Message{
				Topic:   t.topic,
				Key:     []byte(documentID),
				Value:   jsonBytes,
				Headers: t.documentHeaders(event),
			}
		}
	}()
	return messageChan
}

// documentHeaders builds the headers carrying the configured fields of the event full document,
// skipping missing, empty or non-string fields
func (t *ChangeEventKafkaMessageTransformer) documentHeaders(event *ChangeEvent) []kafka.Header {
	var headers []kafka.Header
	for _, header := range t.headers {
		if str, ok := header.value(event.Document); ok && str != "" {
			headers = append(headers, kafka.Header{Key: header.Key, Value: []byte(str)})
		}
	}
	return headers
}

func NewChangeEventKafkaMessageTransformer(topic string, headers []DocumentHeader, logger logger.LoggerInterface) *ChangeEventKafkaMessageTransformer {
	return &ChangeEventKafkaMessageTransformer{
		topic:   topic,
		headers: headers,
		logger:  logger,
	}
}

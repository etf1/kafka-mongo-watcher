package mongo

import (
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
)

// ChangeEventKafkaMessageTransformer transforms mongodb change events into a format that will be used by the kafka client
type ChangeEventKafkaMessageTransformer struct {
	topic  string
	logger logger.LoggerInterface
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
				Topic: t.topic,
				Key:   []byte(documentID),
				Value: jsonBytes,
			}
		}
	}()
	return messageChan
}

func NewChangeEventKafkaMessageTransformer(topic string, logger logger.LoggerInterface) *ChangeEventKafkaMessageTransformer {
	return &ChangeEventKafkaMessageTransformer{
		topic:  topic,
		logger: logger,
	}
}

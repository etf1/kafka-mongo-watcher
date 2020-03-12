package mongo

import (
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/gol4ng/logger"
)

// TransformChangeEventToKafkaMessage transforms mongodb change events into a format that will be used by the kafka client
func TransformChangeEventToKafkaMessage(l logger.LoggerInterface, topic string, events chan *ChangeEvent, messages chan *kafka.Message) {
	for event := range events {
		documentID, err := event.documentID()
		if err != nil {
			l.Error("Mongo transformer: Unable to extract document id from event", logger.Error("error", err))
			continue
		}

		jsonBytes, err := event.marshal()
		if err != nil {
			println(err.Error())
			l.Error("Mongo transformer: Unable to unmarshal change event to json", logger.Error("error", err))
			continue
		}

		l.Info("Mongo transformer: Retrieve event", logger.String("document_id", documentID), logger.ByteString("event", jsonBytes))

		messages <- &kafka.Message{
			Topic: topic,
			Key:   []byte(documentID),
			Value: jsonBytes,
		}
	}
}

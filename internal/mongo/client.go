package mongo

import (
	"context"

	"github.com/gol4ng/logger"
)

type Option func(*client)

type Client interface {
	Oplogs(ctx context.Context, collection CollectionAdapter) (chan *WatchItem, error)
}

type client struct {
	logger logger.LoggerInterface
}

func newClient(options ...Option) *client {
	client := &client{
		logger: logger.NewNopLogger(),
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// WithLogger allows to specify a logger
func WithLogger(logger logger.LoggerInterface) Option {
	return func(c *client) {
		c.logger = logger
	}
}

func (c *client) loop(ctx context.Context, cursor DriverCursor, itemsChan chan *WatchItem) {
	for cursor.Next(ctx) {
		var event ChangeEvent
		if err := cursor.Decode(&event); err != nil {
			c.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
		}

		if err := c.sendIntoChannel(event, itemsChan); err != nil {
			c.logger.Error("Mongo client: Unable to send document", logger.Error("error", err))
		}
	}
}

func (c *client) sendIntoChannel(event ChangeEvent, itemsChan chan *WatchItem) error {
	docID, err := event.documentID()
	if err != nil {
		c.logger.Error("Mongo client: Unable to extract document id from event", logger.Error("error", err))
		return err
	}
	jsonBytes, err := event.marshal()
	if err != nil {
		c.logger.Error("Mongo client: Unable to unmarshal change event to json", logger.Error("error", err))
		return err
	}

	c.logger.Info("Mongo client: Retrieve event", logger.String("document_id", docID), logger.ByteString("event", jsonBytes))

	itemsChan <- &WatchItem{
		Key:   []byte(docID),
		Value: jsonBytes,
	}

	return nil
}

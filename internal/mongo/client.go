package mongo

import (
	"context"

	"github.com/gol4ng/logger"
)

type Option func(*client)

type Client interface {
	Oplogs(ctx context.Context, collection CollectionAdapter) (chan *ChangeEvent, error)
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

func (c *client) sendEvents(ctx context.Context, cursor DriverCursor, events chan *ChangeEvent) {
	for cursor.Next(ctx) {
		event := &ChangeEvent{}
		if err := cursor.Decode(event); err != nil {
			c.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
			continue
		}

		events <- event
	}
}

package mongo

import (
	"context"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionAdapter interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (MongoDriverCursor, error)
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (MongoDriverCursor, error)
	Name() string
}

type collectionAdapter struct {
	collection *mongodriver.Collection
}

func NewCollectionAdapter(collection *mongodriver.Collection) *collectionAdapter {
	return &collectionAdapter{
		collection: collection,
	}
}

func (c *collectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (MongoDriverCursor, error) {
	return c.collection.Aggregate(ctx, pipeline, opts...)
}

func (c *collectionAdapter) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (MongoDriverCursor, error) {
	return c.collection.Watch(ctx, pipeline, opts...)
}

func (c *collectionAdapter) Name() string {
	return c.collection.Name()
}

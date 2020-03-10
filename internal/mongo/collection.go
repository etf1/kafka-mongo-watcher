package mongo

import (
	"context"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DriverDatabase interface {
	Name() string
}

type CollectionAdapter interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (DriverCursor, error)
	Database() DriverDatabase
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (DriverCursor, error)
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

func (c *collectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (DriverCursor, error) {
	return c.collection.Aggregate(ctx, pipeline, opts...)
}

func (c *collectionAdapter) Database() DriverDatabase {
	return c.collection.Database()
}

func (c *collectionAdapter) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (DriverCursor, error) {
	return c.collection.Watch(ctx, pipeline, opts...)
}

func (c *collectionAdapter) Name() string {
	return c.collection.Name()
}

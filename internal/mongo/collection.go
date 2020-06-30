package mongo

import (
	"context"

	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DriverCursor represents a mongo-driver/mongo cursor object
// it can be both a Cursor or a ChangeStream for instance
type DriverCursor interface {
	Decode(val interface{}) error
	Next(ctx context.Context) bool
	TryNext(ctx context.Context) bool
	Err() error
	ID() int64
	Close(ctx context.Context) error
}

// DriverDatabase represents the mongo-driver/mongo database object
type DriverDatabase interface {
	Name() string
}

// CollectionAdapter is a wrapper over the mongo-driver/mongo collection object
// mainly allowing us to use interfaces
type CollectionAdapter interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (DriverCursor, error)
	Database() DriverDatabase
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (DriverCursor, error)
	Name() string
}

type collectionAdapter struct {
	collection *mongodriver.Collection
}

// NewCollectionAdapter returns a mongo-driver/mongo collection wrapper
func NewCollectionAdapter(collection *mongodriver.Collection) *collectionAdapter {
	return &collectionAdapter{
		collection: collection,
	}
}

// Aggregate sends an aggregate query using mongo-driver/mongo
func (c *collectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (DriverCursor, error) {
	return c.collection.Aggregate(ctx, pipeline, opts...)
}

// Database returns the database object using mongo-driver/mongo
func (c *collectionAdapter) Database() DriverDatabase {
	return c.collection.Database()
}

// Watch sends an watch query using mongo-driver/mongo
func (c *collectionAdapter) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (DriverCursor, error) {
	return c.collection.Watch(ctx, pipeline, opts...)
}

// Name returns the current collection name
func (c *collectionAdapter) Name() string {
	return c.collection.Name()
}

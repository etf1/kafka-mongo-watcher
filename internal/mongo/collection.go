package mongo

import (
	"context"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AggregateCursor represents a mongo-driver/mongo Cursor object
type AggregateCursor interface {
	Decode(val interface{}) error
	Next(ctx context.Context) bool
	TryNext(ctx context.Context) bool
	Close(ctx context.Context) error
}

// StreamCursor represents a mongo-driver/mongo ChangeStream object
type StreamCursor interface {
	Decode(val interface{}) error
	Next(ctx context.Context) bool
	TryNext(ctx context.Context) bool
	Close(ctx context.Context) error
	Err() error
	ID() int64
	ResumeToken() bson.Raw
}

// DriverDatabase represents the mongo-driver/mongo database object
type DriverDatabase interface {
	Name() string
}

// CollectionAdapter is a wrapper over the mongo-driver/mongo collection object
// mainly allowing us to use interfaces
type CollectionAdapter interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (AggregateCursor, error)
	Database() DriverDatabase
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (StreamCursor, error)
	Name() string
}

type collectionAdapter struct {
	collection *mongodriver.Collection
}

type cursorCloser interface {
	Close(context.Context) error
}

func closeCursor(cursor cursorCloser) {
	// Guard against both nil interface and typed nil (e.g. (*mongo.ChangeStream)(nil)
	// returned alongside an error by the driver — the interface is non-nil but the
	// underlying pointer is nil, which would panic on Close).
	if cursor == nil || reflect.ValueOf(cursor).IsNil() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Error is intentionally ignored: Close() during cleanup typically fails only when
	// the cursor or connection is already gone, which is acceptable in a shutdown path.
	_ = cursor.Close(ctx)
}

// NewCollectionAdapter returns a mongo-driver/mongo collection wrapper
func NewCollectionAdapter(collection *mongodriver.Collection) *collectionAdapter {
	return &collectionAdapter{
		collection: collection,
	}
}

// Aggregate sends an aggregate query using mongo-driver/mongo
func (c *collectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (AggregateCursor, error) {
	return c.collection.Aggregate(ctx, pipeline, opts...)
}

// Database returns the database object using mongo-driver/mongo
func (c *collectionAdapter) Database() DriverDatabase {
	return c.collection.Database()
}

// Watch sends an watch query using mongo-driver/mongo
func (c *collectionAdapter) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (StreamCursor, error) {
	return c.collection.Watch(ctx, pipeline, opts...)
}

// Name returns the current collection name
func (c *collectionAdapter) Name() string {
	return c.collection.Name()
}

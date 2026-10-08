package checkpoint

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Checkpoint is the position of the last event acknowledged by Kafka
type Checkpoint struct {
	// ClusterTime is the operation time of the event, used to resume the change stream
	ClusterTime bson.Timestamp `bson:"clusterTime"`
	// ResumeToken is the resume token of the event, kept for information
	ResumeToken bson.Raw `bson:"resumeToken,omitempty"`
}

// Store persists the change stream checkpoint
type Store interface {
	// Load returns the stored checkpoint, or nil if there is none
	Load(ctx context.Context) (*Checkpoint, error)
	Save(ctx context.Context, checkpoint Checkpoint) error
}

type mongoStore struct {
	collection *mongodriver.Collection
	id         string
}

// NewMongoStore returns a store saving the checkpoint in a single document of the given collection
func NewMongoStore(collection *mongodriver.Collection, id string) *mongoStore {
	return &mongoStore{
		collection: collection,
		id:         id,
	}
}

// Load returns the stored checkpoint, or nil if there is none. A document without
// clusterTime (written by a previous version) is ignored.
func (s *mongoStore) Load(ctx context.Context) (*Checkpoint, error) {
	var checkpoint Checkpoint
	err := s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: s.id}}).Decode(&checkpoint)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if checkpoint.ClusterTime.IsZero() {
		return nil, nil
	}
	return &checkpoint, nil
}

// Save upserts the checkpoint
func (s *mongoStore) Save(ctx context.Context, checkpoint Checkpoint) error {
	_, err := s.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: s.id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "clusterTime", Value: checkpoint.ClusterTime},
			{Key: "resumeToken", Value: checkpoint.ResumeToken},
			{Key: "updatedAt", Value: time.Now()},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

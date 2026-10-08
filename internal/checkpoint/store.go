package checkpoint

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Store persists the change stream resume token
type Store interface {
	// Load returns the stored resume token, or nil if there is none
	Load(ctx context.Context) (bson.Raw, error)
	Save(ctx context.Context, resumeToken bson.Raw) error
}

type document struct {
	ID          string    `bson:"_id"`
	ResumeToken bson.Raw  `bson:"resumeToken"`
	UpdatedAt   time.Time `bson:"updatedAt"`
}

type mongoStore struct {
	collection *mongodriver.Collection
	id         string
}

// NewMongoStore returns a store saving the resume token in a single document of the given collection
func NewMongoStore(collection *mongodriver.Collection, id string) *mongoStore {
	return &mongoStore{
		collection: collection,
		id:         id,
	}
}

// Load returns the stored resume token, or nil if there is none
func (s *mongoStore) Load(ctx context.Context) (bson.Raw, error) {
	var doc document
	err := s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: s.id}}).Decode(&doc)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc.ResumeToken, nil
}

// Save upserts the resume token
func (s *mongoStore) Save(ctx context.Context, resumeToken bson.Raw) error {
	_, err := s.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: s.id}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "resumeToken", Value: resumeToken},
			{Key: "updatedAt", Value: time.Now()},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

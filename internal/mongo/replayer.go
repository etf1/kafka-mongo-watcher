package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type replayer struct {
	*client
}

// NewReplayer returns a new mongodb client
func NewReplayer(options ...Option) *replayer {
	return &replayer{
		client: newClient(options...),
	}
}

// Do sends an aggregate query into mongodb to generate "oplogs-like" result of all the records
// in the collection
func (r *replayer) Oplogs(ctx context.Context, collection CollectionAdapter) (chan *ChangeEvent, error) {
	var pipeline = bson.A{
		bson.D{{Key: "$replaceRoot", Value: bson.D{
			{
				Key: "newRoot",
				Value: bson.M{
					"_id": bson.M{
						"_id":         "$_id",
						"copyingData": true,
					},
					"operationType": "insert",
					"ns": bson.M{
						"db":   collection.Database().Name(),
						"coll": collection.Name(),
					},
					"documentKey": bson.M{
						"_id": "$_id",
					},
					"fullDocument": "$$ROOT",
				},
			},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var events = make(chan *ChangeEvent)

	go func() {
		defer close(events)
		r.sendEvents(ctx, cursor, events)
	}()

	return events, nil
}

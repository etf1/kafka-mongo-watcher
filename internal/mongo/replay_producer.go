package mongo

import (
	"context"

	"github.com/etf1/kafka-mongo-watcher/internal/mongo/variables"
	"github.com/gol4ng/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type ReplayProducer struct {
	collection     CollectionAdapter
	logger         logger.LoggerInterface
	customPipeline string
}

func (r *ReplayProducer) Produce(ctx context.Context) (chan *ChangeEvent, error) {
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
						"db":   r.collection.Database().Name(),
						"coll": r.collection.Name(),
					},
					"documentKey": bson.M{
						"_id": "$_id",
					},
					"fullDocument": "$$ROOT",
				},
			},
		}}},
	}

	if r.customPipeline != "" {
		var customElements = bson.A{}
		var custom = []byte(variables.Replace(r.customPipeline))

		if err := bson.UnmarshalExtJSON(custom, true, &customElements); err != nil {
			return nil, err
		}

		pipeline = append(customElements, pipeline...)
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var events = make(chan *ChangeEvent)

	go func() {
		defer cursor.Close(ctx)
		defer close(events)

		r.sendEvents(ctx, cursor, events)
	}()

	return events, nil
}

func (r *ReplayProducer) sendEvents(ctx context.Context, cursor AggregateCursor, events chan *ChangeEvent) {
	for cursor.Next(ctx) {
		event := &ChangeEvent{}
		if err := cursor.Decode(event); err != nil {
			r.logger.Error("Mongo client: Unable to decode change event value from cursor", logger.Error("error", err))
			continue
		}

		events <- event
	}
}

func NewReplayProducer(adapter CollectionAdapter, logger logger.LoggerInterface, customPipeline string) *ReplayProducer {
	return &ReplayProducer{
		collection:     adapter,
		logger:         logger,
		customPipeline: customPipeline,
	}
}

package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func giveValidEvent(docKey bson.ObjectID) ChangeEvent {
	e := ChangeEvent{
		DocumentKey: documentKey{
			ID: docKey,
		},
	}
	return e
}

func giveInvalidEvent() ChangeEvent {
	return ChangeEvent{}
}

func Test_documentID(t *testing.T) {
	docKey := bson.NewObjectID()
	event := giveValidEvent(docKey)
	id, err := event.documentID()
	assert.NoError(t, err)
	assert.Equal(t, docKey.Hex(), id)

	event = giveInvalidEvent()
	_, err = event.documentID()
	assert.Error(t, err)
}

func TestChangeEventUnmarshalBSONKeepsClusterTimestamp(t *testing.T) {
	data, err := bson.Marshal(bson.D{
		{Key: "_id", Value: bson.D{{Key: "_data", Value: "826A1B2C3D"}}},
		{Key: "operationType", Value: "insert"},
		{Key: "clusterTime", Value: bson.Timestamp{T: 1791452154, I: 37}},
	})
	assert.Nil(t, err)

	var event ChangeEvent
	assert.Nil(t, bson.Unmarshal(data, &event))

	assert.Equal(t, "insert", event.Operation)
	assert.Equal(t, bson.Timestamp{T: 1791452154, I: 37}, event.ClusterTimestamp())
	assert.Equal(t, int64(1791452154), event.ClusterTime.Unix())
}

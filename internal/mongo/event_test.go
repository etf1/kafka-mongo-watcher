package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func giveValidEvent(docKey primitive.ObjectID) ChangeEvent {
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
	docKey := primitive.NewObjectID()
	event := giveValidEvent(docKey)
	id, err := event.documentID()
	assert.NoError(t, err)
	assert.Equal(t, docKey.Hex(), id)

	event = giveInvalidEvent()
	_, err = event.documentID()
	assert.Error(t, err)
}

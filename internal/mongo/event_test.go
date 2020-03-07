package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func giveValidEvent(hex string) bson.M {
	id, _ := primitive.ObjectIDFromHex(hex)
	idDoc := make(bson.M)
	idDoc[idField] = id
	event := make(bson.M)
	event[documentKeyField] = make(bson.M)
	event[documentKeyField] = idDoc
	return event
}

func giveInvalidEvent() bson.M {
	event := make(bson.M)
	event["invalid"] = "i am invalid"
	return event
}

func Test_isValid(t *testing.T) {
	event := giveValidEvent("5e3985237ed18b8e7c0c9f09")
	err := isValid(event)
	assert.NoError(t, err)

	event = giveInvalidEvent()
	err = isValid(event)
	assert.Error(t, err)
}

func Test_documentID(t *testing.T) {
	event := giveValidEvent("5e3985237ed18b8e7c0c9f09")
	id, err := documentID(event)
	assert.NoError(t, err)
	assert.Equal(t, "5e3985237ed18b8e7c0c9f09", id)

	event = giveInvalidEvent()
	id, err = documentID(event)
	assert.Error(t, err)
}

func Test_marshall(t *testing.T) {
	event := giveValidEvent("5e3985237ed18b8e7c0c9f09")
	_, err := marshall(event)
	assert.NoError(t, err)

	event = giveInvalidEvent()
	_, err = marshall(event)
	assert.Error(t, err)
}

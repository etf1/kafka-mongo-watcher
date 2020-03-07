package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const documentKeyField = "documentKey"
const idField = "_id"

// check if a change event if valid
// A valid event has at least a document key value
func isValid(event bson.M) error {
	_, err := documentID(event)
	return err
}

// marshall event to an array of bytes
func marshall(event bson.M) ([]byte, error) {
	if err := isValid(event); err != nil {
		return nil, fmt.Errorf("%v is not a valid change event : %w", event, err)
	}
	return bson.MarshalExtJSON(event, true, true)
}

// return the document id of the event
func documentID(event bson.M) (string, error) {
	rawKey, ok := event[documentKeyField]
	if !ok {
		return "", fmt.Errorf("%s is missing", documentKeyField)
	}
	key, ok := rawKey.(bson.M)
	if !ok {
		return "", fmt.Errorf("%v invalid type for %s", rawKey, documentKeyField)
	}
	rawID, ok := key[idField]
	if !ok {
		return "", fmt.Errorf("%s.%s is missing", documentKeyField, idField)
	}
	id, ok := rawID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("invalid type for %s.%s", documentKeyField, idField)
	}
	if id.IsZero() {
		return "", fmt.Errorf("%s.%s should not be empty", documentKeyField, idField)
	}
	return id.Hex(), nil
}

package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type documentKey struct {
	ID primitive.ObjectID `bson:"_id"`
}

// change event document according
// https://docs.mongodb.com/manual/reference/change-events/#change-stream-output
type changeEvent struct {
	ID                interface{} `bson:"_id"`
	Operation         string      `bson:"operationType"`
	Document          bson.M      `bson:"fullDocument"`
	Namespace         bson.M      `bson:"ns"`
	NewCollectionName bson.M      `bson:"to,omitempty"`
	DocumentKey       documentKey `bson:"documentKey"`
	Updates           bson.M      `bson:"updateDescription,omitempty"`
	ClusterTime       time.Time   `bson:"clusterTime"`
	Transaction       int64       `bson:"txnNumber,omitempty"`
	SessionID         bson.M      `bson:"lsid,omitempty"`
}

// marshall event to an array of bytes
func (e changeEvent) marshal() ([]byte, error) {
	return bson.MarshalExtJSON(e, true, true)
}

// return the document id of the event
func (e changeEvent) documentID() (string, error) {
	id := e.DocumentKey.ID
	if id.IsZero() {
		return "", fmt.Errorf("documentKey should not be empty")
	}
	return id.Hex(), nil
}

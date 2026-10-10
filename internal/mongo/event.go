package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type documentKey struct {
	ID bson.ObjectID `bson:"_id"`
}

// ChangeEvent document according
// https://docs.mongodb.com/manual/reference/change-events/#change-stream-output
type ChangeEvent struct {
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

	// clusterTimestamp is the raw clusterTime (seconds and increment), ClusterTime
	// only keeps the seconds. It is not part of the produced message.
	clusterTimestamp bson.Timestamp
}

// UnmarshalBSON decodes the event and keeps the raw clusterTime timestamp
func (e *ChangeEvent) UnmarshalBSON(data []byte) error {
	type rawChangeEvent ChangeEvent
	if err := bson.Unmarshal(data, (*rawChangeEvent)(e)); err != nil {
		return err
	}
	if value, err := bson.Raw(data).LookupErr("clusterTime"); err == nil {
		if t, i, ok := value.TimestampOK(); ok {
			e.clusterTimestamp = bson.Timestamp{T: t, I: i}
		}
	}
	return nil
}

// ClusterTimestamp returns the operation time of the event
func (e ChangeEvent) ClusterTimestamp() bson.Timestamp {
	return e.clusterTimestamp
}

// invalidatesStream returns true for the events that close the change stream
// (the collection was dropped or renamed). They have no documentKey.
func (e ChangeEvent) invalidatesStream() bool {
	switch e.Operation {
	case "invalidate", "drop", "rename", "dropDatabase":
		return true
	}
	return false
}

// marshall event to an array of bytes
func (e ChangeEvent) marshal() ([]byte, error) {
	return bson.MarshalExtJSON(e, true, true)
}

// return the document id of the event
func (e ChangeEvent) documentID() (string, error) {
	id := e.DocumentKey.ID
	if id.IsZero() {
		return "", fmt.Errorf("documentKey should not be empty")
	}
	return id.Hex(), nil
}

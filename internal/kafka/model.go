package kafka

import "go.mongodb.org/mongo-driver/v2/bson"

// Message is used over a channel that is filled by kafka transformer
type Message struct {
	Headers []Header
	Topic   string
	Key     []byte
	Value   []byte
	// ResumeToken is the MongoDB change stream resume token (extended JSON) of the event
	ResumeToken []byte
	// ClusterTime is the MongoDB operation time of the event
	ClusterTime bson.Timestamp
}

// Header represents a message header
type Header struct {
	Key   string // Header name (utf-8 string)
	Value []byte // Header value (nil, empty, or binary)
}

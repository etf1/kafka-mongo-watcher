package kafka

// Message is used over a channel that is filled by kafka transformer
type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

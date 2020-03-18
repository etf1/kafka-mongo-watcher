package kafka

// Message is used over a channel that is filled by kafka transformer
type Message struct {
	Headers []Header
	Topic   string
	Key     []byte
	Value   []byte
}

// Header represents a message header
type Header struct {
	Key   string // Header name (utf-8 string)
	Value []byte // Header value (nil, empty, or binary)
}

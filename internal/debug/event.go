package debug

type Event struct {
	Timestamp int64  `json:"timestamp"`
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Value     []byte `json:"value"`
}

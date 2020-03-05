package mongo

type WatchItem struct {
	Key   []byte
	Value []byte
}

type OperationLog struct {
	DocumentKey DocumentKey `json:"documentKey"`
}

type DocumentKey struct {
	ID string `json:"_id"`
}

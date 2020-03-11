package mongo

// WatchItem is used over a channel that is created/filled by mongo client
// and is later read by worker to process data
type WatchItem struct {
	Key   []byte
	Value []byte
}

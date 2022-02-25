package debug

import (
	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

type Debugger struct {
	cfg    *config.Base
	events chan *Event
}

func NewDebugger(cfg *config.Base) *Debugger {
	return &Debugger{
		cfg:    cfg,
		events: make(chan *Event, 1),
	}
}

func (d *Debugger) Add(message *kafka.Message) {
	if message == nil {
		return
	}

	var event *mongo.ChangeEvent
	err := bson.UnmarshalExtJSON(message.Value, true, &event)
	if err != nil {
		return
	}

	var document = event.Document

	if event.Operation == "update" {
		document = event.Updates
	}

	value, err := bson.MarshalExtJSON(document, true, true)
	if err != nil {
		return
	}

	d.events <- &Event{
		Timestamp: event.ClusterTime.Unix(),
		ID:        event.DocumentKey.ID.Hex(),
		Operation: event.Operation,
		Value:     value,
	}
}

func (d *Debugger) Context() map[string]interface{} {
	return map[string]interface{}{
		"collection": d.cfg.MongoDB.CollectionName,
		"database":   d.cfg.MongoDB.DatabaseName,
	}
}

func (d *Debugger) Events() chan *Event {
	return d.events
}

func (d *Debugger) Enabled() bool {
	return d != nil
}

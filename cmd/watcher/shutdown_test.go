package main

import (
	"context"
	"testing"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/stretchr/testify/assert"
)

func TestNotifyWhenClosedForwardsEvents(t *testing.T) {
	events := make(chan *mongo.ChangeEvent)
	forwarded, done := notifyWhenClosed(context.Background(), events)

	event := &mongo.ChangeEvent{Operation: "insert"}
	go func() {
		events <- event
		close(events)
	}()

	assert.Same(t, event, <-forwarded)
	_, ok := <-forwarded
	assert.False(t, ok)
	<-done
}

func TestNotifyWhenClosedDoesNotBlockProducerAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan *mongo.ChangeEvent)
	_, done := notifyWhenClosed(ctx, events)
	cancel()

	// nobody reads the forwarded channel: the producer must still be able to send
	// its last events and close its channel (i.e. close its cursor)
	sent := make(chan struct{})
	go func() {
		events <- &mongo.ChangeEvent{}
		events <- &mongo.ChangeEvent{}
		close(events)
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("producer blocked after cancellation")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("done was not closed")
	}
}

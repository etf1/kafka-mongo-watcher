package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// DeliveryHandler is called for each delivery report of a produced message
type DeliveryHandler func(message *kafka.Message)

// DeliveryDispatcher is the single reader of the producer events channel and
// forwards delivery reports to all registered handlers
type DeliveryDispatcher struct {
	handlers []DeliveryHandler
	done     chan struct{}
}

// NewDeliveryDispatcher returns a new delivery reports dispatcher
func NewDeliveryDispatcher() *DeliveryDispatcher {
	return &DeliveryDispatcher{
		done: make(chan struct{}),
	}
}

// Register adds a handler, it must be called before Run
func (d *DeliveryDispatcher) Register(handler DeliveryHandler) {
	d.handlers = append(d.handlers, handler)
}

// Run reads the events until the channel is closed (producer closed)
func (d *DeliveryDispatcher) Run(events chan kafka.Event) {
	defer close(d.done)
	for e := range events {
		message, ok := e.(*kafka.Message)
		if !ok || message == nil {
			continue
		}
		for _, handler := range d.handlers {
			handler(message)
		}
	}
}

// Done is closed once all events have been dispatched
func (d *DeliveryDispatcher) Done() <-chan struct{} {
	return d.done
}

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/etf1/kafka-mongo-watcher/internal/debug"
	"github.com/gol4ng/logger"
)

type Debugger interface {
	Events() chan *debug.Event
}

type Debug struct {
	logger         logger.LoggerInterface
	debugger       Debugger
	newClients     chan chan *debug.Event
	closingClients chan chan *debug.Event
	clients        map[chan *debug.Event]bool
}

// NewDebug returns the debug HTTP request handler
func NewDebug(logger logger.LoggerInterface, debugger Debugger) http.Handler {
	handler := &Debug{
		logger:         logger,
		debugger:       debugger,
		newClients:     make(chan chan *debug.Event),
		closingClients: make(chan chan *debug.Event),
		clients:        make(map[chan *debug.Event]bool),
	}

	go handler.listen()

	return handler
}

// ServeHTTP handles an HTTP request
func (h *Debug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "streaming not supported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new client message channel.
	messageChan := make(chan *debug.Event)

	// Set the new client message channel as active.
	h.newClients <- messageChan

	// When client disconnects, remove it from the active clients map.
	defer func() {
		h.closingClients <- messageChan
	}()

	// Listen for current active client events.
	for {
		select {
		case event := <-messageChan:
			eventBytes, err := json.Marshal(event)
			if err != nil {
				continue
			}

			data := fmt.Sprintf(
				"event: event\ndata: %v\n\n",
				string(eventBytes),
			)

			fmt.Fprint(w, data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (h *Debug) listen() {
	for {
		select {
		case s := <-h.newClients:

			h.clients[s] = true
			h.logger.Debug("New client connected", logger.Any("length", len(h.clients)))
		case s := <-h.closingClients:

			delete(h.clients, s)
			h.logger.Debug("Client disconnected", logger.Any("length", len(h.clients)))
		case event := <-h.debugger.Events():
			for clientMessageChan := range h.clients {
				clientMessageChan <- event
			}
		}
	}
}

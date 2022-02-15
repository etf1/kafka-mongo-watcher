package handler

import (
	"net/http"

	"github.com/gol4ng/logger"
)

type Liveness struct {
	logger logger.LoggerInterface
}

// NewLiveness returns a liveness HTTP request handler
func NewLiveness(logger logger.LoggerInterface) http.Handler {
	return Liveness{logger: logger}
}

// ServeHTTP handles an HTTP request
func (h Liveness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

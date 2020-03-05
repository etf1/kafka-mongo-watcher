package handler

import (
	"net/http"

	"github.com/gol4ng/logger"
)

type Liveness struct {
	logger logger.LoggerInterface
}

func NewLiveness(logger logger.LoggerInterface) http.Handler {
	return Liveness{logger: logger}
}

func (h Liveness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		respond(w, errorJSON("only GET requests are supported"), http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

func respond(w http.ResponseWriter, body []byte, code int) {
	host, _ := os.Hostname()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if len(host) > 0 {
		w.Header().Set("Hostname", host)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func errorJSON(msg string) []byte {
	buf := bytes.Buffer{}
	_, _ = fmt.Fprintf(&buf, `{"error": "%s"}`, msg)
	return buf.Bytes()
}

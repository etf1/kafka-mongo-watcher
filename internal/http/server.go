package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"text/template"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/debug"
	"github.com/etf1/kafka-mongo-watcher/internal/http/handler"
	"github.com/gol4ng/logger"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Debugger interface {
	Context() map[string]interface{}
	Events() chan *debug.Event
	Enabled() bool
}

type Server struct {
	httpServer *http.Server
	logger     logger.LoggerInterface
	debugger   Debugger
}

// NewServer returns a technical HTTP server that is used for liveness/readiness
// and serving Prometheus metrics for instance
func NewServer(
	logger logger.LoggerInterface,
	debugger Debugger,
	httpTechAddr string,
	readHeaderTimeout, writeTimeout, idleTimeout time.Duration,
	pprofEnabled bool,
) *Server {
	return &Server{
		logger:   logger,
		debugger: debugger,
		httpServer: &http.Server{
			Addr:              httpTechAddr,
			Handler:           getHttpHandler(pprofEnabled, logger, debugger),
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		},
	}
}

// Start starts serving HTTP requests
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("HTTP server started", logger.String("addr", s.httpServer.Addr))
	s.httpServer.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	return s.httpServer.ListenAndServe()
}

// Close shutdowns the HTTP server
func (s *Server) Close(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func getHttpHandler(
	pprofEnabled bool,
	logger logger.LoggerInterface,
	debugger Debugger,
) http.Handler {
	livenessHandler := handler.NewLiveness(logger)

	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/metrics").Handler(promhttp.Handler())
	router.Methods(http.MethodGet).Path("/liveness").Handler(livenessHandler)
	router.Methods(http.MethodGet).Path("/readiness").Handler(livenessHandler)

	if debugger.Enabled() {
		debugHandler := handler.NewDebug(logger, debugger)

		router.Methods(http.MethodGet).Path("/sse/event").Handler(debugHandler)
		router.Methods(http.MethodGet).Path("/ui/component/{file}").HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, fmt.Sprintf("./public/src/component/%s", mux.Vars(r)["file"]))
			},
		)
		router.Methods(http.MethodGet).Path("/").HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				template, err := template.ParseFiles("./public/index.html.tmpl")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				template.Execute(w, debugger.Context())
			},
		)
	}

	if pprofEnabled {
		router.HandleFunc("/debug/pprof/", pprof.Index)
		router.HandleFunc("/debug/pprof/allocs", pprof.Index)
		router.HandleFunc("/debug/pprof/block", pprof.Index)
		router.HandleFunc("/debug/pprof/heap", pprof.Index)
		router.HandleFunc("/debug/pprof/goroutine", pprof.Index)
		router.HandleFunc("/debug/pprof/mutex", pprof.Index)
		router.HandleFunc("/debug/pprof/threadcreate", pprof.Index)

		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return router
}

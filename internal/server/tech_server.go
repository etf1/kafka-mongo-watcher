package server

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/etf1/kafka-mongo-watcher/internal/server/http/handler"
	"github.com/gol4ng/logger"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type TechServer struct {
	httpServer *http.Server
	logger     logger.LoggerInterface
}

func NewTechServer(
	logger logger.LoggerInterface,
	httpTechAddr string,
	readHeaderTimeout, writeTimeout, idleTimeout time.Duration,
	pprofEnabled bool,
) *TechServer {
	return &TechServer{
		logger: logger,
		httpServer: &http.Server{
			Addr:              httpTechAddr,
			Handler:           getTechHttpHandler(pprofEnabled, logger),
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		},
	}
}

func (s *TechServer) Start(ctx context.Context) error {
	s.logger.Info("Tech HTTP server started", logger.String("addr", s.httpServer.Addr))
	s.httpServer.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	return s.httpServer.ListenAndServe()
}

func (s *TechServer) Close(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func getTechHttpHandler(pprofEnabled bool, logger logger.LoggerInterface) http.Handler {
	livenessHandler := handler.NewLiveness(logger)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/liveness", livenessHandler)
	router.Handle("/readiness", livenessHandler)

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

package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/http"
)

func (container *Container) GetHttpServer() *http.Server {
	if container.httpServer == nil {
		container.httpServer = http.NewServer(
			container.GetLogger(),
			container.GetDebugger(),
			container.Cfg.HttpServer.HTTPAddr,
			container.Cfg.HttpServer.ReadHeaderTimeout,
			container.Cfg.HttpServer.WriteTimeout,
			container.Cfg.HttpServer.IdleTimeout,
			container.Cfg.PprofEnabled,
		)
	}

	return container.httpServer
}

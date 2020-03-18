package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/server"
)

func (container *Container) GetTechServer() *server.TechServer {
	if container.techServer == nil {
		container.techServer = server.NewTechServer(
			container.GetLogger(),
			container.Cfg.TechServer.HTTPAddr,
			container.Cfg.TechServer.ReadHeaderTimeout,
			container.Cfg.TechServer.WriteTimeout,
			container.Cfg.TechServer.IdleTimeout,
			container.Cfg.TechServer.PprofEnabled,
		)
	}

	return container.techServer
}

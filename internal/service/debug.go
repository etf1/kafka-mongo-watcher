package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/debug"
)

func (container *Container) GetDebugger() *debug.Debugger {
	if container.Cfg.HttpServer.DebugEnabled && container.debugger == nil {
		container.debugger = debug.NewDebugger(container.Cfg)
	}

	return container.debugger
}

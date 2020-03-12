package service

import (
	"compress/flate"
	"log"
	"net"
	"os"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/gol4ng/logger"
	"github.com/gol4ng/logger/formatter"
	"github.com/gol4ng/logger/handler"
	"github.com/gol4ng/logger/middleware"
	"github.com/gol4ng/logger/writer"
)

func (container *Container) GetLogger() logger.LoggerInterface {
	if container.logger == nil {
		container.logger = logger.NewLogger(container.getLoggerHandler())
	}

	return container.logger
}

func (container *Container) getLoggerHandler() logger.HandlerInterface {
	h := handler.Stream(os.Stdout, formatter.NewDefaultFormatter(formatter.WithColor(true), formatter.WithContext(container.Cfg.LogCliVerbose)))

	if container.Cfg.GraylogEndpoint != "" {
		connection, err := container.getGelfUDPConnection()
		if err != nil {
			log.Printf("GELF logging disabled: %s", err)
		}
		if connection != nil {
			h = handler.Group(
				handler.Stream(
					writer.NewCompressWriter(writer.NewGelfChunkWriter(connection), writer.CompressionType(writer.CompressZlib), writer.CompressionLevel(flate.BestSpeed)),
					formatter.NewGelf(),
				),
				h,
			)
		}
	}

	return container.getLoggerHandlerMiddleware().Decorate(h)
}

func (container *Container) getLoggerHandlerMiddleware() logger.Middlewares {
	return logger.MiddlewareStack(
		middleware.Placeholder(),
		middleware.Context(logger.Ctx("facility", config.AppName).Add("version", config.AppVersion)),
		middleware.MinLevelFilter(logger.LevelString(container.Cfg.LogLevel).Level()),
		middleware.Caller(3),
	)
}

func (container *Container) getGelfUDPConnection() (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", container.Cfg.GraylogEndpoint)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, udpAddr)
}

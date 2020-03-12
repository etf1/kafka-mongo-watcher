package service

import (
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/worker"
)

func (container *Container) GetWorker(mongoClient mongo.Client) worker.Worker {
	if container.worker == nil {
		container.worker = worker.New(
			container.GetLogger(),
			mongoClient,
			container.GetKafkaClient(),
			container.Cfg.WorkerNumber,
		)
	}

	return container.worker
}

package service

import "github.com/etf1/kafka-mongo-watcher/internal/worker"

func (container *Container) GetWorker() worker.Worker {
	if container.worker == nil {
		container.worker = worker.New(
			container.GetLogger(),
			container.GetMongoClient(),
			container.GetKafkaClient(),
			container.Cfg.WorkerNumber,
			container.Cfg.WorkerTimeout,
		)
	}

	return container.worker
}

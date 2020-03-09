package service

import (
	"github.com/prometheus/client_golang/prometheus"
)

func (container *Container) GetMetricsRegistry() prometheus.Registerer {
	return prometheus.DefaultRegisterer
}

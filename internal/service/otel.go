package service

import (
	"time"

	"github.com/etf1/kafka-mongo-watcher/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

func (container *Container) GetTracerProvider() trace.TracerProvider {
	if container.tracerProvider == nil {
		logger := container.GetLogger()
		if container.Cfg.OtelCollectorEndpoint == "" {
			logger.Debug("OpenTelemetry is disabled")
			container.tracerProvider = trace.NewNoopTracerProvider()
			return container.tracerProvider
		}

		logger.Debug("Initializing OpenTelemetry")

		traceExporter, err := otlptracegrpc.New(
			container.baseContext,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(container.Cfg.OtelCollectorEndpoint),
			otlptracegrpc.WithDialOption(
				grpc.WithConnectParams(grpc.ConnectParams{
					Backoff: backoff.Config{
						BaseDelay:  1 * time.Second,
						Multiplier: 1.6,
						MaxDelay:   15 * time.Second,
					},
					MinConnectTimeout: 2 * time.Second,
				}),
			),
		)
		if err != nil {
			panic(err)
		}

		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(container.Cfg.AppName),
				semconv.ServiceVersionKey.String(config.AppVersion),
			)),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
		)

		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})

		container.tracerProvider = tracerProvider
	}

	return container.tracerProvider
}

package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	sampleRate         = 1.0
	maxExportBatchSize = 1000
	batchTimeout       = 100 * time.Millisecond
)

var once sync.Once

// Init initiate global provider for the project
func Init(ctx context.Context, serviceName, jaegerEndpoint string, options ...TraceProviderOption) (*sdktrace.TracerProvider, error) {
	conn, err := grpc.DialContext(ctx, jaegerEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("obesrvability.Init: grpc.DialContext: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("otlptracegrpc.New: %w", err)
	}

	opts := &traceProviderOptions{
		sampleRate:         sampleRate,
		maxExportBatchSize: maxExportBatchSize,
		batchTimeout:       batchTimeout,
	}

	for _, option := range options {
		option(opts)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter, sdktrace.WithMaxExportBatchSize(opts.maxExportBatchSize),
		sdktrace.WithBatchTimeout(opts.batchTimeout))

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
	}
	if opts.env != "" {
		attrs = append(attrs, attribute.String("env", opts.env))
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("resource.New: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(opts.sampleRate)),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	once.Do(func() {
		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)
		otel.SetTracerProvider(tp)
	})

	return tp, nil
}

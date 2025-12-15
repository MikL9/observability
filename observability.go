package observability

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/tracing"
	"github.com/MikL9/observability/utils"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Observability struct {
	prometheus    prometheus.Registerer
	traceProvider *sdktrace.TracerProvider
	serviceID     string
	logFormat     string
}

var instance *Observability

func Init(serviceID string, opts ...Option) error {
	instance = &Observability{
		serviceID:  serviceID,
		prometheus: prometheus.DefaultRegisterer,
	}

	for _, opt := range opts {
		err := opt(instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetRegisterer() prometheus.Registerer {
	return instance.prometheus
}

func Start(ctx *context.Context) *tracing.SpanWrapper {
	caller := utils.GetOriginalCallerFuncName(3)
	spanWrapper := tracing.New(ctx, utils.GetOpNameBySnakeCase(caller))
	spanWrapper.Start()
	*ctx = trace.ContextWithSpanContext(*ctx, spanWrapper.Span.SpanContext())
	spanWrapper.Span.SetAttributes(attribute.String("caller", caller))
	return spanWrapper
}

func StartWithName(ctx *context.Context, name string) *tracing.SpanWrapper {
	caller := utils.GetOriginalCallerFuncName(3)
	spanWrapper := tracing.New(ctx, name)
	spanWrapper.Start()
	*ctx = trace.ContextWithSpanContext(*ctx, spanWrapper.Span.SpanContext())
	spanWrapper.Span.SetAttributes(attribute.String("caller", caller))
	return spanWrapper
}

// WrapError оборачивает ошибку и добавляет префикс названия функции в observability Error
func WrapError(ctx context.Context, err *error) {
	if err == nil {
		return
	}
	caller := utils.GetOriginalCallerFuncName(3)
	opSeparateCase := strings.Join(strings.Split(utils.GetOpNameBySnakeCase(caller), "_"), " ")
	*err = errors.WrapPrefix(ctx, *err, opSeparateCase)
}

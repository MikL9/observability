package observability

import (
	"context"
	slogmulti "github.com/samber/slog-multi"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/MikL9/observability/hide"

	"github.com/MikL9/observability/logger"
	"github.com/MikL9/observability/tracing"
)

type Option func(*Observability) error

func WithJaegerOptions(endpoint string, options ...tracing.TraceProviderOption) Option {
	return func(o *Observability) error {
		var err error
		o.traceProvider, err = tracing.Init(context.Background(), o.serviceID, endpoint, options...)
		return err
	}
}

func WithLoggerOptions(handlers ...slog.Handler) Option {
	return func(o *Observability) error {
		fanout := slogmulti.Fanout(handlers...)

		handler := slog.Handler(fanout)
		if o.logFormat != "" {
			handler = handler.WithAttrs([]slog.Attr{
				slog.String("log_format", o.logFormat),
			})
		}

		return logger.SetupLogger(handler)
	}
}

func WithPrometheus(prom prometheus.Registerer) Option {
	return func(o *Observability) error {
		o.prometheus = prom
		return nil
	}
}

func WithHideConverter(convert *hide.Converter) Option {
	return func(observability *Observability) error {
		hide.SetDefaultConverter(convert)
		return nil
	}
}

func WithLogFormat(format string) Option {
	return func(o *Observability) error {
		o.logFormat = format
		return nil
	}
}

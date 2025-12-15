package client

import (
	"net/http"

	"github.com/MikL9/observability/http/client/logger"
	"github.com/MikL9/observability/http/client/metric"
	"github.com/MikL9/observability/http/client/retry"
	"github.com/MikL9/observability/http/client/tracing"
)

type Option func(rt http.RoundTripper, serviceID string) http.RoundTripper

func WithLog() Option {
	return func(rt http.RoundTripper, serviceID string) http.RoundTripper {
		return logger.NewTransport(rt, serviceID)
	}
}

func WithTracing() Option {
	return func(rt http.RoundTripper, serviceID string) http.RoundTripper {
		return tracing.NewTransport(rt, serviceID)
	}
}

func WithMetrics() Option {
	return func(rt http.RoundTripper, serviceID string) http.RoundTripper {
		return metric.NewTransport(rt, serviceID)
	}
}

func WithRetry(opts ...retry.Option) Option {
	return func(rt http.RoundTripper, serviceID string) http.RoundTripper {
		return retry.NewTransport(rt, opts...)
	}
}

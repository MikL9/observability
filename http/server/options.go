package server

import (
	"regexp"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/MikL9/observability"
	"github.com/MikL9/observability/utils"
)

type Option func(*Middleware)

var excludeLoggingEndpoints atomic.Pointer[regexp.Regexp]

// TODO: deprecated remove in next major release
func WithLogRequest() Option {
	return func(m *Middleware) {
	}
}

func WithLogResponse() Option {
	return func(m *Middleware) {
		m.needToLogResponse = true
	}
}

func WithLogPanic(panicMessage string) Option {
	return func(m *Middleware) {
		m.needToLogPanic = true
		httpRecoveryResponse = panicMessage
	}
}

func WithTracing() Option {
	return func(m *Middleware) {
		m.needToTracing = true
	}
}

func WithMetrics(serviceID string) Option {
	return func(m *Middleware) {
		serviceID = utils.ToSnakeCase(serviceID)
		m.needToMetrics = true
		totalRequests := promauto.With(observability.GetRegisterer()).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "http",
				Subsystem: serviceID,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"path", "app_type", "app_version", "status_code"},
		)
		requestDuration := promauto.With(observability.GetRegisterer()).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "http",
				Subsystem: serviceID,
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"path", "app_type", "app_version", "status_code"},
		)

		m.metrics = &serverMetrics{
			totalRequests:   totalRequests,
			requestDuration: requestDuration,
		}
	}
}

func WithExcludeLoggingEndpoints(endpoints *regexp.Regexp) Option {
	return func(m *Middleware) {
		excludeLoggingEndpoints.Store(endpoints)
	}
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/MikL9/observability"
	"github.com/MikL9/observability/logger"
	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/storage"
	"github.com/MikL9/observability/tracing"
	"github.com/MikL9/observability/utils"
)

type (
	responseWriter struct {
		http.ResponseWriter
		responseData *utils.ResponseData
	}

	serverMetrics struct {
		totalRequests   *prometheus.CounterVec
		requestDuration *prometheus.HistogramVec
	}

	Middleware struct {
		metrics           *serverMetrics
		needToLogResponse bool
		needToLogPanic    bool
		needToTracing     bool
		needToMetrics     bool
	}
)

func (rw *responseWriter) WriteHeader(code int) {
	rw.responseData.Status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseData.Body = b
	return size, err
}

var httpRecoveryResponse string

func NewHTTPMiddleware(opts ...Option) Middleware {
	m := Middleware{}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m *Middleware) HTTPMiddlewareWithParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			span               *tracing.SpanWrapper
			err                error
			excludePathLogging bool
			attrs              = []slog.Attr{utils.KeyRequest(r, false)}
		)
		ctx := r.Context()

		appVersion := r.Header.Get("wb-appversion")
		appType := r.Header.Get("wb-apptype")

		if re := excludeLoggingEndpoints.Load(); re != nil && re.MatchString(r.URL.Path) {
			excludePathLogging = true
		}

		sw := &responseWriter{ResponseWriter: w, responseData: &utils.ResponseData{}}

		if m.needToTracing {
			span = observability.StartWithName(&ctx, r.Method+" "+r.URL.Path)
			traceID := span.SpanContext().TraceID().String()
			defer span.StopWrap(&err)
			ctx = context.WithValue(ctx, "request", r)
			r = r.Clone(ctx)
			sw.Header()["trace_id"] = []string{traceID}
		}

		var timeStart = time.Now()

		if m.needToLogPanic {
			defer func() {
				if rr := recover(); rr != nil {
					sw.WriteHeader(http.StatusInternalServerError)
					sw.Write([]byte(httpRecoveryResponse))

					err = errors.New(ctx, "request completed with panic")
					logger.Error(ctx, err,
						utils.KeyRequest(r, false),
						utils.KeyResponse(sw.responseData, false),
						utils.KeyDuration(timeStart),
						utils.KeyPanic(fmt.Sprint(rr), 4))

					m.metrics.totalRequests.WithLabelValues(r.URL.Path, appType, appVersion, strconv.Itoa(http.StatusInternalServerError)).Add(1)
					m.metrics.requestDuration.WithLabelValues(r.URL.Path, appType, appVersion, strconv.Itoa(http.StatusInternalServerError)).
						Observe(time.Since(timeStart).Seconds())
				}
			}()
		}

		next.ServeHTTP(sw, r)

		attrs = append(attrs,
			utils.KeyResponse(sw.responseData, true),
			utils.KeyDuration(timeStart),
			utils.AppType(appType),
			utils.AppVersion(appVersion),
		)

		if m.needToLogResponse && !excludePathLogging {
			logger.Info(ctx, "request completed", attrs...)
		}

		if m.needToMetrics {
			m.metrics.totalRequests.WithLabelValues(r.URL.Path, appType, appVersion, strconv.Itoa(sw.responseData.Status)).Add(1)
			m.metrics.requestDuration.WithLabelValues(r.URL.Path, appType, appVersion, strconv.Itoa(sw.responseData.Status)).
				Observe(time.Since(timeStart).Seconds())
		}
		if m.needToTracing {
			ctx = storage.SetContextAttr(ctx, attrs...)
		}
	})
}

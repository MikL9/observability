package sentry

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Handler struct {
	level   slog.Leveler
	release string
	env     string
}

func NewHandler(level slog.Leveler, release, env string) *Handler {
	return &Handler{
		level:   level,
		release: release,
		env:     env,
	}
}

func (h *Handler) Enabled(_ context.Context, rec slog.Level) bool {
	return rec.Level() >= h.level.Level()
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	hub := sentry.CurrentHub()
	if hubFromContext := sentry.GetHubFromContext(ctx); hubFromContext != nil {
		hub = hubFromContext
	}

	event := makeEvent(ctx, &rec, h.env, h.release)
	hub.CaptureEvent(event)
	return nil
}

func (h *Handler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *Handler) WithGroup(_ string) slog.Handler {
	return h
}

// Transport для перехвата отправки эвентов в sentry и добавление метрик
type Transport struct {
	rt http.RoundTripper

	RequestsCounter         *prometheus.CounterVec
	RequestsErrorCounter    *prometheus.CounterVec
	RequestsDuration        *prometheus.HistogramVec
	ActiveConnectionCounter prometheus.Gauge
}

func NewTransport(rt http.RoundTripper) *Transport {
	serviceID := "observability_sentry_handler"
	return &Transport{
		rt: rt,
		RequestsCounter: promauto.With(prometheus.DefaultRegisterer).NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_total",
			Help:      "Общее количество запросов",
		}, []string{"status"}),
		RequestsErrorCounter: promauto.With(prometheus.DefaultRegisterer).NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_total_error",
			Help:      "Общее количество ошибочных запросов",
		}, []string{}),
		RequestsDuration: promauto.With(prometheus.DefaultRegisterer).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_duration",
			Help:      "Продолжительность запросов",
			Buckets:   prometheus.DefBuckets,
		}, []string{}),
		ActiveConnectionCounter: promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "active_connection_total",
			Help:      "Количество активных соединений",
		}),
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		timeStart  = time.Now()
		statusCode string
	)
	t.ActiveConnectionCounter.Inc()

	resp, err := t.rt.RoundTrip(req)
	if resp != nil {
		statusCode = strconv.Itoa(resp.StatusCode)
	}
	t.RequestsCounter.WithLabelValues(statusCode).Inc()
	t.ActiveConnectionCounter.Dec()
	t.RequestsDuration.WithLabelValues().Observe(time.Since(timeStart).Seconds())
	if err != nil {
		t.RequestsErrorCounter.WithLabelValues().Inc()
	}
	return resp, err
}

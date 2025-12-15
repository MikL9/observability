package metric

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/MikL9/observability"
)

type Transport struct {
	rt http.RoundTripper

	RequestsCounter         *prometheus.CounterVec
	RequestsErrorCounter    *prometheus.CounterVec
	RequestsDuration        *prometheus.HistogramVec
	ActiveConnectionCounter prometheus.Gauge
}

func NewTransport(rt http.RoundTripper, serviceID string) *Transport {
	return &Transport{
		rt: rt,
		RequestsCounter: promauto.With(observability.GetRegisterer()).NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_total",
			Help:      "Общее количество запросов",
		}, []string{"path", "status"}),
		RequestsErrorCounter: promauto.With(observability.GetRegisterer()).NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_total_error",
			Help:      "Общее количество ошибочных запросов",
		}, []string{"path"}),
		RequestsDuration: promauto.With(observability.GetRegisterer()).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "requests_duration",
			Help:      "Продолжительность запросов",
			Buckets:   prometheus.DefBuckets,
		}, []string{"path"}),
		ActiveConnectionCounter: promauto.With(observability.GetRegisterer()).NewGauge(prometheus.GaugeOpts{
			Namespace: "http",
			Subsystem: serviceID,
			Name:      "active_connection_total",
			Help:      "Количество активных соединений",
		}),
	}
}

func (t *Transport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	var (
		timeStart  = time.Now()
		statusCode string
	)
	t.ActiveConnectionCounter.Inc()

	resp, err = t.rt.RoundTrip(r)

	if resp != nil {
		statusCode = strconv.Itoa(resp.StatusCode)
	}
	t.RequestsCounter.WithLabelValues(r.RequestURI, statusCode).Inc()
	t.ActiveConnectionCounter.Dec()
	t.RequestsDuration.WithLabelValues(r.RequestURI).Observe(time.Since(timeStart).Seconds())
	if err != nil {
		t.RequestsErrorCounter.WithLabelValues(r.RequestURI).Inc()
	}
	return resp, err
}

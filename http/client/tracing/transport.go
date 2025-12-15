package tracing

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MikL9/observability"
	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/storage"
	"github.com/MikL9/observability/utils"
	"go.opentelemetry.io/otel/codes"
)

type Transport struct {
	rt        http.RoundTripper
	serviceID string
}

func NewTransport(rt http.RoundTripper, serviceID string) *Transport {
	return &Transport{rt: rt, serviceID: serviceID}
}

func (t *Transport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	ctx := r.Context()
	var (
		timeStart = time.Now()
		attrs     = make([]slog.Attr, 0, 3)
	)
	attrs = append(attrs, utils.KeyRequest(r, false))
	span := observability.StartWithName(&ctx, "HTTP "+r.Method+" "+t.serviceID)
	defer span.StopWrap(&err)

	resp, err = t.rt.RoundTrip(r)

	attrs = append(attrs, utils.KeyDuration(timeStart))

	if err == nil && resp == nil {
		if err == nil {
			err := errors.New(ctx, "*http.Response and err got nil")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if resp != nil {
		if resp.StatusCode >= 500 {
			err := errors.New(ctx, "Status code is 50x")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		attrs = append(attrs,
			utils.KeyResponse(&utils.ResponseData{Status: resp.StatusCode, Body: utils.GetResponseBodyCopy(resp)}, false),
		)
	}
	ctx = storage.SetContextAttr(ctx, attrs...)
	return resp, err
}

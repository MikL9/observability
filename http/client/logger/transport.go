package logger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MikL9/observability/logger"
	"github.com/MikL9/observability/logger/errors"
	"github.com/MikL9/observability/utils"
)

type Transport struct {
	rt        http.RoundTripper
	serviceID string
}

func NewTransport(rt http.RoundTripper, serviceID string) *Transport {
	return &Transport{rt: rt, serviceID: serviceID}
}

func (t *Transport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	var (
		timeStart = time.Now()
		ctx       = r.Context()
		attrs     = []slog.Attr{utils.KeyRequest(r, true)}
	)

	resp, err = t.rt.RoundTrip(r)

	if err != nil {
		attrs = append(attrs, utils.KeyError(err), utils.KeyDuration(timeStart))
		logger.Error(ctx, errors.New(ctx, t.serviceID+" query"), attrs...)
	} else {
		attrs = append(attrs,
			utils.KeyDuration(timeStart),
			utils.KeyResponse(&utils.ResponseData{Status: resp.StatusCode, Body: utils.GetResponseBodyCopy(resp)}, true))
		logger.Info(ctx, t.serviceID+" query", attrs...)
	}
	return resp, err
}

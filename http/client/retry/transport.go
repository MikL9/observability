package retry

import (
	"context"
	"net/http"
	"time"

	"github.com/MikL9/observability/storage"
	"github.com/MikL9/observability/utils"
)

type (
	CheckRetry func(ctx context.Context, resp *http.Response, err error) (bool, error)

	// Backoff specifies a policy for how long to wait between retries.
	// It is called after a failing request to determine the amount of time
	// that should pass before trying again.
	Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration

	// PrepareRequest is called before RoundTripper operation. It can be used for example to re-sign the request
	PrepareRequest func(ctx context.Context, req *http.Request, attempt bool) error

	// ErrorHandler is called if retries are expired, containing the last status
	// from the http library. If not specified, default behavior for the library is
	// to close the body and return an error indicating how many tries were
	// attempted. If overriding this, be sure to close the body if needed.
	ErrorHandler func(resp *http.Response, err error, numTries int) (*http.Response, error)

	ResponseHandler func(*http.Response) error
)

type Transport struct {
	base http.RoundTripper

	CheckRetry      CheckRetry
	Backoff         Backoff
	PrepareRequest  PrepareRequest
	ErrorHandler    ErrorHandler
	ResponseHandler ResponseHandler

	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
}

func NewTransport(base http.RoundTripper, opts ...Option) *Transport {
	tr := &Transport{
		base:         base,
		CheckRetry:   DefaultRetryPolicy,
		Backoff:      DefaultBackoff,
		RetryMax:     5,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(tr)
	}
	return tr
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var attempt int
	var shouldRetry bool
	var doErr, respErr, checkErr, prepareErr error

	ctx := req.Context()
	body, _, err := getBodyReaderAndContentLength(req.Body)
	if err != nil {
		return nil, err
	}
	for i := 0; ; i++ {
		ctx = storage.SetContextAttr(ctx,
			utils.KeyAttempt(i),
			utils.KeyMaxAttempt(t.RetryMax),
		)
		req = req.Clone(ctx)
		doErr, respErr, prepareErr = nil, nil, nil
		attempt++

		// Always rewind the request body when non-nil.
		if err := rewindBody(req, body); err != nil {
			return nil, err
		}

		if t.PrepareRequest != nil {
			if prepareErr = t.PrepareRequest(req.Context(), req, i > 0); prepareErr != nil {
				break
			}
		}

		// Attempt the request
		resp, doErr = t.base.RoundTrip(req)

		// Check if we should continue with retries.
		shouldRetry, checkErr = t.CheckRetry(req.Context(), resp, doErr)
		if !shouldRetry && doErr == nil && t.ResponseHandler != nil {
			respErr = t.ResponseHandler(resp)
			shouldRetry, checkErr = t.CheckRetry(req.Context(), resp, respErr)
		}

		err = doErr
		if respErr != nil {
			err = respErr
		}

		if !shouldRetry {
			break
		}

		// We do this before drainBody because there's no need for the I/O if
		// we're breaking out
		remain := t.RetryMax - i
		if remain <= 0 {
			break
		}

		// We're going to retry, consume any response to reuse the connection.
		if doErr == nil {
			drainBody(req.Context(), resp.Body)
		}

		wait := t.Backoff(t.RetryWaitMin, t.RetryWaitMax, i, resp)
		timer := time.NewTimer(wait)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}

	// this is the closest we have to success criteria
	if doErr == nil && respErr == nil && checkErr == nil && prepareErr == nil && !shouldRetry {
		return resp, nil
	}

	if prepareErr != nil {
		err = prepareErr
	} else if checkErr != nil {
		err = checkErr
	} else if respErr != nil {
		err = respErr
	} else {
		err = doErr
	}

	if t.ErrorHandler != nil {
		return t.ErrorHandler(resp, err, attempt)
	}

	return resp, err
}

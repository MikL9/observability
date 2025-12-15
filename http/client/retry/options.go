package retry

import "time"

type Option func(*Transport)

func WithCheckRetry(f CheckRetry) Option {
	return func(t *Transport) {
		t.CheckRetry = f
	}
}

func WithBackoff(f Backoff) Option {
	return func(t *Transport) {
		t.Backoff = f
	}
}

func WithPrepareRequest(f PrepareRequest) Option {
	return func(t *Transport) {
		t.PrepareRequest = f
	}
}

func WithErrorHandler(f ErrorHandler) Option {
	return func(t *Transport) {
		t.ErrorHandler = f
	}
}

func WithResponseHandler(f ResponseHandler) Option {
	return func(t *Transport) {
		t.ResponseHandler = f
	}
}

func WithRetryMax(retryMax int) Option {
	return func(t *Transport) {
		t.RetryMax = retryMax
	}
}

func WithRetryWaitMin(waitMin time.Duration) Option {
	return func(t *Transport) {
		t.RetryWaitMin = waitMin
	}
}

func WithRetryWaitMax(waitMax time.Duration) Option {
	return func(t *Transport) {
		t.RetryWaitMax = waitMax
	}
}

package retry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStaticTime(t *testing.T) {
	timeNow = func() time.Time {
		now, err := time.Parse(time.RFC1123, "Fri, 31 Dec 1999 23:59:57 GMT")
		if err != nil {
			panic(err)
		}
		return now
	}
	t.Cleanup(func() {
		timeNow = time.Now
	})
}

func TestParseRetryAfterHeader(t *testing.T) {
	testStaticTime(t)
	tests := []struct {
		name    string
		headers []string
		sleep   time.Duration
		ok      bool
	}{
		{"seconds", []string{"2"}, time.Second * 2, true},
		{"date", []string{"Fri, 31 Dec 1999 23:59:59 GMT"}, time.Second * 2, true},
		{"past-date", []string{"Fri, 31 Dec 1999 23:59:00 GMT"}, 0, true},
		{"nil", nil, 0, false},
		{"two-headers", []string{"2", "3"}, time.Second * 2, true},
		{"empty", []string{""}, 0, false},
		{"negative", []string{"-2"}, 0, false},
		{"bad-date", []string{"Fri, 32 Dec 1999 23:59:59 GMT"}, 0, false},
		{"bad-date-format", []string{"badbadbad"}, 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sleep, ok := parseRetryAfterHeader(test.headers)
			if ok != test.ok {
				t.Fatalf("expected ok=%t, got ok=%t", test.ok, ok)
			}
			if sleep != test.sleep {
				t.Fatalf("expected sleep=%v, got sleep=%v", test.sleep, sleep)
			}
		})
	}
}

func TestClient_DefaultBackoff(t *testing.T) {
	testStaticTime(t)
	tests := []struct {
		name        string
		code        int
		retryHeader string
	}{
		{"http_429_seconds", http.StatusTooManyRequests, "2"},
		{"http_429_date", http.StatusTooManyRequests, "Fri, 31 Dec 1999 23:59:59 GMT"},
		{"http_503_seconds", http.StatusServiceUnavailable, "2"},
		{"http_503_date", http.StatusServiceUnavailable, "Fri, 31 Dec 1999 23:59:59 GMT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", test.retryHeader)
				http.Error(w, fmt.Sprintf("test_%d_body", test.code), test.code)
			}))
			defer ts.Close()

			client := NewTransport(http.DefaultTransport)

			var retryAfter time.Duration
			retryable := false

			client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
				retryable, _ = DefaultRetryPolicy(context.Background(), resp, err)
				retryAfter = DefaultBackoff(client.RetryWaitMin, client.RetryWaitMax, 1, resp)
				return false, nil
			}

			req, err := http.NewRequest("GET", ts.URL, nil)
			require.NoError(t, err)
			_, err = client.RoundTrip(req)
			require.NoError(t, err)
			if !retryable {
				t.Fatal("Since the error is recoverable, the default policy shall return true")
			}
			if retryAfter != 2*time.Second {
				t.Fatalf("The header Retry-After specified 2 seconds, and shall not be %d seconds", retryAfter/time.Second)
			}
		})
	}
}

func TestClient_DefaultRetryPolicy_TLS(t *testing.T) {
	errTxt := "failed to verify certificate: x509: certificate signed by unknown authority"
	retry, err := DefaultRetryPolicy(
		context.TODO(),
		&http.Response{},
		&url.Error{
			Err: &tls.CertificateVerificationError{Err: errors.New(errTxt)},
		},
	)
	require.NoError(t, err)
	assert.False(t, retry, "should no retry")
}

func TestDefaultRetryPolicy(t *testing.T) {
	testCases := []struct {
		name  string
		resp  *http.Response
		retry bool
	}{
		{"0", &http.Response{}, true},
		{"200", &http.Response{StatusCode: 200}, false},
		{"301", &http.Response{StatusCode: 301}, false},
		{"401", &http.Response{StatusCode: 401}, false},
		{"429", &http.Response{StatusCode: 429}, true},
		{"500", &http.Response{StatusCode: 500}, true},
		{"501", &http.Response{StatusCode: 501}, false},
		{"502", &http.Response{StatusCode: 502}, true},
	}
	for _, tt := range testCases {
		retry, err := DefaultRetryPolicy(context.TODO(), tt.resp, nil)
		require.NoError(t, err)
		assert.Equal(t, tt.retry, retry)
	}
}

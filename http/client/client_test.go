package client

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/MikL9/observability/http/client/retry"
)

func TestNewClientWithRetry200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()
	cl := NewClient(
		http.DefaultTransport,
		"test",
		WithRetry(
			retry.WithRetryWaitMin(10*time.Microsecond),
			retry.WithRetryWaitMax(10*time.Microsecond),
			retry.WithRetryMax(3),
		),
	)
	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)
	resp, err := cl.Do(req)
	str := strings.Builder{}
	_, err = io.Copy(&str, resp.Body)
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Equal(t, str.String(), "ok")
}

func TestNewClientWithRetry500(t *testing.T) {
	var attempts int

	body := []byte("test")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(500)
		w.Write([]byte("internal server error"))
		buf, err := io.ReadAll(r.Body)

		require.NoError(t, err)
		assert.Equal(t, body, buf)
	}))
	defer ts.Close()
	cl := NewClient(
		http.DefaultTransport,
		"test",
		WithRetry(
			retry.WithRetryWaitMin(10*time.Microsecond),
			retry.WithRetryWaitMax(10*time.Microsecond),
			retry.WithRetryMax(3),
		),
	)
	req, err := http.NewRequest("GET", ts.URL, bytes.NewBuffer(body))
	require.NoError(t, err)
	resp, err := cl.Do(req)
	str := strings.Builder{}
	_, err = io.Copy(&str, resp.Body)
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 500)
	assert.Equal(t, str.String(), "internal server error")
	assert.Equal(t, 4, attempts)
}

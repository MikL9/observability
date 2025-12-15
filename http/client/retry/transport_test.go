package retry

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emptyReadClose = io.NopCloser(strings.NewReader(""))

func bodyFromString(str string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(str))
}

type Result struct {
	resp *http.Response
	err  error
}

type MockTransport struct {
	results []Result
}

func (t *MockTransport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	result := t.results[0]
	t.results = t.results[1:len(t.results)]
	return result.resp, result.err
}

func TestSuccessRequest(t *testing.T) {
	m := MockTransport{
		results: []Result{
			{&http.Response{StatusCode: 200, Body: emptyReadClose}, nil},
		},
	}
	tr := NewTransport(&m)

	req, _ := http.NewRequest("POST", "/path/to/", bodyFromString("some content"))
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t,
		&http.Response{StatusCode: 200, Body: emptyReadClose},
		resp)
}

func TestSuccessAfterRetries(t *testing.T) {
	m := MockTransport{
		results: []Result{
			{&http.Response{StatusCode: 500, Body: emptyReadClose}, nil},
			{&http.Response{StatusCode: 500, Body: emptyReadClose}, nil},
			{&http.Response{StatusCode: 200, Body: emptyReadClose}, nil},
		},
	}
	tr := NewTransport(&m)

	req, _ := http.NewRequest("GET", "", nil)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t,
		&http.Response{StatusCode: 200, Body: emptyReadClose},
		resp)
}

func TestFailedAfterRetries(t *testing.T) {
	m := MockTransport{
		results: []Result{
			{&http.Response{StatusCode: 500, Body: emptyReadClose}, nil},
			{&http.Response{StatusCode: 500, Body: emptyReadClose}, nil},
			{&http.Response{StatusCode: 500, Body: emptyReadClose}, nil},
			{&http.Response{StatusCode: 500, Body: bodyFromString("test")}, nil},
		},
	}
	tr := NewTransport(&m)
	tr.RetryMax = 3
	tr.RetryWaitMin = 10 * time.Microsecond
	tr.RetryWaitMax = 10 * time.Microsecond

	req, _ := http.NewRequest("GET", "", nil)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t,
		&http.Response{StatusCode: 500, Body: bodyFromString("test")},
		resp)
}

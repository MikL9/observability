package client

import "net/http"

func NewClient(rt http.RoundTripper, serviceID string, opts ...Option) *http.Client {
	for _, opt := range opts {
		rt = opt(rt, serviceID)
	}
	return &http.Client{Transport: rt}
}

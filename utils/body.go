package utils

import (
	"bytes"
	"io"
	"net/http"
)

type ResponseData struct {
	Body   []byte
	Status int
}

func GetRequestBodyCopy(r *http.Request) []byte {
	if r.Body == nil {
		return []byte{}
	}

	var reqBody []byte

	reqBody, _ = io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	return reqBody
}

func GetResponseBodyCopy(r *http.Response) []byte {
	if r.Body == nil {
		return []byte{}
	}

	if r.Header.Get("Content-Type") == "application/pdf" {
		return []byte("pdf file")
	}

	var reqBody []byte

	reqBody, _ = io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	return reqBody
}

package cache

import (
	"io"
	"net/http"
	"sync"
)

func newResponse(headers http.Header, body io.ReadCloser, responseCode int) (*response, error) {
	// TODO: Handle read closer
	return &response{
		headers:      headers,
		responseCode: responseCode,
	}, nil
}

type response struct {
	headers      http.Header
	body         *responseBody
	readLock     *sync.Mutex
	responseCode int
}

type responseBody struct {
	body []byte
}

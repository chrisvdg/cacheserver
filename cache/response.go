package cache

import (
	"io"
	"net/http"
	"sync"
)

func newResponse(headers http.Header, body io.ReadCloser, responseCode int) (*response, error) {

	// TODO: Mark in progress, read body, write into response body buffer, flag when done.
	// Check if body larger than min cache, if not mark no cache, clean response buffer
	// Copy to file, when done mark cached, clean response buffer

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

func (r *response) getReader() io.ReadCloser {

	return nil
}

type responseBody struct {
	body []byte
}

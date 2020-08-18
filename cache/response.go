package cache

import (
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrReadFailed represents an error where reading from proxy failed
	ErrReadFailed = errors.New("Failed to read from proxy target")
)

func newResponse(headers http.Header, body io.ReadCloser, responseCode int) (*response, error) {
	rBody := &responseBody{
		body:           []byte{},
		writeLock:      &sync.Mutex{},
		writeCompleted: false,
	}
	return &response{
		headers:      headers,
		responseCode: responseCode,
		body:         rBody,
		readLock:     &sync.Mutex{},
	}, nil
}

type response struct {
	headers      http.Header
	body         *responseBody
	readLock     *sync.Mutex
	responseCode int
}

func (r *response) cacheBody(body io.ReadCloser, cacheFile string, readWg *sync.WaitGroup) error {
	written, err := io.Copy(r.body, body)
	body.Close()
	r.body.MarkWriteCompleted(written, err)
	if err != nil {
		return errors.Wrap(err, "failed to copy proxy body to cache")
	}

	err = ioutil.WriteFile(cacheFile, r.body.body, filePerm)
	if err != nil {
		return errors.Wrap(err, "failed to write cache to file")
	}

	readWg.Wait()
	r.body = nil

	return nil
}

func (r *response) getReader() io.Reader {
	return r.body.GetReader()
}

type responseBody struct {
	body           []byte
	writeLock      *sync.Mutex
	writeCompleted bool
	readErr        error
	bodySize       int64
}

func (rb *responseBody) Write(p []byte) (int, error) {
	if rb.writeCompleted {
		return 0, errors.New("cache response body has already been written to")
	}
	rb.writeLock.Lock()
	defer rb.writeLock.Unlock()
	rb.body = append(rb.body, p...)
	return len(p), nil
}

// MarkWriteCompleted mark that the full body has been copied
func (rb *responseBody) MarkWriteCompleted(written int64, err error) {
	rb.writeLock.Lock()
	rb.writeCompleted = true
	rb.bodySize = written
	rb.readErr = err
	rb.writeLock.Unlock()
}

func (rb *responseBody) GetReader() *responseBodyReader {
	return &responseBodyReader{
		rb: rb,
		i:  0,
	}
}

type responseBodyReader struct {
	rb *responseBody
	i  int64 // reading index
}

// Read implements io.Read
// When nothing to read it will wait for 100 milliseconds
func (r *responseBodyReader) Read(b []byte) (int, error) {
	if r.rb.readErr != nil {
		err := errors.Wrap(ErrReadFailed, r.rb.readErr.Error())
		return 0, err
	}

	if r.i >= int64(len(r.rb.body)) {
		if r.rb.writeCompleted {
			return 0, io.EOF
		}
		time.Sleep(100 * time.Millisecond)
		return 0, nil
	}
	n := copy(b, r.rb.body[r.i:])
	r.i += int64(n)

	return n, nil
}

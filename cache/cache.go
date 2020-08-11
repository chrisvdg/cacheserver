package cache

import (
	"net/http"

	"github.com/pkg/errors"
)

var (
	// ErrNoCache represents a request that is not cached
	ErrNoCache = errors.New("Entry is not cached")
)

// New returns a new Cache instance
// backendFilePath represents the file on the filesystem where the metadata is stored
// minSize represents the minimum size of the body to be cached (0 caches everything)
func New(backendFilePath string, minSize int) (*Cache, error) {
	b, err := newBackend(backendFilePath)
	if err != nil {
		return nil, err
	}

	if minSize < 0 {
		minSize = 0
	}

	return &Cache{
		b:       b,
		minSize: minSize,
	}, nil
}

// Cache represents a cache instance
type Cache struct {
	b       *backend
	minSize int
}

// CopyFromCache returns reader where the cached (or proxied) body is written to
func (c *Cache) CopyFromCache(res http.ResponseWriter, req *http.Request) error {
	e, err := c.b.FindEntry(req)
	if err != nil && err != ErrEntryNotFound {
		return errors.Wrap(err, "failed to search entry")
	}
	if err == ErrEntryNotFound {
		return c.newEntry(res, req)
	}

	switch e.Status {
	case StateInit:
		return c.initEntry(res, req)
	case StateInProgress:
		return c.progressEntry(res, req)
	case StateCached:
		return c.cashedEntry(res, req)
	case StateNoCache:
		return ErrNoCache
	}

	return nil
}

func (c *Cache) newEntry(res http.ResponseWriter, req *http.Request) error {

	return nil
}

func (c *Cache) initEntry(res http.ResponseWriter, req *http.Request) error {

	return nil
}

func (c *Cache) progressEntry(res http.ResponseWriter, req *http.Request) error {

	return nil
}

func (c *Cache) cashedEntry(res http.ResponseWriter, req *http.Request) error {

	return nil
}

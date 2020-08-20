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
func New(backendFilePath, cacheDir, proxyURL string, minSize int) (*Cache, error) {
	b, err := newBackend(backendFilePath, cacheDir, proxyURL)
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
	e, err := c.b.findEntryByRequest(req)
	if err != nil && err != ErrEntryNotFound {
		return errors.Wrap(err, "failed to search entry")
	}
	if err == ErrEntryNotFound {
		e, err = c.b.addEntry(req.URL.Path, req.URL.Query())
		if err != nil {
			return err
		}
	}

	return c.b.proxy(e, res, req)
}

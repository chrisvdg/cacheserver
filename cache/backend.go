package cache

import (
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const filePerm os.FileMode = 0666

var (
	// ErrEntryNotFound represents an error where a cache entry was not found
	ErrEntryNotFound = errors.New("cache entry not found")
)

func newBackend(filePath string) (*backend, error) {
	if filePath == "" {
		return nil, errors.New("backend file path is empty")
	}
	return &backend{
		filePath: filePath,
		data:     make(map[string]Entry, 0),
	}, nil
}

type backend struct {
	filePath string
	data     map[string]Entry
	m        sync.Mutex
}

func (b *backend) FindEntry(req *http.Request) (*Entry, error) {
	for _, e := range b.data {
		if e.Path == req.URL.Path {
			if reflect.DeepEqual(e.Params, req.URL.Query()) {
				return &e, nil
			}
		}
	}

	return nil, ErrEntryNotFound
}

func (b *backend) AddEntry(path string, params url.Values) (string, Entry, error) {
	b.m.Lock()
	defer b.m.Unlock()

	e := Entry{
		Path:    path,
		Params:  params,
		Created: JSONTime(time.Now()),
		Status:  StateInit,
	}
	id := b.getID()
	b.data[id] = e
	err := b.save()

	return id, e, err
}

// Generates a unique cache entry ID
// Make sure to execute this when backend is locked
func (b *backend) getID() string {
	for {
		id := generateID(25)
		for i := range b.data {
			if i == id {
				continue
			}
		}
		return id
	}
}

// Entry represents a backend entry
type Entry struct {
	ID string `json:"id"`
	// Path represents the request path of the cached entry
	Path string `json:"path"`
	// Params represents the URL request params of the cached entry
	Params url.Values `json:"params"`
	// Created is the timestamp for when the entry was created
	Created JSONTime `json:"created"`
	// Status  represents the entry status
	Status State `json:"Status"`
}

// State represents the cache state of an entry
type State string

const (
	// StateInit represents an initialized cache entry state
	StateInit State = "init"
	// StateInProgress represents a cache entry that is being cached
	StateInProgress State = "in progress"
	// StateCached represents a cache entry that has been cached
	StateCached State = "cached"
	// StateNoCache represents a cache entry that does not need to be cached
	StateNoCache State = "no cache"
)

// JSONTime is a time.Time wrapper that JSON (un)marshals into a unix timestamp
type JSONTime time.Time

// MarshalJSON is used to convert the timestamp to JSON
func (t JSONTime) MarshalJSON() ([]byte, error) {
	unix := time.Time(t).Unix()
	// Negative time stamps make no sense for our use cases
	if unix < 0 {
		unix = 0
	}
	return []byte(strconv.FormatInt(unix, 10)), nil
}

// UnmarshalJSON is used to convert the timestamp from JSON
func (t *JSONTime) UnmarshalJSON(s []byte) (err error) {
	r := string(s)
	q, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	*(*time.Time)(t) = time.Unix(q, 0)
	return nil
}

// Unix returns the unix time stamp of the underlaying time object
func (t JSONTime) Unix() int64 {
	return time.Time(t).Unix()
}

// Time returns the JSON time as a time.Time instance
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}

// String returns time as a formatted string
func (t JSONTime) String() string {
	return t.Time().String()
}

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
		data:     make(map[string]*Entry, 0),
	}, nil
}

type backend struct {
	filePath string
	data     map[string]*Entry
	m        *sync.Mutex
}

func (b *backend) findEntry(req *http.Request) (string, error) {
	for entryID, e := range b.data {
		if e.Path == req.URL.Path {
			if reflect.DeepEqual(e.Params, req.URL.Query()) {
				return entryID, nil
			}
		}
	}

	return "", ErrEntryNotFound
}

func (b *backend) addEntry(path string, params url.Values) (string, error) {
	b.m.Lock()
	defer b.m.Unlock()

	e := Entry{
		Path:    path,
		Params:  params,
		Created: JSONTime(time.Now()),
		Status:  StateInit,
	}
	id := b.generateID()
	b.data[id] = &e
	err := b.save()

	return id, err
}

// generateID generates a unique cache entry ID
// Make sure to execute this when backend is locked
func (b *backend) generateID() string {
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

func (b *backend) setEntryState(id string, state State) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()
	e.Status = state
	return nil
}

func (b *backend) getEntryState(id string) (State, error) {
	e, ok := b.data[id]
	if !ok {
		return "", ErrEntryNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()
	return e.Status, nil
}

func (b *backend) proxy(id string, res http.ResponseWriter) error {
	state, err := b.getEntryState(id)
	if err != nil {
		return err
	}

	switch state {
	case StateInit:
	case StateInProgress:
	case StateCached:
	case StateNoCache:
		return ErrNoCache
	default:
		return errors.Errorf("State %s not supported", state)
	}

	return nil
}

func (b *backend) entryInit(id, res http.ResponseWriter) error {

	return nil
}

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

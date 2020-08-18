package cache

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrEntryNotFound represents an error where a cache entry was not found
	ErrEntryNotFound = errors.New("cache entry not found")
)

func newBackend(filePath, cacheDir, proxyURL string) (*backend, error) {
	if filePath == "" {
		return nil, errors.New("backend file path is empty")
	}
	if cacheDir == "" {
		return nil, errors.New("cache dir not provided")
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.Mkdir(cacheDir, dirPerm)
	}

	return &backend{
		targetBaseURL: proxyURL,
		filePath:      filePath,
		cacheDir:      cacheDir,
		data:          make(map[string]*Entry, 0),
		http:          &http.Client{},
		m:             &sync.Mutex{},
	}, nil
}

type backend struct {
	targetBaseURL string
	filePath      string
	cacheDir      string
	data          map[string]*Entry
	m             *sync.Mutex
	http          *http.Client
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

	e := &Entry{
		Path:    path,
		Params:  params,
		Created: JSONTime(time.Now()),
		Status:  StateInit,
		m:       &sync.Mutex{},
		readWg:  &sync.WaitGroup{},
	}
	id := b.generateID()
	b.data[id] = e

	return id, b.save()
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
	b.m.Lock()
	defer b.m.Unlock()

	err := b.save()
	if err != nil {
		return errors.Wrap(err, "failed to save cache entry state")
	}

	return nil
}

func (b *backend) setEntryCacheFile(id, file string) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}
	e.m.Lock()
	e.CachedFile = file
	e.m.Unlock()
	err := b.save()
	if err != nil {
		return errors.Wrap(err, "failed to save setting cache file name")
	}

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

func (b *backend) proxy(id string, res http.ResponseWriter, req *http.Request) error {
	state, err := b.getEntryState(id)
	if err != nil {
		return err
	}

	switch state {
	case StateInit:
		log.Debugf("Init entry %s", id)
		return b.entryInit(id, res, req)
	case StateInProgress:
		log.Debugf("In progress entry %s", id)
		return b.entryInProgress(id, res)
	case StateCached:
		log.Debugf("Cached entry %s", id)
		return b.entryCached(id, res)
	case StateNoCache:
		log.Debugf("No cache entry %s", id)
		return ErrNoCache
	default:
		return errors.Errorf("State %s not supported", state)
	}
}

func (b *backend) entryInit(id string, res http.ResponseWriter, req *http.Request) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()

	if e.Status == StateInProgress {
		log.Debugf("entry %s seem to already be in progress", id)
		return b.entryInProgress(id, res)
	} else if e.Status != StateInit {
		return errors.Errorf("Entry in unexpected state: %s, expected init", e.Status)
	}

	tURL, err := b.getProxyURL(e.Path)
	if err != nil {
		return errors.Wrap(err, "failed to get target proxy URL")
	}
	targetReq, err := http.NewRequest("GET", tURL, req.Body)
	targetReq.URL.RawQuery = req.URL.RawQuery
	for name, values := range req.Header {
		for _, v := range values {
			targetReq.Header.Add(name, v)
		}
	}
	targetResp, err := b.http.Do(targetReq)
	if err != nil {
		return errors.Wrap(err, "target request failed")
	}
	log.Debugf("%v", targetResp.Header)
	e.resp, err = newResponse(targetResp.Header, targetResp.Body, targetResp.StatusCode)
	if err != nil {
		return errors.Wrap(err, "failed to create cached response")
	}
	cacheFile := b.generateCacheFileName(id)
	err = b.setEntryCacheFile(id, cacheFile)
	if err != nil {
		return nil
	}
	e.Status = StateInProgress
	go b.startCaching(id, e, targetResp.Body)

	return b.entryInProgress(id, res)
}

func (b *backend) generateCacheFileName(id string) string {
	filename := fmt.Sprintf("%s_%s.blob", id, generateID(10))
	return path.Join(b.cacheDir, filename)
}

func (b *backend) startCaching(entryID string, e *Entry, body io.ReadCloser) {
	err := e.resp.cacheBody(body, e.CachedFile, e.readWg)
	log.Debugf("%s downloaded", entryID)
	if err != nil {
		log.Errorf(err.Error())
		b.setEntryState(entryID, StateInit)
		return
	}

	b.setEntryState(entryID, StateCached)

	log.Debug(e.Status)
}

func (b *backend) entryInProgress(id string, res http.ResponseWriter) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}

	for name, values := range e.resp.headers {
		for _, v := range values {
			res.Header().Add(name, v)
		}
	}
	res.WriteHeader(e.resp.responseCode)

	e.readWg.Add(1)
	_, err := io.Copy(res, e.resp.getReader())
	if err != nil {
		return errors.Wrap(err, "failed to get reader from buffer")
	}
	e.readWg.Done()

	return nil
}

// entryCached writes the contents of the cached file to the response writer
func (b *backend) entryCached(id string, res http.ResponseWriter) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}
	cacheFile, err := os.Open(e.CachedFile)
	if err != nil {
		return errors.Wrap(err, "failed to open cached file")
	}
	defer cacheFile.Close()
	_, err = io.Copy(res, cacheFile)
	if err != nil {
		return errors.Wrap(err, "failed to read from cache file")
	}

	return nil
}

func (b *backend) getProxyURL(reqPath string) (string, error) {
	u, err := url.Parse(b.targetBaseURL)
	if err != nil {
		return "", errors.Wrap(err, "Failed to fetch path from request")
	}
	u.Path = path.Join(u.Path, reqPath)

	return u.String(), nil
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

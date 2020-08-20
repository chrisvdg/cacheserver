package cache

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
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
		return nil, errors.New("backend file path is not provided")
	}
	if cacheDir == "" {
		return nil, errors.New("cache dir not provided")
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		log.Debugf("Creating cache dir %s", cacheDir)
		os.Mkdir(cacheDir, dirPerm)
	} else {
		log.Debugf("Deleting contents of cache dir %s", cacheDir)
		removeDirContent(cacheDir)
	}

	b := &backend{
		targetBaseURL:   proxyURL,
		filePath:        filePath,
		cacheDir:        cacheDir,
		data:            make(map[string]*Entry, 0),
		http:            &http.Client{},
		m:               &sync.Mutex{},
		cleanupInterval: 5 * time.Minute,
		cacheExpiration: 10 * time.Minute,
	}

	// start cleanup go routine
	go b.cleanup(nil)

	return b, nil
}

type backend struct {
	targetBaseURL   string
	filePath        string
	cacheDir        string
	data            map[string]*Entry
	m               *sync.Mutex
	http            *http.Client
	cleanupInterval time.Duration
	cacheExpiration time.Duration
}

func (b *backend) findEntryByRequest(req *http.Request) (string, error) {
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
	return b.setEntryStateNoEntryLock(e, state)
}

// setEntryStateNoEntryLock sets the state on provided entry without Entry lock.
// Make sure the entry is locked before calling this function and released after.
func (b *backend) setEntryStateNoEntryLock(e *Entry, state State) error {
	e.Status = state
	b.m.Lock()
	defer b.m.Unlock()
	err := b.save()
	if err != nil {
		return errors.Wrap(err, "failed to save cache entry state")
	}

	return nil
}

func (b *backend) setEntryCacheFile(id, file string, lock bool) error {
	e, ok := b.data[id]
	if !ok {
		return ErrEntryNotFound
	}
	if lock {
		e.m.Lock()
	}
	e.CachedFile = file
	if lock {
		e.m.Unlock()
	}
	err := b.save()
	if err != nil {
		return errors.Wrap(err, "failed to save new cache file name")
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

	if e.Status == StateInProgress {
		log.Debugf("entry %s seem to already be in progress", id)
		e.m.Unlock()
		return b.entryInProgress(id, res)
	} else if e.Status != StateInit {
		e.m.Unlock()
		return errors.Errorf("Entry in unexpected state: %s, expected init", e.Status)
	}

	tURL, err := b.getProxyURL(e.Path)
	if err != nil {
		e.m.Unlock()
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
		e.m.Unlock()
		return errors.Wrap(err, "target request failed")
	}
	e.resp, err = newResponse(targetResp.Header, targetResp.Body, targetResp.StatusCode)
	if err != nil {
		e.m.Unlock()
		return errors.Wrap(err, "failed to create cached response")
	}
	cacheFile := b.generateCacheFileName(id)
	err = b.setEntryCacheFile(id, cacheFile, false)
	if err != nil {
		e.m.Unlock()
		return err
	}

	err = b.setEntryStateNoEntryLock(e, StateInProgress)
	if err != nil {
		e.m.Unlock()
		return err
	}

	go b.startCaching(id, e, targetResp.Body)

	log.Debugf("Entry %s is initialized", id)
	e.m.Unlock()

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

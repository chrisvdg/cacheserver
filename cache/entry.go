package cache

import (
	"net/url"
	"sync"
	"time"
)

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
	// StateInvalid represents an invalid state
	StateInvalid State = ""
)

// Entry represents a backend entry
type Entry struct {
	// Path represents the request path of the cached entry
	Path string `json:"path"`
	// Params represents the URL request params of the cached entry
	Params url.Values `json:"params"`
	// Created is the timestamp for when the entry was initialized
	InitTime JSONTime `json:"innited"`
	// Status  represents the entry status
	Status State `json:"status"`
	// CachedFile represents the file location of the cached request body
	CachedFile string `json:"cached_file"`
	m          *sync.Mutex
	resp       *response
	readWg     *sync.WaitGroup
}

// expired checks if entry is expired
func (e *Entry) expired(expirationDuration time.Duration) bool {
	if e.InitTime.Time().Add(expirationDuration).Unix() < time.Now().Unix() {
		return true
	}

	return false
}

package cache

import (
	"net/url"
	"sync"
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
)

// Entry represents a backend entry
type Entry struct {
	// Path represents the request path of the cached entry
	Path string `json:"path"`
	// Params represents the URL request params of the cached entry
	Params url.Values `json:"params"`
	// Created is the timestamp for when the entry was created
	Created JSONTime `json:"created"`
	// Status  represents the entry status
	Status State `json:"Status"`
	m      *sync.Mutex
}

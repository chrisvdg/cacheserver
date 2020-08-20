package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMarkExpired(t *testing.T) {
	assert := assert.New(t)
	nosave = true

	b := &backend{
		cleanupInterval: 1,
		cacheExpiration: 10 * time.Minute,
		m:               &sync.Mutex{},
		data: map[string]*Entry{
			"1": {
				Status:   StateCached,
				InitTime: JSONTime(time.Now()),
				m:        &sync.Mutex{},
			},
			"2": {
				Status:   StateCached,
				InitTime: JSONTime(time.Now().Add(10 * time.Minute)),
				m:        &sync.Mutex{},
			},
			// should expire
			"3": {
				Status:   StateCached,
				InitTime: JSONTime(time.Now().Local().Add(-15 * time.Minute)),
				m:        &sync.Mutex{},
			},
			"4": {
				Status:   StateInit,
				InitTime: JSONTime(time.Now().Local().Add(-15 * time.Minute)),
				m:        &sync.Mutex{},
			},
			"5": {
				Status:   StateInProgress,
				InitTime: JSONTime(time.Now().Local().Add(-15 * time.Minute)),
				m:        &sync.Mutex{},
			},
			// should expire
			"6": {
				Status:   StateCached,
				InitTime: JSONTime(time.Time{}),
				m:        &sync.Mutex{},
			},
			// should expire
			"7": {
				Status:   StateCached,
				InitTime: JSONTime(time.Now().Local().Add(-601 * time.Second)),
				m:        &sync.Mutex{},
			},
		},
	}

	b.markExpired()
	// assert state cached
	for _, i := range []string{"1", "2"} {
		assert.Equal(StateCached, b.data[i].Status)
	}

	// assert state progress
	for _, i := range []string{"5"} {
		assert.Equal(StateInProgress, b.data[i].Status)
	}

	// assert state init
	for _, i := range []string{"3", "4", "6", "7"} {
		assert.Equal(StateInit, b.data[i].Status)
	}
}

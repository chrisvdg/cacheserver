package cache

import (
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func (b *backend) cleanup(quit <-chan struct{}) {
	if b.cleanupInterval == 0 {
		return
	}
	ticker := time.NewTicker(b.cleanupInterval)
	for {
		select {
		case <-ticker.C:
			b.markExpired()
			b.cleanCacheDir()
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (b *backend) markExpired() {
	if b.cacheExpiration == 0 {
		return
	}
	log.Debug("Started marking expired cache entries.")
	for eID, e := range b.data {
		if e.Status == StateCached {
			if e.expired(b.cacheExpiration) {
				log.Debugf("Entry %s has expired", eID)
				err := b.setEntryState(eID, StateInit, true)
				if err != nil {
					log.Error(err)
					continue
				}
				err = b.setEntryCacheFile(eID, "", true)
				if err != nil {
					log.Error(err)
					continue
				}
			}
		}
	}
	log.Debug("Finished marking expired cache entries.")
}

func (b *backend) cleanCacheDir() {
	log.Debug("Started deleting invalid cache files.")

	filesInUse := []string{}
	for eID, e := range b.data {
		if e.Status == StateCached || e.Status == StateInProgress {
			f := filepath.Base(e.CachedFile)
			filesInUse = append(filesInUse, f)
		} else {
			if e.CachedFile != "" {
				err := b.setEntryCacheFile(eID, "", true)
				if err != nil {
					log.Error(err)
					continue
				}
			}
		}
	}

	if len(filesInUse) == 0 {
		log.Debug("No files currently in use.")
	} else {
		log.Debugf("Current files in use: %s", strings.Join(filesInUse, ", "))
	}

	currentFiles, err := listFiles(b.cacheDir)
	if err != nil {
		log.Errorf("Failed to list cache dir files: %s", err)
	}
	for _, cf := range currentFiles {
		if !inStringSlice(filesInUse, cf) {
			log.Debugf("Deleting file %s", cf)
			err = deletefile(b.cacheDir, cf)
			if err != nil {
				log.Errorf("Failed to delete file %s: %s", cf, err)
			}
		}
	}

	log.Debug("Finished deleting invalid cache files.")
}

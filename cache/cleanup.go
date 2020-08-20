package cache

import (
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
	log.Debug("Started marking expired cache")
	for eID, e := range b.data {
		if e.Status == StateCached {
			if e.Created.Time().Add(b.cacheExpiration).Unix() < time.Now().Unix() {
				log.Debugf("Entry %s has expired", eID)
				err := b.setEntryState(eID, StateInit)
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
	log.Debug("Finished marking expired cache")
}

func (b *backend) cleanCacheDir() {
	log.Debug("Started deleting invalid cache files")

	filesInUse := []string{}
	for eID, e := range b.data {
		if e.Status == StateCached || e.Status == StateInProgress {
			filesInUse = append(filesInUse, e.CachedFile)
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

	currentFiles, err := listFiles(b.cacheDir)
	if err != nil {
		log.Errorf("Failed to list cache dir files: %s", err)
	}
	for _, cf := range currentFiles {
		if !inStringSlice(filesInUse, cf) {
			err = deletefile(b.cacheDir, cf)
			if err != nil {
				log.Errorf("Failed to delete file %s: %s", cf, err)
			}
		}
	}

	log.Debug("Finished deleting invalid cache files")
}

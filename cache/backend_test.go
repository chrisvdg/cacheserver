package cache

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCacheFileName(t *testing.T) {
	assert := assert.New(t)
	entryID := "foobar"
	cacheDir := "/tmp"
	b := &backend{
		cacheDir: cacheDir,
	}

	result1 := b.generateCacheFileName(entryID)
	assert.True(strings.HasPrefix(result1, fmt.Sprintf("%s/%s_", cacheDir, entryID)))
	assert.True(strings.HasSuffix(result1, ".blob"))

	result2 := b.generateCacheFileName(entryID)
	assert.NotEqual(result1, result2)
	assert.True(strings.HasPrefix(result2, fmt.Sprintf("%s/%s_", cacheDir, entryID)))
	assert.True(strings.HasSuffix(result1, ".blob"))
}

package cache

import (
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const base64URLCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// generateID returns a random base64URL string of provided length
// Not guaranteed to be unique
func generateID(length int) string {
	r := make([]byte, length)
	for i := range r {
		r[i] = base64URLCharset[rand.Intn(len(base64URLCharset))]
	}

	return string(r)
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

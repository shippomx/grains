package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// newTempFile returns a new output file in dir with the provided prefix and suffix.
func newTempFile(dir, prefix, suffix string) (*os.File, error) {
	for index := 1; index < 10000; index++ {
		switch f, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("%s%03d%s", prefix, index, suffix)), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666); {
		case err == nil:
			return f, nil
		case !os.IsExist(err):
			return nil, err
		}
	}
	// Give up
	return nil, fmt.Errorf("could not create file of the form %s%03d%s", prefix, 1, suffix)
}

var tempFiles []string
var tempFilesMu = sync.Mutex{}

// deferDeleteTempFile marks a file to be deleted by next call to Cleanup()
func deferDeleteTempFile(path string) {
	tempFilesMu.Lock()
	tempFiles = append(tempFiles, path)
	tempFilesMu.Unlock()
}

// cleanupTempFiles removes any temporary files selected for deferred cleaning.
func cleanupTempFiles() error {
	tempFilesMu.Lock()
	defer tempFilesMu.Unlock()
	var lastErr error
	for _, f := range tempFiles {
		if err := os.Remove(f); err != nil {
			lastErr = err
		}
	}
	tempFiles = nil
	return lastErr
}

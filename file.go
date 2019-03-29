package vimg

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

/**
 * Check the path to make sure it is clean
 *
 * Read file in, managing own buffer/slice size to reduce slice bloat
 */
func Read(path string, size int64) (*[]byte, error) {
	buffer := make([]byte, size)
	cleanPath := filepath.Clean(path)
	if cleanPath == "." {
		return nil, errors.New("File path traversal")
	}
	file, e := os.Open( cleanPath )
	if e != nil {
		return nil, e
	}
	buffer, e = ioutil.ReadAll(file)
	if e != nil {
		return nil, e
	}
	return &buffer, nil
}

// Write writes the given byte buffer into disk
// to the given file path.
func Write(path string, buf []byte) error {
	return ioutil.WriteFile(path, buf, 0644)
}

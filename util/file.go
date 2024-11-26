package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// Default File Mode
	DefFileMode = 0666

	GbUnit uint64 = MbUnit * 1024
	MbUnit uint64 = KbUnit * 1024
	KbUnit uint64 = 1024
)

// Check if file exists
func FileExists(path string) (bool, error) {
	_, e := os.Stat(path)
	if e == nil {
		return true, nil
	}

	if os.IsNotExist(e) {
		return false, nil
	}

	return false, e
}

// Read all content from file.
func ReadFileAll(path string) ([]byte, error) {
	f, err := ReadWriteFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %v, %w", path, err)
	}
	defer f.Close()
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read from file %v, %w", path, err)
	}
	return buf, nil
}

// Open file with 0666 permission.
func OpenFile(name string, flag int) (*os.File, error) {
	return os.OpenFile(name, flag, DefFileMode)
}

// Create appendable file with 0666 permission.
func AppendableFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY)
}

// Create readable & writable file with 0666 permission.
func ReadWriteFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_CREATE|os.O_RDWR)
}

// MkdirAll with 0755 perm.
func MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// MkdirAll but only for the parent directory of the path, perm 0755 is used.
//
// The path should always point to a specific file under some directories,
// as this method always attempts to extract parent dir of the file.
// It the path fails to fulfill this requirement, the output might be unexpected.
func MkdirParentAll(path string) error {
	return MkdirAll(filepath.Dir(path))
}

// Save to temp file, returns temp file path or error.
func SaveTmpFile(tmpDir string, reader io.Reader) (string, error) {
	f, err := os.CreateTemp(tmpDir, "temp_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file, %w", err)
	}
	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("failed to save temp file, %v, %w", f.Name(), err)
	}
	return f.Name(), nil
}

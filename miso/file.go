package miso

import "os"

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

// Open file with 0666 permission.
func OpenFile(name string, flag int) (*os.File, error) {
	return os.OpenFile(name, flag, 0666)
}

// Create writable file with 0666 permisson.
func WritableFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_CREATE|os.O_WRONLY)
}

// Create appendable file with 0666 permisson.
func AppendableFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_CREATE|os.O_APPEND)
}

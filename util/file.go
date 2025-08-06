package util

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
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

// Check if file exists without returning error.
//
// This is mainly a lazy version of FileExists(), it's not recommended for most cases.
func TryFileExists(path string) bool {
	ok, err := FileExists(path)
	if err != nil {
		return false
	}
	return ok
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
//
// If file is missing, new empty file is created.
func ReadWriteFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_CREATE|os.O_RDWR)
}

// Open readable file with 0666 permission.
func OpenRFile(name string) (*os.File, error) {
	return OpenFile(name, os.O_RDONLY)
}

// Open readable & writable file with 0666 permission.
func OpenRWFile(name string, createIfAbsent bool) (*os.File, error) {
	flag := os.O_RDWR
	if createIfAbsent {
		flag = os.O_CREATE | flag
	}
	return OpenFile(name, flag)
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

func FileHasSuffix(name string, ext string) bool {
	if name == "" || ext == "" {
		return false
	}
	return HasSuffixIgnoreCase(name, "."+ext)
}

func FileHasAnySuffix(name string, ext ...string) bool {
	for _, ex := range ext {
		if FileHasSuffix(name, ex) {
			return true
		}
	}
	return false
}

func FileAddSuffix(name string, ext string) string {
	if FileHasSuffix(name, ext) {
		return name
	}
	return name + "." + ext
}

func FileCutSuffix(name string, ext string) (string, bool) {
	if name == "" || ext == "" {
		return name, false
	}
	if strings.HasSuffix(strings.ToLower(name), "."+ext) {
		return name[:len(name)-len(ext)-1], true
	}
	return name, false
}

func FileCutDotSuffix(name string) (s string, suffix string, ok bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return name, "", false
	}
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return name, "", false
	}
	return name[:i], name[i+1:], true
}

func TempFilePath() (string, error) {
	tmpFile, err := os.CreateTemp("/tmp", "temp_*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	return tmpFile.Name(), nil
}

func TempFile() (*os.File, error) {
	tmpFile, err := os.CreateTemp("/tmp", "temp_*")
	if err != nil {
		return nil, err
	}
	return tmpFile, nil
}

func TempFilePathSuffix(suffix string) (string, error) {
	tmpFile, err := os.CreateTemp("/tmp", "temp_*."+suffix)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	return tmpFile.Name(), nil
}

func TempFileSuffix(suffix string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("/tmp", "temp_*."+suffix)
	if err != nil {
		return nil, err
	}
	return tmpFile, nil
}

type WalkFsFile struct {
	Path string
	File fs.FileInfo
}

func WalkDir(n string, suffix ...string) ([]WalkFsFile, error) {
	entries, err := os.ReadDir(n)
	if err != nil {
		return nil, err
	}
	files := make([]WalkFsFile, 0, len(entries))
	for _, et := range entries {
		fi, err := et.Info()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return files, err
		}
		p := path.Join(n, fi.Name())
		if et.IsDir() {
			ff, err := WalkDir(p, suffix...)
			if err == nil {
				files = append(files, ff...)
			}
		} else {
			if len(suffix) < 1 {
				files = append(files, WalkFsFile{File: fi, Path: p})
			} else {
				if FileHasAnySuffix(fi.Name(), suffix...) {
					files = append(files, WalkFsFile{File: fi, Path: p})
				}
			}
		}
	}
	return files, nil
}

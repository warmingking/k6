package fs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

var (
	_ fs.FS = (*AferoBased)(nil)
	_ RWFS  = (*AferoBased)(nil)
)

// AferoBased is the implementation of the fs.RWFS, based on the afero
type AferoBased struct {
	afero afero.Fs
}

// NewAferoBased creates a new Aferobased read-write fs
func NewAferoBased(a afero.Fs) *AferoBased {
	return &AferoBased{afero: a}
}

// NewInMemoryFS .
func NewInMemoryFS() *AferoBased {
	return NewAferoBased(afero.NewMemMapFs())
}

// NewAferoOSFS .
func NewAferoOSFS() *AferoBased {
	return NewAferoBased(afero.NewOsFs())
}

// Afero is the getter for the affero
// deprecated
func (ab AferoBased) Afero() afero.Fs { // nolint:ireturn
	return ab.afero
}

// Open opens the named file.
func (ab *AferoBased) Open(name string) (fs.File, error) {
	// by convention for fs.FS implementations we should perform this check
	// TODO: check why
	// here is an issue that /path/to/same-dir.proto is invalid path
	if !fs.ValidPath(strings.TrimLeft(name, string(filepath.Separator))) {
		return nil, errors.New("invalid path: " + name)
	}

	file, err := ab.afero.Open(name)
	if err != nil {
		return nil, err
	}

	if _, ok := file.(fs.ReadDirFile); !ok {
		return readDirFile{file}, nil
	}

	return file, nil
}

// ReadFile .
func (ab AferoBased) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(ab.afero, path)
}

// WriteFile .
func (ab AferoBased) WriteFile(path string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(ab.afero, path, data, perm)
}

// MkdirAll .
func (ab AferoBased) MkdirAll(path string, perm os.FileMode) error {
	return ab.afero.MkdirAll(path, perm)
}

// OpenFile .
func (ab AferoBased) OpenFile(name string, flag int, perm os.FileMode) (WFile, error) { // nolint:ireturn
	return ab.afero.OpenFile(name, flag, perm)
}

// Stat .
func (ab AferoBased) Stat(name string) (os.FileInfo, error) {
	return ab.afero.Stat(name)
}

// Create .
func (ab AferoBased) Create(name string) (WFile, error) { // nolint:ireturn
	return ab.afero.Create(name)
}

type readDirFile struct {
	afero.File
}

var _ fs.ReadDirFile = readDirFile{}

func (r readDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	items, err := r.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	ret := make([]fs.DirEntry, len(items))
	for i := range items {
		ret[i] = dirEntry{items[i]}
	}

	return ret, nil
}

// dirEntry provides adapter from os.FileInfo to fs.DirEntry
type dirEntry struct {
	fs.FileInfo
}

var _ fs.DirEntry = dirEntry{}

func (d dirEntry) Type() fs.FileMode { return d.FileInfo.Mode().Type() }

func (d dirEntry) Info() (fs.FileInfo, error) { return d.FileInfo, nil }

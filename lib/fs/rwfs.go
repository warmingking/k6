// Package fs contains the abstraction for the file system
package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// FilePathSeparator filepath separator defined by os.Separator.
const FilePathSeparator = string(filepath.Separator)

// WFile combines the file and writer
type WFile interface {
	fs.File

	io.Writer
	Sync() error
}

// RWFS is read-write file system
type RWFS interface {
	fs.FS
	fs.StatFS
	fs.ReadFileFS

	// deprecated
	Afero() afero.Fs

	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (WFile, error)

	WriteFile(path string, data []byte, perm os.FileMode) error

	MkdirAll(path string, perm os.FileMode) error

	// OpenFile opens a file using the given flags and the given mode.
	// instead of the original afero.File we use io.Writer
	OpenFile(name string, flag int, perm os.FileMode) (WFile, error)
}

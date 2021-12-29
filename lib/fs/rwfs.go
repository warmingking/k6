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

// ReadWriteFS is read-write file system
type ReadWriteFS interface {
	fs.FS
	fs.StatFS
	fs.ReadFileFS
	WriteFS

	// Afero is something that will be removed once code will be complete
	Afero() afero.Fs
}

// WriteFS is a FS that can perform write operations
type WriteFS interface {
	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (WritableFile, error)

	// WriteFile writes to the named file, returning the file and an
	// error, if any happens.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that does not exist
	// yet.
	MkdirAll(path string, perm os.FileMode) error

	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flag int, perm os.FileMode) (WritableFile, error)
}

// WritableFile combines the fs.File and io.Writer
// compatable with the afero.File
type WritableFile interface {
	fs.File

	io.Writer
	Sync() error
}

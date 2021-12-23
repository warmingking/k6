// Package fsext .
package fsext

import (
	"io/fs"
	"os"

	"github.com/spf13/afero"
)

var _ fs.FS = (*FS)(nil)

// FS is the wrapper for the file system
type FS struct {
	afero afero.Fs
}

// NewFS creates a new wrapper
func NewFS(a afero.Fs) FS {
	return FS{afero: a}
}

// NewInMemoryFS .
func NewInMemoryFS() FS {
	return NewFS(afero.NewMemMapFs())
}

// Afero is the getter for the affero
// deprecated
func (fs FS) Afero() afero.Fs { // nolint:ireturn
	return fs.afero
}

// Open opens the named file.
//
// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (fs *FS) Open(name string) (fs.File, error) {
	panic("not implemented") // TODO: Implement
}

// ReadFile .
func (fs FS) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(fs.afero, path)
}

// WriteFile .
func (fs FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(fs.afero, path, data, perm)
}

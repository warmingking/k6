package fs

import (
	"io/fs"
	"os"

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
func (fs AferoBased) Afero() afero.Fs { // nolint:ireturn
	return fs.afero
}

// Open opens the named file.
func (fs *AferoBased) Open(name string) (fs.File, error) {
	panic("not implemented") // TODO: Implement
}

// ReadFile .
func (fs AferoBased) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(fs.afero, path)
}

// WriteFile .
func (fs AferoBased) WriteFile(path string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(fs.afero, path, data, perm)
}

// MkdirAll .
func (fs AferoBased) MkdirAll(path string, perm os.FileMode) error {
	return fs.afero.MkdirAll(path, perm)
}

// Package fs contains the abstraction for the file system
package fs

import (
	"io/fs"
	"os"

	"github.com/spf13/afero"
)

// RWFS is read-write file system
type RWFS interface {
	fs.FS

	// deprecated
	Afero() afero.Fs

	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

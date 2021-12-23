/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2019 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package fsext

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestChangePathFs(t *testing.T) {
	t.Parallel()

	aferoFS := afero.NewMemMapFs()

	prefix := "/another/"
	changePathFS := NewChangePathFs(aferoFS, ChangePathFunc(func(name string) (string, error) {
		if !strings.HasPrefix(name, prefix) {
			return "", fmt.Errorf("path %s doesn't  start with `%s`", name, prefix)
		}
		return name[len(prefix):], nil
	}))

	filePath := "/another/path/to/file.txt"

	require.Equal(t, changePathFS.Name(), "ChangePathFs")
	t.Run("Create", func(t *testing.T) {
		f, err := changePathFS.Create(filePath)
		require.NoError(t, err)
		require.Equal(t, filePath, f.Name())

		/** TODO figure out if this is error in MemMapFs
		_, err = c.Create(filePath)
		require.Error(t, err)
		require.True(t, os.IsExist(err))
		*/

		_, err = changePathFS.Create("/notanother/path/to/file.txt")
		checkErrorPath(t, err, "/notanother/path/to/file.txt")
	})

	t.Run("Mkdir", func(t *testing.T) {
		require.NoError(t, changePathFS.Mkdir("/another/path/too", 0o644))
		checkErrorPath(t, changePathFS.Mkdir("/notanother/path/too", 0o644), "/notanother/path/too")
	})

	t.Run("MkdirAll", func(t *testing.T) {
		require.NoError(t, changePathFS.MkdirAll("/another/pattth/too", 0o644))
		checkErrorPath(t, changePathFS.MkdirAll("/notanother/pattth/too", 0o644), "/notanother/pattth/too")
	})

	t.Run("Open", func(t *testing.T) {
		f, err := changePathFS.Open(filePath)
		require.NoError(t, err)
		require.Equal(t, filePath, f.Name())

		_, err = changePathFS.Open("/notanother/path/to/file.txt")
		checkErrorPath(t, err, "/notanother/path/to/file.txt")
	})

	t.Run("OpenFile", func(t *testing.T) {
		f, err := changePathFS.OpenFile(filePath, os.O_RDWR, 0o644)
		require.NoError(t, err)
		require.Equal(t, filePath, f.Name())

		_, err = changePathFS.OpenFile("/notanother/path/to/file.txt", os.O_RDWR, 0o644)
		checkErrorPath(t, err, "/notanother/path/to/file.txt")

		_, err = changePathFS.OpenFile("/another/nonexistant", os.O_RDWR, 0o644)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("Stat Chmod Chtimes", func(t *testing.T) {
		info, err := changePathFS.Stat(filePath)
		require.NoError(t, err)
		require.Equal(t, "file.txt", info.Name())

		sometime := time.Unix(10000, 13)
		require.NotEqual(t, sometime, info.ModTime())
		require.NoError(t, changePathFS.Chtimes(filePath, time.Now(), sometime))
		require.Equal(t, sometime, info.ModTime())

		mode := os.FileMode(0o007)
		require.NotEqual(t, mode, info.Mode())
		require.NoError(t, changePathFS.Chmod(filePath, mode))
		require.Equal(t, mode, info.Mode())

		_, err = changePathFS.Stat("/notanother/path/to/file.txt")
		checkErrorPath(t, err, "/notanother/path/to/file.txt")

		checkErrorPath(t, changePathFS.Chtimes("/notanother/path/to/file.txt", time.Now(), time.Now()), "/notanother/path/to/file.txt")

		checkErrorPath(t, changePathFS.Chmod("/notanother/path/to/file.txt", mode), "/notanother/path/to/file.txt")
	})

	t.Run("LstatIfPossible", func(t *testing.T) {
		info, ok, err := changePathFS.LstatIfPossible(filePath)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, "file.txt", info.Name())

		_, _, err = changePathFS.LstatIfPossible("/notanother/path/to/file.txt")
		checkErrorPath(t, err, "/notanother/path/to/file.txt")
	})

	t.Run("Rename", func(t *testing.T) {
		info, err := changePathFS.Stat(filePath)
		require.NoError(t, err)
		require.False(t, info.IsDir())

		require.NoError(t, changePathFS.Rename(filePath, "/another/path/to/file.doc"))

		_, err = changePathFS.Stat(filePath)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))

		info, err = changePathFS.Stat("/another/path/to/file.doc")
		require.NoError(t, err)
		require.False(t, info.IsDir())

		checkErrorPath(t,
			changePathFS.Rename("/notanother/path/to/file.txt", "/another/path/to/file.doc"),
			"/notanother/path/to/file.txt")

		checkErrorPath(t,
			changePathFS.Rename(filePath, "/notanother/path/to/file.doc"),
			"/notanother/path/to/file.doc")
	})

	t.Run("Remove", func(t *testing.T) {
		removeFilePath := "/another/file/to/remove.txt"
		_, err := changePathFS.Create(removeFilePath)
		require.NoError(t, err)

		require.NoError(t, changePathFS.Remove(removeFilePath))

		_, err = changePathFS.Stat(removeFilePath)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))

		_, err = changePathFS.Create(removeFilePath)
		require.NoError(t, err)

		require.NoError(t, changePathFS.RemoveAll(path.Dir(removeFilePath)))

		_, err = changePathFS.Stat(removeFilePath)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))

		checkErrorPath(t,
			changePathFS.Remove("/notanother/path/to/file.txt"),
			"/notanother/path/to/file.txt")

		checkErrorPath(t,
			changePathFS.RemoveAll("/notanother/path/to"),
			"/notanother/path/to")
	})
}

func checkErrorPath(t *testing.T, err error, path string) {
	require.Error(t, err)
	p, ok := err.(*os.PathError)
	require.True(t, ok)
	require.Equal(t, p.Path, path)
}

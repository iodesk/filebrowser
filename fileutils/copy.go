package fileutils

import (
	"io/fs"
	"os"
	"path"

	"github.com/spf13/afero"
)

// Copy copies a file or folder from one place to another.
func Copy(afs afero.Fs, src, dst string, fileMode, dirMode fs.FileMode) error {
	return CopyOwned(afs, src, dst, fileMode, dirMode, Ownership{})
}

// CopyOwned copies a file or folder and chowns every created entry to own.
func CopyOwned(afs afero.Fs, src, dst string, fileMode, dirMode fs.FileMode, own Ownership) error {
	if src = path.Clean("/" + src); src == "" {
		return os.ErrNotExist
	}

	if dst = path.Clean("/" + dst); dst == "" {
		return os.ErrNotExist
	}

	if src == "/" || dst == "/" {
		// Prohibit copying from or to the virtual root directory.
		return os.ErrInvalid
	}

	if dst == src {
		return os.ErrInvalid
	}

	info, err := afs.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return CopyDirOwned(afs, src, dst, fileMode, dirMode, own)
	}

	return CopyFileOwned(afs, src, dst, fileMode, dirMode, own)
}

package fileutils

import (
	"errors"
	"io/fs"

	"github.com/spf13/afero"
)

// CopyDir copies a directory from source to dest and all
// of its sub-directories. It doesn't stop if it finds an error
// during the copy. Returns an error if any.
func CopyDir(afs afero.Fs, source, dest string, fileMode, dirMode fs.FileMode) error {
	return CopyDirOwned(afs, source, dest, fileMode, dirMode, Ownership{})
}

// CopyDirOwned copies a directory tree from source to dest and chowns every
// newly created entry to own.
func CopyDirOwned(afs afero.Fs, source, dest string, fileMode, dirMode fs.FileMode, own Ownership) error {
	// Get properties of source.
	srcinfo, err := afs.Stat(source)
	if err != nil {
		return err
	}

	// Create the destination directory.
	if err = afs.MkdirAll(dest, srcinfo.Mode()); err != nil {
		return err
	}
	if err = own.Chown(realPath(afs, dest)); err != nil {
		return err
	}

	dir, _ := afs.Open(source)
	obs, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	var errs []error

	for _, obj := range obs {
		fsource := source + "/" + obj.Name()
		fdest := dest + "/" + obj.Name()

		if obj.IsDir() {
			if err = CopyDirOwned(afs, fsource, fdest, fileMode, dirMode, own); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err = CopyFileOwned(afs, fsource, fdest, fileMode, dirMode, own); err != nil {
				errs = append(errs, err)
			}
		}
	}

	var errString string
	for _, err := range errs {
		errString += err.Error() + "\n"
	}

	if errString != "" {
		return errors.New(errString)
	}

	return nil
}

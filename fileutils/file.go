package fileutils

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/afero"
)

// RealPath resolves the on-disk absolute path for a virtual path inside afs.
// When afs is an afero.BasePathFs the path is rebased onto the real root;
// otherwise the path is returned as-is. It also handles filesystems that
// expose a Base() method returning *afero.BasePathFs (e.g. ScopedFs).
func RealPath(afs afero.Fs, p string) string {
	if bfs, ok := afs.(*afero.BasePathFs); ok {
		return afero.FullBaseFsPath(bfs, p)
	}
	// Support ScopedFs or any wrapper that exposes the underlying BasePathFs.
	type baser interface {
		Base() *afero.BasePathFs
	}
	if b, ok := afs.(baser); ok {
		return afero.FullBaseFsPath(b.Base(), p)
	}
	return p
}

// realPath is a package-internal alias kept for backward compatibility.
func realPath(afs afero.Fs, p string) string { return RealPath(afs, p) }

// MoveFile moves file from src to dst.
// By default the rename filesystem system call is used. If src and dst point to different volumes
// the file copy is used as a fallback
func MoveFile(afs afero.Fs, src, dst string, fileMode, dirMode fs.FileMode) error {
	if afs.Rename(src, dst) == nil {
		return nil
	}
	// fallback
	err := Copy(afs, src, dst, fileMode, dirMode)
	if err != nil {
		_ = afs.Remove(dst)
		return err
	}
	if err := afs.RemoveAll(src); err != nil {
		return err
	}
	return nil
}

// CopyFile copies a file from source to dest and returns
// an error if any.
func CopyFile(afs afero.Fs, source, dest string, fileMode, dirMode fs.FileMode) error {
	return CopyFileOwned(afs, source, dest, fileMode, dirMode, Ownership{})
}

// CopyFileOwned copies a file from source to dest and chowns the destination
// to own immediately after creation.
func CopyFileOwned(afs afero.Fs, source, dest string, fileMode, dirMode fs.FileMode, own Ownership) error {
	// Open the source file.
	src, err := afs.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	// Makes the directory needed to create the dst file.
	err = afs.MkdirAll(filepath.Dir(dest), dirMode)
	if err != nil {
		return err
	}

	// Create the destination file.
	dst, err := afs.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the contents of the file.
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Copy the mode.
	info, err := afs.Stat(source)
	if err != nil {
		return err
	}
	if err = afs.Chmod(dest, info.Mode()); err != nil {
		return err
	}

	// Resolve the real on-disk path so os.Lchown works even when afs is a
	// BasePathFs (which returns virtual paths from afs methods).
	realDest := realPath(afs, dest)
	return own.Chown(realDest)
}

// CommonPrefix returns common directory path of provided files
func CommonPrefix(sep byte, paths ...string) string {
	// Handle special cases.
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return path.Clean(paths[0])
	}

	// Note, we treat string as []byte, not []rune as is often
	// done in Go. (And sep as byte, not rune). This is because
	// most/all supported OS' treat paths as string of non-zero
	// bytes. A filename may be displayed as a sequence of Unicode
	// runes (typically encoded as UTF-8) but paths are
	// not required to be valid UTF-8 or in any normalized form
	// (e.g. "é" (U+00C9) and "é" (U+0065,U+0301) are different
	// file names.
	c := []byte(path.Clean(paths[0]))

	// We add a trailing sep to handle the case where the
	// common prefix directory is included in the path list
	// (e.g. /home/user1, /home/user1/foo, /home/user1/bar).
	// path.Clean will have cleaned off trailing / separators with
	// the exception of the root directory, "/" (in which case we
	// make it "//", but this will get fixed up to "/" below).
	c = append(c, sep)

	// Ignore the first path since it's already in c
	for _, v := range paths[1:] {
		// Clean up each path before testing it
		v = path.Clean(v) + string(sep)

		// Find the first non-common byte and truncate c
		if len(v) < len(c) {
			c = c[:len(v)]
		}
		for i := 0; i < len(c); i++ {
			if v[i] != c[i] {
				c = c[:i]
				break
			}
		}
	}

	// Remove trailing non-separator characters and the final separator
	for i := len(c) - 1; i >= 0; i-- {
		if c[i] == sep {
			c = c[:i]
			break
		}
	}

	return string(c)
}

package fbhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"

	"github.com/filebrowser/filebrowser/v2/fileutils"
)

type archiveRequest struct {
	// Files to compress (relative paths within the user's scope)
	Files []string `json:"files"`
	// Destination path for the archive (relative within scope)
	Destination string `json:"destination"`
	// Format: zip, tar, targz, tarbz2, tarxz, tarlz4, tarsz, tarbr, tarzst
	Format string `json:"format"`
}

type extractRequest struct {
	// Source archive file to extract (relative path within scope)
	Source string `json:"source"`
	// Destination directory to extract into (relative within scope)
	Destination string `json:"destination"`
}

// archiveHandler compresses selected files/directories into an archive file
// stored within the user's scope.
var archiveHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Create || !d.user.Perm.Download {
		return http.StatusForbidden, nil
	}

	var req archiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return http.StatusBadRequest, err
	}

	if len(req.Files) == 0 {
		return http.StatusBadRequest, fmt.Errorf("no files specified")
	}
	if req.Destination == "" {
		return http.StatusBadRequest, fmt.Errorf("no destination specified")
	}
	if req.Format == "" {
		req.Format = "zip"
	}

	// Validate destination path
	req.Destination = path.Clean("/" + req.Destination)
	if !d.Check(req.Destination) {
		return http.StatusForbidden, nil
	}

	// Check if destination already exists
	if _, err := d.user.Fs.Stat(req.Destination); err == nil {
		return http.StatusConflict, fmt.Errorf("destination already exists: %s", req.Destination)
	}

	// Resolve archiver from format
	_, archiver, err := parseQueryAlgorithmFromString(req.Format)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Collect files to archive
	var allFiles []archives.FileInfo
	for _, f := range req.Files {
		f = path.Clean("/" + f)
		if !d.Check(f) {
			continue
		}

		commonDir := path.Dir(f)
		archiveFiles, err := getFiles(d, f, commonDir)
		if err != nil {
			log.Printf("[archive] failed to collect files from %s: %v", f, err)
			continue
		}
		allFiles = append(allFiles, archiveFiles...)
	}

	if len(allFiles) == 0 {
		return http.StatusBadRequest, fmt.Errorf("no valid files to archive")
	}

	// Ensure parent directory exists
	destDir := path.Dir(req.Destination)
	if err := d.user.Fs.MkdirAll(destDir, d.settings.DirMode); err != nil {
		return http.StatusInternalServerError, err
	}

	// Create the archive file
	outFile, err := d.user.Fs.OpenFile(req.Destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, d.settings.FileMode)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer outFile.Close()

	// Write archive
	err = archiver.Archive(r.Context(), outFile, allFiles)
	if err != nil {
		// Clean up on failure
		_ = d.user.Fs.Remove(req.Destination)
		return http.StatusInternalServerError, fmt.Errorf("archive creation failed: %w", err)
	}

	// Chown the archive file
	own := d.user.Ownership()
	if own.IsSet() {
		realDst := fileutils.RealPath(d.user.Fs, req.Destination)
		_ = own.Chown(realDst)
	}

	return http.StatusOK, nil
})

// extractHandler extracts an archive file into a directory within the user's scope.
var extractHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Create {
		return http.StatusForbidden, nil
	}

	var req extractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return http.StatusBadRequest, err
	}

	if req.Source == "" {
		return http.StatusBadRequest, fmt.Errorf("no source specified")
	}
	if req.Destination == "" {
		return http.StatusBadRequest, fmt.Errorf("no destination specified")
	}

	req.Source = path.Clean("/" + req.Source)
	req.Destination = path.Clean("/" + req.Destination)

	// Validate paths
	if !d.Check(req.Source) || !d.Check(req.Destination) {
		return http.StatusForbidden, nil
	}

	// Open the source archive
	srcFile, err := d.user.Fs.Open(req.Source)
	if err != nil {
		return errToStatus(err), err
	}
	defer srcFile.Close()

	// Get file info for the archive identification
	srcInfo, err := d.user.Fs.Stat(req.Source)
	if err != nil {
		return errToStatus(err), err
	}

	// Identify the archive format
	format, _, err := archives.Identify(r.Context(), srcInfo.Name(), srcFile)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("could not identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("file is not an extractable archive")
	}

	// Seek back to beginning after Identify read some bytes
	if seeker, ok := srcFile.(interface{ Seek(int64, int) (int64, error) }); ok {
		if _, err := seeker.Seek(0, 0); err != nil {
			return http.StatusInternalServerError, err
		}
	}

	// Ensure destination directory exists
	if err := d.user.Fs.MkdirAll(req.Destination, d.settings.DirMode); err != nil {
		return http.StatusInternalServerError, err
	}

	own := d.user.Ownership()

	// Chown the destination directory itself
	if own.IsSet() {
		realDst := fileutils.RealPath(d.user.Fs, req.Destination)
		_ = own.Chown(realDst)
	}

	// Get the real path of destination for extraction
	realDest := fileutils.RealPath(d.user.Fs, req.Destination)

	// Extract files
	err = extractor.Extract(r.Context(), srcFile, func(ctx context.Context, f archives.FileInfo) error {
		// Security: prevent path traversal in archive entries
		name := filepath.ToSlash(f.NameInArchive)
		if strings.Contains(name, "..") {
			log.Printf("[extract] skipping suspicious path: %s", name)
			return nil
		}

		destPath := filepath.Join(realDest, filepath.FromSlash(name))

		// Ensure the resolved path stays within destination
		if !strings.HasPrefix(destPath, realDest) {
			log.Printf("[extract] path escape attempt: %s", name)
			return nil
		}

		if f.IsDir() {
			if err := d.user.Fs.MkdirAll(path.Join(req.Destination, name), d.settings.DirMode); err != nil {
				return err
			}
			if own.IsSet() {
				_ = own.Chown(destPath)
			}
			return nil
		}

		// Ensure parent directory exists
		parentDir := filepath.Dir(destPath)
		parentVirtual := path.Dir(path.Join(req.Destination, name))
		if err := d.user.Fs.MkdirAll(parentVirtual, d.settings.DirMode); err != nil {
			return err
		}
		if own.IsSet() {
			_ = own.Chown(parentDir)
		}

		// Create the file
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		virtualPath := path.Join(req.Destination, name)
		outFile, err := d.user.Fs.OpenFile(virtualPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, d.settings.FileMode)
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		if err != nil {
			return err
		}

		if own.IsSet() {
			_ = own.Chown(destPath)
		}

		return nil
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("extraction failed: %w", err)
	}

	return http.StatusOK, nil
})

func parseQueryAlgorithmFromString(format string) (string, archives.Archival, error) {
	switch format {
	case "zip", "true", "":
		return ".zip", archives.Zip{}, nil
	case "tar":
		return ".tar", archives.Tar{}, nil
	case "targz":
		return ".tar.gz", archives.CompressedArchive{Compression: archives.Gz{}, Archival: archives.Tar{}}, nil
	case "tarbz2":
		return ".tar.bz2", archives.CompressedArchive{Compression: archives.Bz2{}, Archival: archives.Tar{}}, nil
	case "tarxz":
		return ".tar.xz", archives.CompressedArchive{Compression: archives.Xz{}, Archival: archives.Tar{}}, nil
	case "tarlz4":
		return ".tar.lz4", archives.CompressedArchive{Compression: archives.Lz4{}, Archival: archives.Tar{}}, nil
	case "tarsz":
		return ".tar.sz", archives.CompressedArchive{Compression: archives.Sz{}, Archival: archives.Tar{}}, nil
	case "tarbr":
		return ".tar.br", archives.CompressedArchive{Compression: archives.Brotli{}, Archival: archives.Tar{}}, nil
	case "tarzst":
		return ".tar.zst", archives.CompressedArchive{Compression: archives.Zstd{}, Archival: archives.Tar{}}, nil
	default:
		return "", nil, fmt.Errorf("unsupported archive format: %s", format)
	}
}

// isArchiveFile checks if a filename has an archive extension.
func isArchiveFile(name string) bool {
	lower := strings.ToLower(name)
	archiveExts := []string{
		".zip", ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2",
		".tar.xz", ".txz", ".tar.lz4", ".tar.sz", ".tar.br", ".tar.zst",
		".rar", ".7z",
	}
	for _, ext := range archiveExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

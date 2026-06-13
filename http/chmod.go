package fbhttp

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filebrowser/filebrowser/v2/fileutils"
)

type chmodRequest struct {
	Path      string `json:"path"`
	Mode      string `json:"mode"`
	DirMode   string `json:"dirMode"`
	Recursive bool   `json:"recursive"`
}

var chmodHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	// Only users with Modify permission can change file permissions.
	if !d.user.Perm.Modify {
		return http.StatusForbidden, nil
	}

	var req chmodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return http.StatusBadRequest, err
	}

	// Parse the octal mode string for files (e.g. "0644" or "644").
	modeVal, err := strconv.ParseUint(req.Mode, 8, 32)
	if err != nil {
		log.Printf("[chmod] invalid mode %q: %v", req.Mode, err)
		return http.StatusBadRequest, err
	}
	fileMode := fs.FileMode(modeVal)

	// Parse dir mode (defaults to same as file mode if not specified).
	dirMode := fileMode
	if req.DirMode != "" {
		dirModeVal, err := strconv.ParseUint(req.DirMode, 8, 32)
		if err != nil {
			log.Printf("[chmod] invalid dirMode %q: %v", req.DirMode, err)
			return http.StatusBadRequest, err
		}
		dirMode = fs.FileMode(dirModeVal)
	}

	// Validate that the path is within the user's scope.
	if !d.Check(req.Path) {
		log.Printf("[chmod] DENIED path %q failed scope check for user %q", req.Path, d.user.Username)
		return http.StatusForbidden, nil
	}

	// Resolve the real filesystem path via the scoped FS base.
	realPath := fileutils.RealPath(d.user.Fs.Base(), req.Path)

	if req.Recursive {
		err = filepath.Walk(realPath, func(p string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return os.Chmod(p, dirMode)
			}
			return os.Chmod(p, fileMode)
		})
	} else {
		err = os.Chmod(realPath, fileMode)
	}

	if err != nil {
		log.Printf("[chmod] FAILED %q: %v", realPath, err)
		return errToStatus(err), err
	}

	return http.StatusOK, nil
})

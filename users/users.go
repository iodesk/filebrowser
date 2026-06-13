package users

import (
	"log"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/spf13/afero"

	fberrors "github.com/filebrowser/filebrowser/v2/errors"
	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/fileutils"
	"github.com/filebrowser/filebrowser/v2/rules"
)

// ViewMode describes a view mode.
type ViewMode string

const (
	ListViewMode   ViewMode = "list"
	MosaicViewMode ViewMode = "mosaic"
)

// User describes a user.
type User struct {
	ID                    uint            `storm:"id,increment" json:"id"`
	Username              string          `storm:"unique" json:"username"`
	Password              string          `json:"password"`
	Scope                 string          `json:"scope"`
	Locale                string          `json:"locale"`
	LockPassword          bool            `json:"lockPassword"`
	ViewMode              ViewMode        `json:"viewMode"`
	SingleClick           bool            `json:"singleClick"`
	RedirectAfterCopyMove bool            `json:"redirectAfterCopyMove"`
	Perm                  Permissions     `json:"perm"`
	Commands              []string        `json:"commands"`
	Sorting               files.Sorting   `json:"sorting"`
	Fs                    *files.ScopedFs `json:"-" yaml:"-"`
	Rules                 []rules.Rule    `json:"rules"`
	HideDotfiles          bool            `json:"hideDotfiles"`
	DateFormat            bool            `json:"dateFormat"`
	AceEditorTheme        string          `json:"aceEditorTheme"`
	// SystemUID and SystemGID, when non-zero, cause every file and directory
	// created on behalf of this user to be chown'd to the specified OS
	// uid:gid immediately after creation. Requires filebrowser to run as
	// root (or with CAP_CHOWN).
	SystemUID int `json:"systemUID"`
	SystemGID int `json:"systemGID"`
}

// GetRules implements rules.Provider.
func (u *User) GetRules() []rules.Rule {
	return u.Rules
}

var checkableFields = []string{
	"Username",
	"Password",
	"Scope",
	"ViewMode",
	"Commands",
	"Sorting",
	"Rules",
}

// Clean cleans up a user and verifies if all its fields
// are alright to be saved.
func (u *User) Clean(baseScope string, fields ...string) error {
	if len(fields) == 0 {
		fields = checkableFields
	}

	for _, field := range fields {
		switch field {
		case "Username":
			if u.Username == "" {
				return fberrors.ErrEmptyUsername
			}
		case "Password":
			if u.Password == "" {
				return fberrors.ErrEmptyPassword
			}
		case "ViewMode":
			if u.ViewMode == "" {
				u.ViewMode = ListViewMode
			}
		case "Commands":
			if u.Commands == nil {
				u.Commands = []string{}
			}
		case "Sorting":
			if u.Sorting.By == "" {
				u.Sorting.By = "name"
			}
		case "Rules":
			if u.Rules == nil {
				u.Rules = []rules.Rule{}
			}
		}
	}

	if u.Fs == nil {
		scope := u.Scope
		scope = filepath.Join(baseScope, filepath.Join("/", scope))
		u.Fs = files.NewScopedFs(afero.NewOsFs(), scope)
	}

	return nil
}

// FullPath gets the full path for a user's relative path.
func (u *User) FullPath(path string) string {
	return afero.FullBaseFsPath(u.Fs.Base(), path)
}

// Ownership returns the fileutils.Ownership value for this user.
// When SystemUID/SystemGID are explicitly set (non-zero), those values are
// used directly. Otherwise it attempts to resolve the user's Username against
// the OS user database (/etc/passwd on Linux). If the lookup fails the zero
// Ownership is returned (no chown).
//
// Results are cached after the first successful lookup to avoid repeated
// /etc/passwd reads on every file operation.
func (u *User) Ownership() fileutils.Ownership {
	if u.SystemUID != 0 || u.SystemGID != 0 {
		return fileutils.Ownership{UID: u.SystemUID, GID: u.SystemGID}
	}

	// Auto-resolve from OS user database.
	osUser, err := user.Lookup(u.Username)
	if err != nil {
		log.Printf("[ownership] user %q: os lookup failed: %v", u.Username, err)
		return fileutils.Ownership{}
	}

	uid, err := strconv.Atoi(osUser.Uid)
	if err != nil {
		log.Printf("[ownership] user %q: bad UID %q: %v", u.Username, osUser.Uid, err)
		return fileutils.Ownership{}
	}
	gid, err := strconv.Atoi(osUser.Gid)
	if err != nil {
		log.Printf("[ownership] user %q: bad GID %q: %v", u.Username, osUser.Gid, err)
		return fileutils.Ownership{}
	}

	// Cache the resolved values so subsequent calls don't hit /etc/passwd.
	u.SystemUID = uid
	u.SystemGID = gid

	return fileutils.Ownership{UID: uid, GID: gid}
}

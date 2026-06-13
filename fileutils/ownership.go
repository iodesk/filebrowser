package fileutils

import (
	"log"
	"os"
)

// Ownership carries the OS uid/gid that newly created files and directories
// should be assigned to. A zero value (both fields 0) means "no chown" —
// files are left with whatever owner the process inherits from the OS.
type Ownership struct {
	UID int
	GID int
}

// IsSet reports whether an explicit ownership has been configured.
// UID 0 and GID 0 together mean root:root on Linux, which is also the
// conventional "not set" value here — callers that genuinely need root
// ownership should be running as root and will get it for free without
// needing this feature.
func (o Ownership) IsSet() bool {
	return o.UID != 0 || o.GID != 0
}

// Chown calls os.Lchown on path when ownership is set. It uses Lchown so
// that symlinks themselves are re-owned rather than their targets, which
// keeps the scope-escape guards intact.
func (o Ownership) Chown(path string) error {
	if !o.IsSet() {
		return nil
	}
	if err := os.Lchown(path, o.UID, o.GID); err != nil {
		log.Printf("[chown] FAILED %s uid=%d gid=%d: %v", path, o.UID, o.GID, err)
		return err
	}
	return nil
}

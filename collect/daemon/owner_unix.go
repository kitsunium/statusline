//go:build unix

package daemon

import (
	"os"
	"syscall"
)

// ownedByMe compares the directory's owner with this process's user.
func ownedByMe(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}

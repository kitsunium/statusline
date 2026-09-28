package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

// dirPerm is the only mode an instance directory may have.
const dirPerm os.FileMode = 0o700

// privateDir creates the instance directory and its per-user parent 0700,
// and refuses one that is a link or that another user owns.
func privateDir(dir string) error {
	for _, d := range []string{filepath.Dir(dir), dir} {
		if err := os.MkdirAll(d, dirPerm); err != nil {
			return err
		}
		info, err := os.Lstat(d)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s: not a directory", d)
		}
		if !ownedByMe(info) {
			return fmt.Errorf("%s: owned by another user", d)
		}
		if info.Mode().Perm() != dirPerm {
			if err := os.Chmod(d, dirPerm); err != nil {
				return err
			}
		}
	}
	return nil
}

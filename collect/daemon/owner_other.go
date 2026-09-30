//go:build !unix

package daemon

import "os"

// ownedByMe cannot be answered from a FileInfo off Unix; the directory lives
// under the user's own profile there.
func ownedByMe(os.FileInfo) bool { return true }

//go:build !windows

package daemon

import "syscall"

// terminate is the signal the framework's main stops the app on.
var terminate = syscall.SIGTERM

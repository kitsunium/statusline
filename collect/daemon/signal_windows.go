//go:build windows

package daemon

import "os"

// terminate stops the app on Windows, where SIGTERM cannot be sent.
var terminate = os.Interrupt

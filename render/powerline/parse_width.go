// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package powerline

import (
	"strconv"
	"strings"
)

// parseWidth: the host gives the status line its room in COLUMNS; the
// process's own stdout is a pipe, and /dev/tty would report the whole window.
func parseWidth(value string) int {
	width, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || width <= 0 || width > maxWidth {
		return defaultWidth
	}
	return width
}

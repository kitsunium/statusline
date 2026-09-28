// Package renderer provides status line rendering.
package renderer

import (
	"os"
	"strings"
)

// Path display constants.
const (
	// pathEllipsis is the ellipsis used for truncated paths.
	pathEllipsis string = "..."
	// pathSeparator is the path separator character.
	pathSeparator string = "/"
	// defaultMaxPath is the default maximum path length.
	defaultMaxPath int = 30
)

// TruncatePath shortens a path by removing leading segments.
//
// Params:
//   - path: the full path to truncate
//   - maxLen: maximum allowed length (0 uses default)
//
// Returns:
//   - string: truncated path with ellipsis prefix if shortened
func TruncatePath(path string, maxLen int) string {
	// Use default if maxLen is zero or negative
	if maxLen <= 0 {
		maxLen = defaultMaxPath
	}

	// Replace home directory with tilde
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}

	// Return as-is if already short enough
	if len(path) <= maxLen {
		// Return original path
		return path
	}

	// Split path into segments
	segments := strings.Split(path, pathSeparator)
	// Remove leading empty segment if path starts with /
	if len(segments) > 0 && segments[0] == "" {
		segments = segments[1:]
	}

	// Try removing segments from the left until it fits
	for len(segments) > 1 {
		// Remove first segment
		segments = segments[1:]
		candidate := pathEllipsis + pathSeparator + strings.Join(segments, pathSeparator)
		// Return if it fits
		if len(candidate) <= maxLen {
			// Return truncated path
			return candidate
		}
	}

	// Return ellipsis + last segment if even single segment is too long
	if len(segments) == 1 {
		// Return minimal truncated path
		return pathEllipsis + pathSeparator + segments[0]
	}
	// Fallback: return original path
	return path
}

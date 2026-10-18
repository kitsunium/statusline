// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package powerline

import "github.com/kitsunium/statusline/render/powerline/internal/renderer"

// condensed asks the renderer which segments gave way.
func condensed(frame Frame) []string {
	return renderer.Condensed(toData(frame))
}

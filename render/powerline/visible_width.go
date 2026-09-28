// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package powerline

import "github.com/kitsunium/statusline/render/powerline/internal/renderer"

func visibleWidth(s string) int {
	return renderer.VisibleWidth(s)
}

// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package powerline

import "github.com/kitsunium/statusline/render/powerline/internal/renderer"

func visibleWidth(s string) int {
	return renderer.VisibleWidth(s)
}

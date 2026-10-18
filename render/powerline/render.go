// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package powerline

import "github.com/kitsunium/statusline/render/powerline/internal/renderer"

// render converts the frame to the renderer's vocabulary and draws it.
func render(frame Frame) string {
	return renderer.NewPowerline().Render(toData(frame))
}

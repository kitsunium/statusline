// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import "runtime/debug"

// version is set at release time:
// -ldflags "-X github.com/kitsunium/statusline/ipc.version=v1.2.3".
var version string

// buildVersion falls back on the module version go install records; a
// local build is "(devel)", a development build, which never updates.
func buildVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return ""
	}
	return info.Main.Version
}

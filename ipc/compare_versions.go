// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

// compareVersions: dev < any release; numbers first; a release without a
// pre-release suffix is newer than one with; suffixes compare as strings.
func compareVersions(a, b string) int {
	va, vb := parseVersion(a), parseVersion(b)
	switch {
	case va.dev && vb.dev:
		return 0
	case va.dev:
		return -1
	case vb.dev:
		return 1
	}
	for i := range va.parts {
		if va.parts[i] != vb.parts[i] {
			if va.parts[i] < vb.parts[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case va.pre == vb.pre:
		return 0
	case va.pre == "":
		return 1
	case vb.pre == "":
		return -1
	case va.pre < vb.pre:
		return -1
	default:
		return 1
	}
}

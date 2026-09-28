// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package powerline

// iconsFromEnv: every icon is on unless its switch says false, 0 or no.
func iconsFromEnv(env map[string]string) Icons {
	icons := Icons{OS: true, Path: true, Git: true, Model: true}
	for name, dst := range map[string]*bool{
		"STATUSLINE_ICON_OS":    &icons.OS,
		"STATUSLINE_ICON_PATH":  &icons.Path,
		"STATUSLINE_ICON_GIT":   &icons.Git,
		"STATUSLINE_ICON_MODEL": &icons.Model,
	} {
		if val := env[name]; val != "" {
			*dst = parseBool(val)
		}
	}
	return icons
}

package snapshot

import "strings"

// pluginToolPrefix opens the tool-name key of a plugin's server:
// plugin_<plugin>_<server>.
const pluginToolPrefix string = "plugin_"

// indexOfKey finds the server a tool-name key designates: an exact match
// (plain, or plugin and server both matching) wins; otherwise the first
// server whose bare name matches.
func indexOfKey(s []MCPServer, key string) int {
	bare := bareName(key)
	fallback := -1
	for i, srv := range s {
		name := toolKey(srv.Name)
		if name == key || (srv.Plugin != "" && pluginToolPrefix+toolKey(srv.Plugin)+"_"+name == key) {
			return i
		}
		if fallback < 0 && (name == bare || srv.Name == bare) {
			fallback = i
		}
	}
	return fallback
}

// bareName strips the plugin prefix of a key: plugin_<plugin>_<server>.
func bareName(key string) string {
	rest, isPlugin := strings.CutPrefix(key, pluginToolPrefix)
	if !isPlugin {
		return key
	}
	_, server, found := strings.Cut(rest, "_")
	if !found || server == "" {
		return key
	}
	return server
}

func (l TaskList) count(status string) int {
	n := 0
	for _, item := range l.Items {
		if item.Status == status {
			n++
		}
	}
	return n
}

// Component states as published by the status page.
const (
	componentDegraded string = "degraded_performance"
	componentPartial  string = "partial_outage"
	componentMajor    string = "major_outage"
	// degradedForDown is how many impaired components make the service down.
	degradedForDown int = 2
)

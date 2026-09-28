package snapshot

import "strings"

// isInRepo: a branch name is only found inside a work tree.
func (s GitStatus) isInRepo() bool { return s.Branch != "" }

func (c Changes) hasChanges() bool { return c.Added > 0 || c.Removed > 0 }

func (c Changes) hasAdded() bool { return c.Added > 0 }

func (c Changes) hasRemoved() bool { return c.Removed > 0 }

// withSource tags in place: each source slice is built fresh for one call.
func (s MCPServers) withSource(src MCPSource) MCPServers {
	for i := range s {
		s[i].Source = src
	}
	return s
}

// pluginToolPrefix opens the tool-name key of a plugin's server:
// plugin_<plugin>_<server>.
const pluginToolPrefix string = "plugin_"

// withBusy lights the servers whose tool-name key is being called. A key no
// configuration names is still a server in use: it is appended, lit,
// without a scope, and counted enabled.
func (s MCPServers) withBusy(keys []string) MCPServers {
	out := make(MCPServers, len(s), len(s)+len(keys))
	copy(out, s)
	for _, key := range keys {
		idx := out.indexOfKey(key)
		if idx < 0 {
			out = append(out, MCPServer{Name: bareName(key), Enabled: true})
			idx = len(out) - 1
		}
		out[idx].Busy = true
	}
	return out
}

// indexOfKey finds the server a tool-name key designates: an exact match
// (plain, or plugin and server both matching) wins; otherwise the first
// server whose bare name matches.
func (s MCPServers) indexOfKey(key string) int {
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

// toolKey replaces every character outside [A-Za-z0-9_-] by an underscore.
func toolKey(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
}

func (l TaskList) total() int { return len(l.Items) }

func (l TaskList) count(status string) int {
	n := 0
	for _, item := range l.Items {
		if item.Status == status {
			n++
		}
	}
	return n
}

func (l TaskList) done() int { return l.count(TaskCompleted) }

func (l TaskList) active() int { return l.count(TaskInProgress) }

func (l TaskList) waiting() int { return l.count(TaskWaiting) }

// headline picks the task under way, else the first one waiting on the
// user, else the next one to start.
func (l TaskList) headline() (string, string) {
	for _, status := range []string{TaskInProgress, TaskWaiting, TaskPending} {
		for _, item := range l.Items {
			if item.Status == status {
				return item.Subject, status
			}
		}
	}
	return "", ""
}

func (l TaskList) current() string {
	for _, item := range l.Items {
		if item.Status == TaskInProgress {
			return item.Subject
		}
	}
	return ""
}

func (l TaskList) isActive() bool { return l.total() > 0 && l.done() < l.total() }

func (b TaskBoard) isEmpty() bool { return len(b.Epics) == 0 && b.Unattributed == 0 }

// Component states as published by the status page.
const (
	componentDegraded string = "degraded_performance"
	componentPartial  string = "partial_outage"
	componentMajor    string = "major_outage"
	// degradedForDown is how many impaired components make the service down.
	degradedForDown int = 2
)

// classifyHealth: one impaired component is degraded, two or one major
// outage is down; maintenance does not count; nothing to judge is unknown.
func classifyHealth(states []string) Health {
	if len(states) == 0 {
		return HealthUnknown
	}
	degraded := 0
	for _, state := range states {
		switch state {
		case componentMajor:
			return HealthDown
		case componentDegraded, componentPartial:
			degraded++
		}
	}
	switch {
	case degraded >= degradedForDown:
		return HealthDown
	case degraded == 1:
		return HealthDegraded
	default:
		return HealthOK
	}
}

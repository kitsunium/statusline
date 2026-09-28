// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

func (l TaskList) active() int { return l.count(TaskInProgress) }

func (l TaskList) current() string {
	for _, item := range l.Items {
		if item.Status == TaskInProgress {
			return item.Subject
		}
	}
	return ""
}

func (l TaskList) done() int { return l.count(TaskCompleted) }

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

func (l TaskList) isActive() bool { return l.total() > 0 && l.done() < l.total() }

func (l TaskList) total() int { return len(l.Items) }

func (l TaskList) waiting() int { return l.count(TaskWaiting) }

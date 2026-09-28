package snapshot

import (
	"testing"

	"pgregory.net/rapid"
)

// seeded logs the design seed: rapid v1.2 takes its own from -rapid.seed.
func seeded(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
}

var statuses = []string{TaskPending, TaskInProgress, TaskWaiting, TaskCompleted, "odd"}

func listGen() *rapid.Generator[TaskList] {
	return rapid.Custom(func(t *rapid.T) TaskList {
		var l TaskList
		for i, s := range rapid.SliceOfN(rapid.SampledFrom(statuses), 0, 12).Draw(t, "statuses") {
			l.Items = append(l.Items, TaskItem{ID: string(rune('a' + i)), Subject: s + "-subject", Status: s})
		}
		return l
	})
}

func count(l TaskList, status string) int {
	n := 0
	for _, it := range l.Items {
		if it.Status == status {
			n++
		}
	}
	return n
}

func changesGen() *rapid.Generator[Changes] {
	return rapid.Custom(func(t *rapid.T) Changes {
		return Changes{Added: rapid.IntRange(-3, 50).Draw(t, "added"), Removed: rapid.IntRange(-3, 50).Draw(t, "removed")}
	})
}

func propertyChangesHasAddedPositive(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		c := changesGen().Draw(t, "c")
		if c.HasAdded() != (c.Added > 0) {
			t.Fatal("HasAdded")
		}
	})
}

func propertyChangesHasRemovedPositive(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		c := changesGen().Draw(t, "c")
		if c.HasRemoved() != (c.Removed > 0) {
			t.Fatal("HasRemoved")
		}
	})
}

func propertyChangesHasChangesEitherSide(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		c := changesGen().Draw(t, "c")
		if c.HasChanges() != (c.HasAdded() || c.HasRemoved()) {
			t.Fatal("HasChanges")
		}
	})
}

func propertyGitStatusIsInRepoMatchesBranch(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		g := GitStatus{Branch: rapid.StringMatching(`[a-z/]{0,5}`).Draw(t, "b"), Modified: rapid.IntRange(0, 5).Draw(t, "m")}
		if g.IsInRepo() != (g.Branch != "") {
			t.Fatal("IsInRepo")
		}
	})
}

func propertyTaskBoardIsEmptyMatchesContent(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		b := TaskBoard{Unattributed: rapid.IntRange(0, 2).Draw(t, "u")}
		for i := rapid.IntRange(0, 2).Draw(t, "epics"); i > 0; i-- {
			b.Epics = append(b.Epics, Epic{ID: i})
		}
		if b.IsEmpty() != (len(b.Epics) == 0 && b.Unattributed == 0) {
			t.Fatal("IsEmpty")
		}
	})
}

func propertyTaskListTotalCountsItems(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		if l.Total() != len(l.Items) {
			t.Fatal("Total")
		}
	})
}

func propertyTaskListActiveCountsInProgress(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		if l.Active() != count(l, TaskInProgress) {
			t.Fatal("Active")
		}
	})
}

func propertyTaskListWaitingCountsWaiting(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		if l.Waiting() != count(l, TaskWaiting) {
			t.Fatal("Waiting")
		}
	})
}

func propertyTaskListHeadlineNeverCompleted(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		subject, status := l.Headline()
		if status == TaskCompleted || (subject == "") != (status == "") {
			t.Fatalf("Headline() = %q, %q", subject, status)
		}
		if l.Active() > 0 && status != TaskInProgress {
			t.Fatal("a task under way must headline the list")
		}
	})
}

func propertyTaskListCurrentInProgressOnly(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		if cur := l.Current(); (cur != "") != (l.Active() > 0) {
			t.Fatalf("Current() = %q with %d in progress", cur, l.Active())
		}
	})
}

func propertyTaskListIsActiveMatchesCounts(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := listGen().Draw(t, "l")
		if l.IsActive() != (l.Total() > 0 && l.Done() < l.Total()) {
			t.Fatal("IsActive")
		}
	})
}

func propertyWithSourceTagsEvery(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		servers := serversGen().Draw(t, "servers")
		src := rapid.SampledFrom([]string{MCPSourceCLI, MCPSourceUser, MCPSourcePlugin}).Draw(t, "src")
		got := WithSource(servers, src)
		if len(got) != len(servers) {
			t.Fatal("WithSource changed the length")
		}
		for _, s := range got {
			if s.Source != src {
				t.Fatalf("server %q not tagged", s.Name)
			}
		}
	})
}

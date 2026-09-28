package snapshot_test

import (
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestTaskList(t *testing.T) {
	list := snapshot.TaskList{Items: []snapshot.TaskItem{
		{ID: "1", Subject: "a", Status: snapshot.TaskCompleted},
		{ID: "2", Subject: "b", Status: snapshot.TaskInProgress},
		{ID: "3", Subject: "c", Status: snapshot.TaskInProgress},
		{ID: "4", Subject: "d", Status: snapshot.TaskPending},
	}}
	if list.Total() != 4 || list.Done() != 1 || list.Active() != 2 {
		t.Errorf("Total/Done/Active = %d/%d/%d, want 4/1/2", list.Total(), list.Done(), list.Active())
	}
	if got := list.Current(); got != "b" {
		t.Errorf("Current() = %q, want the earliest started task", got)
	}
	if !list.IsActive() {
		t.Error("IsActive() = false with open tasks")
	}
	finished := snapshot.TaskList{Items: []snapshot.TaskItem{{ID: "1", Status: snapshot.TaskCompleted}}}
	if finished.IsActive() || (snapshot.TaskList{}).IsActive() {
		t.Error("a finished or empty list must not be active")
	}
}

func TestTaskListHeadlineAndWaiting(t *testing.T) {
	list := snapshot.TaskList{Items: []snapshot.TaskItem{
		{ID: "1", Subject: "next", Status: snapshot.TaskPending},
		{ID: "2", Subject: "blocked", Status: snapshot.TaskWaiting},
	}}
	if got := list.Waiting(); got != 1 {
		t.Errorf("Waiting() = %d, want 1", got)
	}
	if subject, status := list.Headline(); subject != "blocked" || status != snapshot.TaskWaiting {
		t.Errorf("Headline() = %q, %q, want the waiting task", subject, status)
	}
	if subject, status := (snapshot.TaskList{}).Headline(); subject != "" || status != "" {
		t.Errorf("empty list Headline() = %q, %q", subject, status)
	}
	if !list.IsActive() {
		t.Error("a list with waiting tasks is still active")
	}
}

func TestTaskBoardIsEmpty(t *testing.T) {
	if !(snapshot.TaskBoard{}).IsEmpty() {
		t.Error("a zero board is not empty")
	}
	if (snapshot.TaskBoard{Unattributed: 1}).IsEmpty() || (snapshot.TaskBoard{Epics: []snapshot.Epic{{ID: snapshot.NoEpic}}}).IsEmpty() {
		t.Error("a board with a subagent or an epic reads as empty")
	}
}

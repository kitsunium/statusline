package collector

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

var t0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// clock is a settable clock.
type clock struct{ now time.Time }

func (c *clock) Now(context.Context) (time.Time, error) { return c.now, nil }

// calls counts port calls by name, safely: collection is concurrent.
type calls struct {
	mu sync.Mutex
	n  map[string]int
}

func (c *calls) hit(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n == nil {
		c.n = map[string]int{}
	}
	c.n[name]++
}

func (c *calls) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, v := range c.n {
		n += v
	}
	return n
}

// sources answers every collect port.
type sources struct {
	calls
	gitDir string
}

func (s *sources) Dir(_ context.Context, transcript, fallback string) (string, error) {
	s.hit("workdir")
	if transcript == "" {
		return fallback, nil
	}
	return "/work/elsewhere", nil
}

func (s *sources) Lookup(context.Context, string) (state.HostSession, bool, error) {
	s.hit("sessions")
	return state.HostSession{PID: 42, Working: true}, true, nil
}

func (s *sources) Status(_ context.Context, dir string) (snapshot.GitStatus, error) {
	s.hit("git")
	s.mu.Lock()
	s.gitDir = dir
	s.mu.Unlock()
	return snapshot.GitStatus{Branch: "main"}, nil
}

func (s *sources) DiffStats(context.Context, string) (snapshot.Changes, error) {
	s.hit("diff")
	return snapshot.Changes{Added: 1}, nil
}

func (s *sources) Servers(_ context.Context, _ string, pid int) ([]snapshot.MCPServer, error) {
	s.hit("mcp")
	if pid != 42 {
		return nil, nil
	}
	return []snapshot.MCPServer{{Name: "github", Enabled: true}}, nil
}

func (s *sources) Busy(context.Context, string, string, time.Time) ([]string, error) {
	s.hit("calls")
	return []string{"github"}, nil
}

func (s *sources) Board(context.Context, string, string, time.Time) (snapshot.TaskBoard, error) {
	s.hit("tasks")
	return snapshot.TaskBoard{Unattributed: 1}, nil
}

func (s *sources) Info(context.Context) (snapshot.System, error) {
	s.hit("system")
	return snapshot.System{OS: snapshot.OSDarwin}, nil
}

// memCache is a SessionCache and a SnapshotCache in memory.
type memCache struct {
	mu     sync.Mutex
	snaps  map[ipc.Key]snapshot.Snapshot
	at     map[ipc.Key]time.Time
	saved  []ipc.Key
	touchd int
}

func newMemCache() *memCache {
	return &memCache{snaps: map[ipc.Key]snapshot.Snapshot{}, at: map[ipc.Key]time.Time{}}
}

func (m *memCache) Get(_ context.Context, key ipc.Key, now time.Time) (snapshot.Snapshot, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	at, ok := m.at[key]
	if !ok || now.Sub(at) >= state.Freshness {
		return snapshot.Snapshot{}, false, nil
	}
	return m.snaps[key], true, nil
}

func (m *memCache) Put(_ context.Context, key ipc.Key, snap snapshot.Snapshot, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snaps[key], m.at[key] = snap, now
	return nil
}

func (m *memCache) Touch(context.Context, ipc.Key, time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touchd++
	return nil
}

func (m *memCache) Evict(context.Context, time.Time) (int, error)  { return 0, nil }
func (m *memCache) Len(context.Context) (int, error)               { return len(m.snaps), nil }
func (m *memCache) LastRequest(context.Context) (time.Time, error) { return time.Time{}, nil }
func (m *memCache) SaveSnapshot(_ context.Context, key ipc.Key, _ snapshot.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, key)
	return nil
}

// network is a NetworkStore in memory, counting its loads and saves.
type network struct {
	calls
	n state.Network
}

func (s *network) LoadNetwork(context.Context) (state.Network, error) {
	s.hit("load")
	return s.n, nil
}

func (s *network) SaveNetwork(_ context.Context, n state.Network) error {
	s.hit("save")
	s.n = n
	return nil
}

// endpoints are the token store, the usage API and the status page.
type endpoints struct {
	calls
	token    string
	usage    quota.Set
	usageErr error
	health   int
	pageErr  error
}

func (e *endpoints) Token(context.Context) (string, error) {
	e.hit("token")
	if e.token == "" {
		return "", errors.New("no token")
	}
	return e.token, nil
}

func (e *endpoints) Fetch(_ context.Context, token string) (quota.Set, error) {
	e.hit("usage")
	return e.usage, e.usageErr
}

// page is the status page side of endpoints.
type page struct{ e *endpoints }

func (p page) Fetch(context.Context) (int, error) {
	p.e.hit("health")
	return p.e.health, p.e.pageErr
}

func weekly(p int) quota.Set {
	return quota.Set{Weekly: quota.NewLimit(quota.KindWeekly, "weekly", p, t0.Add(time.Hour), quota.WeeklyWindow, quota.SourceAPI)}
}

// releases is an update host: a store, a release source.
type releases struct {
	calls
	stored   state.Update
	latest   string
	probeErr error
}

func (r *releases) LoadUpdate(context.Context) (state.Update, error) {
	r.hit("load")
	return r.stored, nil
}

func (r *releases) SaveUpdate(_ context.Context, u state.Update) error {
	r.hit("save")
	r.stored = u
	return nil
}

func (r *releases) Latest(context.Context) (state.Release, error) {
	r.hit("latest")
	return state.Release{Version: r.latest}, nil
}

func (r *releases) Install(context.Context, state.Release) error {
	r.hit("install")
	return nil
}

func (r *releases) Probe(context.Context) error {
	r.hit("probe")
	return r.probeErr
}

func (r *releases) Rollback(context.Context) error {
	r.hit("rollback")
	return nil
}

// stateWithScoped is network bookkeeping with a scoped quota and a
// degraded status page, both fresh at t0.
func stateWithScoped() state.Network {
	set := quota.Set{Scoped: []quota.Limit{quota.NewLimit(quota.KindScoped, "opus", 5, t0.Add(time.Hour), quota.WeeklyWindow, quota.SourceAPI)}}
	return state.Network{}.RecordUsage(set, t0).RecordHealth(snapshot.HealthDegraded, t0)
}

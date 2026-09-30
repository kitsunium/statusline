package usageapi

import (
	"encoding/json"
	"testing"

	"github.com/kitsunium/statusline/quota"
)

// noWeeklyPlan is the payload of an account whose plan carries no plan-wide
// weekly cap. Every legacy top-level bucket is null and the only weekly signal
// lives in limits[], scoped to a model family. Parsing five_hour/seven_day
// alone loses the weekly quota entirely on such an account.
const noWeeklyPlan = `{
  "five_hour": {"utilization": 15.0, "resets_at": "2026-09-18T22:30:00.555838+00:00"},
  "seven_day": null,
  "seven_day_opus": null,
  "nimbus_quill": {"utilization": 0.0, "resets_at": null},
  "extra_usage": {"is_enabled": false, "monthly_limit": 100, "used_credits": 0.0,
                  "utilization": 0.0, "currency": "USD", "decimal_places": 2},
  "limits": [
    {"kind": "session", "group": "session", "percent": 15, "severity": "normal",
     "resets_at": "2026-09-18T22:30:00.555838+00:00", "scope": null, "is_active": true},
    {"kind": "weekly_scoped", "group": "weekly", "percent": 13, "severity": "normal",
     "resets_at": "2026-09-19T15:59:59.556013+00:00",
     "scope": {"model": {"id": null, "display_name": "Fable"}}, "is_active": false}
  ]
}`

// weeklyPlan is the payload of an account that does carry a plan-wide weekly
// cap, expressed both in limits[] and in the legacy buckets.
const weeklyPlan = `{
  "five_hour": {"utilization": 42.0, "resets_at": "2026-09-18T22:30:00Z"},
  "seven_day": {"utilization": 61.0, "resets_at": "2026-09-21T15:59:59Z"},
  "extra_usage": {"is_enabled": true, "monthly_limit": 5000, "used_credits": 1250.0,
                  "utilization": 25.0, "currency": "USD", "decimal_places": 2},
  "limits": [
    {"kind": "session", "group": "session", "percent": 42,
     "resets_at": "2026-09-18T22:30:00Z", "is_active": true},
    {"kind": "weekly", "group": "weekly", "percent": 61,
     "resets_at": "2026-09-21T15:59:59Z", "scope": null, "is_active": true}
  ]
}`

// legacyOnly has no limits array at all, as older API responses did.
const legacyOnly = `{
  "five_hour": {"utilization": 30.0, "resets_at": "2026-09-18T22:30:00Z"},
  "seven_day": {"utilization": 55.0, "resets_at": "2026-09-21T15:59:59Z"}
}`

func parse(t *testing.T, raw string) quota.Set {
	t.Helper()
	var resp usageResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}
	return resp.toLimitSet()
}

func TestToLimitSet_NoWeeklyPlan(t *testing.T) {
	set := parse(t, noWeeklyPlan)

	// The session quota must survive the null-riddled payload
	if !set.Session.IsValid() || set.Session.Percent != 15 {
		t.Errorf("Session = %+v, want a valid 15%% quota", set.Session)
	}
	// A plan with no weekly cap must expose none, not a zero-percent one
	if set.Weekly.IsValid() {
		t.Errorf("Weekly = %+v, want absent on a plan without a weekly cap", set.Weekly)
	}
	// The weekly signal that does exist lives in limits[] and must be kept
	if len(set.Scoped) != 1 {
		t.Fatalf("Scoped = %d entries, want 1", len(set.Scoped))
	}
	if set.Scoped[0].Label != "fable" || set.Scoped[0].Percent != 13 {
		t.Errorf("Scoped[0] = %+v, want fable at 13%%", set.Scoped[0])
	}
	// is_active false must be preserved, not silently dropped
	if set.Scoped[0].Active {
		t.Error("Scoped[0].Active = true, want false")
	}
	// Credits that are switched off must not render
	if set.Extra.IsValid() {
		t.Errorf("Extra = %+v, want invalid when disabled", set.Extra)
	}
}

func TestToLimitSet_WeeklyPlan(t *testing.T) {
	set := parse(t, weeklyPlan)

	// Both quotas must be present on a plan that exposes both
	if !set.Session.IsValid() || set.Session.Percent != 42 {
		t.Errorf("Session = %+v, want a valid 42%% quota", set.Session)
	}
	if !set.Weekly.IsValid() || set.Weekly.Percent != 61 {
		t.Errorf("Weekly = %+v, want a valid 61%% quota", set.Weekly)
	}
	// Window durations drive the pace arithmetic and must not be swapped
	if set.Session.Window != quota.SessionWindow {
		t.Errorf("Session.Window = %v, want %v", set.Session.Window, quota.SessionWindow)
	}
	if set.Weekly.Window != quota.WeeklyWindow {
		t.Errorf("Weekly.Window = %v, want %v", set.Weekly.Window, quota.WeeklyWindow)
	}
	// An enabled credit balance must render with its amounts intact
	if !set.Extra.IsValid() || set.Extra.UsedMinor != 1250 || set.Extra.LimitMinor != 5000 {
		t.Errorf("Extra = %+v, want 1250/5000 enabled", set.Extra)
	}
}

func TestToLimitSet_LegacyOnly(t *testing.T) {
	set := parse(t, legacyOnly)

	// Without a limits array the legacy buckets must still be honoured
	if !set.Session.IsValid() || set.Session.Percent != 30 {
		t.Errorf("Session = %+v, want a valid 30%% quota from the legacy bucket", set.Session)
	}
	if !set.Weekly.IsValid() || set.Weekly.Percent != 55 {
		t.Errorf("Weekly = %+v, want a valid 55%% quota from the legacy bucket", set.Weekly)
	}
}

func TestToLimitSet_EmptyPayload(t *testing.T) {
	set := parse(t, `{}`)

	// An empty payload must yield no quota at all rather than zero-percent bars
	if set.HasTimed() {
		t.Errorf("Timed() = %+v, want no quota from an empty payload", set.Timed())
	}
	if set.Extra.IsValid() {
		t.Error("Extra must be invalid on an empty payload")
	}
}

func TestScopeLabel(t *testing.T) {
	tests := []struct {
		name  string
		scope *apiScope
		want  string
	}{
		{name: "display name wins", scope: &apiScope{Model: &apiScopeModel{ID: "id", DisplayName: "Opus"}}, want: "opus"},
		{name: "falls back to id", scope: &apiScope{Model: &apiScopeModel{ID: "claude-Fable-5"}}, want: "claude-fable-5"},
		{name: "nil scope", scope: nil, want: "model"},
		{name: "nil model", scope: &apiScope{}, want: "model"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scopeLabel(tt.scope); got != tt.want {
				t.Errorf("scopeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

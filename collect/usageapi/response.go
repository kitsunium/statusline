// Package usage provides the Anthropic API usage adapter.
package usageapi

import (
	"strings"

	"github.com/kitsunium/statusline/quota"
)

// API limit kinds and groups as returned by the usage endpoint.
const (
	// groupSession marks the rolling five-hour bucket.
	groupSession string = "session"
	// groupWeekly marks the rolling seven-day buckets.
	groupWeekly string = "weekly"
	// kindWeeklyScoped marks a seven-day bucket restricted to one model.
	kindWeeklyScoped string = "weekly_scoped"
	// centsPerUnit converts minor currency units to a display amount.
	centsPerUnit int = 100
)

// usageResponse is the payload of the OAuth usage endpoint.
// The legacy top-level buckets are null on plans that do not expose them,
// so limits is the authoritative, plan-agnostic list and the buckets are
// only a fallback for older API responses.
type usageResponse struct {
	Limits     []apiLimit   `json:"limits"`
	FiveHour   *usagePeriod `json:"five_hour"`
	SevenDay   *usagePeriod `json:"seven_day"`
	ExtraUsage *extraUsage  `json:"extra_usage"`
}

// apiLimit is one entry of the limits array.
// kind and group describe the bucket; scope narrows it to a model family.
type apiLimit struct {
	Kind     string          `json:"kind"`
	Group    string          `json:"group"`
	Percent  *float64        `json:"percent"`
	Severity string          `json:"severity"`
	ResetsAt quota.Timestamp `json:"resets_at"`
	Scope    *apiScope       `json:"scope"`
	IsActive *bool           `json:"is_active"`
}

// apiScope narrows a limit to a model family or a surface.
type apiScope struct {
	Model *apiScopeModel `json:"model"`
}

// apiScopeModel names the model family a scoped limit applies to.
type apiScopeModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// usagePeriod is a legacy top-level bucket.
type usagePeriod struct {
	Utilization float64         `json:"utilization"`
	ResetsAt    quota.Timestamp `json:"resets_at"`
}

// extraUsage is the monthly credit balance.
type extraUsage struct {
	IsEnabled    bool    `json:"is_enabled"`
	MonthlyLimit int     `json:"monthly_limit"`
	UsedCredits  float64 `json:"used_credits"`
	Utilization  float64 `json:"utilization"`
	Currency     string  `json:"currency"`
	Decimals     int     `json:"decimal_places"`
}

// toLimitSet maps the API payload onto the domain limit set.
// Buckets absent from the response stay absent from the set, so a plan
// without a weekly cap never renders a fabricated zero-percent bar.
//
// Returns:
//   - quota.Set: session, weekly, scoped and extra quotas
func (u usageResponse) toLimitSet() quota.Set {
	var set quota.Set

	// The limits array is the plan-agnostic source; walk it first
	for _, limit := range u.Limits {
		// An entry without a percentage carries no usable signal
		if limit.Percent == nil {
			continue
		}
		percent := int(*limit.Percent)
		active := limit.IsActive == nil || *limit.IsActive

		switch {
		// The five-hour bucket, whatever kind spelling the API uses
		case limit.Group == groupSession:
			mapped := quota.NewLimit(quota.KindSession, groupSession, percent, limit.ResetsAt.Time, quota.SessionWindow, quota.SourceAPI)
			mapped.Active = active
			set.Session = mapped

		// A seven-day bucket restricted to one model family
		case limit.Kind == kindWeeklyScoped:
			mapped := quota.NewLimit(quota.KindScoped, scopeLabel(limit.Scope), percent, limit.ResetsAt.Time, quota.WeeklyWindow, quota.SourceAPI)
			mapped.Active = active
			set.Scoped = append(set.Scoped, mapped)

		// The plan-wide seven-day bucket
		case limit.Group == groupWeekly:
			mapped := quota.NewLimit(quota.KindWeekly, groupWeekly, percent, limit.ResetsAt.Time, quota.WeeklyWindow, quota.SourceAPI)
			mapped.Active = active
			set.Weekly = mapped
		}
	}

	// Fall back to the legacy session bucket when limits carried none
	if !set.Session.IsValid() && u.FiveHour != nil {
		set.Session = quota.NewLimit(quota.KindSession, groupSession, int(u.FiveHour.Utilization), u.FiveHour.ResetsAt.Time, quota.SessionWindow, quota.SourceAPI)
	}
	// Fall back to the legacy weekly bucket when limits carried none
	if !set.Weekly.IsValid() && u.SevenDay != nil {
		set.Weekly = quota.NewLimit(quota.KindWeekly, groupWeekly, int(u.SevenDay.Utilization), u.SevenDay.ResetsAt.Time, quota.WeeklyWindow, quota.SourceAPI)
	}

	set.Extra = u.toExtraUsage()
	return set
}

// toExtraUsage maps the credit balance onto the domain value object.
//
// Returns:
//   - quota.Extra: credit balance, zero when absent
func (u usageResponse) toExtraUsage() quota.Extra {
	// An absent block means the account has no credit balance
	if u.ExtraUsage == nil {
		return quota.Extra{}
	}
	return quota.Extra{
		Enabled:    u.ExtraUsage.IsEnabled,
		Percent:    int(u.ExtraUsage.Utilization),
		UsedMinor:  int(u.ExtraUsage.UsedCredits),
		LimitMinor: u.ExtraUsage.MonthlyLimit,
		Currency:   u.ExtraUsage.Currency,
		Exponent:   u.ExtraUsage.Decimals,
		Source:     quota.SourceAPI,
	}
}

// scopeLabel derives a short label from a scoped limit's model scope.
//
// Params:
//   - scope: scope block from the API, possibly nil
//
// Returns:
//   - string: lowercase model label, "model" when unnamed
func scopeLabel(scope *apiScope) string {
	// An unscoped entry gets a neutral label
	if scope == nil || scope.Model == nil {
		return "model"
	}
	// Prefer the human display name
	if scope.Model.DisplayName != "" {
		return strings.ToLower(scope.Model.DisplayName)
	}
	// Fall back to the model identifier
	if scope.Model.ID != "" {
		return strings.ToLower(scope.Model.ID)
	}
	return "model"
}

// Package session reads the host's stdin payload and what it says about the
// session: the model, the effort, the context window, the rate limits.
//
// Exported API of design/domains/render.yaml (render/component/session).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package session

import "github.com/kitsunium/statusline/quota"

// Payload is the JSON the host pipes on stdin.
type Payload struct {
	Model         PayloadModel      `json:"model"`
	Workspace     PayloadWorkspace  `json:"workspace"`
	ContextWindow PayloadContext    `json:"context_window"`
	Cost          PayloadCost       `json:"cost"`
	RateLimits    PayloadRateLimits `json:"rate_limits"`
	Effort        PayloadEffort     `json:"effort"`
	Thinking      PayloadThinking   `json:"thinking"`
	OutputStyle   PayloadStyle      `json:"output_style"`
	SessionName   string            `json:"session_name"`
	Transcript    string            `json:"transcript_path"`
	SessionID     string            `json:"session_id"`
	Version       string            `json:"version"`
	FastMode      bool              `json:"fast_mode"`
	Exceeds200k   bool              `json:"exceeds_200k_tokens"`
}

// PayloadModel names the model.
type PayloadModel struct {
	DisplayName string `json:"display_name"`
	ID          string `json:"id"`
}

// PayloadWorkspace holds the session's directories.
type PayloadWorkspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

// PayloadContext is the context window.
type PayloadContext struct {
	CurrentUsage        PayloadUsage `json:"current_usage"`
	TotalInputTokens    int          `json:"total_input_tokens"`
	TotalOutputTokens   int          `json:"total_output_tokens"`
	ContextWindowSize   int          `json:"context_window_size"`
	UsedPercentage      *float64     `json:"used_percentage"`
	RemainingPercentage *float64     `json:"remaining_percentage"`
}

// PayloadUsage is the live token composition of the context window.
type PayloadUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// PayloadCost is the session's cost and line counts.
type PayloadCost struct {
	TotalCostUSD       float64 `json:"total_cost_usd"`
	TotalDurationMs    int64   `json:"total_duration_ms"`
	TotalAPIDurationMs int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

// PayloadRateLimits are the rate limits the host knows.
type PayloadRateLimits struct {
	FiveHour *PayloadRateLimit `json:"five_hour"`
	SevenDay *PayloadRateLimit `json:"seven_day"`
}

// PayloadRateLimit is one rate limit bucket.
type PayloadRateLimit struct {
	UsedPercentage *float64        `json:"used_percentage"`
	Utilization    *float64        `json:"utilization"`
	ResetsAt       quota.Timestamp `json:"resets_at"`
}

// Percent returns the consumption, false when neither field is present.
func (r *PayloadRateLimit) Percent() (int, bool) { return r.percent() }

// PayloadEffort is the reasoning effort.
type PayloadEffort struct {
	Level string `json:"level"`
}

// PayloadThinking says whether extended thinking is on.
type PayloadThinking struct {
	Enabled bool `json:"enabled"`
}

// PayloadStyle names the output style.
type PayloadStyle struct {
	Name string `json:"name"`
}

// Parse decodes a payload and never fails: invalid JSON is the zero
// Payload, a mistyped field leaves the fields decoded around it.
func Parse(data []byte) Payload { return parse(data) }

// ModelInfo returns the model's name and version.
func (p *Payload) ModelInfo() ModelInfo { return p.modelInfo() }

// WorkingDir returns the session's directory, "~" when absent.
func (p *Payload) WorkingDir() string { return p.workingDir() }

// ContextWindowSize returns the window size, 200000 when absent.
func (p *Payload) ContextWindowSize() int { return p.contextWindowSize() }

// TotalTokens returns input plus output tokens.
func (p *Payload) TotalTokens() int { return p.totalTokens() }

// Progress returns the context usage.
func (p *Payload) Progress() quota.Progress { return p.progress() }

// StdinLimits returns the context window and the rate limits of stdin.
func (p *Payload) StdinLimits() quota.Set { return p.stdinLimits() }

// ModelInfo is a model's name and version.
type ModelInfo struct {
	Name    string
	Version string
}

// FullName returns the name with its version.
func (m ModelInfo) FullName() string { return m.fullName() }

// ShortName returns the name with its bare version number.
func (m ModelInfo) ShortName() string { return m.shortName() }

// Effort levels, cheapest first.
const (
	EffortLow    string = "low"
	EffortMedium string = "medium"
	EffortHigh   string = "high"
	EffortXHigh  string = "xhigh"
	EffortMax    string = "max"
)

// EffortRank returns 1 for the cheapest level up to EffortSteps for the
// highest, false for a level off the scale.
func EffortRank(level string) (int, bool) { return effortRank(level) }

// EffortSteps returns the number of known effort levels.
func EffortSteps() int { return effortSteps() }

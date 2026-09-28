// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package session

import (
	"testing"

	"pgregory.net/rapid"
)

// propertyParseTotal (render/property/payload-parse-total).
func propertyParseTotal(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.OneOf(
			rapid.SliceOf(rapid.Byte()),
			rapid.Map(rapid.StringMatching(`\{"model":\{"display_name":("[A-Za-z0-9 ]{0,12}"|[0-9]+|null)\},"context_window":(\{"used_percentage":-?[0-9]{1,4}\}|"x"|null)\}`), func(s string) []byte { return []byte(s) }),
		).Draw(t, "raw")
		p := Parse(raw)
		// A payload without a model name is drawn as the default model; a name
		// is otherwise split at its first space, as the legacy product did
		if p.Model.DisplayName == "" && p.ModelInfo().Name != "Claude" {
			t.Fatalf("no display name gives %q", p.ModelInfo().Name)
		}
		if pr := p.Progress(); pr.Percent > 100 {
			t.Fatalf("Progress = %d", pr.Percent)
		}
		if p.WorkingDir() == "" {
			t.Fatal("WorkingDir is never empty")
		}
		set := p.StdinLimits()
		if !set.Context.IsValid() {
			t.Fatal("the context window is always present")
		}
	})
}

// propertyEffortRankScale (render/property/effort-rank-scale).
func propertyEffortRankScale(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		level := rapid.OneOf(rapid.SampledFrom([]string{"low", "medium", "high", "xhigh", "max"}), rapid.String()).Draw(t, "level")
		rank, ok := EffortRank(level)
		if ok != (rank >= 1 && rank <= EffortSteps()) {
			t.Fatalf("EffortRank(%q) = %d, %v", level, rank, ok)
		}
	})
}

func TestParseKeepsWhatDecodesAroundAMistypedField(t *testing.T) {
	p := Parse([]byte(`{"model":{"display_name":5},"session_id":"s","workspace":{"current_dir":"/w"}}`))
	if p.SessionID != "s" || p.WorkingDir() != "/w" || p.ModelInfo().Name != "Claude" {
		t.Errorf("Parse() = %+v", p)
	}
	if p := Parse([]byte(`{"session_id":"s"`)); p.SessionID != "" {
		t.Errorf("invalid JSON decoded something: %+v", p)
	}
}

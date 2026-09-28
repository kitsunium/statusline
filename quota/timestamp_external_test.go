package quota_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kitsunium/statusline/quota"
)

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantSec int64
		wantSet bool
	}{
		{name: "unix epoch as sent on stdin", raw: `1789770600`, wantSec: 1789770600, wantSet: true},
		{name: "rfc3339 as returned by the api", raw: `"2026-09-18T22:30:00Z"`, wantSec: 1789770600, wantSet: true},
		{name: "rfc3339 with offset and fraction", raw: `"2026-09-18T22:30:00.555838+00:00"`, wantSec: 1789770600, wantSet: true},
		{name: "json null", raw: `null`, wantSet: false},
		{name: "empty string", raw: `""`, wantSet: false},
		{name: "epoch zero means unset", raw: `0`, wantSet: false},
		{name: "unparseable string", raw: `"not a date"`, wantSet: false},
		{name: "boolean", raw: `true`, wantSet: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts quota.Timestamp
			// Decoding must never fail: an unusable value yields the zero instant
			if err := json.Unmarshal([]byte(tt.raw), &ts); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v, want nil", err)
			}
			if ts.IsZero() == tt.wantSet {
				t.Fatalf("UnmarshalJSON() IsZero = %v, want set = %v", ts.IsZero(), tt.wantSet)
			}
			if tt.wantSet && ts.UTC().Unix() != tt.wantSec {
				t.Errorf("UnmarshalJSON() unix = %d, want %d", ts.UTC().Unix(), tt.wantSec)
			}
		})
	}
}

func TestTimestamp_BothFormatsAgree(t *testing.T) {
	var epoch, rfc quota.Timestamp
	// Both spellings of the same instant must decode identically, otherwise a
	// quota read from stdin and the same quota read from the API disagree
	if err := json.Unmarshal([]byte(`1789770600`), &epoch); err != nil {
		t.Fatalf("epoch decode error = %v", err)
	}
	if err := json.Unmarshal([]byte(`"2026-09-18T22:30:00Z"`), &rfc); err != nil {
		t.Fatalf("rfc decode error = %v", err)
	}
	if !epoch.Equal(rfc.Time) {
		t.Errorf("epoch %v and rfc3339 %v decode to different instants", epoch.Time, rfc.Time)
	}
	if epoch.Time.Location() == nil || rfc.Time.Location() == nil {
		t.Error("decoded instants must carry a location")
	}
	_ = time.Now()
}

package snapshot_test

import (
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestClassifyHealth(t *testing.T) {
	tests := []struct {
		name   string
		states []string
		want   int
	}{
		{name: "nothing to judge", states: nil, want: snapshot.HealthUnknown},
		{name: "all operational", states: []string{"operational", "operational"}, want: snapshot.HealthOK},
		{name: "maintenance is not an outage", states: []string{"under_maintenance", "operational"}, want: snapshot.HealthOK},
		{name: "one degraded", states: []string{"degraded_performance", "operational"}, want: snapshot.HealthDegraded},
		{name: "one partial outage", states: []string{"partial_outage", "operational"}, want: snapshot.HealthDegraded},
		{name: "two degraded", states: []string{"degraded_performance", "partial_outage"}, want: snapshot.HealthDown},
		{name: "one major outage", states: []string{"operational", "major_outage"}, want: snapshot.HealthDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := snapshot.ClassifyHealth(tt.states); got != tt.want {
				t.Errorf("ClassifyHealth(%v) = %v, want %v", tt.states, got, tt.want)
			}
		})
	}
}

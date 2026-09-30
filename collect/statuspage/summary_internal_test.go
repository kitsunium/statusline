package statuspage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		body string
		want snapshot.Health
	}{
		{name: "all up, government ignored", body: `{"components":[{"name":"claude.ai","status":"operational"},{"name":"CLI","status":"operational"},{"name":"Claude for Government","status":"major_outage"}]}`, want: snapshot.HealthOK},
		{name: "one partial", body: `{"components":[{"name":"claude.ai","status":"partial_outage"},{"name":"CLI","status":"operational"}]}`, want: snapshot.HealthDegraded},
		{name: "two degraded", body: `{"components":[{"name":"a","status":"degraded_performance"},{"name":"b","status":"partial_outage"}]}`, want: snapshot.HealthDown},
		{name: "one major", body: `{"components":[{"name":"a","status":"major_outage"}]}`, want: snapshot.HealthDown},
		{name: "maintenance does not count", body: `{"components":[{"name":"a","status":"under_maintenance"}]}`, want: snapshot.HealthOK},
		{name: "groups only", body: `{"components":[{"name":"g","status":"major_outage","group":true}]}`, want: snapshot.HealthUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := classify([]byte(tt.body))
			if err != nil || got != tt.want {
				t.Errorf("classify() = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
	if _, err := classify([]byte(`{"components":"nope"}`)); err == nil {
		t.Error("an undecodable summary was accepted")
	}
}

// TestClassifyObserved replays the observed shape (design/evidence.yaml,
// status-page-summary-shape).
func TestClassifyObserved(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "design", "evidence", "observed", "status-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := classify(data); err != nil || got != snapshot.HealthDegraded {
		t.Errorf("classify(observed) = %v, %v; want degraded", got, err)
	}
}

func TestFetch(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"components":[{"name":"a","status":"operational"}]}`))
	}))
	defer srv.Close()
	c := &StatusPage{newPage(func(key string) string {
		if key == urlEnv {
			return srv.URL
		}
		return ""
	})}
	if got, err := c.Fetch(context.Background()); err != nil || got != snapshot.HealthOK {
		t.Errorf("Fetch() = %v, %v", got, err)
	}
	status = http.StatusBadGateway
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Error("a 502 was accepted")
	}
	if newPage(func(string) string { return "" }).url != defaultURL {
		t.Error("the default URL is not used without an override")
	}
}

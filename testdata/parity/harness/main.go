// Command harness is the parity oracle of statusline.
//
// It materialises each synthetic scenario of ../scenarios in a sandbox (a
// home directory, a configuration directory, git repositories, a host
// stand-in, cached or served network payloads), runs a status line binary in
// it with a closed environment, and records or compares what it printed.
//
//	harness capture -bin <legacy status-line> -flavour legacy
//	harness check   -bin <statusline>         -flavour kit
//	harness latency -bin <binary> -flavour legacy -runs 300
//
// The goldens were captured from kodflow/status-line at the commit named in
// ../README.md; they are the behaviour the new product must reproduce.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ansi matches the escapes the status line writes.
var ansi = regexp.MustCompile("\033\\[[0-9;]*m")

// Entry is one golden in the manifest.
type Entry struct {
	Scenario string            `json:"scenario"`
	Variant  string            `json:"variant"`
	Scope    string            `json:"scope"`
	Env      map[string]string `json:"env,omitempty"`
	SHA256   string            `json:"sha256"`
	Exit     int               `json:"exit"`
	Stderr   string            `json:"stderr,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	bin := fs.String("bin", "", "status line binary to run")
	flavour := fs.String("flavour", string(Kit), "legacy | kit")
	dir := fs.String("dir", "..", "parity directory (scenarios/, golden/)")
	only := fs.String("only", "", "regexp on scenario/variant")
	runs := fs.Int("runs", 300, "latency: runs per measure")
	scenario := fs.String("scenario", "busy/default", "latency: scenario/variant measured")
	out := fs.String("out", "", "latency: file written")
	_ = fs.Parse(os.Args[2:])
	if *bin == "" {
		usage()
	}
	abs, err := filepath.Abs(*bin)
	if err != nil {
		fatal(err)
	}
	scenarios, err := loadScenarios(filepath.Join(*dir, "scenarios"))
	if err != nil {
		fatal(err)
	}
	var filter *regexp.Regexp
	if *only != "" {
		filter = regexp.MustCompile(*only)
	}
	switch cmd {
	case "capture":
		capture(abs, Flavour(*flavour), *dir, scenarios, filter)
	case "check":
		os.Exit(check(abs, Flavour(*flavour), *dir, scenarios, filter))
	case "latency":
		latency(abs, Flavour(*flavour), scenarios, *scenario, *runs, *out)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: harness capture|check|latency -bin <binary> [-flavour legacy|kit] [-dir ..] [-only re]")
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "harness:", err)
	os.Exit(1)
}

// render runs one variant and returns the line the host would display.
func render(bin string, flavour Flavour, s Scenario, v Variant) (Result, error) {
	sb, err := buildSandbox(s, flavour, time.Now())
	if err != nil {
		return Result{}, err
	}
	defer sb.Close()
	if flavour == Kit {
		defer func() { _, _ = sb.run(bin, v, "daemon", "stop") }()
		return warm(sb, bin, s, v)
	}
	if s.EvenSecond {
		waitEvenSecond()
	}
	return sb.run(bin, v)
}

// warm runs the new binary until its daemon serves a stable line: the first
// render after a cold start is the cached one by design, the parity is on the
// warm one.
func warm(sb *Sandbox, bin string, s Scenario, v Variant) (Result, error) {
	deadline := time.Now().Add(5 * time.Second)
	prev, err := sb.run(bin, v)
	if err != nil {
		return prev, err
	}
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		if s.EvenSecond {
			waitEvenSecond()
		}
		cur, err := sb.run(bin, v)
		if err != nil {
			return cur, err
		}
		if bytes.Equal(cur.Stdout, prev.Stdout) {
			return cur, nil
		}
		prev = cur
	}
	return prev, nil
}

// waitEvenSecond sleeps until just after the next even wall-clock second.
func waitEvenSecond() {
	now := time.Now()
	next := now.Truncate(time.Second).Add(time.Second)
	if next.Unix()%2 != 0 {
		next = next.Add(time.Second)
	}
	time.Sleep(time.Until(next) + 40*time.Millisecond)
}

// goldenPath names the golden of a variant.
func goldenPath(dir string, s Scenario, v Variant, ext string) string {
	return filepath.Join(dir, "golden", s.Name, v.Name+ext)
}

// capture records the goldens and the manifest.
func capture(bin string, flavour Flavour, dir string, scenarios []Scenario, filter *regexp.Regexp) {
	var manifest []Entry
	for _, s := range scenarios {
		for _, v := range s.Variants {
			id := s.Name + "/" + v.Name
			if filter != nil && !filter.MatchString(id) {
				continue
			}
			res, err := render(bin, flavour, s, v)
			if err != nil {
				fatal(fmt.Errorf("%s: %w", id, err))
			}
			if err := os.MkdirAll(filepath.Dir(goldenPath(dir, s, v, "")), 0o755); err != nil {
				fatal(err)
			}
			if err := os.WriteFile(goldenPath(dir, s, v, ".ansi"), res.Stdout, 0o644); err != nil {
				fatal(err)
			}
			plain := ansi.ReplaceAllString(string(res.Stdout), "")
			if err := os.WriteFile(goldenPath(dir, s, v, ".txt"), []byte(plain), 0o644); err != nil {
				fatal(err)
			}
			sum := sha256.Sum256(res.Stdout)
			manifest = append(manifest, Entry{
				Scenario: s.Name, Variant: v.Name, Scope: s.Scope, Env: v.Env,
				SHA256: hex.EncodeToString(sum[:]), Exit: res.Exit, Stderr: string(res.Stderr),
			})
			fmt.Printf("captured %-40s %s\n", id, strings.SplitN(plain, "\n", 2)[0])
		}
	}
	if filter != nil {
		manifest = mergeManifest(filepath.Join(dir, "golden", "manifest.json"), manifest)
	}
	sort.Slice(manifest, func(i, j int) bool {
		if manifest[i].Scenario != manifest[j].Scenario {
			return manifest[i].Scenario < manifest[j].Scenario
		}
		return manifest[i].Variant < manifest[j].Variant
	})
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "golden", "manifest.json"), append(data, '\n'), 0o644); err != nil {
		fatal(err)
	}
}

// mergeManifest keeps the entries of a partial capture's untouched variants.
func mergeManifest(path string, fresh []Entry) []Entry {
	data, err := os.ReadFile(path)
	if err != nil {
		return fresh
	}
	var old []Entry
	if json.Unmarshal(data, &old) != nil {
		return fresh
	}
	seen := map[string]bool{}
	for _, e := range fresh {
		seen[e.Scenario+"/"+e.Variant] = true
	}
	for _, e := range old {
		if !seen[e.Scenario+"/"+e.Variant] {
			fresh = append(fresh, e)
		}
	}
	return fresh
}

// check compares a binary against the goldens; it returns the exit code.
func check(bin string, flavour Flavour, dir string, scenarios []Scenario, filter *regexp.Regexp) int {
	failed, passed, skipped := 0, 0, 0
	for _, s := range scenarios {
		for _, v := range s.Variants {
			id := s.Name + "/" + v.Name
			if filter != nil && !filter.MatchString(id) {
				continue
			}
			if flavour == Kit && s.Scope == "legacy-only" {
				skipped++
				fmt.Printf("SKIP %-40s legacy-only: %s\n", id, s.Note)
				continue
			}
			want, err := os.ReadFile(goldenPath(dir, s, v, ".ansi"))
			if err != nil {
				failed++
				fmt.Printf("FAIL %-40s no golden: %v\n", id, err)
				continue
			}
			res, err := render(bin, flavour, s, v)
			if err != nil {
				failed++
				fmt.Printf("FAIL %-40s run: %v\n", id, err)
				continue
			}
			if !bytes.Equal(res.Stdout, want) || res.Exit != 0 {
				failed++
				fmt.Printf("FAIL %-40s exit=%d\n  want %s\n  got  %s\n", id, res.Exit, strconv.Quote(string(want)), strconv.Quote(string(res.Stdout)))
				if len(res.Stderr) > 0 {
					fmt.Printf("  stderr %s\n", strconv.Quote(string(res.Stderr)))
				}
				continue
			}
			passed++
			fmt.Printf("ok   %s\n", id)
		}
	}
	fmt.Printf("parity: %d passed, %d failed, %d skipped (legacy-only)\n", passed, failed, skipped)
	if failed > 0 {
		return 1
	}
	return 0
}

// Latency is the measured client latency of one binary.
type Latency struct {
	Binary   string  `json:"binary"`
	Flavour  Flavour `json:"flavour"`
	Scenario string  `json:"scenario"`
	Runs     int     `json:"runs"`
	P50Ms    float64 `json:"p50_ms"`
	P95Ms    float64 `json:"p95_ms"`
	P99Ms    float64 `json:"p99_ms"`
	MaxMs    float64 `json:"max_ms"`
	Host     string  `json:"host"`
	Measured string  `json:"measured"`
}

// latency measures the wall time of one render, process start to exit.
func latency(bin string, flavour Flavour, scenarios []Scenario, id string, runs int, out string) {
	name, variant, _ := strings.Cut(id, "/")
	var s *Scenario
	for i := range scenarios {
		if scenarios[i].Name == name {
			s = &scenarios[i]
		}
	}
	if s == nil {
		fatal(fmt.Errorf("no scenario %q", name))
	}
	v := Variant{Name: "default"}
	for _, cand := range s.Variants {
		if cand.Name == variant {
			v = cand
		}
	}
	sb, err := buildSandbox(*s, flavour, time.Now())
	if err != nil {
		fatal(err)
	}
	defer sb.Close()
	if flavour == Kit {
		defer func() { _, _ = sb.run(bin, v, "daemon", "stop") }()
		if _, err := warm(sb, bin, *s, v); err != nil {
			fatal(err)
		}
	} else if _, err := sb.run(bin, v); err != nil {
		fatal(err)
	}
	took := make([]float64, 0, runs)
	for i := 0; i < runs; i++ {
		res, err := sb.run(bin, v)
		if err != nil {
			fatal(err)
		}
		took = append(took, float64(res.Took.Microseconds())/1000)
	}
	sort.Float64s(took)
	pick := func(q float64) float64 { return took[min(len(took)-1, int(q*float64(len(took))))] }
	l := Latency{
		Binary: filepath.Base(bin), Flavour: flavour, Scenario: id, Runs: runs,
		P50Ms: pick(0.50), P95Ms: pick(0.95), P99Ms: pick(0.99), MaxMs: took[len(took)-1],
		Host: hostLabel(), Measured: time.Now().UTC().Format("2006-01-02"),
	}
	data, _ := json.MarshalIndent(l, "", "  ")
	fmt.Println(string(data))
	if out != "" {
		if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
			fatal(err)
		}
	}
}

// hostLabel names the machine class, never the machine.
func hostLabel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return "unknown"
}

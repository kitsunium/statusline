#!/usr/bin/env bash
# latency-ab.sh — warm client latency of two binaries, measured in alternation.
#
#   testdata/parity/latency-ab.sh -a <before> -b <after> [-rounds 6] [-runs 300]
#       [-scenario busy/default] [-maxload 3] [-target 9.2] [-out <dir>]
#
# Each round runs the harness's warm latency measure once per binary (the
# order flips every round, so neither side always runs second), and writes
# the 1/5/15-minute load averages before and after each measure next to its
# p50/p95/p99. A measure is only started while the 1-minute load is below
# -maxload: above it the script stops, says so, and keeps what it has.
# The result is the median, across rounds, of each binary's p50.
#
# Build the two binaries first, the same way a release would (make build);
# the harness is built here, through ~/.local/bin/lourd when it exists. The
# measures themselves run outside the heavy slice: its CPU quota and nice
# level would be measured too.
set -euo pipefail
# printf %f reads its argument in the locale: a decimal comma locale refuses
# the harness's 7.221.
export LC_ALL=C

a="" b="" rounds=6 runs=300 scenario=busy/default maxload=3 target=9.2 out=""
while [[ $# -gt 0 ]]; do
  case $1 in
    -a) a=$2; shift 2 ;;
    -b) b=$2; shift 2 ;;
    -rounds) rounds=$2; shift 2 ;;
    -runs) runs=$2; shift 2 ;;
    -scenario) scenario=$2; shift 2 ;;
    -maxload) maxload=$2; shift 2 ;;
    -target) target=$2; shift 2 ;;
    -out) out=$2; shift 2 ;;
    *) sed -n '2,17p' "$0" >&2; exit 2 ;;
  esac
done
[[ -x $a && -x $b ]] || { echo "latency-ab: -a and -b must be executables" >&2; exit 2; }
a=$(realpath "$a") b=$(realpath "$b")

here=$(cd "$(dirname "$0")" && pwd)
out=${out:-$(mktemp -d "${TMPDIR:-/tmp}/latency-ab.XXXXXX")}
mkdir -p "$out"
harness=$out/harness
heavy=()
[[ -x $HOME/.local/bin/lourd ]] && heavy=("$HOME/.local/bin/lourd")
(cd "$here/harness" && "${heavy[@]}" go build -o "$harness" .)

load() { cut -d' ' -f1-3 /proc/loadavg; }
below() { awk -v l="$(cut -d' ' -f1 /proc/loadavg)" -v m="$maxload" 'BEGIN { exit !(l < m) }'; }

tsv=$out/measures.tsv
printf 'round\tside\tbinary_sha256\tload_before_1_5_15\tload_after_1_5_15\tp50_ms\tp95_ms\tp99_ms\truns\tscenario\n' >"$tsv"
declare -A sha=([a]=$(sha256sum "$a" | cut -c1-12) [b]=$(sha256sum "$b" | cut -c1-12))
declare -A bin=([a]=$a [b]=$b)
stopped=""

for ((r = 1; r <= rounds; r++)); do
  order=(a b)
  ((r % 2 == 0)) && order=(b a)
  for side in "${order[@]}"; do
    if ! below; then
      stopped="round $r, side $side: 1-minute load $(load | cut -d' ' -f1) >= $maxload, no measure started"
      break 2
    fi
    before=$(load)
    json=$out/r$r-$side.json
    (cd "$here/harness" && "$harness" latency -bin "${bin[$side]}" -flavour kit \
      -runs "$runs" -scenario "$scenario" -out "$json" >/dev/null)
    after=$(load)
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$r" "$side" "${sha[$side]}" "$before" "$after" \
      "$(jq -r '[.p50_ms, .p95_ms, .p99_ms, .runs, .scenario] | @tsv' "$json")" >>"$tsv"
    printf 'round %d  %s  p50 %6.3f ms   load %s -> %s\n' "$r" "$side" \
      "$(jq -r .p50_ms "$json")" "$before" "$after"
  done
done

echo
column -t -s $'\t' "$tsv"
echo
[[ -n $stopped ]] && echo "stopped: $stopped"
python3 - "$tsv" "$target" <<'EOF'
import csv, statistics, sys
rows = list(csv.DictReader(open(sys.argv[1]), delimiter="\t"))
target = float(sys.argv[2])
med = {}
for side, label in (("a", "before"), ("b", "after")):
    p50 = [float(r["p50_ms"]) for r in rows if r["side"] == side]
    loads = [r["load_before_1_5_15"].split()[0] for r in rows if r["side"] == side]
    if not p50:
        print(f"{side} ({label}): no measure")
        continue
    med[side] = statistics.median(p50)
    print(f"{side} ({label}): median p50 {med[side]:.3f} ms over {len(p50)} rounds, "
          f"min {min(p50):.3f}, max {max(p50):.3f}, 1-min load at start {' '.join(loads)}")
if "a" in med and "b" in med:
    d = med["b"] - med["a"]
    print(f"after - before: {d:+.3f} ms; after {'<=' if med['b'] <= target else '>'} target {target} ms")
EOF
echo "measures kept in $out"
[[ -z $stopped ]]

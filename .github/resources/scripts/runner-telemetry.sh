#!/usr/bin/env bash
#
# Self-contained runner telemetry: samples host CPU, memory, and load from
# /proc and renders mermaid charts plus a stats table into the GitHub job
# summary. Exists because CI has no metrics-server and the repo cannot ship
# metric samples to third-party chart services; everything here stays on the
# runner and in GitHub.
#
#   runner-telemetry.sh sample <csv>   # loop forever appending samples
#   runner-telemetry.sh render <csv>   # stop sampler, render to job summary
#
# The sampler is started in the background at the beginning of test-and-report
# and rendered near the end, so the window covers deploy + test execution.
# Best-effort: render never fails the caller.

set -u

INTERVAL_SECONDS=5
PID_FILE="/tmp/runner-telemetry.pid"

cpu_totals() {
    # Prints "<busy> <total>" jiffies from the aggregate cpu line.
    awk '/^cpu / {
        idle = $5 + $6
        total = 0
        for (i = 2; i <= 9; i++) total += $i
        print total - idle, total
    }' /proc/stat
}

sample_loop() {
    local csv="$1"
    echo "epoch,cpu_pct,mem_used_mb,load1" > "$csv"
    local prev_busy prev_total
    read -r prev_busy prev_total < <(cpu_totals)
    while true; do
        sleep "$INTERVAL_SECONDS"
        local busy total
        read -r busy total < <(cpu_totals)
        local cpu_pct=0
        if [[ $((total - prev_total)) -gt 0 ]]; then
            cpu_pct=$(( 100 * (busy - prev_busy) / (total - prev_total) ))
        fi
        prev_busy=$busy; prev_total=$total
        local mem_used_mb
        mem_used_mb=$(awk '/MemTotal/ {t=$2} /MemAvailable/ {a=$2} END {print int((t-a)/1024)}' /proc/meminfo)
        local load1
        load1=$(cut -d' ' -f1 /proc/loadavg)
        echo "$(date +%s),$cpu_pct,$mem_used_mb,$load1" >> "$csv"
    done
}

render() {
    local csv="$1"
    if [[ -f "$PID_FILE" ]]; then
        kill "$(cat "$PID_FILE")" 2>/dev/null || true
        rm -f "$PID_FILE"
    fi
    local out="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
    if [[ ! -r "$csv" ]] || [[ $(wc -l < "$csv") -lt 3 ]]; then
        echo "_Runner telemetry: no samples collected._" >> "$out"
        return 0
    fi
    if ! command -v python3 >/dev/null 2>&1; then
        echo "_Runner telemetry: python3 unavailable; raw samples in ${csv}._" >> "$out"
        return 0
    fi
    python3 - "$csv" >> "$out" <<'PY'
import sys

rows = []
with open(sys.argv[1]) as f:
    next(f)  # header
    for line in f:
        parts = line.strip().split(",")
        if len(parts) == 4:
            try:
                rows.append((int(parts[0]), int(parts[1]), int(parts[2]), float(parts[3])))
            except ValueError:
                continue

if len(rows) < 2:
    print("_Runner telemetry: not enough samples to render._")
    sys.exit(0)

start = rows[0][0]
minutes = (rows[-1][0] - start) / 60


def downsample(values, cap=50):
    if len(values) <= cap:
        return values
    step = len(values) / cap
    return [values[int(i * step)] for i in range(cap)]


cpu = [r[1] for r in rows]
mem = [r[2] for r in rows]
load = [r[3] for r in rows]
ncpu = 0
try:
    import os
    ncpu = os.cpu_count() or 0
except Exception:
    pass

print("\n## Runner telemetry\n")
print(
    f"_{len(rows)} samples over {minutes:.0f} min"
    + (f", {ncpu} CPUs" if ncpu else "")
    + "._\n"
)
print("| Metric | Avg | Peak |")
print("|---|---:|---:|")
print(f"| CPU busy % | {sum(cpu)/len(cpu):.0f}% | {max(cpu)}% |")
print(f"| Memory used MB | {sum(mem)/len(mem):.0f} | {max(mem)} |")
print(f"| Load (1 min) | {sum(load)/len(load):.1f} | {max(load):.1f} |")

cpu_ds = downsample(cpu)
mem_ds = downsample(mem)
print("\n```mermaid")
print("xychart-beta")
print('    title "Runner CPU busy %"')
print(f"    x-axis 0 --> {max(len(cpu_ds) - 1, 1)}")
print('    y-axis "CPU %" 0 --> 100')
print(f"    line [{', '.join(str(v) for v in cpu_ds)}]")
print("```")
print("\n```mermaid")
print("xychart-beta")
print('    title "Runner memory used (MB)"')
print(f"    x-axis 0 --> {max(len(mem_ds) - 1, 1)}")
print(f'    y-axis "MB" 0 --> {max(mem) + 256}')
print(f"    line [{', '.join(str(v) for v in mem_ds)}]")
print("```")
PY
    return 0
}

case "${1:-}" in
    sample)
        sample_loop "${2:?csv path required}"
        ;;
    render)
        render "${2:?csv path required}"
        ;;
    *)
        echo "usage: $0 {sample|render} <csv>"
        exit 1
        ;;
esac

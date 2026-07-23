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
PROC_ROOT="${RUNNER_TELEMETRY_PROC_ROOT:-/proc}"
CGROUP_ROOT="${RUNNER_TELEMETRY_CGROUP_ROOT:-/sys/fs/cgroup}"

# Job summaries are not retrievable through the logs API; mirror rendered
# output to stdout so the job log carries the numbers too.
publish() {
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        tee -a "$GITHUB_STEP_SUMMARY"
    else
        cat
    fi
}

cpu_totals() {
    # Prints "<busy> <total>" jiffies from the aggregate cpu line.
    awk '/^cpu / {
        idle = $5 + $6
        total = 0
        for (i = 2; i <= 9; i++) total += $i
        print total - idle, total
    }' /proc/stat
}

contention_snapshot() {
    local output="$1"
    : > "$output"
    printf 'epoch_us\t%s\n' "$(python3 -c 'import time; print(time.time_ns() // 1000)')" >> "$output"

    if [[ -r "$PROC_ROOT/pressure/cpu" ]]; then
        awk '
            ($1 == "some" || $1 == "full") {
                for (i = 2; i <= NF; i++) {
                    split($i, field, "=")
                    if (field[1] == "total") print "cpu_psi_" $1 "_us\t" field[2]
                }
            }
        ' "$PROC_ROOT/pressure/cpu" >> "$output"
    fi

    local cpu_stat=""
    if [[ -r "$PROC_ROOT/self/cgroup" ]]; then
        local cgroup_path
        cgroup_path=$(awk -F: '$1 == "0" {print $3; exit}' "$PROC_ROOT/self/cgroup")
        if [[ -n "$cgroup_path" && -r "$CGROUP_ROOT$cgroup_path/cpu.stat" ]]; then
            cpu_stat="$CGROUP_ROOT$cgroup_path/cpu.stat"
        else
            cgroup_path=$(awk -F: '$2 ~ /(^|,)cpu(,|$)/ {print $3; exit}' "$PROC_ROOT/self/cgroup")
            for candidate in \
                "$CGROUP_ROOT/cpu,cpuacct$cgroup_path/cpu.stat" \
                "$CGROUP_ROOT/cpu$cgroup_path/cpu.stat"; do
                if [[ -n "$cgroup_path" && -r "$candidate" ]]; then
                    cpu_stat="$candidate"
                    break
                fi
            done
        fi
    fi
    # A cgroup namespace commonly exposes the process cgroup as the mount root.
    if [[ -z "$cpu_stat" && -r "$CGROUP_ROOT/cpu.stat" ]]; then
        cpu_stat="$CGROUP_ROOT/cpu.stat"
    fi

    if [[ -n "$cpu_stat" ]]; then
        awk '
            $1 == "nr_periods" || $1 == "nr_throttled" || $1 == "throttled_usec" {
                print "runner_cgroup_" $1 "\t" $2
            }
            $1 == "throttled_time" {
                print "runner_cgroup_throttled_usec\t" int($2 / 1000)
            }
        ' "$cpu_stat" >> "$output"
    fi
}

render_contention_deltas() {
    local baseline="$1"
    if [[ -z "$baseline" || ! -r "$baseline" ]]; then
        printf '\n## Test-window contention deltas\n\n_Baseline unavailable; test setup ended before the Ginkgo window._\n' | publish
        return 0
    fi

    local end_snapshot="${baseline}.end"
    contention_snapshot "$end_snapshot"
    python3 - "$baseline" "$end_snapshot" <<'PY' | publish
import sys


def read_snapshot(path):
    values = {}
    with open(path) as snapshot:
        for line in snapshot:
            key, separator, value = line.rstrip().partition("\t")
            if separator and value.isdigit():
                values[key] = int(value)
    return values


baseline = read_snapshot(sys.argv[1])
end = read_snapshot(sys.argv[2])
elapsed_us = end.get("epoch_us", 0) - baseline.get("epoch_us", 0)
metrics = (
    ("CPU PSI some stall", "cpu_psi_some_us", "time"),
    ("CPU PSI full stall", "cpu_psi_full_us", "time"),
    ("Runner cgroup CPU periods", "runner_cgroup_nr_periods", "count"),
    ("Runner cgroup throttled periods", "runner_cgroup_nr_throttled", "count"),
    ("Runner cgroup throttled time", "runner_cgroup_throttled_usec", "time"),
)

print("\n## Test-window contention deltas\n")
print("| Metric | Baseline | End | Delta | Status |")
print("|---|---:|---:|---:|---|")
for label, key, kind in metrics:
    before = baseline.get(key)
    after = end.get(key)
    if before is None or after is None:
        print(f"| {label} | {before if before is not None else '—'} | "
              f"{after if after is not None else '—'} | — | unavailable |")
        continue
    if after < before:
        print(f"| {label} | {before} | {after} | — | counter reset |")
        continue
    delta = after - before
    if kind == "time":
        rendered_delta = f"{delta / 1000:.1f} ms"
        if elapsed_us > 0 and key.startswith("cpu_psi_"):
            rendered_delta += f" ({100 * delta / elapsed_us:.2f}% of window)"
    else:
        rendered_delta = str(delta)
    print(f"| {label} | {before} | {after} | {rendered_delta} | ok |")

period_keys = ("runner_cgroup_nr_periods", "runner_cgroup_nr_throttled")
if all(key in baseline and key in end for key in period_keys):
    periods = end[period_keys[0]] - baseline[period_keys[0]]
    throttled = end[period_keys[1]] - baseline[period_keys[1]]
else:
    periods = throttled = -1
if periods > 0 and throttled >= 0:
    print(f"\n_Runner cgroup throttled periods: {100 * throttled / periods:.1f}% of test-window periods._")
print("\n_CPU PSI is host-wide. Runner-cgroup counters do not include sibling Kind cgroups on every runner layout._")
PY
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
    local contention_baseline="${2:-}"
    if [[ -f "$PID_FILE" ]]; then
        kill "$(cat "$PID_FILE")" 2>/dev/null || true
        rm -f "$PID_FILE"
    fi
    render_contention_deltas "$contention_baseline"
    if [[ ! -r "$csv" ]] || [[ $(wc -l < "$csv") -lt 3 ]]; then
        echo "_Runner telemetry: no samples collected._" | publish
        return 0
    fi
    if ! command -v python3 >/dev/null 2>&1; then
        echo "_Runner telemetry: python3 unavailable; raw samples in ${csv}._" | publish
        return 0
    fi
    python3 - "$csv" <<'PY' | publish
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
    # Bucket-max, not every-Nth: a short saturation spike between picks must
    # stay visible in the chart, and for forensics the max is the signal.
    if len(values) <= cap:
        return values
    n = len(values)
    out = []
    for i in range(cap):
        lo = i * n // cap
        hi = max(lo + 1, (i + 1) * n // cap)
        out.append(max(values[lo:hi]))
    return out


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
        render "${2:?csv path required}" "${3:-}"
        ;;
    contention-baseline)
        contention_snapshot "${2:?snapshot path required}"
        ;;
    *)
        echo "usage: $0 {sample|render|contention-baseline} <path> [contention-baseline]"
        exit 1
        ;;
esac

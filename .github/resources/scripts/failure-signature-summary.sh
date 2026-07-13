#!/usr/bin/env bash
#
# Classifies a failed E2E/integration lane by grepping the collected pod logs
# and JUnit reports for the recurring failure signatures seen in this repo's
# CI, and writes a compact markdown table to the job summary. Triage then
# starts from "which known class is this (or is it new)?" instead of raw log
# archaeology.
#
# Signature catalog (kept in sync with observed flake classes):
#   - ClusterIP dial timeout (per VIP)  -> service dataplane / backend saturation
#   - client deadline / timeout         -> slow server (e.g. MLflow queueing)
#   - connection refused / reset        -> process down or restarting
#   - HTTP 429                          -> upstream rate limiting
#   - OOMKilled / CrashLoopBackOff      -> pod lifecycle
#   - nf_conntrack table full           -> node conntrack exhaustion
#
# Best-effort: missing files or tools never fail the caller.

set -u

LOG_FILE=""
REPORTS_DIR=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --log-file) LOG_FILE="${2:?--log-file requires a value}"; shift ;;
        --reports-dir) REPORTS_DIR="${2:?--reports-dir requires a value}"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

OUT="${GITHUB_STEP_SUMMARY:-/dev/stdout}"

count_in_log() {
    # $1 = extended regex; prints 0 when the log is missing.
    if [[ -n "$LOG_FILE" && -r "$LOG_FILE" ]]; then
        grep -cE "$1" "$LOG_FILE" 2>/dev/null || true
    else
        echo 0
    fi
}

{
    echo ""
    echo "## Failure signature summary"
    echo ""

    if [[ -z "$LOG_FILE" || ! -r "$LOG_FILE" ]]; then
        echo "_No collected pod-log file at \`${LOG_FILE:-<unset>}\`; signature scan skipped._"
    else
        echo "| Signature | Count | Points at |"
        echo "|---|---:|---|"
        echo "| ClusterIP dial \`i/o timeout\` | $(count_in_log 'dial tcp [0-9.]+:[0-9]+: i/o timeout') | service dataplane / backend accept saturation |"
        echo "| Client deadline / timeout | $(count_in_log 'context deadline exceeded|Client\.Timeout exceeded') | slow server (queueing, saturation) |"
        echo "| Connection refused / reset | $(count_in_log 'connection refused|connection reset by peer') | process down or restarting |"
        echo "| HTTP 429 rate limit | $(count_in_log 'status 429|429 Too Many Requests') | upstream rate limiting |"
        echo "| OOMKilled | $(count_in_log 'OOMKilled') | pod memory limits |"
        echo "| CrashLoopBackOff | $(count_in_log 'CrashLoopBackOff') | pod lifecycle |"
        echo "| conntrack table full | $(count_in_log 'nf_conntrack: table full') | node conntrack exhaustion |"

        # Per-VIP breakdown makes multi-service dataplane failures (SeaweedFS
        # :9000 vs MLflow :8443 ...) visible at a glance.
        vips=$(grep -oE 'dial tcp [0-9.]+:[0-9]+: i/o timeout' "$LOG_FILE" 2>/dev/null \
            | grep -oE '[0-9.]+:[0-9]+' | sort | uniq -c | sort -rn | head -5 || true)
        if [[ -n "$vips" ]]; then
            echo ""
            echo "**Dial-timeout targets (top 5):**"
            echo ""
            echo '```'
            printf '%s\n' "$vips"
            echo '```'
        fi
    fi

    # Failed JUnit test cases, so the summary names the tests, not just the lane.
    if [[ -n "$REPORTS_DIR" && -d "$REPORTS_DIR" ]] && command -v python3 >/dev/null 2>&1; then
        failed=$(python3 - "$REPORTS_DIR" <<'PY' 2>/dev/null || true
import sys, glob, os
import xml.etree.ElementTree as ET

names = []
for path in glob.glob(os.path.join(sys.argv[1], "*.xml")):
    try:
        root = ET.parse(path).getroot()
    except ET.ParseError:
        continue
    for case in root.iter("testcase"):
        if case.find("failure") is not None or case.find("error") is not None:
            names.append(case.get("name") or "<unnamed>")
for name in names[:10]:
    print(f"- {name}")
if len(names) > 10:
    print(f"- ... and {len(names) - 10} more")
PY
)
        if [[ -n "$failed" ]]; then
            echo ""
            echo "**Failed test cases:**"
            echo ""
            printf '%s\n' "$failed"
        fi
    fi
    echo ""
} >> "$OUT"

exit 0

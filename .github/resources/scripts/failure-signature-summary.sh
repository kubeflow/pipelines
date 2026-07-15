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

# Signature scan corpus: the collected pod logs PLUS the JUnit failure/error
# bodies. Client-side failures (MLflow timeouts, connection refused from the
# test process) live only in the JUnit XML, never in pod logs, so scanning the
# pod log alone under-reports exactly the client-observed classes.
SCAN_SOURCES=()
SOURCE_LABELS=()
if [[ -n "$LOG_FILE" && -r "$LOG_FILE" ]]; then
    SCAN_SOURCES+=("$LOG_FILE")
    SOURCE_LABELS+=("pod logs")
fi
JUNIT_TEXT=""
if [[ -n "$REPORTS_DIR" && -d "$REPORTS_DIR" ]] && command -v python3 >/dev/null 2>&1; then
    JUNIT_TEXT=$(mktemp)
    python3 - "$REPORTS_DIR" > "$JUNIT_TEXT" 2>/dev/null <<'PY' || true
import sys, glob, os
import xml.etree.ElementTree as ET

for path in glob.glob(os.path.join(sys.argv[1], "*.xml")):
    try:
        root = ET.parse(path).getroot()
    except ET.ParseError:
        continue
    for case in root.iter("testcase"):
        for node_name in ("failure", "error"):
            node = case.find(node_name)
            if node is None:
                continue
            # Ginkgo writes the same failure text to both the message
            # attribute and the element body; emitting both double-counts
            # every signature. Prefer the body, fall back to the message.
            body = (node.text or "").strip()
            if body:
                print(body)
            elif node.get("message"):
                print(node.get("message"))
PY
    if [[ -s "$JUNIT_TEXT" ]]; then
        SCAN_SOURCES+=("$JUNIT_TEXT")
        SOURCE_LABELS+=("JUnit failure bodies")
    fi
fi

count_signature() {
    # $1 = extended regex, counted across every scan source.
    if [[ ${#SCAN_SOURCES[@]} -eq 0 ]]; then
        echo 0
        return
    fi
    cat "${SCAN_SOURCES[@]}" 2>/dev/null | grep -cE "$1" || true
}

{
    echo ""
    echo "## Failure signature summary"
    echo ""

    if [[ ${#SCAN_SOURCES[@]} -eq 0 ]]; then
        echo "_No pod-log file or JUnit reports found; signature scan skipped._"
    else
        scanned=$(printf '%s, ' "${SOURCE_LABELS[@]}")
        echo "_Scanned: ${scanned%, }._"
        echo ""
        echo "| Signature | Count | Points at |"
        echo "|---|---:|---|"
        echo "| ClusterIP dial \`i/o timeout\` | $(count_signature 'dial tcp [0-9.]+:[0-9]+: i/o timeout') | service dataplane / backend accept saturation |"
        echo "| Client deadline / timeout | $(count_signature 'context deadline exceeded|Client\.Timeout exceeded') | slow server (queueing, saturation) |"
        echo "| Connection refused / reset | $(count_signature 'connection refused|connection reset by peer') | process down or restarting |"
        echo "| HTTP 429 rate limit | $(count_signature 'status 429|429 Too Many Requests') | upstream rate limiting |"
        echo "| OOMKilled | $(count_signature 'OOMKilled') | pod memory limits |"
        echo "| CrashLoopBackOff | $(count_signature 'CrashLoopBackOff') | pod lifecycle |"
        echo "| conntrack table full | $(count_signature 'nf_conntrack: table full') | node conntrack exhaustion |"

        # Per-VIP breakdown makes multi-service dataplane failures (SeaweedFS
        # :9000 vs MLflow :8443 ...) visible at a glance.
        vips=$(cat "${SCAN_SOURCES[@]}" 2>/dev/null \
            | grep -oE 'dial tcp [0-9.]+:[0-9]+: i/o timeout' \
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

[[ -n "$JUNIT_TEXT" ]] && rm -f "$JUNIT_TEXT"
exit 0

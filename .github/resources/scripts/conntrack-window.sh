#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -u

publish() {
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        tee -a "$GITHUB_STEP_SUMMARY"
    else
        cat
    fi
}

discover_nodes() {
    kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        2>/dev/null || true
}

capture_node() {
    local output="$1"
    local node="$2"
    local container_id netns stats

    if ! container_id=$(docker inspect --format '{{.Id}}' "$node" 2>/dev/null); then
        printf 'node\t%s\t-\t-\tnot_docker\n' "$node" >> "$output"
        return
    fi
    if ! netns=$(docker exec "$node" readlink /proc/self/ns/net 2>/dev/null); then
        printf 'node\t%s\t%s\t-\texec_unavailable\n' "$node" "$container_id" >> "$output"
        return
    fi
    if ! stats=$(docker exec "$node" conntrack -S 2>/dev/null); then
        printf 'node\t%s\t%s\t%s\tconntrack_unavailable\n' \
            "$node" "$container_id" "$netns" >> "$output"
        return
    fi

    printf 'node\t%s\t%s\t%s\tok\n' "$node" "$container_id" "$netns" >> "$output"
    # Python integers preserve exact cumulative counters even on long-lived
    # nodes where awk's floating-point arithmetic could lose precision.
    python3 -c '
import collections
import sys

totals = collections.Counter()
for line in sys.stdin:
    for field in line.split():
        key, separator, value = field.partition("=")
        if separator and key != "cpu" and value.isdecimal():
            totals[key] += int(value)
for counter, value in sorted(totals.items()):
    print("metric", *sys.argv[1:], counter, value, sep="\t")
' "$node" "$container_id" "$netns" <<< "$stats" >> "$output"
}

capture() {
    local output="$1"
    local baseline="${2:-}"
    : > "$output"
    printf '# conntrack-window-v1\n' >> "$output"

    if ! command -v docker >/dev/null 2>&1; then
        printf 'status\tdocker_unavailable\n' >> "$output"
        return 0
    fi

    local nodes
    nodes=$(
        {
            discover_nodes
            if [[ -n "$baseline" && -r "$baseline" ]]; then
                awk -F '\t' '$1 == "node" {print $2}' "$baseline"
            fi
        } | sort -u
    )
    if [[ -z "$nodes" ]]; then
        printf 'status\tno_nodes\n' >> "$output"
        return 0
    fi

    local node
    for node in $nodes; do
        capture_node "$output" "$node"
    done
}

report() {
    local baseline="$1"
    if [[ ! -r "$baseline" ]]; then
        printf '\n## Test-window conntrack deltas\n\n_Baseline unavailable; test setup ended before the Ginkgo window._\n' | publish
        return 0
    fi

    local end_snapshot="${baseline}.end"
    capture "$end_snapshot" "$baseline"
    python3 - "$baseline" "$end_snapshot" <<'PY' | publish
import sys


def read_snapshot(path):
    nodes = {}
    metrics = {}
    with open(path) as snapshot:
        for line in snapshot:
            fields = line.rstrip().split("\t")
            if fields[0] == "node" and len(fields) == 5:
                nodes[fields[1]] = tuple(fields[2:])
            elif fields[0] == "metric" and len(fields) == 6:
                try:
                    metrics[(fields[1], fields[4])] = int(fields[5])
                except ValueError:
                    pass
    return nodes, metrics


baseline_nodes, baseline_metrics = read_snapshot(sys.argv[1])
end_nodes, end_metrics = read_snapshot(sys.argv[2])
counters = ("insert", "insert_failed", "drop", "early_drop", "error", "search_restart", "clash")

print("\n## Test-window conntrack deltas\n")
print("| Node | Counter | Baseline | End | Delta | Status |")
print("|---|---|---:|---:|---:|---|")
rows = 0
for node in sorted(set(baseline_nodes) | set(end_nodes)):
    before_node = baseline_nodes.get(node)
    after_node = end_nodes.get(node)
    if before_node is None or after_node is None:
        status = "appeared after baseline" if before_node is None else "missing at end"
        print(f"| {node} | — | — | — | — | {status} |")
        rows += 1
        continue
    before_id, before_netns, before_status = before_node
    after_id, after_netns, after_status = after_node
    if before_status != "ok" or after_status != "ok":
        print(f"| {node} | — | — | — | — | {before_status} → {after_status} |")
        rows += 1
        continue
    if (before_id, before_netns) != (after_id, after_netns):
        print(f"| {node} | — | — | — | — | node/netns replaced; delta unavailable |")
        rows += 1
        continue
    for counter in counters:
        before = baseline_metrics.get((node, counter))
        after = end_metrics.get((node, counter))
        if before is None and after is None:
            continue
        rows += 1
        if before is None or after is None:
            print(f"| {node} | {counter} | {before if before is not None else '—'} | "
                  f"{after if after is not None else '—'} | — | unavailable |")
        elif after < before:
            print(f"| {node} | {counter} | {before} | {after} | — | counter reset |")
        else:
            print(f"| {node} | {counter} | {before} | {after} | {after - before} | ok |")
if not rows:
    print("| — | — | — | — | — | no Kind node conntrack counters available |")
print("\n_Nonzero `insert_failed`/`drop` deltas are attributed to the test window; table pressure still requires `early_drop` or table-full evidence._")
PY
}

case "${1:-}" in
    capture)
        capture "${2:?snapshot path required}"
        ;;
    report)
        report "${2:?baseline path required}"
        ;;
    *)
        echo "usage: $0 {capture|report} <snapshot>"
        exit 1
        ;;
esac

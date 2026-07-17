#!/usr/bin/env python3
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
"""Sample runner and Kind-node network counters throughout a test window."""

from __future__ import annotations

import argparse
from collections import defaultdict
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import signal
import subprocess
import threading
import time

import service_path_probe


COUNTER_METRICS = {
    'conntrack.clash',
    'conntrack.clash_resolve',
    'conntrack.drop',
    'conntrack.early_drop',
    'conntrack.error',
    'conntrack.insert',
    'conntrack.insert_failed',
    'conntrack.search_restart',
    'Ip.InDiscards',
    'Ip.OutDiscards',
    'Tcp.RetransSegs',
    'TcpExt.ListenDrops',
    'TcpExt.ListenOverflows',
    'TcpExt.TCPBacklogDrop',
    'TcpExt.TCPReqQFullDoCookies',
    'TcpExt.TCPReqQFullDrop',
    'TcpExt.TCPSynRetrans',
    'TcpExt.TCPTimeouts',
    'softnet.dropped',
    'softnet.processed',
    'softnet.time_squeeze',
}
GAUGE_METRICS = {
    'conntrack.count',
    'conntrack.max',
    *service_path_probe.SELECTED_GAUGES,
}
CORRELATION_METRICS = COUNTER_METRICS - {
    'conntrack.insert',
    'softnet.processed',
}
STOP_EVENT = threading.Event()
NODE_COMMAND = r'''
echo __NETNS__
readlink /proc/self/ns/net 2>/dev/null || true
echo __CONNTRACK__
conntrack -S 2>/dev/null || true
echo __CONNTRACK_GAUGES__
count=$(cat /proc/sys/net/netfilter/nf_conntrack_count 2>/dev/null || true)
max=$(cat /proc/sys/net/netfilter/nf_conntrack_max 2>/dev/null || true)
printf 'count=%s max=%s\n' "$count" "$max"
echo __SOCKSTAT__
cat /proc/net/sockstat 2>/dev/null || true
echo __NETSTAT__
cat /proc/net/netstat 2>/dev/null || true
echo __SNMP__
cat /proc/net/snmp 2>/dev/null || true
'''


def run_command(arguments: list[str], timeout_seconds: float = 3.0) -> str | None:
    try:
        result = subprocess.run(
            arguments,
            check=True,
            capture_output=True,
            text=True,
            timeout=timeout_seconds,
            stdin=subprocess.DEVNULL,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    return result.stdout


def discover_nodes() -> list[str]:
    output = run_command([
        'kubectl',
        'get',
        'nodes',
        '-o',
        r'jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}',
    ], timeout_seconds=5.0)
    return sorted(set(output.split())) if output else []


def parse_softnet(text: str) -> dict[str, int]:
    totals = {'processed': 0, 'dropped': 0, 'time_squeeze': 0}
    rows = 0
    for line in text.splitlines():
        fields = line.split()
        if len(fields) < 3:
            continue
        try:
            totals['processed'] += int(fields[0], 16)
            totals['dropped'] += int(fields[1], 16)
            totals['time_squeeze'] += int(fields[2], 16)
            rows += 1
        except ValueError:
            continue
    return {f'softnet.{key}': value for key, value in totals.items()} if rows else {}


def parse_conntrack(text: str) -> dict[str, int]:
    totals: dict[str, int] = defaultdict(int)
    for line in text.splitlines():
        for field in line.split():
            key, separator, value = field.partition('=')
            if separator and key != 'cpu' and value.isdecimal():
                totals[f'conntrack.{key}'] += int(value)
    return dict(totals)


def split_sections(text: str) -> dict[str, str]:
    sections: dict[str, list[str]] = defaultdict(list)
    current = ''
    for line in text.splitlines():
        if line.startswith('__') and line.endswith('__'):
            current = line.strip('_').lower()
        elif current:
            sections[current].append(line)
    return {key: '\n'.join(lines) for key, lines in sections.items()}


def parse_node_output(text: str) -> tuple[str, dict[str, int]]:
    sections = split_sections(text)
    network_namespace = sections.get('netns', '').strip() or 'unavailable'
    metrics = parse_conntrack(sections.get('conntrack', ''))
    for field in sections.get('conntrack_gauges', '').split():
        key, separator, value = field.partition('=')
        if separator and value.isdecimal():
            metrics[f'conntrack.{key}'] = int(value)
    proc_values = {}
    proc_values.update(
        service_path_probe.parse_proc_table(sections.get('netstat', ''))
    )
    proc_values.update(service_path_probe.parse_proc_table(sections.get('snmp', '')))
    proc_values.update(service_path_probe.parse_sockstat(sections.get('sockstat', '')))
    metrics.update({
        key: value
        for key, value in proc_values.items()
        if key in COUNTER_METRICS or key in GAUGE_METRICS
    })
    return network_namespace, metrics


def host_metrics(proc_root: Path) -> dict[str, int]:
    metrics = {}
    try:
        metrics.update(
            parse_softnet(
                (proc_root / 'net' / 'softnet_stat').read_text(encoding='utf-8')
            )
        )
    except OSError:
        pass
    metrics.update(service_path_probe.snapshot_network_metrics(proc_root))
    return metrics


def metric_event(
    timestamp_ms: int,
    sample_number: int,
    source: str,
    identity: str,
    metric: str,
    value: int,
) -> dict[str, object]:
    return {
        'type': 'metric',
        'timestamp_ms': timestamp_ms,
        'sample': sample_number,
        'source': source,
        'identity': identity,
        'metric': metric,
        'value': value,
    }


def capture_sample(nodes: list[str], proc_root: Path, sample_number: int) -> list[dict[str, object]]:
    timestamp_ms = time.time_ns() // 1_000_000
    events = [
        metric_event(timestamp_ms, sample_number, 'runner-host', 'host', metric, value)
        for metric, value in host_metrics(proc_root).items()
    ]
    for node in nodes:
        container_id = run_command(
            ['docker', 'inspect', '--format', '{{.Id}}', node]
        )
        node_output = run_command(['docker', 'exec', node, 'sh', '-c', NODE_COMMAND])
        node_timestamp_ms = time.time_ns() // 1_000_000
        if not container_id or node_output is None:
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': f'node:{node}',
                'status': 'docker_unavailable',
            })
            continue
        network_namespace, metrics = parse_node_output(node_output)
        identity = f'{container_id.strip()}|{network_namespace}'
        if network_namespace == 'unavailable':
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': f'node:{node}',
                'identity': identity,
                'status': 'netns_unavailable',
            })
        for metric, value in metrics.items():
            events.append(
                metric_event(
                    node_timestamp_ms,
                    sample_number,
                    f'node:{node}',
                    identity,
                    metric,
                    value,
                )
            )
    return events


def sample(output: Path, interval_seconds: float, max_samples: int) -> None:
    nodes = discover_nodes()
    proc_root = Path(os.environ.get('NETWORK_WINDOW_PROC_ROOT', '/proc'))
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open('w', encoding='utf-8') as event_file:
        event_file.write(json.dumps({
            'type': 'meta',
            'version': 1,
            'nodes': nodes,
            'interval_seconds': interval_seconds,
        }, sort_keys=True) + '\n')
        event_file.flush()
        sample_number = 0
        started_ns = time.monotonic_ns()
        while not STOP_EVENT.is_set() and (
            max_samples <= 0 or sample_number < max_samples
        ):
            scheduled_ns = started_ns + int(
                sample_number * interval_seconds * 1_000_000_000
            )
            remaining_seconds = (scheduled_ns - time.monotonic_ns()) / 1_000_000_000
            if remaining_seconds > 0 and STOP_EVENT.wait(remaining_seconds):
                break
            for event in capture_sample(nodes, proc_root, sample_number):
                event_file.write(json.dumps(event, sort_keys=True) + '\n')
            event_file.flush()
            sample_number += 1


def read_events(path: Path) -> list[dict[str, object]]:
    events = []
    if not path.is_file():
        return events
    with path.open(encoding='utf-8') as event_file:
        for line in event_file:
            try:
                events.append(json.loads(line))
            except (json.JSONDecodeError, TypeError):
                continue
    return events


def format_timestamp(timestamp_ms: int | None) -> str:
    if timestamp_ms is None:
        return '—'
    return datetime.fromtimestamp(timestamp_ms / 1000, timezone.utc).strftime(
        '%Y-%m-%d %H:%M:%S.%f'
    )[:-3] + 'Z'


def summarize_metric_points(points: list[dict[str, object]], is_gauge: bool) -> dict[str, object]:
    points = sorted(points, key=lambda point: int(point['sample']))
    values = [int(point['value']) for point in points]
    if is_gauge:
        maximum_index = max(range(len(values)), key=values.__getitem__)
        return {
            'window': 'gauge',
            'peak': values[maximum_index],
            'peak_time': int(points[maximum_index]['timestamp_ms']),
            'status': 'ok',
            'spikes': [],
        }
    deltas = []
    reset = False
    for previous, current in zip(points, points[1:]):
        delta = int(current['value']) - int(previous['value'])
        if delta < 0:
            reset = True
            continue
        deltas.append((delta, int(current['timestamp_ms'])))
    peak = max(deltas, default=(0, None), key=lambda item: item[0])
    return {
        'window': sum(delta for delta, _timestamp in deltas),
        'peak': peak[0],
        'peak_time': peak[1],
        'status': 'counter reset' if reset else 'ok',
        'spikes': deltas,
    }


def correlation_rows(
    metric_events: list[dict[str, object]], probe_path: Path | None
) -> list[str]:
    if probe_path is None:
        return []
    failures = [
        event
        for event in read_events(probe_path)
        if event.get('type') == 'probe' and event.get('result') != 'ok'
    ]
    if not failures or not metric_events:
        return []
    by_series: dict[tuple[str, str, str], list[dict[str, object]]] = defaultdict(list)
    for event in metric_events:
        if event.get('metric') in CORRELATION_METRICS:
            by_series[(
                str(event['source']),
                str(event['identity']),
                str(event['metric']),
            )].append(event)
    rows = []
    for failure in failures[:20]:
        timestamp_ms = int(failure['timestamp_ms'])
        increases = []
        following_distance = None
        for (source, _identity, metric), points in by_series.items():
            ordered = sorted(points, key=lambda point: int(point['timestamp_ms']))
            following_index = next(
                (
                    index
                    for index, point in enumerate(ordered)
                    if int(point['timestamp_ms']) >= timestamp_ms
                ),
                None,
            )
            if following_index is None or following_index == 0:
                continue
            distance = int(ordered[following_index]['timestamp_ms']) - timestamp_ms
            if distance > 10_000:
                continue
            following_distance = distance if following_distance is None else min(
                following_distance, distance
            )
            delta = int(ordered[following_index]['value']) - int(
                ordered[following_index - 1]['value']
            )
            if delta > 0:
                increases.append(f'{source} {metric} +{delta}')
        pair_path = f'{failure.get("pair", "unknown")}/{failure.get("path", "unknown")}'
        rows.append(
            f'| {format_timestamp(timestamp_ms)} | {pair_path} | '
            f'{failure.get("result", "unknown")} | '
            f'{following_distance if following_distance is not None else "—"} ms | '
            f'{"; ".join(increases[:8]) if increases else "no nearby counter increase"} |'
        )
    return rows


def build_report(path: Path, probe_path: Path | None = None) -> str:
    events = read_events(path)
    meta = next((event for event in events if event.get('type') == 'meta'), {})
    metrics = [event for event in events if event.get('type') == 'metric']
    statuses = [event for event in events if event.get('type') == 'status']
    lines = ['', '## Test-window network counter time series', '']
    if not metrics:
        lines.append('_No runner/Kind network counter samples were collected._')
        return '\n'.join(lines) + '\n'
    sample_count = len({int(event['sample']) for event in metrics})
    lines.append(
        f'_Collected {sample_count} samples at approximately '
        f'{meta.get("interval_seconds", "?")} second intervals; Kind nodes were '
        f'{", ".join(meta.get("nodes", [])) or "unavailable"}._'
    )
    lines.extend([
        '',
        '| Source | Identity | Metric | Samples | Window delta/type | Peak interval delta/value | Peak time | Status |',
        '|---|---|---|---:|---:|---:|---|---|',
    ])
    grouped: dict[tuple[str, str, str], list[dict[str, object]]] = defaultdict(list)
    for event in metrics:
        grouped[(
            str(event['source']),
            str(event['identity']),
            str(event['metric']),
        )].append(event)
    identities_by_source: dict[str, set[str]] = defaultdict(set)
    spike_rows = []
    for (source, identity, metric), points in sorted(grouped.items()):
        identities_by_source[source].add(identity)
        summary = summarize_metric_points(points, metric in GAUGE_METRICS)
        status = str(summary['status'])
        lines.append(
            f'| {source} | `{identity[:24]}` | {metric} | {len(points)} | '
            f'{summary["window"]} | {summary["peak"]} | '
            f'{format_timestamp(summary["peak_time"])} | {status} |'
        )
        if metric in CORRELATION_METRICS:
            for delta, timestamp_ms in summary['spikes']:
                if delta > 0:
                    spike_rows.append((delta, timestamp_ms, source, metric))
    replaced_sources = [
        source for source, identities in identities_by_source.items() if len(identities) > 1
    ]
    if replaced_sources:
        lines.append(
            '\n_Node/container network namespace replacement detected for: '
            + ', '.join(sorted(replaced_sources))
            + '; deltas are segmented by identity._'
        )
    if statuses:
        status_counts: dict[tuple[str, str], int] = defaultdict(int)
        for event in statuses:
            status_counts[(str(event['source']), str(event['status']))] += 1
        lines.extend(['', '### Sampling gaps', '', '| Source | Status | Samples |', '|---|---|---:|'])
        for (source, status), count in sorted(status_counts.items()):
            lines.append(f'| {source} | {status} | {count} |')
    if spike_rows:
        lines.extend([
            '',
            '### Largest counter increases between samples',
            '',
            '| Timestamp | Source | Metric | Increase |',
            '|---|---|---|---:|',
        ])
        for delta, timestamp_ms, source, metric in sorted(
            spike_rows, reverse=True
        )[:20]:
            lines.append(
                f'| {format_timestamp(timestamp_ms)} | {source} | {metric} | +{delta} |'
            )
    correlations = correlation_rows(metrics, probe_path)
    if correlations:
        lines.extend([
            '',
            '### Probe failures and following counter sample',
            '',
            '| Timestamp | Probe | Result | Following sample | Counter increases |',
            '|---|---|---|---:|---|',
            *correlations,
        ])
    lines.append('')
    return '\n'.join(lines)


def publish(report: str) -> None:
    print(report, end='')
    summary_path = os.environ.get('GITHUB_STEP_SUMMARY')
    if summary_path:
        with open(summary_path, 'a', encoding='utf-8') as summary:
            summary.write(report)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)
    sample_parser = subparsers.add_parser('sample')
    sample_parser.add_argument('--output', type=Path, required=True)
    sample_parser.add_argument('--interval', type=float, default=5.0)
    sample_parser.add_argument('--max-samples', type=int, default=0)
    report_parser = subparsers.add_parser('report')
    report_parser.add_argument('--input', type=Path, required=True)
    report_parser.add_argument('--probe-input', type=Path)
    return parser.parse_args()


def main() -> None:
    arguments = parse_args()
    if arguments.command == 'sample':
        signal.signal(signal.SIGTERM, lambda _signum, _frame: STOP_EVENT.set())
        signal.signal(signal.SIGINT, lambda _signum, _frame: STOP_EVENT.set())
        sample(arguments.output, arguments.interval, arguments.max_samples)
    else:
        publish(build_report(arguments.input, arguments.probe_input))


if __name__ == '__main__':
    main()

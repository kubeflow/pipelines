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
import sys
import threading
import time

import service_path_probe


COUNTER_METRICS = {
    'cgroup.cpu.nr_periods',
    'cgroup.cpu.nr_throttled',
    'cgroup.cpu.throttled_us',
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
    'psi.cpu.full_stall_us',
    'psi.cpu.some_stall_us',
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
    'listen_queue.listeners',
    'listen_queue.rx',
    'listen_queue.tx',
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


def parse_pressure(text: str) -> dict[str, int]:
    metrics = {}
    for line in text.splitlines():
        fields = line.split()
        if not fields or fields[0] not in {'some', 'full'}:
            continue
        for field in fields[1:]:
            key, separator, value = field.partition('=')
            if key == 'total' and separator and value.isdecimal():
                metrics[f'psi.cpu.{fields[0]}_stall_us'] = int(value)
                break
    return metrics


def parse_cpu_stat(text: str) -> dict[str, int]:
    metrics = {}
    for line in text.splitlines():
        fields = line.split()
        if len(fields) != 2 or not fields[1].isdecimal():
            continue
        key, value = fields[0], int(fields[1])
        if key in {'nr_periods', 'nr_throttled'}:
            metrics[f'cgroup.cpu.{key}'] = value
        elif key == 'throttled_usec':
            metrics['cgroup.cpu.throttled_us'] = value
        elif key == 'throttled_time':
            metrics['cgroup.cpu.throttled_us'] = value // 1000
    return metrics


def current_cpu_stat_path(proc_root: Path, cgroup_root: Path) -> Path | None:
    try:
        cgroup_lines = (proc_root / 'self' / 'cgroup').read_text(
            encoding='utf-8'
        ).splitlines()
    except OSError:
        cgroup_lines = []
    candidates = []
    for line in cgroup_lines:
        fields = line.split(':', 2)
        if len(fields) != 3:
            continue
        hierarchy, controllers, cgroup_path = fields
        relative_path = cgroup_path.lstrip('/')
        if hierarchy == '0':
            candidates.append(cgroup_root / relative_path / 'cpu.stat')
        elif 'cpu' in controllers.split(','):
            candidates.extend([
                cgroup_root / 'cpu,cpuacct' / relative_path / 'cpu.stat',
                cgroup_root / 'cpu' / relative_path / 'cpu.stat',
            ])
    candidates.append(cgroup_root / 'cpu.stat')
    return next((path for path in candidates if path.is_file()), None)


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


def parse_listen_queues(text: str, port: int) -> dict[str, int]:
    """Parse aggregate LISTEN socket queues for one port from /proc/net/tcp*."""
    listeners = receive_queue = transmit_queue = 0
    valid_table = False
    for line in text.splitlines():
        fields = line.split()
        if 'local_address' in fields and 'st' in fields:
            valid_table = True
            continue
        if len(fields) < 5 or fields[3] != '0A':
            continue
        try:
            local_port = int(fields[1].rsplit(':', 1)[1], 16)
            transmit_hex, receive_hex = fields[4].split(':', 1)
            transmit = int(transmit_hex, 16)
            receive = int(receive_hex, 16)
        except (IndexError, ValueError):
            continue
        if local_port == port:
            listeners += 1
            transmit_queue += transmit
            receive_queue += receive
    if not valid_table:
        return {}
    return {
        'listen_queue.listeners': listeners,
        'listen_queue.rx': receive_queue,
        'listen_queue.tx': transmit_queue,
    }


def parse_target_output(text: str, port: int) -> tuple[str, dict[str, int]]:
    """Parse a target-pod network snapshot emitted by the shell collector."""
    sections = split_sections(text)
    pod = sections.get('pod', '').strip() or 'unavailable'
    network_namespace = sections.get('netns', '').strip() or 'unavailable'
    metrics = {}
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
    metrics.update(
        parse_listen_queues(
            sections.get('tcp', '') + '\n' + sections.get('tcp6', ''), port
        )
    )
    return f'{pod}|{network_namespace}', metrics


def host_metrics(proc_root: Path, cgroup_root: Path) -> dict[str, int]:
    metrics = {}
    try:
        metrics.update(
            parse_softnet(
                (proc_root / 'net' / 'softnet_stat').read_text(encoding='utf-8')
            )
        )
    except OSError:
        pass
    try:
        metrics.update(
            parse_pressure(
                (proc_root / 'pressure' / 'cpu').read_text(encoding='utf-8')
            )
        )
    except OSError:
        pass
    cpu_stat_path = current_cpu_stat_path(proc_root, cgroup_root)
    if cpu_stat_path is not None:
        try:
            metrics.update(parse_cpu_stat(cpu_stat_path.read_text(encoding='utf-8')))
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


def capture_sample(
    nodes: list[str],
    proc_root: Path,
    cgroup_root: Path,
    sample_number: int,
) -> list[dict[str, object]]:
    timestamp_ms = time.time_ns() // 1_000_000
    events = [
        metric_event(timestamp_ms, sample_number, 'runner-host', 'host', metric, value)
        for metric, value in host_metrics(proc_root, cgroup_root).items()
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
    cgroup_root = Path(
        os.environ.get('NETWORK_WINDOW_CGROUP_ROOT', '/sys/fs/cgroup')
    )
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
            for event in capture_sample(
                nodes, proc_root, cgroup_root, sample_number
            ):
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


def wait_for_following_sample(
    network_path: Path, probe_path: Path, timeout_seconds: float
) -> tuple[bool, str]:
    probe_timestamps = [
        int(event['timestamp_ms'])
        for event in read_events(probe_path)
        if event.get('type') == 'probe' and 'timestamp_ms' in event
    ]
    if not probe_timestamps:
        return False, 'unavailable: no probe timestamp to bound the final sample'
    final_probe_timestamp = max(probe_timestamps)
    deadline = time.monotonic() + timeout_seconds
    while True:
        metric_timestamps = [
            int(event['timestamp_ms'])
            for event in read_events(network_path)
            if event.get('type') == 'metric' and 'timestamp_ms' in event
        ]
        if metric_timestamps and max(metric_timestamps) > final_probe_timestamp:
            return (
                True,
                'confirmed: counter sample follows final probe timestamp '
                f'({max(metric_timestamps)} > {final_probe_timestamp})',
            )
        if time.monotonic() >= deadline:
            return (
                False,
                'unavailable: no counter sample followed the final probe '
                f'timestamp within {timeout_seconds:g}s',
            )
        time.sleep(0.1)


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
    # Collapse the VIP/Endpoint results from one pair/cycle so they do not
    # consume two rows, then retain each pair's onset plus the failures with
    # the strongest nearby counter increases. Chronological first-20 sampling
    # hid the later, high-signal conntrack spikes in the settled failures.
    grouped_failures: dict[tuple[int, str], list[dict[str, object]]] = defaultdict(list)
    for failure in failures:
        grouped_failures[(
            int(failure['timestamp_ms']), str(failure.get('pair', 'unknown'))
        )].append(failure)
    candidates = []
    for (timestamp_ms, pair), grouped in sorted(grouped_failures.items()):
        increases = []
        correlation_score = 0
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
                correlation_score += delta
        paths = '+'.join(sorted(str(failure.get('path', 'unknown')) for failure in grouped))
        results = '+'.join(
            sorted(
                set(
                    str(failure.get('result', 'unknown'))
                    for failure in grouped
                )
            )
        )
        candidates.append({
            'timestamp_ms': timestamp_ms,
            'pair': pair,
            'paths': paths,
            'results': results,
            'following_distance': following_distance,
            'increases': increases,
            'score': correlation_score,
        })

    selected = []
    seen_pairs = set()
    for candidate in candidates:
        if candidate['pair'] not in seen_pairs:
            selected.append(candidate)
            seen_pairs.add(candidate['pair'])
    selected_keys = {
        (candidate['timestamp_ms'], candidate['pair']) for candidate in selected
    }
    for candidate in sorted(
        candidates,
        key=lambda item: (-int(item['score']), int(item['timestamp_ms'])),
    ):
        key = (candidate['timestamp_ms'], candidate['pair'])
        if key not in selected_keys:
            selected.append(candidate)
            selected_keys.add(key)
        if len(selected) >= 20:
            break

    rows = []
    for candidate in sorted(selected, key=lambda item: int(item['timestamp_ms'])):
        following_distance = candidate['following_distance']
        increases = candidate['increases']
        rows.append(
            f'| {format_timestamp(int(candidate["timestamp_ms"]))} | '
            f'{candidate["pair"]}/{candidate["paths"]} | '
            f'{candidate["results"]} | '
            f'{following_distance if following_distance is not None else "—"} ms | '
            f'{"; ".join(increases[:8]) if increases else "no nearby counter increase"} |'
        )
    return rows


def build_target_report(
    baseline_path: Path, end_path: Path, port: int, label: str
) -> str:
    lines = ['', f'## {label} target network-namespace snapshots', '']
    if not baseline_path.is_file() or not end_path.is_file():
        lines.append('_Target network snapshot unavailable._')
        return '\n'.join(lines) + '\n'
    baseline_identity, baseline = parse_target_output(
        baseline_path.read_text(encoding='utf-8'), port
    )
    end_identity, end = parse_target_output(end_path.read_text(encoding='utf-8'), port)
    if not baseline or not end or 'unavailable' in {
        baseline_identity.split('|')[-1],
        end_identity.split('|')[-1],
    }:
        lines.append('_Target network counters unavailable._')
        return '\n'.join(lines) + '\n'
    lines.append(
        f'_Baseline identity `{baseline_identity}`; end identity `{end_identity}`._'
    )
    if baseline_identity != end_identity:
        lines.append(
            '_Pod/network namespace replacement detected; counter deltas are unavailable._'
        )
    queue_metrics = {
        'listen_queue.listeners',
        'listen_queue.rx',
        'listen_queue.tx',
    }
    if not queue_metrics.intersection(baseline) or not queue_metrics.intersection(end):
        lines.append(
            '_Listener queue table unavailable in at least one target snapshot._'
        )
    lines.extend([
        '',
        '| Metric | Baseline | End | Delta/type | Status |',
        '|---|---:|---:|---:|---|',
    ])
    for metric in sorted(set(baseline) | set(end)):
        before = baseline.get(metric)
        after = end.get(metric)
        if before is None or after is None:
            lines.append(
                f'| {metric} | {before if before is not None else "—"} | '
                f'{after if after is not None else "—"} | — | unavailable |'
            )
        elif baseline_identity != end_identity:
            lines.append(f'| {metric} | {before} | {after} | — | identity changed |')
        elif metric in GAUGE_METRICS:
            lines.append(f'| {metric} | {before} | {after} | gauge | ok |')
        elif after < before:
            lines.append(f'| {metric} | {before} | {after} | — | counter reset |')
        else:
            lines.append(f'| {metric} | {before} | {after} | {after - before} | ok |')
    lines.append('')
    return '\n'.join(lines)


def build_report(path: Path, probe_path: Path | None = None) -> str:
    events = read_events(path)
    meta = next((event for event in events if event.get('type') == 'meta'), {})
    metrics = [event for event in events if event.get('type') == 'metric']
    statuses = [event for event in events if event.get('type') == 'status']
    lines = ['', '## Test-window network and contention counter time series', '']
    if not metrics:
        lines.append(
            '_No runner/Kind network or contention counter samples were collected._'
        )
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
    target_report_parser = subparsers.add_parser('target-report')
    target_report_parser.add_argument('--baseline', type=Path, required=True)
    target_report_parser.add_argument('--end', type=Path, required=True)
    target_report_parser.add_argument('--port', type=int, required=True)
    target_report_parser.add_argument('--label', required=True)
    wait_parser = subparsers.add_parser('wait-for-following-sample')
    wait_parser.add_argument('--input', type=Path, required=True)
    wait_parser.add_argument('--probe-input', type=Path, required=True)
    wait_parser.add_argument('--timeout', type=float, default=8.0)
    return parser.parse_args()


def main() -> None:
    arguments = parse_args()
    if arguments.command == 'sample':
        signal.signal(signal.SIGTERM, lambda _signum, _frame: STOP_EVENT.set())
        signal.signal(signal.SIGINT, lambda _signum, _frame: STOP_EVENT.set())
        sample(arguments.output, arguments.interval, arguments.max_samples)
    elif arguments.command == 'report':
        publish(build_report(arguments.input, arguments.probe_input))
    elif arguments.command == 'target-report':
        publish(
            build_target_report(
                arguments.baseline,
                arguments.end,
                arguments.port,
                arguments.label,
            )
        )
    else:
        success, message = wait_for_following_sample(
            arguments.input,
            arguments.probe_input,
            arguments.timeout,
        )
        print(message)
        if not success:
            sys.exit(1)


if __name__ == '__main__':
    main()

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
"""Continuously compare Service VIP and direct Endpoint TCP connectivity."""

from __future__ import annotations

import argparse
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import json
import math
import os
from pathlib import Path
import signal
import socket
import sys
import threading
import time
from typing import Iterable


SELECTED_COUNTERS = {
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
}
SELECTED_GAUGES = {
    'sockstat.sockets.used',
    'sockstat.TCP.alloc',
    'sockstat.TCP.inuse',
    'sockstat.TCP.orphan',
    'sockstat.TCP.tw',
}
STOP_EVENT = threading.Event()


def load_targets(path: Path) -> list[dict[str, object]]:
    targets = []
    with path.open(encoding='utf-8') as target_file:
        for line in target_file:
            if not line.strip() or line.startswith('#'):
                continue
            fields = line.rstrip().split('\t')
            if len(fields) != 4 or fields[1] not in {'vip', 'endpoint'}:
                continue
            try:
                port = int(fields[3])
            except ValueError:
                continue
            targets.append({
                'pair': fields[0],
                'path': fields[1],
                'host': fields[2],
                'port': port,
            })
    return targets


def resolve_pair(
    pair: str,
    service: dict[str, object],
    endpoint_slices: dict[str, object],
    port_name: str,
) -> list[tuple[str, str, str, int]]:
    """Resolve one immutable VIP/Endpoint pair from Kubernetes API objects."""
    service_spec = service.get('spec', {})
    if not isinstance(service_spec, dict):
        return []
    cluster_ip = service_spec.get('clusterIP')
    service_port = None
    for port in service_spec.get('ports', []):
        if isinstance(port, dict) and port.get('name') == port_name:
            service_port = port.get('port')
            break
    if not isinstance(cluster_ip, str) or not isinstance(service_port, int):
        return []

    for endpoint, endpoint_port in ready_endpoint_candidates(
        endpoint_slices, port_name
    ):
        addresses = endpoint.get('addresses', [])
        if addresses and isinstance(addresses[0], str):
            return [
                (pair, 'vip', cluster_ip, service_port),
                (pair, 'endpoint', addresses[0], endpoint_port),
            ]
    return []


def ready_endpoint_candidates(
    endpoint_slices: dict[str, object], port_name: str
) -> Iterable[tuple[dict[str, object], int]]:
    """Yield ready, serving, non-terminating endpoints for a named port."""
    items = endpoint_slices.get('items', [])
    if not isinstance(items, list):
        return
    for endpoint_slice in sorted(
        (item for item in items if isinstance(item, dict)),
        key=lambda item: str(item.get('metadata', {}).get('name', '')),
    ):
        endpoint_port = next(
            (
                port.get('port')
                for port in endpoint_slice.get('ports', [])
                if isinstance(port, dict) and port.get('name') == port_name
            ),
            None,
        )
        if not isinstance(endpoint_port, int):
            continue
        for endpoint in endpoint_slice.get('endpoints', []):
            if not isinstance(endpoint, dict):
                continue
            conditions = endpoint.get('conditions', {})
            if not isinstance(conditions, dict):
                conditions = {}
            if (
                conditions.get('ready') is False
                or conditions.get('serving') is False
                or conditions.get('terminating') is True
            ):
                continue
            yield endpoint, endpoint_port


def resolve_ready_target_pod(
    endpoint_slices: dict[str, object], port_name: str
) -> str | None:
    """Resolve the Pod backing the same ready EndpointSlice port we probe."""
    for endpoint, _endpoint_port in ready_endpoint_candidates(
        endpoint_slices, port_name
    ):
        target_ref = endpoint.get('targetRef', {})
        if (
            isinstance(target_ref, dict)
            and target_ref.get('kind') == 'Pod'
            and isinstance(target_ref.get('name'), str)
        ):
            return str(target_ref['name'])
    return None


def resolve_from_files(
    pair: str, service_path: Path, endpoint_slices_path: Path, port_name: str
) -> None:
    try:
        service = json.loads(service_path.read_text(encoding='utf-8'))
        endpoint_slices = json.loads(
            endpoint_slices_path.read_text(encoding='utf-8')
        )
    except (OSError, json.JSONDecodeError):
        return
    for target in resolve_pair(pair, service, endpoint_slices, port_name):
        print(*target, sep='\t')


def resolve_pod_from_file(endpoint_slices_path: Path, port_name: str) -> None:
    try:
        endpoint_slices = json.loads(
            endpoint_slices_path.read_text(encoding='utf-8')
        )
    except (OSError, json.JSONDecodeError):
        return
    pod_name = resolve_ready_target_pod(endpoint_slices, port_name)
    if pod_name:
        print(pod_name)


def parse_proc_table(text: str) -> dict[str, int]:
    """Parse paired header/value rows from /proc/net/{netstat,snmp}."""
    rows = [line.split() for line in text.splitlines() if line.strip()]
    values: dict[str, int] = {}
    for header, data in zip(rows[0::2], rows[1::2]):
        if not header or not data or header[0] != data[0]:
            continue
        section = header[0].rstrip(':')
        for key, value in zip(header[1:], data[1:]):
            try:
                values[f'{section}.{key}'] = int(value)
            except ValueError:
                continue
    return values


def parse_sockstat(text: str) -> dict[str, int]:
    values: dict[str, int] = {}
    for line in text.splitlines():
        fields = line.split()
        if len(fields) < 3:
            continue
        section = fields[0].rstrip(':')
        for index in range(1, len(fields) - 1, 2):
            try:
                values[f'sockstat.{section}.{fields[index]}'] = int(
                    fields[index + 1]
                )
            except ValueError:
                continue
    return values


def snapshot_network_metrics(proc_root: Path = Path('/proc')) -> dict[str, int]:
    values: dict[str, int] = {}
    for name in ('netstat', 'snmp'):
        try:
            values.update(
                parse_proc_table((proc_root / 'net' / name).read_text(encoding='utf-8'))
            )
        except OSError:
            pass
    try:
        values.update(
            parse_sockstat(
                (proc_root / 'net' / 'sockstat').read_text(encoding='utf-8')
            )
        )
    except OSError:
        pass
    return {
        key: value
        for key, value in values.items()
        if key in SELECTED_COUNTERS or key in SELECTED_GAUGES
    }


def probe_target(target: dict[str, object], timeout_seconds: float) -> dict[str, object]:
    started_ns = time.monotonic_ns()
    result = 'ok'
    error_number = None
    error_type = None
    try:
        connection = socket.create_connection(
            (str(target['host']), int(target['port'])), timeout=timeout_seconds
        )
        connection.close()
    except socket.timeout as error:
        result = 'timeout'
        error_number = getattr(error, 'errno', None)
        error_type = type(error).__name__
    except OSError as error:
        result = 'error'
        error_number = error.errno
        error_type = type(error).__name__
    return {
        **target,
        'result': result,
        'errno': error_number,
        'error_type': error_type,
        'latency_ms': round((time.monotonic_ns() - started_ns) / 1_000_000, 3),
    }


def write_event(event: dict[str, object]) -> None:
    print(json.dumps(event, sort_keys=True), flush=True)


def sample(
    targets_path: Path,
    interval_seconds: float,
    timeout_seconds: float,
    max_cycles: int,
) -> None:
    targets = load_targets(targets_path)
    try:
        network_namespace = os.readlink('/proc/self/ns/net')
    except OSError:
        network_namespace = 'unavailable'
    write_event({
        'type': 'meta',
        'version': 1,
        'source_pod': os.environ.get('POD_NAME', os.environ.get('HOSTNAME', 'unknown')),
        'source_pod_ip': os.environ.get('POD_IP', 'unknown'),
        'source_node': os.environ.get('NODE_NAME', 'unknown'),
        'network_namespace': network_namespace,
        'interval_seconds': interval_seconds,
        'timeout_seconds': timeout_seconds,
        'targets': targets,
    })
    if not targets:
        return

    started_ns = time.monotonic_ns()
    cycle = 0
    with ThreadPoolExecutor(max_workers=len(targets)) as executor:
        while not STOP_EVENT.is_set() and (max_cycles <= 0 or cycle < max_cycles):
            scheduled_ns = started_ns + int(cycle * interval_seconds * 1_000_000_000)
            remaining_seconds = (scheduled_ns - time.monotonic_ns()) / 1_000_000_000
            if remaining_seconds > 0 and STOP_EVENT.wait(remaining_seconds):
                break
            actual_ns = time.monotonic_ns()
            timestamp_ms = time.time_ns() // 1_000_000
            write_event({
                'type': 'cycle',
                'cycle': cycle,
                'timestamp_ms': timestamp_ms,
                'loop_lag_ms': round(max(0, actual_ns - scheduled_ns) / 1_000_000, 3),
            })
            for metric, value in snapshot_network_metrics().items():
                write_event({
                    'type': 'metric',
                    'cycle': cycle,
                    'timestamp_ms': timestamp_ms,
                    'metric': metric,
                    'value': value,
                })
            futures = [
                executor.submit(probe_target, target, timeout_seconds)
                for target in targets
            ]
            for future in futures:
                write_event({
                    'type': 'probe',
                    'cycle': cycle,
                    'timestamp_ms': timestamp_ms,
                    **future.result(),
                })
            cycle += 1


def percentile(values: list[float], percentage: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, math.ceil(percentage * len(ordered)) - 1)
    return ordered[index]


def longest_failure_streak(records: Iterable[dict[str, object]]) -> int:
    longest = current = 0
    for record in sorted(records, key=lambda item: int(item['cycle'])):
        if record['result'] == 'ok':
            current = 0
        else:
            current += 1
            longest = max(longest, current)
    return longest


def format_timestamp(timestamp_ms: int | None) -> str:
    if timestamp_ms is None:
        return '—'
    return datetime.fromtimestamp(timestamp_ms / 1000, timezone.utc).strftime(
        '%Y-%m-%d %H:%M:%S.%f'
    )[:-3] + 'Z'


def metric_summary(records: list[dict[str, object]]) -> list[str]:
    grouped: dict[str, list[dict[str, object]]] = defaultdict(list)
    for record in records:
        grouped[str(record['metric'])].append(record)
    rows = []
    for metric in sorted(grouped):
        points = sorted(grouped[metric], key=lambda item: int(item['cycle']))
        values = [int(point['value']) for point in points]
        if metric in SELECTED_GAUGES:
            rows.append(
                f'| {metric} | {len(values)} | gauge | {max(values)} | — | ok |'
            )
            continue
        deltas = []
        reset = False
        peak_time = None
        for previous, current in zip(points, points[1:]):
            delta = int(current['value']) - int(previous['value'])
            if delta < 0:
                reset = True
                continue
            deltas.append(delta)
            if peak_time is None or delta > peak_time[0]:
                peak_time = (delta, int(current['timestamp_ms']))
        rows.append(
            f'| {metric} | {len(values)} | {sum(deltas)} | '
            f'{max(deltas, default=0)} | '
            f'{format_timestamp(peak_time[1] if peak_time else None)} | '
            f'{"counter reset" if reset else "ok"} |'
        )
    return rows


def build_report(path: Path) -> str:
    events = []
    if path.is_file():
        with path.open(encoding='utf-8') as event_file:
            for line in event_file:
                try:
                    events.append(json.loads(line))
                except (json.JSONDecodeError, TypeError):
                    continue
    meta = next((event for event in events if event.get('type') == 'meta'), {})
    probes = [event for event in events if event.get('type') == 'probe']
    metrics = [event for event in events if event.get('type') == 'metric']
    cycles = [event for event in events if event.get('type') == 'cycle']

    lines = ['', '## Service VIP versus direct Endpoint probes', '']
    if not probes:
        lines.append('_No in-cluster Service-path probe samples were collected._')
        return '\n'.join(lines) + '\n'
    lines.append(
        f'_Source: pod `{meta.get("source_pod", "unknown")}` on node '
        f'`{meta.get("source_node", "unknown")}`, pod IP '
        f'`{meta.get("source_pod_ip", "unknown")}`, netns '
        f'`{meta.get("network_namespace", "unknown")}`._'
    )
    if cycles:
        max_lag = max(float(cycle.get('loop_lag_ms', 0)) for cycle in cycles)
        lines.append(
            f'_Collected {len(cycles)} cycles; maximum sampler scheduling lag '
            f'was {max_lag:.1f} ms._'
        )
    lines.extend([
        '',
        '| Pair | Path | Address | Attempts | Failures | Timeouts | p50 connect | p95 connect | Longest failure streak |',
        '|---|---|---|---:|---:|---:|---:|---:|---:|',
    ])
    by_target: dict[tuple[str, str], list[dict[str, object]]] = defaultdict(list)
    for probe in probes:
        by_target[(str(probe['pair']), str(probe['path']))].append(probe)
    for (pair, path_name), records in sorted(by_target.items()):
        successful_latencies = [
            float(record['latency_ms'])
            for record in records
            if record['result'] == 'ok'
        ]
        failures = sum(record['result'] != 'ok' for record in records)
        timeouts = sum(record['result'] == 'timeout' for record in records)
        p50 = percentile(successful_latencies, 0.50)
        p95 = percentile(successful_latencies, 0.95)
        address = f'{records[0]["host"]}:{records[0]["port"]}'
        lines.append(
            f'| {pair} | {path_name} | `{address}` | {len(records)} | '
            f'{failures} | {timeouts} | '
            f'{f"{p50:.1f} ms" if p50 is not None else "—"} | '
            f'{f"{p95:.1f} ms" if p95 is not None else "—"} | '
            f'{longest_failure_streak(records)} |'
        )

    by_cycle_pair: dict[tuple[int, str], dict[str, str]] = defaultdict(dict)
    timestamps: dict[tuple[int, str], int] = {}
    for probe in probes:
        key = (int(probe['cycle']), str(probe['pair']))
        by_cycle_pair[key][str(probe['path'])] = str(probe['result'])
        timestamps[key] = int(probe['timestamp_ms'])
    classifications: dict[str, dict[str, int]] = defaultdict(
        lambda: defaultdict(int)
    )
    failure_windows = []
    for key, paths in sorted(by_cycle_pair.items()):
        vip = paths.get('vip')
        endpoint = paths.get('endpoint')
        if vip is None or endpoint is None:
            classification = 'incomplete'
        elif vip != 'ok' and endpoint == 'ok':
            classification = 'VIP failed / Endpoint succeeded'
        elif vip == 'ok' and endpoint != 'ok':
            classification = 'VIP succeeded / Endpoint failed'
        elif vip != 'ok' and endpoint != 'ok':
            classification = 'both failed'
        else:
            classification = 'both succeeded'
        classifications[key[1]][classification] += 1
        if classification != 'both succeeded':
            failure_windows.append((timestamps[key], key[1], classification))
    lines.extend([
        '',
        '### Paired outcomes',
        '',
        '| Pair | Outcome | Cycles | Interpretation |',
        '|---|---|---:|---|',
    ])
    interpretations = {
        'VIP failed / Endpoint succeeded': 'Service/netfilter path isolated',
        'both failed': 'Shared source/host/backend path',
        'VIP succeeded / Endpoint failed': 'Direct route, endpoint, or policy path',
        'both succeeded': 'No failure observed in that cycle',
        'incomplete': 'Probe result unavailable',
    }
    for pair in sorted(classifications):
        for classification, interpretation in interpretations.items():
            lines.append(
                f'| {pair} | {classification} | '
                f'{classifications[pair][classification]} | {interpretation} |'
            )
    if failure_windows:
        lines.extend([
            '',
            '### First paired failure windows',
            '',
            '| Timestamp | Pair | Outcome |',
            '|---|---|---|',
        ])
        for timestamp_ms, pair, classification in failure_windows[:20]:
            lines.append(
                f'| {format_timestamp(timestamp_ms)} | {pair} | {classification} |'
            )
    if metrics:
        lines.extend([
            '',
            '### Probe-netns network counter deltas',
            '',
            '| Metric | Samples | Window delta/type | Peak interval delta/value | Peak time | Status |',
            '|---|---:|---:|---:|---|---|',
            *metric_summary(metrics),
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
    sample_parser.add_argument('--targets', type=Path, required=True)
    sample_parser.add_argument('--interval', type=float, default=5.0)
    sample_parser.add_argument('--timeout', type=float, default=1.0)
    sample_parser.add_argument('--max-cycles', type=int, default=0)
    report_parser = subparsers.add_parser('report')
    report_parser.add_argument('--input', type=Path, required=True)
    resolve_parser = subparsers.add_parser('resolve')
    resolve_parser.add_argument('--pair', required=True)
    resolve_parser.add_argument('--service', type=Path, required=True)
    resolve_parser.add_argument('--endpoint-slices', type=Path, required=True)
    resolve_parser.add_argument('--port-name', required=True)
    resolve_pod_parser = subparsers.add_parser('resolve-pod')
    resolve_pod_parser.add_argument('--endpoint-slices', type=Path, required=True)
    resolve_pod_parser.add_argument('--port-name', required=True)
    return parser.parse_args()


def main() -> None:
    arguments = parse_args()
    if arguments.command == 'sample':
        signal.signal(signal.SIGTERM, lambda _signum, _frame: STOP_EVENT.set())
        signal.signal(signal.SIGINT, lambda _signum, _frame: STOP_EVENT.set())
        sample(
            arguments.targets,
            arguments.interval,
            arguments.timeout,
            arguments.max_cycles,
        )
    elif arguments.command == 'report':
        publish(build_report(arguments.input))
    elif arguments.command == 'resolve':
        resolve_from_files(
            arguments.pair,
            arguments.service,
            arguments.endpoint_slices,
            arguments.port_name,
        )
    else:
        resolve_pod_from_file(
            arguments.endpoint_slices,
            arguments.port_name,
        )


if __name__ == '__main__':
    main()

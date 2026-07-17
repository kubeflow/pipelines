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
import re
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
    'kindnet.cgroup.cpu.nr_periods',
    'kindnet.cgroup.cpu.nr_throttled',
    'kindnet.cgroup.cpu.throttled_us',
    'kindnet.sched.runtime_ns',
    'kindnet.sched.timeslices',
    'kindnet.sched.wait_ns',
    'kindnet.nft.counter.bytes',
    'kindnet.nft.counter.packets',
    'nfqueue.id_sequence',
    'nfqueue.queue_dropped',
    'nfqueue.user_dropped',
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
    'kindnet.cgroup.cpu.period_us',
    'kindnet.cgroup.cpu.quota_us',
    'nfqueue.queue_total',
    *service_path_probe.SELECTED_GAUGES,
}
CORRELATION_METRICS = COUNTER_METRICS - {
    'conntrack.insert',
    'kindnet.cgroup.cpu.nr_periods',
    'kindnet.nft.counter.bytes',
    'kindnet.nft.counter.packets',
    'kindnet.sched.runtime_ns',
    'kindnet.sched.timeslices',
    'nfqueue.id_sequence',
    'softnet.processed',
}
STOP_EVENT = threading.Event()
KINDNET_TIMESTAMP_PATTERN = re.compile(
    r'\b(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z)\b'
)
KINDNET_ELAPSED_PATTERN = re.compile(
    r'elapsed="([0-9]+(?:\.[0-9]+)?)(ns|us|µs|ms|s)"'
)
KINDNET_LOG_SIGNALS = {
    'NFQUEUE verdict-send failure': 'failed to set verdict with label',
    'NFQUEUE receive failure': 'Could not receive message',
}
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

KINDNET_COMMAND = r'''
echo __NFQUEUE__
if test -r /proc/net/netfilter/nfnetlink_queue; then
  if test -s /proc/net/netfilter/nfnetlink_queue; then
    cat /proc/net/netfilter/nfnetlink_queue 2>/dev/null || echo status=unavailable
  else
    echo status=empty
  fi
else
  echo status=unavailable
fi

kindnet_id=""
kindnet_pid=""
kindnet_start_time=""
kindnet_cgroup_path=""
kindnet_cpu_stat=""
if command -v crictl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1 \
  && command -v timeout >/dev/null 2>&1; then
  kindnet_id=$(timeout 1 crictl ps --state Running --name kindnet-cni -q \
    2>/dev/null | head -n 1 || true)
  if test -n "$kindnet_id"; then
    kindnet_pid=$(timeout 1 crictl inspect "$kindnet_id" 2>/dev/null \
      | jq -r '.info.pid // empty' 2>/dev/null || true)
  fi
fi
if test -n "$kindnet_pid" && test -r "/proc/$kindnet_pid/stat"; then
  kindnet_start_time=$(awk '{print $22}' "/proc/$kindnet_pid/stat" \
    2>/dev/null || true)
  while IFS=: read -r hierarchy controllers cgroup_path; do
    if test "$hierarchy" = "0"; then
      candidate="/sys/fs/cgroup${cgroup_path}/cpu.stat"
      if test -r "$candidate"; then
        kindnet_cgroup_path="$cgroup_path"
        kindnet_cpu_stat="$candidate"
        break
      fi
    else
      case ",$controllers," in
        *,cpu,*)
          for cgroup_root in /sys/fs/cgroup/cpu,cpuacct /sys/fs/cgroup/cpu; do
            candidate="${cgroup_root}${cgroup_path}/cpu.stat"
            if test -r "$candidate"; then
              kindnet_cgroup_path="$cgroup_path"
              kindnet_cpu_stat="$candidate"
              break 2
            fi
          done
          ;;
      esac
    fi
  done < "/proc/$kindnet_pid/cgroup"
fi

echo __KINDNET_IDENTITY__
if test -n "$kindnet_id" && test -n "$kindnet_pid" \
  && test -n "$kindnet_start_time"; then
  printf 'container_id=%s pid=%s start_time=%s cgroup=%s\n' \
    "$kindnet_id" "$kindnet_pid" "$kindnet_start_time" \
    "${kindnet_cgroup_path:-unavailable}"
else
  echo status=unavailable
fi
echo __KINDNET_CPU_STAT__
if test -n "$kindnet_cpu_stat"; then
  cat "$kindnet_cpu_stat" 2>/dev/null || echo status=unavailable
  cpu_max=${kindnet_cpu_stat%/cpu.stat}/cpu.max
  if test -r "$cpu_max"; then
    read -r quota period < "$cpu_max" || true
    printf 'quota_us %s\nperiod_us %s\n' "$quota" "$period"
  else
    cgroup_directory=${kindnet_cpu_stat%/cpu.stat}
    quota=$(cat "$cgroup_directory/cpu.cfs_quota_us" 2>/dev/null || true)
    period=$(cat "$cgroup_directory/cpu.cfs_period_us" 2>/dev/null || true)
    printf 'quota_us %s\nperiod_us %s\n' "$quota" "$period"
  fi
else
  echo status=unavailable
fi
echo __KINDNET_SCHEDSTAT__
if test -n "$kindnet_pid" && test -d "/proc/$kindnet_pid/task"; then
  runtime_ns=0
  wait_ns=0
  timeslices=0
  threads=0
  for thread_stat in "/proc/$kindnet_pid"/task/*/schedstat; do
    test -r "$thread_stat" || continue
    read -r thread_runtime thread_wait thread_timeslices _ < "$thread_stat" \
      || continue
    case "$thread_runtime:$thread_wait:$thread_timeslices" in
      *[!0-9:]*) continue ;;
    esac
    runtime_ns=$((runtime_ns + thread_runtime))
    wait_ns=$((wait_ns + thread_wait))
    timeslices=$((timeslices + thread_timeslices))
    threads=$((threads + 1))
  done
  if test "$threads" -gt 0; then
    printf '%s %s %s\n' "$runtime_ns" "$wait_ns" "$timeslices"
  else
    echo status=unavailable
  fi
else
  echo status=unavailable
fi
echo __KINDNET_NFT__
if command -v nft >/dev/null 2>&1 && command -v timeout >/dev/null 2>&1; then
  timeout 0.5 nft -a -j list table inet kindnet-network-policies \
    2>/dev/null || echo status=unavailable
else
  echo status=unavailable
fi
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


def parse_cpu_stat(
    text: str, prefix: str = 'cgroup.cpu'
) -> dict[str, int]:
    metrics = {}
    for line in text.splitlines():
        fields = line.split()
        if len(fields) != 2 or not fields[1].isdecimal():
            continue
        key, value = fields[0], int(fields[1])
        if key in {'nr_periods', 'nr_throttled'}:
            metrics[f'{prefix}.{key}'] = value
        elif key == 'throttled_usec':
            metrics[f'{prefix}.throttled_us'] = value
        elif key == 'throttled_time':
            metrics[f'{prefix}.throttled_us'] = value // 1000
    return metrics


def parse_kindnet_cpu(text: str) -> dict[str, int]:
    if any(line.startswith('status=') for line in text.splitlines()):
        return {}
    metrics = parse_cpu_stat(text, 'kindnet.cgroup.cpu')
    for line in text.splitlines():
        fields = line.split()
        if len(fields) != 2 or not fields[1].isdecimal():
            continue
        if fields[0] in {'quota_us', 'period_us'}:
            metrics[f'kindnet.cgroup.cpu.{fields[0]}'] = int(fields[1])
    return metrics


def parse_kindnet_identity(text: str) -> tuple[str, str | None]:
    fields = {}
    for field in text.split():
        key, separator, value = field.partition('=')
        if separator:
            fields[key] = value
    if fields.get('status'):
        return 'unavailable', f'kindnet_{fields["status"]}'
    required = {'container_id', 'pid', 'start_time'}
    if not required.issubset(fields):
        return 'unavailable', 'kindnet_identity_unavailable'
    cgroup = fields.get('cgroup', 'unavailable')
    identity = '|'.join([fields['container_id'], fields['pid'], fields['start_time'], cgroup])
    return identity, (
        'kindnet_cgroup_unavailable' if cgroup == 'unavailable' else None
    )


def parse_schedstat(text: str) -> dict[str, int]:
    for line in text.splitlines():
        fields = line.split()
        if len(fields) >= 3 and all(field.isdecimal() for field in fields[:3]):
            return {
                'kindnet.sched.runtime_ns': int(fields[0]),
                'kindnet.sched.wait_ns': int(fields[1]),
                'kindnet.sched.timeslices': int(fields[2]),
            }
    return {}


def parse_nfqueue(
    text: str, queue_number: int = 101
) -> tuple[str, dict[str, int], str | None]:
    status = next(
        (
            line.split('=', 1)[1]
            for line in text.splitlines()
            if line.startswith('status=')
        ),
        None,
    )
    if status == 'empty':
        return 'unavailable', {}, 'nfqueue_no_active_queues'
    if status:
        return 'unavailable', {}, f'nfqueue_{status}'
    saw_well_formed_row = False
    for line in text.splitlines():
        fields = line.split()
        if len(fields) < 9 or not all(field.isdecimal() for field in fields[:9]):
            continue
        saw_well_formed_row = True
        if int(fields[0]) != queue_number:
            continue
        identity = f'queue={fields[0]}|peer={fields[1]}'
        return identity, {
            'nfqueue.queue_total': int(fields[2]),
            'nfqueue.queue_dropped': int(fields[5]),
            'nfqueue.user_dropped': int(fields[6]),
            'nfqueue.id_sequence': int(fields[7]),
        }, None
    if saw_well_formed_row:
        return 'unavailable', {}, f'nfqueue_{queue_number}_not_active'
    return 'unavailable', {}, 'nfqueue_malformed'


def parse_nft_counters(text: str) -> tuple[str, dict[str, int], str | None]:
    if any(line.startswith('status=') for line in text.splitlines()):
        return 'unavailable', {}, 'kindnet_nft_unavailable'
    try:
        ruleset = json.loads(text)
    except (json.JSONDecodeError, TypeError):
        return 'unavailable', {}, 'kindnet_nft_malformed'
    if not isinstance(ruleset, dict):
        return 'unavailable', {}, 'kindnet_nft_malformed'
    table_handle = None
    rule_handles = []
    packets = bytes_count = 0
    counters = 0
    for item in ruleset.get('nftables', []):
        if not isinstance(item, dict):
            continue
        table = item.get('table')
        if isinstance(table, dict) and table.get('name') == 'kindnet-network-policies':
            table_handle = table.get('handle')
        rule = item.get('rule')
        if not isinstance(rule, dict) or rule.get('table') != 'kindnet-network-policies':
            continue
        rule_has_counter = False
        for expression in rule.get('expr', []):
            counter = expression.get('counter') if isinstance(expression, dict) else None
            if not isinstance(counter, dict):
                continue
            counter_packets = counter.get('packets')
            counter_bytes = counter.get('bytes')
            if not isinstance(counter_packets, int) or not isinstance(counter_bytes, int):
                continue
            packets += counter_packets
            bytes_count += counter_bytes
            counters += 1
            rule_has_counter = True
        if rule_has_counter:
            rule_handles.append(str(rule.get('handle', 'unavailable')))
    if table_handle is None:
        return 'unavailable', {}, 'kindnet_nft_table_unavailable'
    if not counters:
        return f'table={table_handle}|rules=none', {}, 'kindnet_nft_no_counters'
    return (
        f'table={table_handle}|rules={",".join(sorted(rule_handles))}',
        {
            'kindnet.nft.counter.packets': packets,
            'kindnet.nft.counter.bytes': bytes_count,
        },
        None,
    )


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
        kindnet_output = run_command(
            ['docker', 'exec', node, 'sh', '-c', KINDNET_COMMAND],
            timeout_seconds=5.0,
        )
        sections = split_sections(kindnet_output or '')
        if kindnet_output is None:
            sections = {
                section: 'status=unavailable'
                for section in (
                    'nfqueue',
                    'kindnet_identity',
                    'kindnet_cpu_stat',
                    'kindnet_schedstat',
                    'kindnet_nft',
                )
            }
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': f'kindnet:{node}',
                'identity': identity,
                'status': 'kindnet_collection_unavailable',
            })

        kindnet_identity, kindnet_status = parse_kindnet_identity(
            sections.get('kindnet_identity', '')
        )
        kindnet_metrics = parse_kindnet_cpu(
            sections.get('kindnet_cpu_stat', '')
        )
        kindnet_metrics.update(
            parse_schedstat(sections.get('kindnet_schedstat', ''))
        )
        kindnet_source = f'kindnet:{node}'
        full_kindnet_identity = f'{identity}|{kindnet_identity}'
        if kindnet_status:
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': kindnet_source,
                'identity': full_kindnet_identity,
                'status': kindnet_status,
            })
        if kindnet_identity == 'unavailable':
            kindnet_metrics = {}
        elif not {
            'kindnet.cgroup.cpu.nr_throttled',
            'kindnet.cgroup.cpu.throttled_us',
        }.issubset(kindnet_metrics) and kindnet_status != 'kindnet_cgroup_unavailable':
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': kindnet_source,
                'identity': full_kindnet_identity,
                'status': 'kindnet_cgroup_unavailable',
            })
        if kindnet_identity != 'unavailable' and not any(
            metric.startswith('kindnet.sched.') for metric in kindnet_metrics
        ):
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': kindnet_source,
                'identity': full_kindnet_identity,
                'status': 'kindnet_schedstat_unavailable',
            })
        for metric, value in kindnet_metrics.items():
            events.append(
                metric_event(
                    node_timestamp_ms,
                    sample_number,
                    kindnet_source,
                    full_kindnet_identity,
                    metric,
                    value,
                )
            )

        queue_identity, queue_metrics, queue_status = parse_nfqueue(
            sections.get('nfqueue', '')
        )
        queue_source = f'nfqueue:{node}:101'
        full_queue_identity = f'{identity}|{queue_identity}'
        if queue_status:
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': queue_source,
                'identity': full_queue_identity,
                'status': queue_status,
            })
        for metric, value in queue_metrics.items():
            events.append(
                metric_event(
                    node_timestamp_ms,
                    sample_number,
                    queue_source,
                    full_queue_identity,
                    metric,
                    value,
                )
            )

        nft_identity, nft_metrics, nft_status = parse_nft_counters(
            sections.get('kindnet_nft', '')
        )
        nft_source = f'kindnet-nft:{node}'
        full_nft_identity = f'{identity}|{nft_identity}'
        if nft_status:
            events.append({
                'type': 'status',
                'timestamp_ms': node_timestamp_ms,
                'sample': sample_number,
                'source': nft_source,
                'identity': full_nft_identity,
                'status': nft_status,
            })
        for metric, value in nft_metrics.items():
            events.append(
                metric_event(
                    node_timestamp_ms,
                    sample_number,
                    nft_source,
                    full_nft_identity,
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


def parse_rfc3339_timestamp(value: str) -> int | None:
    try:
        return int(
            datetime.fromisoformat(value.replace('Z', '+00:00')).timestamp()
            * 1000
        )
    except (ValueError, TypeError, OverflowError):
        return None


def parse_kindnet_log(
    text: str,
) -> tuple[dict[str, list[int | None]], list[tuple[int | None, float]]]:
    signals: dict[str, list[int | None]] = {
        label: [] for label in KINDNET_LOG_SIGNALS
    }
    nft_syncs: list[tuple[int | None, float]] = []
    unit_to_milliseconds = {
        'ns': 0.000001,
        'us': 0.001,
        'µs': 0.001,
        'ms': 1.0,
        's': 1000.0,
    }
    for line in text.splitlines():
        timestamp_match = KINDNET_TIMESTAMP_PATTERN.search(line)
        timestamp_ms = (
            parse_rfc3339_timestamp(timestamp_match.group(1))
            if timestamp_match
            else None
        )
        for label, phrase in KINDNET_LOG_SIGNALS.items():
            if phrase in line:
                signals[label].append(timestamp_ms)
        if 'Syncing nftables rules' not in line:
            continue
        elapsed_match = KINDNET_ELAPSED_PATTERN.search(line)
        if elapsed_match:
            nft_syncs.append((
                timestamp_ms,
                float(elapsed_match.group(1))
                * unit_to_milliseconds[elapsed_match.group(2)],
            ))
    return signals, nft_syncs


def build_kindnet_log_report(
    path: Path, window_start_path: Path, probe_path: Path | None = None
) -> str:
    lines = ['', '### Kind NetworkPolicy log signals', '']
    if not path.is_file() or not path.stat().st_size:
        lines.append('_Recent kindnet logs unavailable or empty._')
        return '\n'.join(lines) + '\n'
    try:
        text = path.read_text(encoding='utf-8')
    except OSError:
        lines.append('_Recent kindnet logs unavailable or unreadable._')
        return '\n'.join(lines) + '\n'
    try:
        window_start_text = window_start_path.read_text(encoding='utf-8').strip()
    except OSError:
        window_start_text = ''
    window_start_ms = parse_rfc3339_timestamp(window_start_text)
    signals, nft_syncs = parse_kindnet_log(text)
    if window_start_ms is None:
        lines.append('_Observer start timestamp unavailable; signals are not windowed._')
    else:
        lines.append(f'_Observer started at {format_timestamp(window_start_ms)}._')
    lines.extend([
        '',
        '| Signal | Before observer | During observer | Timestamp unavailable | '
        'First seen | Last seen |',
        '|---|---:|---:|---:|---|---|',
    ])
    for label, timestamps in signals.items():
        known = [timestamp for timestamp in timestamps if timestamp is not None]
        before = (
            str(sum(timestamp < window_start_ms for timestamp in known))
            if window_start_ms is not None
            else '—'
        )
        during = (
            str(sum(timestamp >= window_start_ms for timestamp in known))
            if window_start_ms is not None
            else '—'
        )
        lines.append(
            f'| {label} | {before} | {during} | {len(timestamps) - len(known)} | '
            f'{format_timestamp(min(known) if known else None)} | '
            f'{format_timestamp(max(known) if known else None)} |'
        )

    if nft_syncs:
        slow_syncs = [duration for _timestamp, duration in nft_syncs if duration >= 1000]
        maximum_duration = max(duration for _timestamp, duration in nft_syncs)
        lines.extend([
            '',
            f'_nftables sync completions: {len(nft_syncs)}; at least 1 s: '
            f'{len(slow_syncs)}; maximum: {maximum_duration:.1f} ms._',
        ])
    else:
        lines.extend(['', '_No timed nftables sync completions found._'])

    probe_failures = []
    if probe_path is not None:
        probe_failures = [
            int(event['timestamp_ms'])
            for event in read_events(probe_path)
            if event.get('type') == 'probe'
            and event.get('result') != 'ok'
            and 'timestamp_ms' in event
        ]
    kindnet_errors = sorted(
        timestamp
        for timestamps in signals.values()
        for timestamp in timestamps
        if timestamp is not None
    )
    if probe_failures and kindnet_errors:
        first_probe_failure = min(probe_failures)
        nearest_error = min(
            kindnet_errors,
            key=lambda timestamp: abs(timestamp - first_probe_failure),
        )
        delta_ms = nearest_error - first_probe_failure
        if delta_ms == 0:
            relationship = 'at the same timestamp'
        elif delta_ms < 0:
            relationship = f'{abs(delta_ms)} ms before'
        else:
            relationship = f'{delta_ms} ms after'
        lines.extend([
            '',
            f'_First probe failure: {format_timestamp(first_probe_failure)}; nearest '
            f'kindnet NFQUEUE error: {format_timestamp(nearest_error)} '
            f'({relationship})._',
        ])
    lines.append('')
    return '\n'.join(lines)


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
    kindnet_report_parser = subparsers.add_parser('kindnet-log-report')
    kindnet_report_parser.add_argument('--input', type=Path, required=True)
    kindnet_report_parser.add_argument('--window-start', type=Path, required=True)
    kindnet_report_parser.add_argument('--probe-input', type=Path)
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
    elif arguments.command == 'kindnet-log-report':
        publish(
            build_kindnet_log_report(
                arguments.input,
                arguments.window_start,
                arguments.probe_input,
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

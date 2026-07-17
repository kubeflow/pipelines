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

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import network_window


SCRIPT = Path(__file__).with_name('network_window.py')


class NetworkWindowTest(unittest.TestCase):

    def test_parses_softnet_conntrack_tcp_and_socket_counters(self):
        softnet = network_window.parse_softnet(
            '0000000a 00000002 00000003 0 0 0 0 0 0 0 0\n'
            '00000010 00000004 00000005 0 0 0 0 0 0 0 0\n'
        )
        node_output = _node_output(7, 9)
        network_namespace, metrics = network_window.parse_node_output(node_output)

        self.assertEqual(softnet['softnet.processed'], 26)
        self.assertEqual(softnet['softnet.dropped'], 6)
        self.assertEqual(softnet['softnet.time_squeeze'], 8)
        self.assertEqual(network_namespace, 'net:[7]')
        self.assertEqual(metrics['conntrack.insert_failed'], 7)
        self.assertEqual(metrics['conntrack.drop'], 9)
        self.assertEqual(metrics['conntrack.count'], 12)
        self.assertEqual(metrics['TcpExt.TCPSynRetrans'], 4)
        self.assertEqual(metrics['sockstat.TCP.inuse'], 8)

    def test_sampler_preserves_short_conntrack_spike(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            state_file = temporary_path / 'state'
            state_file.write_text('0', encoding='utf-8')
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(bin_directory / 'docker', _FAKE_DOCKER)
            proc_root = self._write_proc(temporary_path)
            output_path = temporary_path / 'network.jsonl'
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_STATE_FILE'] = str(state_file)
            environment['NETWORK_WINDOW_PROC_ROOT'] = str(proc_root)

            subprocess.run(
                [
                    'python3',
                    str(SCRIPT),
                    'sample',
                    '--output',
                    str(output_path),
                    '--interval',
                    '0',
                    '--max-samples',
                    '3',
                ],
                check=True,
                env=environment,
            )
            report = network_window.build_report(output_path)

        self.assertIn('| conntrack.insert_failed | 3 | 11 | 9 |', report)
        self.assertIn('| conntrack.drop | 3 | 13 | 10 |', report)
        self.assertIn('Largest counter increases between samples', report)

    def test_report_segments_replaced_node_and_never_emits_negative_delta(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            path = Path(temporary_directory) / 'events.jsonl'
            events = [
                {'type': 'meta', 'nodes': ['kind-control-plane'], 'interval_seconds': 5},
                self._metric(0, 'container-a|net:[1]', 8),
                self._metric(1, 'container-a|net:[1]', 3),
                self._metric(2, 'container-b|net:[2]', 4),
                self._metric(3, 'container-b|net:[2]', 7),
            ]
            path.write_text(
                ''.join(json.dumps(event) + '\n' for event in events),
                encoding='utf-8',
            )
            report = network_window.build_report(path)

        self.assertIn('counter reset', report)
        self.assertIn('replacement detected for: node:kind-control-plane', report)
        self.assertNotIn('| -5 |', report)

    def test_report_mirrors_stdout_and_correlates_probe_failure(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            input_path = temporary_path / 'network.jsonl'
            probe_path = temporary_path / 'probe.jsonl'
            summary_path = temporary_path / 'summary.md'
            input_path.write_text(
                ''.join(json.dumps(event) + '\n' for event in [
                    {'type': 'meta', 'nodes': ['kind-control-plane'], 'interval_seconds': 5},
                    self._metric(0, 'container-a|net:[1]', 2),
                    self._metric(1, 'container-a|net:[1]', 7),
                ]),
                encoding='utf-8',
            )
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': 1_700_000_005_000,
                    'cycle': 1,
                    'pair': 'kubernetes-api',
                    'path': 'vip',
                    'result': 'timeout',
                }) + '\n',
                encoding='utf-8',
            )
            environment = os.environ.copy()
            environment['GITHUB_STEP_SUMMARY'] = str(summary_path)
            result = subprocess.run(
                [
                    'python3',
                    str(SCRIPT),
                    'report',
                    '--input',
                    str(input_path),
                    '--probe-input',
                    str(probe_path),
                ],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )
            summary_contents = summary_path.read_text(encoding='utf-8')

        self.assertEqual(result.stdout, summary_contents)
        self.assertIn('kubernetes-api/vip', result.stdout)
        self.assertIn('conntrack.insert_failed +5', result.stdout)

    def test_correlation_uses_sample_after_failure_not_closer_prior_sample(self):
        failure_timestamp = 1_700_000_000_200
        metric_events = [
            self._metric(0, 'container-a|net:[1]', 2),
            self._metric(1, 'container-a|net:[1]', 7),
        ]
        with tempfile.TemporaryDirectory() as temporary_directory:
            probe_path = Path(temporary_directory) / 'probe.jsonl'
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': failure_timestamp,
                    'cycle': 0,
                    'pair': 'seaweedfs-s3',
                    'path': 'vip',
                    'result': 'timeout',
                }) + '\n',
                encoding='utf-8',
            )

            rows = network_window.correlation_rows(metric_events, probe_path)

        self.assertEqual(len(rows), 1)
        self.assertIn('| 4800 ms |', rows[0])
        self.assertIn('conntrack.insert_failed +5', rows[0])

    @staticmethod
    def _metric(sample, identity, value):
        return {
            'type': 'metric',
            'timestamp_ms': 1_700_000_000_000 + sample * 5000,
            'sample': sample,
            'source': 'node:kind-control-plane',
            'identity': identity,
            'metric': 'conntrack.insert_failed',
            'value': value,
        }

    @staticmethod
    def _write_executable(path, contents):
        path.write_text(contents, encoding='utf-8')
        path.chmod(0o755)

    @staticmethod
    def _write_proc(temporary_path):
        proc_root = temporary_path / 'proc'
        (proc_root / 'net').mkdir(parents=True)
        (proc_root / 'net' / 'softnet_stat').write_text(
            '0000000a 00000000 00000000 0 0 0 0 0 0 0 0\n',
            encoding='utf-8',
        )
        (proc_root / 'net' / 'netstat').write_text('', encoding='utf-8')
        (proc_root / 'net' / 'snmp').write_text('', encoding='utf-8')
        (proc_root / 'net' / 'sockstat').write_text('', encoding='utf-8')
        return proc_root


def _node_output(insert_failed, drop):
    return f'''__NETNS__
net:[7]
__CONNTRACK__
cpu=0 insert_failed={insert_failed} drop={drop} early_drop=0 error=0 search_restart=1
__CONNTRACK_GAUGES__
count=12 max=262144
__SOCKSTAT__
sockets: used 20
TCP: inuse 8 orphan 0 tw 3 alloc 10 mem 1
__NETSTAT__
TcpExt: ListenOverflows ListenDrops TCPBacklogDrop TCPSynRetrans TCPTimeouts
TcpExt: 0 0 0 4 2
__SNMP__
Ip: InDiscards OutDiscards
Ip: 0 0
Tcp: RetransSegs
Tcp: 6
'''


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
echo "kind-control-plane"
'''


_FAKE_DOCKER = r'''#!/usr/bin/env bash
case "$1" in
  inspect)
    echo "container-a"
    ;;
  exec)
    count=$(<"$FAKE_STATE_FILE")
    count=$((count + 1))
    echo "$count" > "$FAKE_STATE_FILE"
    case "$count" in
      1) insert_failed=1; drop=2 ;;
      2) insert_failed=10; drop=12 ;;
      *) insert_failed=12; drop=15 ;;
    esac
    cat <<EOF
__NETNS__
net:[1]
__CONNTRACK__
cpu=0 insert_failed=$insert_failed drop=$drop early_drop=0 error=0 search_restart=0
__CONNTRACK_GAUGES__
count=10 max=262144
__SOCKSTAT__
sockets: used 4
TCP: inuse 2 orphan 0 tw 1 alloc 3 mem 0
__NETSTAT__
TcpExt: ListenOverflows ListenDrops TCPBacklogDrop TCPSynRetrans TCPTimeouts
TcpExt: 0 0 0 0 0
__SNMP__
Ip: InDiscards OutDiscards
Ip: 0 0
Tcp: RetransSegs
Tcp: 0
EOF
    ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

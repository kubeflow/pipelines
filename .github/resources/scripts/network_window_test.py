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
import threading
import unittest
from unittest import mock

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
            environment['NETWORK_WINDOW_CGROUP_ROOT'] = str(
                temporary_path / 'cgroup'
            )

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
        self.assertIn('| kindnet.cgroup.cpu.nr_throttled | 3 | 10 | 9 |', report)
        self.assertIn('| nfqueue.queue_dropped | 3 | 8 | 7 |', report)
        self.assertIn('| kindnet.nft.counter.packets | 3 | 11 | 9 |', report)
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

    def test_target_report_attributes_counters_and_listen_queue(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            baseline_path = temporary_path / 'baseline.txt'
            end_path = temporary_path / 'end.txt'
            baseline_path.write_text(
                _target_output(2, 3, receive_queue=1), encoding='utf-8'
            )
            end_path.write_text(
                _target_output(7, 9, receive_queue=4), encoding='utf-8'
            )

            report = network_window.build_target_report(
                baseline_path, end_path, 8333, 'SeaweedFS'
            )

        self.assertIn('SeaweedFS target network-namespace snapshots', report)
        self.assertIn('| TcpExt.ListenDrops | 2 | 7 | 5 | ok |', report)
        self.assertIn('| TcpExt.ListenOverflows | 3 | 9 | 6 | ok |', report)
        self.assertIn('| listen_queue.listeners | 1 | 1 | gauge | ok |', report)
        self.assertIn('| listen_queue.rx | 1 | 4 | gauge | ok |', report)

    def test_missing_tcp_table_is_unavailable_not_zero_queue(self):
        self.assertEqual(network_window.parse_listen_queues('', 8333), {})
        self.assertEqual(
            network_window.parse_listen_queues('malformed tcp state', 8333),
            {},
        )

    def test_samples_cpu_psi_and_current_cgroup_throttling(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            proc_root = self._write_proc(temporary_path)
            (proc_root / 'pressure').mkdir()
            (proc_root / 'pressure' / 'cpu').write_text(
                'some avg10=1.00 avg60=2.00 avg300=3.00 total=12345\n'
                'full avg10=0.00 avg60=0.00 avg300=0.00 total=67\n',
                encoding='utf-8',
            )
            (proc_root / 'self').mkdir()
            (proc_root / 'self' / 'cgroup').write_text(
                '0::/actions_job/step\n', encoding='utf-8'
            )
            cgroup_root = temporary_path / 'cgroup'
            cpu_stat_directory = cgroup_root / 'actions_job' / 'step'
            cpu_stat_directory.mkdir(parents=True)
            (cpu_stat_directory / 'cpu.stat').write_text(
                'nr_periods 50\nnr_throttled 7\nthrottled_usec 890\n',
                encoding='utf-8',
            )

            metrics = network_window.host_metrics(proc_root, cgroup_root)

        self.assertEqual(metrics['psi.cpu.some_stall_us'], 12345)
        self.assertEqual(metrics['psi.cpu.full_stall_us'], 67)
        self.assertEqual(metrics['cgroup.cpu.nr_periods'], 50)
        self.assertEqual(metrics['cgroup.cpu.nr_throttled'], 7)
        self.assertEqual(metrics['cgroup.cpu.throttled_us'], 890)

    def test_parses_kindnet_cgroup_v2_v1_and_scheduler_counters(self):
        v2_metrics = network_window.parse_kindnet_cpu(
            'nr_periods 50\nnr_throttled 7\nthrottled_usec 890\n'
            'quota_us 10000\nperiod_us 100000\n'
        )
        v1_metrics = network_window.parse_kindnet_cpu(
            'nr_periods 5\nnr_throttled 2\nthrottled_time 91000\n'
        )
        scheduler = network_window.parse_schedstat('123 456 7\n')

        self.assertEqual(v2_metrics['kindnet.cgroup.cpu.nr_throttled'], 7)
        self.assertEqual(v2_metrics['kindnet.cgroup.cpu.throttled_us'], 890)
        self.assertEqual(v2_metrics['kindnet.cgroup.cpu.quota_us'], 10000)
        self.assertEqual(v2_metrics['kindnet.cgroup.cpu.period_us'], 100000)
        self.assertEqual(v2_metrics['kindnet.cgroup.cpu.unlimited'], 0)
        self.assertEqual(v1_metrics['kindnet.cgroup.cpu.throttled_us'], 91)
        self.assertEqual(scheduler['kindnet.sched.runtime_ns'], 123)
        self.assertEqual(scheduler['kindnet.sched.wait_ns'], 456)
        self.assertEqual(scheduler['kindnet.sched.timeslices'], 7)

    def test_parses_unlimited_kindnet_cgroup_quota(self):
        for quota in ('max', '-1'):
            with self.subTest(quota=quota):
                metrics = network_window.parse_kindnet_cpu(
                    f'nr_periods 50\nnr_throttled 0\nquota_us {quota}\n'
                    'period_us 100000\n'
                )

                self.assertEqual(metrics['kindnet.cgroup.cpu.unlimited'], 1)
                self.assertNotIn('kindnet.cgroup.cpu.quota_us', metrics)
                self.assertEqual(
                    metrics['kindnet.cgroup.cpu.period_us'], 100000
                )

    def test_kindnet_cpu_unavailable_does_not_publish_quota_as_health(self):
        metrics = network_window.parse_kindnet_cpu(
            'status=unavailable\nquota_us 10000\nperiod_us 100000\n'
        )

        self.assertEqual(metrics, {})

    def test_kindnet_log_report_windows_nfqueue_errors_and_slow_syncs(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            log_path = temporary_path / 'kindnet.log'
            start_path = temporary_path / 'start.txt'
            probe_path = temporary_path / 'probe.jsonl'
            log_path.write_text(
                '[pod/kindnet-a/kindnet-cni] 2026-07-17T16:18:59.500Z '
                '"failed to set verdict with label" err="netlink send: i/o timeout"\n'
                '2026-07-17T16:19:01.250Z "Could not receive message" '
                'error="netlink receive: no such file or directory"\n'
                '2026-07-17T16:19:02.000Z "Syncing nftables rules" '
                'elapsed="2.5s"\n'
                'without-a-timestamp "failed to set verdict with label"\n',
                encoding='utf-8',
            )
            start_path.write_text(
                '2026-07-17T16:19:00Z\n', encoding='utf-8'
            )
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': 1784305141000,
                    'result': 'timeout',
                }) + '\n',
                encoding='utf-8',
            )

            report = network_window.build_kindnet_log_report(
                log_path, start_path, probe_path
            )

        self.assertIn('| NFQUEUE verdict-send failure | 1 | 0 | 1 |', report)
        self.assertIn('| NFQUEUE receive failure | 0 | 1 | 0 |', report)
        self.assertIn('at least 1 s: 1; maximum: 2500.0 ms', report)
        self.assertIn('nearest kindnet NFQUEUE error', report)

    def test_kindnet_log_report_distinguishes_empty_from_zero_errors(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            start_path = temporary_path / 'start.txt'
            start_path.write_text('2026-07-17T16:19:00Z\n', encoding='utf-8')
            missing_report = network_window.build_kindnet_log_report(
                temporary_path / 'missing.log', start_path
            )
            log_path = temporary_path / 'kindnet.log'
            log_path.write_text(
                '2026-07-17T16:19:01Z policy engine is ready\n',
                encoding='utf-8',
            )
            healthy_report = network_window.build_kindnet_log_report(
                log_path, start_path
            )

        self.assertIn('logs unavailable or empty', missing_report)
        self.assertIn(
            '| NFQUEUE verdict-send failure | 0 | 0 | 0 |', healthy_report
        )
        self.assertNotIn('logs unavailable', healthy_report)

    def test_kindnet_scheduler_command_aggregates_all_threads(self):
        self.assertIn('/task/*/schedstat', network_window.KINDNET_COMMAND)
        self.assertIn('runtime_ns=$((runtime_ns + thread_runtime))',
                      network_window.KINDNET_COMMAND)

    def test_kindnet_identity_segments_restart_and_reports_missing_cgroup(self):
        first, first_status = network_window.parse_kindnet_identity(
            'container_id=abc pid=12 start_time=100 cgroup=/kindnet-a\n'
        )
        second, second_status = network_window.parse_kindnet_identity(
            'container_id=def pid=13 start_time=200 cgroup=/kindnet-b\n'
        )
        partial, partial_status = network_window.parse_kindnet_identity(
            'container_id=abc pid=12 start_time=100 cgroup=unavailable\n'
        )

        self.assertNotEqual(first, second)
        self.assertIsNone(first_status)
        self.assertIsNone(second_status)
        self.assertNotEqual(partial, 'unavailable')
        self.assertEqual(partial_status, 'kindnet_cgroup_unavailable')

    def test_parses_nfqueue_101_depth_and_drop_counters(self):
        identity, metrics, status = network_window.parse_nfqueue(
            '    0   2000     0 2 65535     0     0       10  1\n'
            '  101   2538     3 2 65535     7     5      106  1\n'
        )

        self.assertEqual(identity, 'queue=101|peer=2538')
        self.assertIsNone(status)
        self.assertEqual(metrics['nfqueue.queue_total'], 3)
        self.assertEqual(metrics['nfqueue.queue_dropped'], 7)
        self.assertEqual(metrics['nfqueue.user_dropped'], 5)
        self.assertEqual(metrics['nfqueue.id_sequence'], 106)

    def test_kindnet_command_reads_zero_size_nfqueue_stream(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            fifo_path = Path(temporary_directory) / 'nfnetlink_queue'
            os.mkfifo(fifo_path)

            def write_queue_row():
                with fifo_path.open('w', encoding='utf-8') as fifo:
                    fifo.write('101 2538 3 2 65535 7 5 106 1\n')

            writer = threading.Thread(target=write_queue_row, daemon=True)
            writer.start()
            contents = self._run_kindnet_command(fifo_path)
            writer.join(timeout=5)

        self.assertFalse(writer.is_alive())
        identity, metrics, status = network_window.parse_nfqueue(contents)
        self.assertEqual(identity, 'queue=101|peer=2538')
        self.assertIsNone(status)
        self.assertEqual(metrics['nfqueue.queue_total'], 3)
        self.assertEqual(metrics['nfqueue.queue_dropped'], 7)
        self.assertEqual(metrics['nfqueue.user_dropped'], 5)
        self.assertEqual(metrics['nfqueue.id_sequence'], 106)

    def test_kindnet_command_marks_empty_nfqueue_stream(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            queue_path = Path(temporary_directory) / 'nfnetlink_queue'
            queue_path.touch()
            contents = self._run_kindnet_command(queue_path)

        self.assertEqual(contents, 'status=empty')
        identity, metrics, status = network_window.parse_nfqueue(contents)
        self.assertEqual(identity, 'unavailable')
        self.assertEqual(metrics, {})
        self.assertEqual(status, 'nfqueue_no_active_queues')

    def test_kindnet_command_marks_missing_nfqueue_stream_unavailable(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            queue_path = Path(temporary_directory) / 'missing'
            contents = self._run_kindnet_command(queue_path)

        self.assertEqual(contents, 'status=unavailable')
        identity, metrics, status = network_window.parse_nfqueue(contents)
        self.assertEqual(identity, 'unavailable')
        self.assertEqual(metrics, {})
        self.assertEqual(status, 'nfqueue_unavailable')

    def test_nfqueue_unavailable_states_are_not_synthetic_zeroes(self):
        for text, expected_status in (
            ('status=empty\n', 'nfqueue_no_active_queues'),
            ('status=unavailable\n', 'nfqueue_unavailable'),
            ('malformed row\n', 'nfqueue_malformed'),
            ('0 1 2 3 4 5 6 7 8\n', 'nfqueue_101_not_active'),
        ):
            identity, metrics, status = network_window.parse_nfqueue(text)
            self.assertEqual(identity, 'unavailable')
            self.assertEqual(metrics, {})
            self.assertEqual(status, expected_status)

    def test_aggregates_bounded_kindnet_nft_counters_and_tracks_handles(self):
        identity, metrics, status = network_window.parse_nft_counters(
            json.dumps({
                'nftables': [
                    {'table': {
                        'family': 'inet',
                        'name': 'kindnet-network-policies',
                        'handle': 9,
                    }},
                    {'rule': {
                        'family': 'inet',
                        'table': 'kindnet-network-policies',
                        'chain': 'forward',
                        'handle': 11,
                        'expr': [{'counter': {'packets': 7, 'bytes': 700}}],
                    }},
                    {'rule': {
                        'family': 'inet',
                        'table': 'unrelated',
                        'chain': 'forward',
                        'handle': 99,
                        'expr': [{'counter': {'packets': 999, 'bytes': 9999}}],
                    }},
                ],
            })
        )

        self.assertEqual(identity, 'table=9|rules=11')
        self.assertIsNone(status)
        self.assertEqual(metrics['kindnet.nft.counter.packets'], 7)
        self.assertEqual(metrics['kindnet.nft.counter.bytes'], 700)
        self.assertEqual(len(metrics), 2)

    def test_optional_kindnet_sources_do_not_hide_core_node_metrics(self):
        node_output = _node_output(7, 9)
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            proc_root = self._write_proc(temporary_path)
            with mock.patch.object(
                network_window,
                'run_command',
                side_effect=['container-a', node_output, None],
            ):
                events = network_window.capture_sample(
                    ['kind-control-plane'],
                    proc_root,
                    temporary_path / 'cgroup',
                    0,
                )

        node_metrics = {
            event.get('metric'): event.get('value')
            for event in events
            if event.get('source') == 'node:kind-control-plane'
        }
        statuses = {event.get('status') for event in events}
        self.assertEqual(node_metrics['conntrack.insert_failed'], 7)
        self.assertEqual(node_metrics['conntrack.drop'], 9)
        self.assertIn('kindnet_collection_unavailable', statuses)
        self.assertIn('nfqueue_unavailable', statuses)
        self.assertIn('kindnet_unavailable', statuses)
        self.assertIn('kindnet_nft_unavailable', statuses)

    def test_waits_for_counter_sample_after_final_probe(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            probe_path = temporary_path / 'probe.jsonl'
            network_path = temporary_path / 'network.jsonl'
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': 1_700_000_000_000,
                }) + '\n',
                encoding='utf-8',
            )
            network_path.write_text(
                json.dumps({
                    'type': 'metric',
                    'timestamp_ms': 1_700_000_000_001,
                }) + '\n',
                encoding='utf-8',
            )

            success, message = network_window.wait_for_following_sample(
                network_path, probe_path, 0
            )

        self.assertTrue(success)
        self.assertIn('confirmed', message)

    def test_reports_missing_following_counter_sample(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            probe_path = temporary_path / 'probe.jsonl'
            network_path = temporary_path / 'network.jsonl'
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': 1_700_000_000_001,
                }) + '\n',
                encoding='utf-8',
            )
            network_path.write_text(
                json.dumps({
                    'type': 'metric',
                    'timestamp_ms': 1_700_000_000_000,
                }) + '\n',
                encoding='utf-8',
            )

            success, message = network_window.wait_for_following_sample(
                network_path, probe_path, 0
            )

        self.assertFalse(success)
        self.assertIn('unavailable', message)

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

    def test_correlation_keeps_onset_and_late_counter_spike(self):
        base_timestamp = 1_700_000_000_000
        metric_events = []
        value = 0
        for sample in range(27):
            if sample == 25:
                value += 100
            else:
                value += 1
            metric_events.append({
                **self._metric(sample, 'container-a|net:[1]', value),
                'timestamp_ms': base_timestamp + sample * 5000,
            })
        with tempfile.TemporaryDirectory() as temporary_directory:
            probe_path = Path(temporary_directory) / 'probe.jsonl'
            events = []
            for cycle in range(25):
                for path in ('vip', 'endpoint'):
                    events.append({
                        'type': 'probe',
                        'timestamp_ms': base_timestamp + cycle * 5000 + 200,
                        'cycle': cycle,
                        'pair': 'seaweedfs-s3',
                        'path': path,
                        'result': 'timeout',
                    })
            probe_path.write_text(
                ''.join(json.dumps(event) + '\n' for event in events),
                encoding='utf-8',
            )

            rows = network_window.correlation_rows(metric_events, probe_path)

        self.assertEqual(len(rows), 20)
        self.assertIn('endpoint+vip', rows[0])
        self.assertTrue(
            any('conntrack.insert_failed +100' in row for row in rows)
        )

    def test_correlation_includes_kindnet_throttle_and_nfqueue_drops(self):
        base_timestamp = 1_700_000_000_000
        metric_events = []
        for metric, before, after, source in (
            (
                'kindnet.cgroup.cpu.nr_throttled',
                2,
                7,
                'kindnet:kind-control-plane',
            ),
            (
                'nfqueue.queue_dropped',
                1,
                4,
                'nfqueue:kind-control-plane:101',
            ),
        ):
            for sample, value in enumerate((before, after)):
                metric_events.append({
                    'type': 'metric',
                    'timestamp_ms': base_timestamp + sample * 5000,
                    'sample': sample,
                    'source': source,
                    'identity': 'stable',
                    'metric': metric,
                    'value': value,
                })
        with tempfile.TemporaryDirectory() as temporary_directory:
            probe_path = Path(temporary_directory) / 'probe.jsonl'
            probe_path.write_text(
                json.dumps({
                    'type': 'probe',
                    'timestamp_ms': base_timestamp + 200,
                    'pair': 'seaweedfs-s3',
                    'path': 'endpoint',
                    'result': 'timeout',
                }) + '\n',
                encoding='utf-8',
            )

            rows = network_window.correlation_rows(metric_events, probe_path)

        self.assertEqual(len(rows), 1)
        self.assertIn('kindnet.cgroup.cpu.nr_throttled +5', rows[0])
        self.assertIn('nfqueue.queue_dropped +3', rows[0])

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
    def _run_kindnet_command(queue_path):
        environment = os.environ.copy()
        environment['NETWORK_WINDOW_NFQUEUE_PATH'] = str(queue_path)
        result = subprocess.run(
            ['sh', '-c', network_window.KINDNET_COMMAND],
            check=True,
            capture_output=True,
            env=environment,
            text=True,
            timeout=10,
        )
        return network_window.split_sections(result.stdout)['nfqueue']

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


def _target_output(listen_drops, listen_overflows, receive_queue):
    return f'''__POD__
seaweedfs-abc
__NETNS__
net:[42]
__SOCKSTAT__
sockets: used 20
TCP: inuse 8 orphan 0 tw 3 alloc 10 mem 1
__NETSTAT__
TcpExt: ListenOverflows ListenDrops TCPBacklogDrop TCPSynRetrans TCPTimeouts
TcpExt: {listen_overflows} {listen_drops} 0 4 2
__SNMP__
Ip: InDiscards OutDiscards
Ip: 0 0
Tcp: RetransSegs
Tcp: 6
__TCP__
  sl  local_address rem_address   st tx_queue:rx_queue tr tm->when retrnsmt
   0: 00000000:208D 00000000:0000 0A 00000000:{receive_queue:08X} 00:00000000 00000000
__TCP6__
  sl  local_address rem_address   st tx_queue:rx_queue tr tm->when retrnsmt
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
    if [[ "$5" == *"__NETNS__"* ]]; then
      count=$(<"$FAKE_STATE_FILE")
      count=$((count + 1))
      echo "$count" > "$FAKE_STATE_FILE"
    else
      count=$(<"$FAKE_STATE_FILE")
    fi
    case "$count" in
      1)
        insert_failed=1; drop=2; kindnet_throttled=1; kindnet_throttled_us=100
        nfqueue_depth=0; nfqueue_drop=1; nfqueue_user_drop=0; nft_packets=3
        ;;
      2)
        insert_failed=10; drop=12; kindnet_throttled=10; kindnet_throttled_us=1000
        nfqueue_depth=3; nfqueue_drop=8; nfqueue_user_drop=5; nft_packets=12
        ;;
      *)
        insert_failed=12; drop=15; kindnet_throttled=11; kindnet_throttled_us=1100
        nfqueue_depth=0; nfqueue_drop=9; nfqueue_user_drop=5; nft_packets=14
        ;;
    esac
    if [[ "$5" == *"__NETNS__"* ]]; then
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
    else
    cat <<EOF
__NFQUEUE__
101 2538 $nfqueue_depth 2 65535 $nfqueue_drop $nfqueue_user_drop $((count * 100)) 1
__KINDNET_IDENTITY__
container_id=kindnet-a pid=42 start_time=100 cgroup=/kindnet-a
__KINDNET_CPU_STAT__
nr_periods $((count * 10))
nr_throttled $kindnet_throttled
throttled_usec $kindnet_throttled_us
quota_us 10000
period_us 100000
__KINDNET_SCHEDSTAT__
$((count * 1000)) $((count * 100)) $((count * 10))
__KINDNET_NFT__
{"nftables":[{"table":{"family":"inet","name":"kindnet-network-policies","handle":9}},{"rule":{"family":"inet","table":"kindnet-network-policies","chain":"forward","handle":11,"expr":[{"counter":{"packets":$nft_packets,"bytes":$((nft_packets * 100))}}]}}]}
EOF
    fi
    ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Regression tests for failure-signature-summary.sh and runner-telemetry.sh.

Covers the conntrack signature classification (the collect-seaweedfs
diagnostics header must never count as a kernel table-full event) and the
stdout/summary mirroring contract (job summaries have no API, so the job log
must carry byte-identical output).

Run with:  cd .github/resources/scripts && python3 -m unittest -v observability_output_test
"""

import os
import json
import subprocess
import tempfile
import textwrap
import time
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).parent
SIGNATURE_SCRIPT = SCRIPTS_DIR / 'failure-signature-summary.sh'
TELEMETRY_SCRIPT = SCRIPTS_DIR / 'runner-telemetry.sh'


def run_signature(log_text, summary_path=None):
    with tempfile.TemporaryDirectory() as temporary_directory:
        log_file = Path(temporary_directory) / 'pod.log'
        log_file.write_text(log_text, encoding='utf-8')
        environment = os.environ.copy()
        environment.pop('GITHUB_STEP_SUMMARY', None)
        if summary_path is not None:
            environment['GITHUB_STEP_SUMMARY'] = str(summary_path)
        return subprocess.run(
            ['bash', str(SIGNATURE_SCRIPT), '--log-file', str(log_file),
             '--reports-dir', str(Path(temporary_directory) / 'none')],
            check=True, capture_output=True, text=True, env=environment,
        ).stdout


def conntrack_count(output):
    for line in output.splitlines():
        if line.startswith('| conntrack table full |'):
            return int(line.split('|')[2].strip())
    raise AssertionError(f'conntrack row missing in:\n{output}')


class ConntrackClassificationTest(unittest.TestCase):

    def test_diagnostics_header_counts_zero(self):
        # collect-seaweedfs-diagnostics tees this header into the pod log; it
        # must never be classified as a kernel table-full event.
        output = run_signature(textwrap.dedent('''\
            kernel 'nf_conntrack: table full' events:
            (no table-full event logged)
        '''))
        self.assertEqual(conntrack_count(output), 0)

    def test_real_kernel_messages_count_correctly(self):
        output = run_signature(textwrap.dedent('''\
            kernel 'nf_conntrack: table full' events:
            [111.1] nf_conntrack: table full, dropping packet
            [222.2] nf_conntrack: table full, dropping packet
        '''))
        self.assertEqual(conntrack_count(output), 2)

    def test_json_output_matches_markdown_counts(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            log_file = Path(temporary_directory) / 'pod.log'
            json_file = Path(temporary_directory) / 'signatures.json'
            log_file.write_text(
                'dial tcp 10.1.2.3:9000: i/o timeout\n'
                'connection reset by peer\n',
                encoding='utf-8',
            )
            result = subprocess.run(
                [
                    'bash', str(SIGNATURE_SCRIPT),
                    '--log-file', str(log_file),
                    '--reports-dir', str(Path(temporary_directory) / 'none'),
                    '--json-output', str(json_file),
                ],
                check=True, capture_output=True, text=True,
            )
            signatures = json.loads(json_file.read_text(encoding='utf-8'))

        self.assertEqual(signatures['clusterip_dial_timeout'], 1)
        self.assertEqual(signatures['connection_refused_or_reset'], 1)
        self.assertIn('| ClusterIP dial `i/o timeout` | 1 |', result.stdout)


class OutputMirroringTest(unittest.TestCase):
    """stdout must be byte-identical to what gets appended to the summary."""

    def test_signature_stdout_matches_summary(self):
        with tempfile.NamedTemporaryFile('r', suffix='.md') as summary:
            stdout = run_signature(
                'dial tcp 10.96.1.1:9000: i/o timeout\n',
                summary_path=summary.name,
            )
            self.assertEqual(stdout, Path(summary.name).read_text())
            self.assertIn('Failure signature summary', stdout)

    def test_telemetry_stdout_matches_summary(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            csv = Path(temporary_directory) / 't.csv'
            csv.write_text(
                'epoch,cpu_pct,mem_used_mb,load1\n'
                + ''.join(f'{1700000000 + i * 5},{i * 10},1000,1.0\n'
                          for i in range(5)),
                encoding='utf-8',
            )
            summary = Path(temporary_directory) / 'summary.md'
            environment = os.environ.copy()
            environment['GITHUB_STEP_SUMMARY'] = str(summary)
            # The render step kills any recorded sampler pid; point the pid
            # file lookup at a path that does not exist.
            result = subprocess.run(
                ['bash', str(TELEMETRY_SCRIPT), 'render', str(csv)],
                check=True, capture_output=True, text=True, env=environment,
            )
            self.assertEqual(result.stdout, summary.read_text())
            self.assertIn('Runner telemetry', result.stdout)
            self.assertIn('CPU busy %', result.stdout)

    def test_telemetry_stdout_only_without_summary(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            csv = Path(temporary_directory) / 't.csv'
            csv.write_text(
                'epoch,cpu_pct,mem_used_mb,load1\n'
                '1700000000,10,1000,1.0\n1700000005,20,1000,1.0\n'
                '1700000010,30,1000,1.0\n',
                encoding='utf-8',
            )
            environment = os.environ.copy()
            environment.pop('GITHUB_STEP_SUMMARY', None)
            result = subprocess.run(
                ['bash', str(TELEMETRY_SCRIPT), 'render', str(csv)],
                check=True, capture_output=True, text=True, env=environment,
            )
            self.assertIn('Runner telemetry', result.stdout)


class ContentionDeltaTest(unittest.TestCase):

    def test_reads_current_v2_cgroup_instead_of_mount_root(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            proc_root = temporary_path / 'proc'
            cgroup_root = temporary_path / 'cgroup'
            (proc_root / 'self').mkdir(parents=True)
            (cgroup_root / 'actions-job').mkdir(parents=True)
            (proc_root / 'self' / 'cgroup').write_text(
                '0::/actions-job\n', encoding='utf-8')
            (cgroup_root / 'cpu.stat').write_text(
                'nr_throttled 999\n', encoding='utf-8')
            (cgroup_root / 'actions-job' / 'cpu.stat').write_text(
                'nr_throttled 7\n', encoding='utf-8')
            snapshot = temporary_path / 'contention.tsv'
            environment = os.environ.copy()
            environment['RUNNER_TELEMETRY_PROC_ROOT'] = str(proc_root)
            environment['RUNNER_TELEMETRY_CGROUP_ROOT'] = str(cgroup_root)

            subprocess.run(
                ['bash', str(TELEMETRY_SCRIPT), 'contention-baseline', str(snapshot)],
                check=True,
                env=environment,
            )

            contents = snapshot.read_text(encoding='utf-8')
            self.assertIn('runner_cgroup_nr_throttled\t7\n', contents)
            self.assertNotIn('runner_cgroup_nr_throttled\t999\n', contents)

    def test_reports_psi_and_cgroup_v2_deltas(self):
        output = self._render_contention(
            baseline={
                'cpu_psi_some_us': 100000,
                'cpu_psi_full_us': 50000,
                'runner_cgroup_nr_periods': 100,
                'runner_cgroup_nr_throttled': 10,
                'runner_cgroup_throttled_usec': 1000,
            },
            pressure='some avg10=0.00 total=300000\nfull avg10=0.00 total=50000\n',
            cpu_stat='nr_periods 120\nnr_throttled 15\nthrottled_usec 6000\n',
        )

        self.assertIn('| CPU PSI some stall | 100000 | 300000 | 200.0 ms', output)
        self.assertIn('| Runner cgroup throttled periods | 10 | 15 | 5 | ok |', output)
        self.assertIn('| Runner cgroup throttled time | 1000 | 6000 | 5.0 ms | ok |', output)
        self.assertIn('25.0% of test-window periods', output)

    def test_normalizes_cgroup_v1_throttled_time(self):
        output = self._render_contention(
            baseline={'runner_cgroup_throttled_usec': 1000},
            pressure='',
            cpu_stat='nr_periods 2\nnr_throttled 1\nthrottled_time 5000000\n',
        )

        self.assertIn('| Runner cgroup throttled time | 1000 | 5000 | 4.0 ms | ok |', output)

    def test_reports_unavailable_and_reset_without_negative_delta(self):
        output = self._render_contention(
            baseline={
                'cpu_psi_some_us': 500,
                'cpu_psi_full_us': 100,
                'runner_cgroup_nr_throttled': 9,
            },
            pressure='some avg10=0.00 total=400\n',
            cpu_stat='nr_throttled 2\n',
        )

        self.assertIn('| CPU PSI some stall | 500 | 400 | — | counter reset |', output)
        self.assertIn('| CPU PSI full stall | 100 | — | — | unavailable |', output)
        self.assertIn('| Runner cgroup throttled periods | 9 | 2 | — | counter reset |', output)

    def _render_contention(self, baseline, pressure, cpu_stat):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            proc_root = temporary_path / 'proc'
            cgroup_root = temporary_path / 'cgroup'
            (proc_root / 'pressure').mkdir(parents=True)
            cgroup_root.mkdir()
            (proc_root / 'pressure' / 'cpu').write_text(pressure, encoding='utf-8')
            (cgroup_root / 'cpu.stat').write_text(cpu_stat, encoding='utf-8')
            baseline_path = temporary_path / 'baseline.tsv'
            baseline_values = {'epoch_us': time.time_ns() // 1000 - 10_000_000}
            baseline_values.update(baseline)
            baseline_path.write_text(
                ''.join(f'{key}\t{value}\n' for key, value in baseline_values.items()),
                encoding='utf-8',
            )
            csv = temporary_path / 'telemetry.csv'
            csv.write_text(
                'epoch,cpu_pct,mem_used_mb,load1\n'
                '1700000000,10,1000,1.0\n'
                '1700000005,20,1000,1.0\n'
                '1700000010,30,1000,1.0\n',
                encoding='utf-8',
            )
            environment = os.environ.copy()
            environment.pop('GITHUB_STEP_SUMMARY', None)
            environment['RUNNER_TELEMETRY_PROC_ROOT'] = str(proc_root)
            environment['RUNNER_TELEMETRY_CGROUP_ROOT'] = str(cgroup_root)
            result = subprocess.run(
                ['bash', str(TELEMETRY_SCRIPT), 'render', str(csv), str(baseline_path)],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )
            return result.stdout


if __name__ == '__main__':
    unittest.main()

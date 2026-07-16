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
import subprocess
import tempfile
import textwrap
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


if __name__ == '__main__':
    unittest.main()

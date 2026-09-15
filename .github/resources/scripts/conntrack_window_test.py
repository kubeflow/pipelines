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

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).with_name('conntrack-window.sh')


class ConntrackWindowTest(unittest.TestCase):

    def test_aggregates_cpu_counters_and_reports_window_deltas(self):
        output, summary = self._capture_and_report('end-api-down')

        self.assertEqual(output, summary)
        self.assertIn('| kind-control-plane | insert_failed | 3 | 8 | 5 | ok |', output)
        self.assertIn('| kind-control-plane | drop | 4 | 10 | 6 | ok |', output)
        self.assertIn('| kind-control-plane | early_drop | 0 | 0 | 0 | ok |', output)

    def test_replaced_node_suppresses_deltas(self):
        output, _ = self._capture_and_report('replaced')

        self.assertIn('node/netns replaced; delta unavailable', output)
        self.assertNotIn('| insert_failed |', output)

    def test_counter_reset_never_reports_a_negative_delta(self):
        output, _ = self._capture_and_report('reset')

        self.assertIn('| kind-control-plane | insert_failed | 3 | 1 | — | counter reset |', output)
        self.assertNotIn('| -2 |', output)

    def test_missing_baseline_is_distinct_from_zero(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            environment = os.environ.copy()
            environment.pop('GITHUB_STEP_SUMMARY', None)
            result = subprocess.run(
                ['bash', str(SCRIPT), 'report', f'{temporary_directory}/missing'],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )

        self.assertIn('Baseline unavailable', result.stdout)

    def _capture_and_report(self, end_phase):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            phase_file = temporary_path / 'phase'
            phase_file.write_text('baseline', encoding='utf-8')
            baseline = temporary_path / 'baseline.tsv'
            summary = temporary_path / 'summary.md'
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(bin_directory / 'docker', _FAKE_DOCKER)
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_PHASE_FILE'] = str(phase_file)
            environment['GITHUB_STEP_SUMMARY'] = str(summary)

            subprocess.run(
                ['bash', str(SCRIPT), 'capture', str(baseline)],
                check=True,
                env=environment,
            )
            phase_file.write_text(end_phase, encoding='utf-8')
            result = subprocess.run(
                ['bash', str(SCRIPT), 'report', str(baseline)],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )
            return result.stdout, summary.read_text(encoding='utf-8')

    @staticmethod
    def _write_executable(path, contents):
        path.write_text(contents, encoding='utf-8')
        path.chmod(0o755)


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
phase=$(<"$FAKE_PHASE_FILE")
if [[ "$phase" == "end-api-down" ]]; then
  exit 1
fi
echo "kind-control-plane"
'''

_FAKE_DOCKER = r'''#!/usr/bin/env bash
phase=$(<"$FAKE_PHASE_FILE")
case "$1" in
  inspect)
    [[ "$phase" == "replaced" ]] && echo "container-b" || echo "container-a"
    ;;
  exec)
    if [[ "$3" == "readlink" ]]; then
      [[ "$phase" == "replaced" ]] && echo "net:[2]" || echo "net:[1]"
    elif [[ "$3" == "conntrack" ]]; then
      case "$phase" in
        baseline)
          echo "cpu=0 insert=10 insert_failed=1 drop=2 early_drop=0 error=0 search_restart=3 clash=1"
          echo "cpu=1 insert=20 insert_failed=2 drop=2 early_drop=0 error=0 search_restart=3 clash=1"
          ;;
        reset)
          echo "cpu=0 insert=5 insert_failed=1 drop=1 early_drop=0 error=0 search_restart=1 clash=0"
          ;;
        *)
          echo "cpu=0 insert=12 insert_failed=4 drop=5 early_drop=0 error=0 search_restart=4 clash=2"
          echo "cpu=1 insert=23 insert_failed=4 drop=5 early_drop=0 error=0 search_restart=5 clash=2"
          ;;
      esac
    fi
    ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

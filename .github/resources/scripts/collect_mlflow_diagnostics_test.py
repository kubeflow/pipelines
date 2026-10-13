#!/usr/bin/env python3

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from pathlib import Path
import subprocess
import unittest

import collect_mlflow_diagnostics as diagnostics
import yaml


class MLflowDiagnosticsTest(unittest.TestCase):

    def test_migration_evidence_is_read_only_and_bounded(self):
        calls = []
        output = []

        def run(command, **kwargs):
            calls.append(command)
            self.assertEqual(kwargs['timeout'], 15)
            self.assertEqual(kwargs['stdin'], subprocess.DEVNULL)
            self.assertEqual(kwargs['stderr'], subprocess.STDOUT)
            kwargs['stdout'].write(b'migration pending')
            return subprocess.CompletedProcess(command, 0)

        diagnostics.collect(runner=run, output=output.append)
        self.assertEqual(len(calls), 6)
        self.assertTrue(all(c[4] in ('get', 'logs') for c in calls))
        self.assertTrue(
            all(c[1:4] == ['--request-timeout=10s', '-n', 'opendatahub']
                for c in calls))
        self.assertTrue(
            any('jobs' in c and diagnostics.MIGRATION_SELECTOR in c
                for c in calls))
        self.assertTrue(
            any('deployment/mlflow-operator-controller-manager' in c
                for c in calls))
        self.assertTrue(all('--tail=100' in c for c in calls if c[4] == 'logs'))
        self.assertNotIn('secret', ' '.join(' '.join(c) for c in calls).lower())
        self.assertEqual(output.count('migration pending'), 6)

    def test_unavailable_and_timed_out_commands_preserve_later_evidence(self):
        output = []
        calls = []

        def run(command, **kwargs):
            calls.append(command)
            kwargs['stdout'].write(b'partial evidence')
            if len(calls) == 1:
                raise subprocess.TimeoutExpired(command, 15)
            if len(calls) == 2:
                raise FileNotFoundError()
            return subprocess.CompletedProcess(command, 1)

        diagnostics.collect(runner=run, output=output.append)
        self.assertEqual(len(calls), 6)
        self.assertEqual(output.count('partial evidence'), 6)
        self.assertTrue(any('TimeoutExpired' in value for value in output))
        self.assertTrue(any('FileNotFoundError' in value for value in output))
        self.assertEqual(output.count('Diagnostic command exited 1.'), 4)

    def test_output_is_capped_and_database_credentials_are_redacted(self):
        output = []

        def run(command, **kwargs):
            kwargs['stdout'].write(b'postgresql://user:secret@database/db\n' +
                                   b'x' * diagnostics.MAX_OUTPUT_BYTES)
            return subprocess.CompletedProcess(command, 0)

        diagnostics.collect(runner=run, output=output.append)
        self.assertNotIn('user:secret', '\n'.join(output))
        self.assertIn('postgresql://[redacted]@database/db', '\n'.join(output))
        self.assertEqual(output.count('[diagnostic output truncated]'), 6)

    def test_workflow_collects_after_failed_deploy_without_masking_failure(
            self):
        root = Path(__file__).resolve().parents[3]
        workflow = yaml.safe_load(
            (root / '.github/workflows/e2e-test.yml').read_text())
        steps = next(job['steps'] for job in workflow['jobs'].values() if any(
            step.get('id') == 'deploy-mlflow' for step in job.get('steps', [])))
        deploy = next(i for i, step in enumerate(steps)
                      if step.get('id') == 'deploy-mlflow')
        diagnostic = steps[deploy + 1]
        self.assertEqual(
            diagnostic['if'],
            "${{ failure() && steps.deploy-mlflow.outcome == 'failure' }}")
        self.assertEqual(diagnostic['timeout-minutes'], 2)
        self.assertIn('collect_mlflow_diagnostics.py', diagnostic['run'])
        self.assertNotIn('continue-on-error', steps[deploy])


if __name__ == '__main__':
    unittest.main()

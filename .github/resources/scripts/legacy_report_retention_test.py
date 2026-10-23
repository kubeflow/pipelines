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
"""Successful qualification runs must retain complete raw test evidence."""

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class LegacyReportRetentionTest(unittest.TestCase):
    def setUp(self):
        self.api = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text()
        )['runs']['steps']
        self.legacy = yaml.safe_load(
            (
                ROOT / '.github/workflows/legacy-v2-api-integration-tests.yml'
            ).read_text()
        )['jobs']['api-integration-tests-v2']['steps']

    def step(self, steps, name):
        matches = [step for step in steps if step['name'] == name]
        self.assertEqual(
            len(matches), 1, f'Missing qualification evidence step: {name}'
        )
        return matches[0]

    def test_passing_api_reports_are_required_and_retained(self):
        validate = self.step(self.api, 'Validate successful API reports')
        upload = self.step(self.api, 'Retain successful API reports')
        for step in (validate, upload):
            self.assertIn("steps.run-tests.outcome == 'success'", step['if'])
            self.assertIn(
                "inputs.test_directory == './backend/test/v2/api'", step['if']
            )
            self.assertIn('!cancelled()', step['if'])
            self.assertFalse(step.get('continue-on-error', False))
        self.assertEqual(
            upload['with']['path'].splitlines(),
            [
                '${{ inputs.test_directory }}/reports/api.json',
                '${{ inputs.test_directory }}/reports/junit.xml',
                '${{ inputs.test_directory }}/reports/provenance.json',
                '${{ inputs.test_directory }}/reports/legacy-v2-migration-specs.json',
                '${{ inputs.test_directory }}/reports/legacy-v2-service-account-specs.json',
            ],
        )
        self.assert_upload(upload)
        self.assertIn('github.run_id', upload['with']['name'])
        self.assertIn('github.run_attempt', upload['with']['name'])
        self.assertIn(
            'steps.api-report-evidence.outputs.suffix', upload['with']['name']
        )
        self.assertIn('uuidgen', validate['run'])
        self.check_validation(
            validate,
            'REPORT_DIRECTORY',
            [
                'api.json',
                'junit.xml',
                'legacy-v2-migration-specs.json',
                'legacy-v2-service-account-specs.json',
            ],
        )

    def test_passing_legacy_output_is_required_and_retained(self):
        validate = self.step(self.legacy, 'Validate successful legacy report')
        upload = self.step(self.legacy, 'Retain successful legacy report')
        for step in (validate, upload):
            self.assertIn("steps.tests.outcome == 'success'", step['if'])
            self.assertIn('!cancelled()', step['if'])
            self.assertFalse(step.get('continue-on-error', False))
        self.assertEqual(
            upload['with']['path'].splitlines(),
            [
                '${{ runner.temp }}/go-test-output.log',
                '${{ runner.temp }}/legacy-provenance.json',
            ],
        )
        self.assert_upload(upload)
        for dimension in (
            'github.run_id',
            'github.run_attempt',
            'matrix.pipeline_store',
            'matrix.k8s_version',
            'matrix.pod_to_pod_tls_enabled',
            'matrix.db_type',
            'matrix.test_group',
        ):
            self.assertIn(dimension, upload['with']['name'])
        self.check_validation(
            validate, 'REPORT_DIRECTORY', ['go-test-output.log']
        )

    def assert_upload(self, upload):
        self.assertEqual(
            upload['uses'], './.github/actions/upload-artifact-with-retry'
        )
        self.assertEqual(upload['with']['if-no-files-found'], 'error')
        self.assertEqual(upload['with']['retention-days'], 30)

    def check_validation(self, step, directory_variable, filenames):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            env = dict(os.environ, **{directory_variable: directory})
            env['GITHUB_OUTPUT'] = str(root / 'output')
            for key in step['env']:
                if key != directory_variable:
                    env[key] = '{}' if key == 'MATRIX_JSON' else 'test'
            manifests = root / 'manifests'
            manifests.mkdir()
            env['MANIFEST_DIRECTORY'] = str(manifests)
            for manifest in (
                'legacy-v2-migration-specs.json',
                'legacy-v2-service-account-specs.json',
            ):
                (manifests / manifest).write_text('[]')

            def path(filename):
                return (
                    manifests if filename.startswith('legacy-v2-') else root
                ) / filename

            for absent in filenames:
                for filename in filenames:
                    path(filename).write_text('test report\n')
                path(absent).unlink()
                missing = subprocess.run(
                    ['bash', '-e', '-c', step['run']],
                    env=env,
                    capture_output=True,
                    text=True,
                )
                self.assertNotEqual(missing.returncode, 0)
                self.assertIn(absent, missing.stdout)
                path(absent).touch()
                empty = subprocess.run(
                    ['bash', '-e', '-c', step['run']],
                    env=env,
                    capture_output=True,
                )
                self.assertNotEqual(empty.returncode, 0)
            for filename in filenames:
                path(filename).write_text('test report\n')
            complete = subprocess.run(
                ['bash', '-e', '-c', step['run']],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(complete.returncode, 0, complete.stderr)
            provenance_path = root / (
                'provenance.json'
                if 'api.json' in filenames
                else 'legacy-provenance.json'
            )
            provenance = json.loads(provenance_path.read_text())
            for key in (
                'source_sha',
                'head_sha',
                'run_id',
                'run_attempt',
                'job_name',
            ):
                self.assertEqual(provenance[key], 'test')
            self.assertIn('merge commit', provenance['source_sha_kind'])
            if 'api.json' in filenames:
                for manifest in manifests.iterdir():
                    self.assertEqual(
                        (root / manifest.name).read_bytes(),
                        manifest.read_bytes(),
                    )

    def test_failure_diagnostics_remain_available(self):
        for name in ('Upload JUnit XML on failure', 'Upload Kind cluster logs'):
            self.assertIn(
                "steps.run-tests.outcome != 'success'",
                self.step(self.api, name)['if'],
            )
        self.assertEqual(
            self.step(self.legacy, 'Collect test results')['if'],
            'failure() || cancelled()',
        )


if __name__ == '__main__':
    unittest.main()

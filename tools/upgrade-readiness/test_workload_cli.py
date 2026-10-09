# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import readiness
from test_source_observation import evidence as observation_evidence
import workload_inventory


class WorkloadCliTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.kube = self.root / 'kube.json'
        self.kube.write_text('{"items": []}')
        self.workloads = self.root / 'workloads.json'
        self.data = {key: [] for key in workload_inventory.RESOURCES}
        self.data['experiments'] = [
            dict(experiment_id='exp', namespace='team-a')
        ]
        self.data['runs'] = [
            dict(
                run_id='r',
                experiment_id='exp',
                pipeline_spec={
                    'pipelineInfo': {
                        'name': 'private-pipeline-name'
                    },
                    'root': {}
                },
                runtime_config={
                    'parameters': {
                        'password': 'secret-param-value'
                    }
                })
        ]
        self.workloads.write_text(json.dumps(self.data))
        self.args = [
            '--inventory',
            str(self.kube), '--system-namespace', 'team-a', '--source-version',
            '2.17.2', '--format', 'json'
        ]

    def invoke(self, extra):
        output, errors = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(
                errors):
            try:
                code = readiness.main(self.args + extra)
            except SystemExit as error:
                code = error.code
        return code, output.getvalue(), errors.getvalue()

    def test_offline_report_is_scoped_sanitized_and_not_green(self):
        code, output, errors = self.invoke([
            '--include-workloads', '--workload-inventory',
            str(self.workloads)
        ])
        self.assertEqual((code, errors), (2, ''))
        report = json.loads(output)
        self.assertEqual(report['assessment'], 'incomplete')
        self.assertEqual(report['source']['workload_summary']['v2_ir'], 1)
        self.assertFalse(report['source']['workload_collection']['complete'])
        self.assertEqual(
            report['source']['workload_collection']['record_counts']['runs'], 1)
        self.assertTrue(report['control_coverage'])
        for value in ('private-pipeline-name', 'secret-param-value',
                      'runtime_config'):
            self.assertNotIn(value, output)

    def test_network_inventory_requires_opt_in_and_cannot_mix_exports(self):
        for options in [['--workload-inventory',
                         str(self.workloads)], ['--include-workloads'],
                        ['--include-shared-pipelines'],
                        ['--source-single-user'],
                        [
                            '--include-workloads', '--workload-inventory',
                            str(self.workloads), '--kfp-endpoint',
                            'http://127.0.0.1:1234'
                        ],
                        [
                            '--include-workloads', '--workload-inventory',
                            str(self.workloads), '--source-single-user',
                            '--namespace', 'team-b'
                        ]]:
            with self.subTest(options=options):
                self.assertEqual(self.invoke(options)[0], 1)

    def test_live_inventory_needs_no_policy_or_schedule_scan(self):
        normalized = workload_inventory.validate(self.data, ['team-a'])
        coverage = dict(
            record_counts={
                key: len(value) for key, value in normalized.items()
            },
            failed_checks=1,
            complete=False,
            requested_namespaces=['team-a'])
        failure = dict(
            namespace='team-a',
            resource='pipeline_versions/p',
            reason='http_403')
        with mock.patch.object(readiness,
                               'Client') as client, mock.patch.object(
                                   workload_inventory,
                                   'collect',
                                   return_value=(normalized, [failure],
                                                 coverage)) as collect:
            code, output, _ = self.invoke([
                '--include-workloads', '--kfp-endpoint',
                'https://source.test/pipeline'
            ])
        self.assertEqual(code, 2)
        collect.assert_called_once_with(client.return_value, ['team-a'], False,
                                        None)
        self.assertIn('http_403', output)
        self.assertFalse(json.loads(output)['source']['schedules_requested'])

    def test_scope_forgery_does_not_become_evidence(self):
        self.data['experiments'][0]['namespace'] = 'team-b'
        self.data['runs'][0]['_readiness_namespace_evidence'] = 'api_namespace'
        self.data['runs'][0]['namespace'] = 'team-a'
        self.workloads.write_text(json.dumps(self.data))
        code, output, errors = self.invoke([
            '--include-workloads', '--workload-inventory',
            str(self.workloads)
        ])
        self.assertEqual((code, output), (1, ''))
        self.assertIn('No readiness conclusion', errors)

    def test_markdown_exposes_coverage_not_raw_records(self):
        self.args[-1] = 'markdown'
        code, output, _ = self.invoke([
            '--include-workloads', '--workload-inventory',
            str(self.workloads)
        ])
        self.assertEqual(code, 2)
        self.assertIn('Stored workload collection: 2 records', output)
        self.assertIn('Per-control coverage:', output)
        self.assertNotIn('secret-param-value', output)

    def test_observation_import_keeps_raw_callers_out_of_cli_report(self):
        data = observation_evidence()
        data.update(
            started_at='2020-01-01T00:00:00Z',
            planned_end_at='2020-01-01T00:01:00Z',
            checkpoint_at='2020-01-01T00:01:00Z',
            ended_at='2020-01-01T00:01:00Z')
        data['operations']['authorization'] = 1
        data['unobserved_operations'].remove('authorization')
        data['records'] = [
            dict(
                operation='authorization',
                observed_at='2020-01-01T00:00:10Z',
                caller='private-caller')
        ]
        path = self.root / 'observation.json'
        path.write_text(json.dumps(data))
        code, output, errors = self.invoke(['--source-observation', str(path)])
        self.assertEqual((code, errors), (2, ''))
        self.assertEqual(
            json.loads(output)['source']['observation']
            ['authenticated_callers_observed'], 1)
        self.assertNotIn('private-caller', output)


if __name__ == '__main__':
    unittest.main()

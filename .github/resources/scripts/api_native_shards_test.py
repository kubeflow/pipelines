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
"""Preserve native API coverage while splitting independent runner jobs."""

import json
import os
import subprocess
import tempfile
import unittest
from itertools import product
from pathlib import Path

import api_multi_user_shards_test as multi_user
import verify_migrated_specs
import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/api-server-tests.yml'


class ApiNativeShardsTest(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text())
        self.job = self.workflow['jobs']['api-test-k8s-native']
        self.run_tests = next(
            s for s in self.job['steps'] if s.get('id') == 'test-run'
        )

    def matrix_rows(self, event):
        return multi_user.ApiMultiUserShardsTest.matrix_rows(self, event)

    def test_all_native_configurations_keep_both_automatic_shards(self):
        configurations = list(
            product(
                ['v1.36.1', 'v1.33.12'],
                ['true', 'false'],
                ['v4.0.13', 'v4.1.6'],
            )
        )
        for event, shards in [
            ('pull_request', ['A', 'B']),
            ('workflow_dispatch', ['all']),
        ]:
            with self.subTest(event=event):
                rows = self.matrix_rows(event)
                actual = [
                    (
                        r['k8s_version'],
                        r['uploadPipelinesWithKubernetesClient'],
                        r['argo_version'],
                        r['test_shard'],
                    )
                    for r in rows
                ]
                self.assertCountEqual(
                    actual,
                    [(*c, shard) for c in configurations for shard in shards],
                )
                for row in rows:
                    self.assertEqual(row['cache_enabled'], 'true')
                    self.assertEqual(row['pipeline_store'], 'kubernetes')
        self.assertFalse(self.job['strategy']['fail-fast'])

    def configure(self, row, event):
        step = next(s for s in self.job['steps'] if s.get('id') == 'configure')
        context = {
            'matrix': row,
            'github': {'event_name': event},
            'env': self.workflow['env'],
            'inputs': {
                'test_label': 'DO_NOT_APPLY',
                'number_of_parallel_tests': '99',
            },
        }
        environment = dict(os.environ)
        environment.update(
            {
                key: multi_user.render_references(value, context)
                for key, value in step.get('env', {}).items()
            }
        )
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            environment['GITHUB_OUTPUT'] = str(output)
            result = subprocess.run(
                [
                    'bash',
                    '-e',
                    '-o',
                    'pipefail',
                    '-c',
                    multi_user.render_references(step['run'], context),
                ],
                env=environment,
                cwd=directory,
                text=True,
                capture_output=True,
                timeout=10,
                check=False,
            )
            self.assertEqual(
                result.returncode, 0, result.stdout + result.stderr
            )
            return dict(
                line.split('=', 1) for line in output.read_text().splitlines()
            )

    def test_executable_filters_partition_full_suite_and_preserve_manual(self):
        expected = {
            'A': '(ApiServerTests) && (ApiPipelineShardA || Pipeline)',
            'B': '(ApiServerTests) && !(ApiPipelineShardA || Pipeline)',
            'all': 'ApiServerTests',
        }
        for event in ('pull_request', 'workflow_dispatch'):
            for row in self.matrix_rows(event):
                with self.subTest(event=event, row=row):
                    outputs = self.configure(row, event)
                    self.assertEqual(
                        outputs['TEST_LABEL'], expected[row['test_shard']]
                    )
        self.assertEqual(
            self.run_tests['with']['test_label'],
            '${{ steps.configure.outputs.TEST_LABEL }}',
        )
        self.assertEqual(
            self.run_tests['with']['num_parallel_nodes'],
            '${{ env.NUMBER_OF_PARALLEL_NODES }}',
        )
        self.assertEqual(self.workflow['env']['NUMBER_OF_PARALLEL_NODES'], 5)

    def test_fixtures_and_required_migration_follow_the_complement(self):
        expected = "${{ matrix.test_shard != 'A' }}"
        self.assertEqual(
            self.run_tests['with']['prepare_cache_fixtures'],
            "${{ matrix.cache_enabled == 'true' && matrix.test_shard != 'A' }}",
        )
        self.assertEqual(
            self.run_tests['with']['required_migration_specs'], expected
        )
        self.assertEqual(
            self.run_tests['with']['upload_pipelines_with_kubernetes_client'],
            '${{ matrix.uploadPipelinesWithKubernetesClient }}',
        )
        self.assertEqual(
            self.run_tests['with']['pipeline_store'],
            '${{ matrix.pipeline_store }}',
        )

    @unittest.skipUnless(
        os.environ.get('NATIVE_API_DRY_RUN_REPORTS'),
        'Set NATIVE_API_DRY_RUN_REPORTS to real Ginkgo dry-run reports',
    )
    def test_real_ginkgo_inventory_is_complete_and_disjoint(self):
        directory = Path(os.environ['NATIVE_API_DRY_RUN_REPORTS'])
        reports = {
            name: json.loads((directory / f'{name}.json').read_text())
            for name in ('all', 'A', 'B')
        }
        selected, registered = {}, {}
        for name, suites in reports.items():
            specs = [
                spec
                for suite in suites
                for spec in suite['SpecReports']
                if spec['LeafNodeType'] == 'It'
            ]
            identities = [
                tuple(spec['ContainerHierarchyTexts'] + [spec['LeafNodeText']])
                for spec in specs
            ]
            self.assertTrue(identities, f'Empty {name} inventory')
            self.assertEqual(
                len(identities), len(set(identities)), f'Duplicate {name} specs'
            )
            registered[name] = {
                key: (
                    spec.get('ContainerHierarchyLabels', []),
                    spec.get('LeafNodeLabels', []),
                )
                for key, spec in zip(identities, specs)
            }
            fixture_specs = {
                key
                for key, spec in zip(identities, specs)
                if 'LegacyCache'
                in (
                    spec.get('LeafNodeLabels', [])
                    + [
                        label
                        for labels in spec.get('ContainerHierarchyLabels', [])
                        for label in labels
                    ]
                )
                and spec['State'] == 'passed'
            }
            if name == 'A':
                self.assertFalse(
                    fixture_specs, 'Shard A requires cache fixtures'
                )
            else:
                self.assertTrue(
                    fixture_specs, f'Missing {name} cache fixture specs'
                )
            selected[name] = {
                key
                for key, spec in zip(identities, specs)
                if spec['State'] == 'passed'
            }
            self.assertTrue(selected[name], f'Empty {name} selection')
            self.assertTrue(
                all(
                    spec['State'] in ('passed', 'skipped', 'pending')
                    for spec in specs
                ),
                f'Failed {name} dry run',
            )
        self.assertEqual(registered['all'], registered['A'])
        self.assertEqual(registered['all'], registered['B'])
        self.assertFalse(selected['A'] & selected['B'])
        self.assertEqual(selected['all'], selected['A'] | selected['B'])
        required = json.loads(
            (
                ROOT / '.github/resources/legacy-v2-migration-specs.json'
            ).read_text()
        )
        self.assertEqual(
            verify_migrated_specs.verify(reports['all'], required), {}
        )
        self.assertEqual(
            verify_migrated_specs.verify(reports['B'], required), {}
        )
        self.assertEqual(
            set(verify_migrated_specs.verify(reports['A'], required)),
            set(required),
        )

    def test_job_and_report_names_remain_unique(self):
        for event in ('pull_request', 'workflow_dispatch'):
            rows = self.matrix_rows(event)
            for template in (
                self.job['name'],
                self.run_tests['with']['report_name'],
            ):
                names = [
                    multi_user.render_references(template, {'matrix': row})
                    for row in rows
                ]
                self.assertEqual(len(names), len(set(names)))
                self.assertIn('${{ matrix.test_shard }}', template)


if __name__ == '__main__':
    unittest.main()

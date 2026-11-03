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
"""Keep API multi-user sharding complete and manual inputs unchanged."""

import json
import os
import re
import subprocess
import tempfile
import unittest
from itertools import product
from pathlib import Path

import verify_migrated_specs
import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/api-server-tests.yml'
BASE_FILTERS = {
    'true': 'ApiServerTests && (PipelineUpload || PipelineCacheEnabled)',
    'false': 'ApiServerTests && !(PipelineUpload || PipelineCacheEnabled)',
}
CONFIGURATIONS = {('mysql', 'true'), ('mysql', 'false'), ('pgx', 'true')}
SHARD_FILTERS = {
    'A': '(ApiPipelineShardA || Pipeline)',
    'B': '!(ApiPipelineShardA || Pipeline)',
    'C': '!(ApiPipelineShardA || Pipeline) && !LegacyMigration',
}


def render_references(value, context):
    """Render only the simple context references used by this job."""

    def substitute(match):
        expression = match.group(1).strip()
        if not re.fullmatch(r'[\w-]+(?:\.[\w-]+)+', expression):
            raise AssertionError(f'Unsupported expression: {expression}')
        result = context
        for key in expression.split('.'):
            result = result.get(key, '') if isinstance(result, dict) else ''
        return str(result)

    return re.sub(r'\$\{\{(.*?)\}\}', substitute, str(value))


class ApiMultiUserShardsTest(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))
        self.job = self.workflow['jobs']['api-test-multi-user']

    def test_constants_changes_trigger_api_workflow(self):
        workflow = yaml.load(
            WORKFLOW.read_text(encoding='utf-8'), Loader=yaml.BaseLoader
        )
        self.assertIn(
            'backend/test/constants/**', workflow['on']['pull_request']['paths']
        )

    def matrix_rows(self, event):
        matrix = self.job['strategy']['matrix']
        self.assertIn('test_shard', matrix)
        # Match the two literal dispatch branches, without pretending to
        # implement the GitHub Actions expression language.
        branches = re.fullmatch(
            r"\$\{\{ fromJSON\(github\.event_name == 'workflow_dispatch' && "
            r"'([^']+)' \|\| '([^']+)'\) \}\}",
            matrix['test_shard'],
        )
        self.assertIsNotNone(branches)
        shard_values = json.loads(
            branches.group(1 if event == 'workflow_dispatch' else 2)
        )
        if event == 'workflow_dispatch':
            self.assertEqual(shard_values, ['all'])
        axes = {
            key: shard_values if key == 'test_shard' else value
            for key, value in matrix.items()
            if key not in ('include', 'exclude')
        }
        originals = [
            dict(zip(axes, values)) for values in product(*axes.values())
        ]
        originals = [
            row
            for row in originals
            if not any(
                all(row.get(key) == value for key, value in rule.items())
                for rule in matrix.get('exclude', [])
            )
        ]
        rows = [row.copy() for row in originals]
        additions = []
        for rule in matrix.get('include', []):
            matches = [
                index
                for index, row in enumerate(originals)
                if all(
                    key not in row or row[key] == value
                    for key, value in rule.items()
                )
            ]
            if not matches:
                additions.append(rule.copy())
            for index in matches:
                rows[index].update(rule)
        return rows + additions

    def test_automatic_and_manual_matrices_preserve_deployment_coverage(self):
        for event, shards in (
            ('pull_request', ['A', 'B', 'C']),
            ('workflow_dispatch', ['all']),
        ):
            with self.subTest(event=event):
                rows = self.matrix_rows(event)
                self.assertCountEqual(
                    [
                        (
                            row['db_type'],
                            row['cache_enabled'],
                            row['test_shard'],
                        )
                        for row in rows
                    ],
                    [
                        (*configuration, shard)
                        for configuration in CONFIGURATIONS
                        for shard in shards
                        if not (configuration[1] == 'false' and shard == 'C')
                    ],
                )
                for row in rows:
                    self.assertEqual(row['multi_user'], 'true')
                    self.assertEqual(row['k8s_version'], 'v1.36.1')
                    self.assertEqual(
                        row['test_label'], BASE_FILTERS[row['cache_enabled']]
                    )
        self.assertFalse(self.job['strategy']['fail-fast'])

    def test_job_and_report_names_are_unique_for_each_matrix_row(self):
        report = next(
            step for step in self.job['steps'] if step.get('id') == 'test-run'
        )['with']['report_name']
        for event in ('pull_request', 'workflow_dispatch'):
            for template in (self.job['name'], report):
                with self.subTest(event=event, template=template):
                    names = [
                        render_references(template, {'matrix': row})
                        for row in self.matrix_rows(event)
                    ]
                    self.assertEqual(len(names), len(set(names)), names)

    def configure(self, row, event, inputs):
        step = next(
            step for step in self.job['steps'] if step.get('id') == 'configure'
        )
        context = {
            'matrix': row,
            'github': {'event_name': event},
            'env': self.workflow['env'],
            'inputs': inputs,
        }
        environment = os.environ.copy()
        environment.update(
            {
                key: render_references(value, context)
                for key, value in step.get('env', {}).items()
            }
        )
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'github-output'
            environment['GITHUB_OUTPUT'] = str(output)
            result = subprocess.run(
                [
                    'bash',
                    '-e',
                    '-o',
                    'pipefail',
                    '-c',
                    render_references(step['run'], context),
                ],
                cwd=directory,
                env=environment,
                capture_output=True,
                text=True,
                timeout=10,
                check=False,
            )
            self.assertEqual(
                result.returncode, 0, result.stdout + result.stderr
            )
            return dict(
                line.split('=', 1) for line in output.read_text().splitlines()
            )

    def test_manual_dispatch_preserves_requested_label_nodes_and_namespace(
        self,
    ):
        inputs = {
            'number_of_parallel_tests': '7',
            'test_label': 'ApiServerTests && (PipelineUpload || PipelineRun) && !Smoke',
            'namespace': 'custom-kfp-namespace',
        }
        for row in self.matrix_rows('workflow_dispatch'):
            with self.subTest(row=row):
                actual = self.configure(row, 'workflow_dispatch', inputs)
                self.assertEqual(
                    actual,
                    {
                        'NUMBER_OF_NODES': inputs['number_of_parallel_tests'],
                        'TEST_LABEL': (
                            f'({inputs["test_label"]}) && !LegacyCache'
                            if row['cache_enabled'] == 'false'
                            else inputs['test_label']
                        ),
                        'NAMESPACE': inputs['namespace'],
                    },
                )

    def test_manual_default_and_explicit_cache_labels_respect_disabled_cache(
        self,
    ):
        for label in ('ApiServerTests', 'LegacyCache'):
            for row in self.matrix_rows('workflow_dispatch'):
                with self.subTest(label=label, row=row):
                    actual = self.configure(
                        row,
                        'workflow_dispatch',
                        {
                            'number_of_parallel_tests': '5',
                            'test_label': label,
                            'namespace': 'kubeflow',
                        },
                    )
                    expected = (
                        f'({label}) && !LegacyCache'
                        if row['cache_enabled'] == 'false'
                        else label
                    )
                    self.assertEqual(actual['TEST_LABEL'], expected)

    def test_automatic_runs_combine_cache_shards_and_legacy_compatibility(self):
        # Populated dispatch inputs must have no effect on an automatic run.
        inputs = {
            'number_of_parallel_tests': '7',
            'test_label': 'Smoke',
            'namespace': 'unused-dispatch-namespace',
        }
        for row in self.matrix_rows('pull_request'):
            with self.subTest(row=row):
                actual = self.configure(row, 'pull_request', inputs)
                expected_label = (
                    f'({BASE_FILTERS[row["cache_enabled"]]}) && '
                    f'{SHARD_FILTERS[row["test_shard"]]}'
                )
                if row['test_shard'] == 'B' and row['cache_enabled'] == 'true':
                    expected_label += ' && LegacyMigration'
                expected_label = f'({expected_label}) || LegacyAPICompatibility'
                if row['cache_enabled'] == 'false':
                    expected_label = f'({expected_label}) && !LegacyCache'
                self.assertEqual(
                    actual,
                    {
                        'NUMBER_OF_NODES': '2',
                        'TEST_LABEL': expected_label,
                        'NAMESPACE': 'kubeflow',
                    },
                )
        test_inputs = next(
            step for step in self.job['steps'] if step.get('id') == 'test-run'
        )['with']
        self.assertEqual(
            test_inputs['test_label'],
            '${{ steps.configure.outputs.TEST_LABEL }}',
        )
        self.assertEqual(
            test_inputs['num_parallel_nodes'],
            '${{ steps.configure.outputs.NUMBER_OF_NODES }}',
        )
        self.assertEqual(
            test_inputs['default_namespace'],
            '${{ steps.configure.outputs.NAMESPACE }}',
        )

    def test_fixture_and_schedule_ownership_remain_on_cached_b(self):
        steps = self.job['steps']
        run = next(step for step in steps if step.get('id') == 'test-run')
        self.assertEqual(
            run['with']['prepare_cache_fixtures'],
            "${{ matrix.cache_enabled == 'true' && (github.event_name == "
            "'workflow_dispatch' || matrix.test_shard == 'B') }}",
        )
        schedule = next(
            step for step in steps if step.get('id') == 'schedule-account'
        )
        self.assertEqual(
            schedule['if'],
            "${{ steps.deploy.outcome == 'success' && github.event_name != "
            "'workflow_dispatch' && matrix.cache_enabled == 'true' && "
            "matrix.test_shard == 'B' }}",
        )
        self.assertEqual(
            run['with']['required_service_account_specs'],
            "${{ steps.schedule-account.outcome == 'success' }}",
        )

    @unittest.skipUnless(
        os.environ.get('MULTI_USER_API_DRY_RUN_REPORTS'),
        'Set MULTI_USER_API_DRY_RUN_REPORTS to real Ginkgo dry-run reports',
    )
    def test_real_ginkgo_inventory_preserves_every_original_spec(self):
        directory = Path(os.environ['MULTI_USER_API_DRY_RUN_REPORTS'])
        required = json.loads(
            (
                ROOT / '.github/resources/legacy-v2-service-account-specs.json'
            ).read_text()
        )
        for cache in ('true', 'false'):
            with self.subTest(cache=cache):
                names = ['baseline-A', 'baseline-B', 'A', 'B']
                if cache == 'true':
                    names.append('C')
                selected, registered, reports = {}, {}, {}
                compatibility, fixtures = set(), set()
                for name in names:
                    reports[name] = json.loads(
                        (directory / cache / f'{name}.json').read_text()
                    )
                    specs = [
                        spec
                        for suite in reports[name]
                        for spec in suite['SpecReports']
                        if spec['LeafNodeType'] == 'It'
                    ]
                    identities = [
                        tuple(
                            spec['ContainerHierarchyTexts']
                            + [spec['LeafNodeText']]
                        )
                        for spec in specs
                    ]
                    self.assertTrue(identities, f'Empty {name} inventory')
                    self.assertEqual(len(identities), len(set(identities)))
                    registered[name] = {}
                    selected[name] = set()
                    for key, spec in zip(identities, specs):
                        labels = set(spec.get('LeafNodeLabels', [])) | {
                            label
                            for labels in spec.get(
                                'ContainerHierarchyLabels', []
                            )
                            for label in labels
                        }
                        registered[name][key] = labels
                        self.assertIn(
                            spec['State'], ('passed', 'skipped', 'pending')
                        )
                        if spec['State'] != 'passed':
                            continue
                        selected[name].add(key)
                        if 'LegacyAPICompatibility' in labels:
                            compatibility.add(key)
                        if 'LegacyCache' in labels:
                            fixtures.add(key)
                    self.assertTrue(selected[name], f'Empty {name} selection')
                    self.assertEqual(registered['baseline-A'], registered[name])
                self.assertTrue(compatibility)
                self.assertEqual(selected['baseline-A'], selected['A'])
                new_b = selected['B'] | selected.get('C', set())
                self.assertEqual(selected['baseline-B'], new_b)
                old = selected['baseline-A'] | selected['baseline-B']
                current = selected['A'] | new_b
                self.assertEqual(old, current)
                candidate_names = ['A', 'B'] + (
                    ['C'] if cache == 'true' else []
                )
                for index, name in enumerate(candidate_names):
                    self.assertTrue(compatibility <= selected[name])
                    for other in candidate_names[index + 1 :]:
                        self.assertEqual(
                            selected[name] & selected[other], compatibility
                        )
                if cache == 'true':
                    self.assertTrue(fixtures)
                    self.assertTrue(fixtures <= selected['B'])
                    self.assertFalse(fixtures & (selected['A'] | selected['C']))
                    self.assertEqual(
                        verify_migrated_specs.verify(reports['B'], required), {}
                    )
                    for name in ('A', 'C'):
                        self.assertEqual(
                            set(
                                verify_migrated_specs.verify(
                                    reports[name], required
                                )
                            ),
                            set(required),
                        )
                else:
                    self.assertFalse(fixtures)


if __name__ == '__main__':
    unittest.main()

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

from itertools import product
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

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

    def matrix_rows(self, event):
        matrix = self.job['strategy']['matrix']
        self.assertIn('test_shard', matrix)
        # Match the two literal dispatch branches, without pretending to
        # implement the GitHub Actions expression language.
        branches = re.fullmatch(
            r"\$\{\{ fromJSON\(github\.event_name == 'workflow_dispatch' && "
            r"'([^']+)' \|\| '([^']+)'\) \}\}", matrix['test_shard'])
        self.assertIsNotNone(branches)
        shard_values = json.loads(
            branches.group(1 if event == 'workflow_dispatch' else 2))
        self.assertEqual(
            shard_values,
            ['all'] if event == 'workflow_dispatch' else ['A', 'B'])
        axes = {
            key: shard_values if key == 'test_shard' else value
            for key, value in matrix.items()
            if key not in ('include', 'exclude')
        }
        originals = [
            dict(zip(axes, values)) for values in product(*axes.values())
        ]
        originals = [
            row for row in originals if not any(
                all(row.get(key) == value
                    for key, value in rule.items())
                for rule in matrix.get('exclude', []))
        ]
        rows = [row.copy() for row in originals]
        additions = []
        for rule in matrix.get('include', []):
            matches = [
                index for index, row in enumerate(originals)
                if all(key not in row or row[key] == value
                       for key, value in rule.items())
            ]
            if not matches:
                additions.append(rule.copy())
            for index in matches:
                rows[index].update(rule)
        return rows + additions

    def test_automatic_and_manual_matrices_preserve_deployment_coverage(self):
        for event, shards in (('pull_request', ['A', 'B']),
                              ('workflow_dispatch', ['all'])):
            with self.subTest(event=event):
                rows = self.matrix_rows(event)
                self.assertCountEqual(
                    [(row['db_type'], row['cache_enabled'], row['test_shard'])
                     for row in rows], [(*configuration, shard)
                                        for configuration in CONFIGURATIONS
                                        for shard in shards])
                for row in rows:
                    self.assertEqual(row['multi_user'], 'true')
                    self.assertEqual(row['k8s_version'], 'v1.36.1')
                    self.assertEqual(row['test_label'],
                                     BASE_FILTERS[row['cache_enabled']])
        self.assertFalse(self.job['strategy']['fail-fast'])

    def test_job_and_report_names_are_unique_for_each_matrix_row(self):
        report = next(step for step in self.job['steps']
                      if step.get('id') == 'test-run')['with']['report_name']
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
            step for step in self.job['steps'] if step.get('id') == 'configure')
        context = {
            'matrix': row,
            'github': {
                'event_name': event
            },
            'env': self.workflow['env'],
            'inputs': inputs,
        }
        environment = os.environ.copy()
        environment.update({
            key: render_references(value, context)
            for key, value in step.get('env', {}).items()
        })
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'github-output'
            environment['GITHUB_OUTPUT'] = str(output)
            result = subprocess.run([
                'bash', '-e', '-o', 'pipefail', '-c',
                render_references(step['run'], context)
            ],
                                    cwd=directory,
                                    env=environment,
                                    capture_output=True,
                                    text=True,
                                    timeout=10,
                                    check=False)
            self.assertEqual(result.returncode, 0,
                             result.stdout + result.stderr)
            return dict(
                line.split('=', 1) for line in output.read_text().splitlines())

    def test_manual_dispatch_preserves_requested_label_nodes_and_namespace(
            self):
        inputs = {
            'number_of_parallel_tests':
                '7',
            'test_label':
                'ApiServerTests && (PipelineUpload || PipelineRun) && !Smoke',
            'namespace':
                'custom-kfp-namespace',
        }
        for row in self.matrix_rows('workflow_dispatch'):
            with self.subTest(row=row):
                actual = self.configure(row, 'workflow_dispatch', inputs)
                self.assertEqual(
                    actual, {
                        'NUMBER_OF_NODES': inputs['number_of_parallel_tests'],
                        'TEST_LABEL': inputs['test_label'],
                        'NAMESPACE': inputs['namespace'],
                    })

    def test_automatic_runs_combine_cache_and_shard_filters(self):
        # Populated dispatch inputs must have no effect on an automatic run.
        inputs = {
            'number_of_parallel_tests': '7',
            'test_label': 'Smoke',
            'namespace': 'unused-dispatch-namespace',
        }
        for row in self.matrix_rows('pull_request'):
            with self.subTest(row=row):
                actual = self.configure(row, 'pull_request', inputs)
                self.assertEqual(
                    actual, {
                        'NUMBER_OF_NODES': '2',
                        'TEST_LABEL':
                            f"({BASE_FILTERS[row['cache_enabled']]}) && "
                            f"{SHARD_FILTERS[row['test_shard']]}",
                        'NAMESPACE': 'kubeflow',
                    })
        test_inputs = next(step for step in self.job['steps']
                           if step.get('id') == 'test-run')['with']
        self.assertEqual(test_inputs['test_label'],
                         '${{ steps.configure.outputs.TEST_LABEL }}')
        self.assertEqual(test_inputs['num_parallel_nodes'],
                         '${{ steps.configure.outputs.NUMBER_OF_NODES }}')
        self.assertEqual(test_inputs['default_namespace'],
                         '${{ steps.configure.outputs.NAMESPACE }}')


if __name__ == '__main__':
    unittest.main()

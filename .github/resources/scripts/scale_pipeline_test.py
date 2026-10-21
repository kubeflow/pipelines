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
"""Contracts for the dedicated DAG scale lane and its compiled fixture."""

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]


class ScalePipelineTest(unittest.TestCase):

    def test_exactly_120_uncached_container_tasks_with_layered_dependencies(
            self):
        fixture = ROOT / 'test_data/sdk_compiled_pipelines/valid/scale/dag_120.yaml'
        pipeline = yaml.safe_load(fixture.read_text())
        tasks = pipeline['root']['dag']['tasks']
        self.assertEqual(len(tasks), 120)
        by_id = {
            int(task['inputs']['parameters']['node_id']['runtimeValue']
                ['constant']):
                name for name, task in tasks.items()
        }
        self.assertEqual(set(by_id), set(range(120)))
        for node_id, name in by_id.items():
            with self.subTest(node_id=node_id):
                task = tasks[name]
                self.assertEqual(task['taskInfo']['name'],
                                 f'scale-node-{node_id:03d}')
                self.assertFalse(
                    task.get('cachingOptions', {}).get('enableCache', False))
                expected_dependencies = []
                if node_id >= 20:
                    layer, column = divmod(node_id, 20)
                    expected_dependencies = [
                        by_id[node_id - 20],
                        by_id[(layer - 1) * 20 + (column + 1) % 20],
                    ]
                self.assertCountEqual(
                    task.get('dependentTasks', []), expected_dependencies)
                component = pipeline['components'][task['componentRef']['name']]
                self.assertNotIn('dag', component)
                container = pipeline['deploymentSpec']['executors'][
                    component['executorLabel']]['container']
                self.assertEqual(container['image'], 'docker.io/alpine:3.23')
                self.assertEqual(container['command'], ['echo'])
                self.assertEqual(container['args'],
                                 ["{{$.inputs.parameters['node_id']}}"])

    def test_scale_lane_is_dedicated_and_uses_shared_setup(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text())
        job = workflow['jobs']['dag-scale-test']
        self.assertNotIn('strategy', job)
        self.assertEqual(job['timeout-minutes'], 60)
        actions = {step.get('uses'): step for step in job['steps']}
        self.assertIn('./.github/actions/create-cluster', actions)
        deploy = actions['./.github/actions/deploy']['with']
        self.assertEqual(deploy['cache_enabled'], 'false')
        self.assertEqual(deploy['image_path'], 'images_${{ github.run_id }}')
        test = actions['./.github/actions/test-and-report']['with']
        self.assertEqual(test['test_label'], 'E2EScale')
        self.assertEqual(test['num_parallel_nodes'], '1')
        self.assertEqual(test['cache_enabled'], 'false')
        self.assertIn('build', workflow['jobs'])


if __name__ == '__main__':
    unittest.main()

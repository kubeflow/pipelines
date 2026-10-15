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

import importlib.util
from pathlib import Path
import sys
from types import SimpleNamespace
import unittest
from unittest import mock

SCRIPT_DIR = Path(__file__).parent / 'kfp-readiness'


class WaitForPodsTest(unittest.TestCase):

    def setUp(self):
        spec = importlib.util.spec_from_file_location(
            'wait_for_pods', SCRIPT_DIR / 'wait_for_pods.py')
        assert spec is not None and spec.loader is not None
        self.module = importlib.util.module_from_spec(spec)
        # Import the real readiness and diagnostics code without cluster access.
        with mock.patch.dict(sys.modules, {
                'kubernetes': mock.MagicMock(),
                'urllib3': mock.MagicMock(),
        }), mock.patch.object(sys, 'path', [str(SCRIPT_DIR), *sys.path]):
            spec.loader.exec_module(self.module)

    def test_agent_classification_preserves_readiness_and_diagnostics(self):
        component_label = 'workflows.argoproj.io/component'
        cases = [
            ('workflow-agent', {
                component_label: 'agent'
            }, True),
            ('renamed-workflow-pod', {
                component_label: 'agent'
            }, True),
            ('custom-agent', None, False),
            ('custom-agent', {}, False),
            ('custom-agent', {
                component_label: 'controller'
            }, False),
            ('ml-pipeline-persistenceagent-abc', None, False),
            ('workflow-system-container-executor', None, True),
        ]
        ready_pod = SimpleNamespace(
            metadata=SimpleNamespace(
                name='ml-pipeline-abc', uid='ready', labels=None),
            status=SimpleNamespace(
                phase='Running',
                container_statuses=[
                    SimpleNamespace(
                        name='main',
                        ready=True,
                        state=SimpleNamespace(waiting=None))
                ],
                init_container_statuses=None,
            ),
        )
        for name, labels, excluded in cases:
            with self.subTest(name=name, labels=labels):
                pending_pod = SimpleNamespace(
                    metadata=SimpleNamespace(
                        name=name, uid='pending', labels=labels),
                    status=SimpleNamespace(
                        phase='Pending',
                        container_statuses=[
                            SimpleNamespace(
                                name='main',
                                ready=False,
                                state=SimpleNamespace(
                                    waiting=SimpleNamespace(
                                        reason='ContainerCreating',
                                        message='starting')))
                        ],
                        init_container_statuses=None,
                    ),
                )
                self.module.v1.list_namespaced_pod.return_value = SimpleNamespace(
                    items=[ready_pod, pending_pod])

                statuses, diagnostic_pods = self.module.get_pod_statuses()

                self.assertEqual(self.module.all_pods_ready(statuses), excluded)
                self.assertEqual(
                    set(statuses), {'ml-pipeline-abc'}
                    if excluded else {'ml-pipeline-abc', name})
                self.assertEqual(diagnostic_pods,
                                 set() if excluded else {(name, 'pending')})
                if not excluded:
                    self.assertEqual(statuses[name], ('Pending', 0, 1, [
                        'Waiting on Container: main - ContainerCreating: starting'
                    ]))


if __name__ == '__main__':
    unittest.main()

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
from types import SimpleNamespace
import subprocess
import unittest

MODULE_PATH = (Path(__file__).parent / 'kfp-readiness' /
               'slow_pod_diagnostics.py')
SPEC = importlib.util.spec_from_file_location('slow_pod_diagnostics',
                                              MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)
SlowContainerCreatingDiagnostics = MODULE.SlowContainerCreatingDiagnostics
container_creating_pod_identities = MODULE.container_creating_pod_identities


class SlowContainerCreatingDiagnosticsTest(unittest.TestCase):

    def setUp(self):
        self.now = 0.0
        self.commands = []
        self.output = []

        def run(command, **_kwargs):
            self.commands.append(command)
            return subprocess.CompletedProcess(command, 0, stdout='captured')

        self.diagnostics = SlowContainerCreatingDiagnostics(
            threshold_seconds=60,
            clock=lambda: self.now,
            command_runner=run,
            output=self.output.append,
        )

    def test_captures_describe_and_events_once_after_threshold(self):
        pod = ('cache-server-abc', 'pod-uid')

        self.diagnostics.observe('kubeflow', [pod])
        self.now = 59
        self.diagnostics.observe('kubeflow', [pod])
        self.assertEqual(self.commands, [])

        self.now = 60
        self.diagnostics.observe('kubeflow', [pod])
        self.now = 120
        self.diagnostics.observe('kubeflow', [pod])

        self.assertEqual(len(self.commands), 2)
        self.assertIn('describe', self.commands[0])
        self.assertIn('involvedObject.uid=pod-uid', self.commands[1])
        self.assertIn('ContainerCreating', self.output[0])

    def test_reports_a_new_episode_after_the_condition_clears(self):
        pod = ('cache-server-abc', 'pod-uid')
        self.diagnostics.observe('kubeflow', [pod])
        self.now = 60
        self.diagnostics.observe('kubeflow', [pod])
        self.diagnostics.observe('kubeflow', [])

        self.now = 70
        self.diagnostics.observe('kubeflow', [pod])
        self.now = 130
        self.diagnostics.observe('kubeflow', [pod])

        self.assertEqual(len(self.commands), 4)

    def test_recreated_pod_uid_has_an_independent_threshold(self):
        old_pod = ('cache-server-abc', 'old-uid')
        new_pod = ('cache-server-abc', 'new-uid')
        self.diagnostics.observe('kubeflow', [old_pod])
        self.now = 60
        self.diagnostics.observe('kubeflow', [old_pod])

        self.now = 70
        self.diagnostics.observe('kubeflow', [new_pod])
        self.now = 129
        self.diagnostics.observe('kubeflow', [new_pod])
        self.assertEqual(len(self.commands), 2)
        self.now = 130
        self.diagnostics.observe('kubeflow', [new_pod])
        self.assertEqual(len(self.commands), 4)

    def test_command_failures_are_best_effort_and_not_retried(self):
        calls = 0

        def fail(command, **_kwargs):
            nonlocal calls
            calls += 1
            if calls == 1:
                return subprocess.CompletedProcess(command, 1, stdout='failed')
            raise subprocess.TimeoutExpired(command, 15)

        diagnostics = SlowContainerCreatingDiagnostics(
            threshold_seconds=0,
            clock=lambda: 0,
            command_runner=fail,
            output=self.output.append,
        )
        pod = ('cache-server-abc', 'pod-uid')

        diagnostics.observe('kubeflow', [pod])
        diagnostics.observe('kubeflow', [pod])

        self.assertEqual(calls, 2)
        self.assertTrue(any('status 1' in line for line in self.output))
        self.assertTrue(any('unable to collect events' in line
                            for line in self.output))

    def test_detects_regular_and_init_containers_being_created(self):
        def pod(name, uid, regular_reason=None, init_reason=None):
            def status(reason):
                return SimpleNamespace(state=SimpleNamespace(
                    waiting=(SimpleNamespace(reason=reason)
                             if reason else None)))

            return SimpleNamespace(
                metadata=SimpleNamespace(name=name, uid=uid),
                status=SimpleNamespace(
                    container_statuses=[status(regular_reason)],
                    init_container_statuses=[status(init_reason)],
                ),
            )

        identities = container_creating_pod_identities([
            pod('regular', 'uid-1', regular_reason='ContainerCreating'),
            pod('init', 'uid-2', init_reason='ContainerCreating'),
            pod('ready', 'uid-3', regular_reason='Running'),
        ])

        self.assertEqual(identities, {
            ('regular', 'uid-1'),
            ('init', 'uid-2'),
        })


if __name__ == '__main__':
    unittest.main()

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
"""Cache fixture optimization preserves the IR and executes the current SDK."""

import copy
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import prepare_cache_fixtures as prepare
import yaml

ROOT = prepare.ROOT


def source(name):
    return list(
        yaml.safe_load_all(
            (ROOT / 'backend/test/v2/resources' / name).read_text()
        )
    )


class CacheFixtureImageTest(unittest.TestCase):
    def test_sdk_source_changes_trigger_fixture_consumers(self):
        paths = [
            'sdk/python/kfp/__init__.py',
            'sdk/python/kfp/version.py',
            'sdk/python/kfp/dsl/__init__.py',
            'sdk/python/kfp/dsl/executor.py',
            'sdk/python/kfp/dsl/executor_main.py',
            'sdk/python/kfp/dsl/types/type_annotations.py',
        ]
        script = """
const gate = require('./.github/resources/scripts/ci_expected_workflows.js');
const {inventory} = gate.loadLocalInventory(process.cwd());
const paths = JSON.parse(process.argv[1]);
console.log(JSON.stringify(paths.map(path => inventory.workflows.filter(workflow =>
  workflow.pull_request !== null && gate.applicable(workflow.pull_request,
    'master', [path])).map(workflow => workflow.path))));
"""
        result = subprocess.run(
            ['node', '-e', script, json.dumps(paths)],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
        consumers = {
            '.github/workflows/api-server-tests.yml',
            '.github/workflows/ci-scripts-tests.yml',
        }
        for path, selected in zip(
            paths, json.loads(result.stdout), strict=True
        ):
            with self.subTest(path=path):
                self.assertTrue(consumers.issubset(selected), selected)

    def test_only_image_and_installer_prefix_change(self):
        for name in prepare.FIXTURES:
            with self.subTest(name=name):
                original = source(name)
                actual = prepare.transform(
                    name, original, 'kfp-cache-fixture:ci'
                )
                expected = copy.deepcopy(original)
                for executor in expected[0]['deploymentSpec'][
                    'executors'
                ].values():
                    executor['container']['image'] = 'kfp-cache-fixture:ci'
                    executor['container']['command'] = executor['container'][
                        'command'
                    ][3:]
                self.assertEqual(expected, actual)
                self.assertEqual(source(name), original)
                # Includes the separate PVC platform document, not only the IR.
                self.assertEqual(original[1:], actual[1:])

    def test_unknown_or_extended_installer_is_rejected(self):
        name = 'pvc-mount.yaml'
        for mutation in (
            'extra_dependency',
            'extra_executor',
            'different_image',
            'different_command',
        ):
            with self.subTest(mutation=mutation):
                documents = source(name)
                executors = documents[0]['deploymentSpec']['executors']
                container = executors['exec-producer']['container']
                if mutation == 'extra_dependency':
                    container['command'][2] += 'pip install other-package'
                elif mutation == 'extra_executor':
                    executors['unexpected'] = copy.deepcopy(
                        executors['exec-producer']
                    )
                elif mutation == 'different_image':
                    container['image'] = 'other:latest'
                else:
                    container['command'][5] = 'echo bypass-runtime'
                with self.assertRaises(ValueError):
                    prepare.transform(name, documents, 'kfp-cache-fixture:ci')
        with self.assertRaises(ValueError):
            prepare.transform(
                'unknown.yaml', source(name), 'kfp-cache-fixture:ci'
            )

    def test_compiled_components_execute_without_installed_packages(self):
        # -S excludes site packages, proving the copied current SDK needs no pip.
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            for name in prepare.FIXTURES:
                documents = prepare.transform(
                    name, source(name), 'kfp-cache-fixture:ci'
                )
                executors = documents[0]['deploymentSpec']['executors']
                # The consumer must run after the producer, as in the PVC DAG.
                for key in sorted(executors, reverse=True):
                    container = executors[key]['container']
                    component = work / 'component.py'
                    component.write_text(
                        container['command'][-1].replace(
                            '/data/file.txt', str(work / 'file.txt')
                        )
                    )
                    output = work / f'{key}.json'
                    parameters = (
                        {'message': 'same-cache-input'}
                        if key == 'exec-comp'
                        else {}
                    )
                    executor_input = {
                        'inputs': {'parameterValues': parameters},
                        'outputs': {'outputFile': str(output)},
                    }
                    result = subprocess.run(
                        [
                            sys.executable,
                            '-S',
                            '-m',
                            'kfp.dsl.executor_main',
                            '--component_module_path',
                            str(component),
                            '--function_to_execute',
                            container['args'][-1],
                            '--executor_input',
                            json.dumps(executor_input),
                        ],
                        env={
                            **os.environ,
                            'PYTHONPATH': str(ROOT / 'sdk/python'),
                            '_KFP_RUNTIME': 'true',
                        },
                        capture_output=True,
                        text=True,
                        check=True,
                    )
                    if key == 'exec-comp':
                        self.assertEqual(
                            'same-cache-input',
                            json.loads(output.read_text())['parameterValues'][
                                'Output'
                            ],
                        )
                    elif key == 'exec-producer':
                        self.assertEqual(
                            'hello',
                            json.loads(output.read_text())['parameterValues'][
                                'Output'
                            ],
                        )
                        self.assertEqual(
                            'hello', (work / 'file.txt').read_text()
                        )
                    else:
                        self.assertIn('hello', result.stdout)

    def test_image_preparation_stops_on_export_or_load_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            binaries = work / 'bin'
            binaries.mkdir()
            for command in ('docker', 'kind', 'python'):
                stub = binaries / command
                stub.write_text(
                    '#!/bin/bash\n'
                    'echo "$(basename "$0") $*" >> "$COMMAND_LOG"\n'
                    'if [[ "$(basename "$0")" == kind && "$1" == get ]]; then\n'
                    '  echo test-control-plane\n'
                    'fi\n'
                    'if [[ "$(basename "$0") $*" == *"$FAIL_COMMAND"* ]]; then exit 1; fi\n'
                )
                stub.chmod(0o755)
            for failure in ('docker exec', 'kind load'):
                with self.subTest(failure=failure):
                    log = work / 'commands'
                    log.write_text('')
                    result = subprocess.run(
                        [
                            'bash',
                            str(
                                ROOT
                                / '.github/resources/scripts/build-cache-fixture-image.sh'
                            ),
                            'test',
                            str(work / 'output'),
                        ],
                        cwd=ROOT,
                        env={
                            **os.environ,
                            'PATH': str(binaries)
                            + os.pathsep
                            + os.environ['PATH'],
                            'COMMAND_LOG': str(log),
                            'FAIL_COMMAND': failure,
                        },
                        capture_output=True,
                        text=True,
                    )
                    self.assertNotEqual(0, result.returncode)
                    self.assertNotIn('python ', log.read_text())
                    if failure == 'docker exec':
                        self.assertNotIn('docker build', log.read_text())

    def test_shell_reuses_preloaded_base_without_network(self):
        script = (
            ROOT / '.github/resources/scripts/build-cache-fixture-image.sh'
        ).read_text()
        dockerfile = (
            ROOT / 'backend/test/v2/resources/cache-image/Dockerfile'
        ).read_text()
        self.assertIn(
            'images export --platform linux/amd64 - docker.io/library/python:3.12',
            script,
        )
        self.assertIn(
            'DOCKER_BUILDKIT=0 docker build --network=none --pull=false', script
        )
        self.assertIn(
            'kind load docker-image --name "$cluster_name" kfp-cache-fixture:ci',
            script,
        )
        self.assertIn('COPY sdk/python/kfp /opt/kfp-sdk/kfp', dockerfile)
        self.assertIn('ENV PYTHONPATH=/opt/kfp-sdk', dockerfile)
        self.assertNotIn('pip install', dockerfile)


if __name__ == '__main__':
    unittest.main()

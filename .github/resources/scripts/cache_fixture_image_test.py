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
"""Cache fixture optimization preserves the IR and retains released SDK
runtimes."""

import copy
import io
import json
import os
import subprocess
import sys
import tarfile
import tempfile
import unittest
import zipfile
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
    def test_kind_load_is_retried(self):
        script = (
            ROOT / '.github/resources/scripts/build-cache-fixture-image.sh'
        ).read_text()
        self.assertIn(
            'source .github/resources/scripts/helper-functions.sh', script
        )
        loads = [
            line.strip()
            for line in script.splitlines()
            if 'kind load' in line and not line.lstrip().startswith('#')
        ]
        self.assertEqual(
            loads,
            [
                'retry 3 "${KIND_LOAD_RETRY_INTERVAL:-10}" kind load '
                'docker-image --name "$cluster_name" "$image"'
            ],
        )

    def test_preserves_released_sdk_and_python_matrix(self):
        with tempfile.TemporaryDirectory() as directory:
            subprocess.run(
                [sys.executable, str(Path(prepare.__file__)), directory],
                check=True,
            )
            images = set()
            for name, executors in prepare.FIXTURES.items():
                prepared = list(
                    yaml.safe_load_all((Path(directory) / name).read_text())
                )
                for executor, (base, version) in executors.items():
                    expected = f'kfp-cache-fixture:{version}-python{base.split(":")[1]}'
                    actual = prepared[0]['deploymentSpec']['executors'][
                        executor
                    ]['container']['image']
                    self.assertEqual(expected, actual)
                    images.add(actual)
            self.assertEqual(
                images,
                {
                    'kfp-cache-fixture:2.13.0-python3.12',
                    'kfp-cache-fixture:2.14.2-python3.11',
                },
            )

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

    @unittest.skipUnless(
        os.environ.get('KFP_CACHE_WHEELHOUSE'),
        'requires prepared released wheels and original Python interpreters',
    )
    def test_compiled_components_execute_without_installed_packages(self):
        # Unpack the same released wheels installed in the offline images; -S excludes site packages.
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            for name in prepare.FIXTURES:
                documents = prepare.transform(
                    name, source(name), 'kfp-cache-fixture:ci'
                )
                base, version = next(iter(prepare.FIXTURES[name].values()))
                wheel = next(
                    Path(os.environ['KFP_CACHE_WHEELHOUSE']).glob(
                        f'kfp-{version}-*.whl'
                    )
                )
                sdk = work / version
                with zipfile.ZipFile(wheel) as archive:
                    archive.extractall(sdk)
                interpreter = os.environ[
                    'KFP_CACHE_PYTHON_' + base.split(':')[1].replace('.', '')
                ]
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
                            interpreter,
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
                            'PYTHONPATH': str(sdk),
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

    @staticmethod
    def image_archive():
        output = io.BytesIO()
        with tarfile.open(fileobj=output, mode='w') as archive:
            for name, data in (
                (
                    'manifest.json',
                    b'[{"Config":"config.json","Layers":["layer.tar"]}]',
                ),
                ('config.json', b'{}'),
                ('layer.tar', b'layer contents' * 100),
            ):
                member = tarfile.TarInfo(name)
                member.size = len(data)
                archive.addfile(member, io.BytesIO(data))
        return output.getvalue()

    def run_image_preparation(
        self, archive, failure='', platform='linux/amd64'
    ):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            binaries = work / 'bin'
            binaries.mkdir()
            exported = work / 'export.tar'
            exported.write_bytes(archive)
            log = work / 'commands'
            log.touch()
            for command in ('docker', 'kind', 'python'):
                stub = binaries / command
                stub.write_text(
                    '#!/bin/bash\n'
                    'command="$(basename "$0")"\n'
                    'echo "$command $*" >> "$COMMAND_LOG"\n'
                    'if [[ -n "$FAIL_COMMAND" && "$command $*" == *"$FAIL_COMMAND"* ]]; then exit 1; fi\n'
                    'if [[ "$command $1" == "kind get" ]]; then echo test-control-plane; fi\n'
                    'if [[ "$command $1" == "docker inspect" ]]; then echo node-image; fi\n'
                    'if [[ "$command $1 $2" == "docker image inspect" ]]; then echo "$NODE_PLATFORM"; fi\n'
                    'if [[ "$command $1" == "docker exec" ]]; then cat "$EXPORT_ARCHIVE"; fi\n'
                    # Confirm validation does not alter the loaded bytes.
                    'if [[ "$command $1" == "docker load" ]]; then cmp "$EXPORT_ARCHIVE" "$3" || exit 2; fi\n'
                    'if [[ "$*" == *--image-matrix* ]]; then echo "python:3.12 2.13.0 kfp-cache-fixture:2.13.0-python3.12"; fi\n'
                )
                stub.chmod(0o755)
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
                    'PATH': str(binaries) + os.pathsep + os.environ['PATH'],
                    'COMMAND_LOG': str(log),
                    'FAIL_COMMAND': failure,
                    'EXPORT_ARCHIVE': str(exported),
                    'NODE_PLATFORM': platform,
                    'KIND_LOAD_RETRY_INTERVAL': '0',
                },
                capture_output=True,
                text=True,
            )
            return result, log.read_text()

    def test_image_preparation_loads_complete_archive_for_node_platform(self):
        for platform in ('linux/amd64', 'linux/arm64'):
            with self.subTest(platform=platform):
                result, commands = self.run_image_preparation(
                    self.image_archive(), platform=platform
                )
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIn(
                    f'images export --local --platform {platform}', commands
                )
                self.assertLess(
                    commands.index('docker exec'), commands.index('docker load')
                )
                self.assertLess(
                    commands.index('docker load'),
                    commands.index('docker build'),
                )
                self.assertIn('kind load docker-image', commands)
                self.assertIn('prepare_cache_fixtures.py ', commands)
                self.assertNotIn('docker pull', commands)

    def test_image_preparation_rejects_invalid_archive_before_load(self):
        ordinary_tar = io.BytesIO()
        with tarfile.open(fileobj=ordinary_tar, mode='w') as archive:
            archive.addfile(tarfile.TarInfo('ordinary-file'))
        for name, archive in (
            ('empty', b''),
            ('not-tar', b'Exporting image\n'),
            ('not-image', ordinary_tar.getvalue()),
            # Cut within the final layer payload, after manifest.json exists.
            ('truncated-layer', self.image_archive()[:2700]),
        ):
            with self.subTest(archive=name):
                result, commands = self.run_image_preparation(archive)
                self.assertNotEqual(0, result.returncode)
                self.assertIn(
                    'Invalid or incomplete image archive', result.stderr
                )
                self.assertNotIn('docker load', commands)
                self.assertNotIn('docker build', commands)

    def test_image_preparation_stops_on_export_or_load_failure(self):
        for failure, diagnostic in (
            ('docker exec', 'Failed to export preloaded image'),
            ('docker load', 'Failed to load exported image'),
            ('python -m pip wheel', ''),
            ('kind load', ''),
        ):
            with self.subTest(failure=failure):
                result, commands = self.run_image_preparation(
                    self.image_archive(), failure=failure
                )
                self.assertNotEqual(0, result.returncode)
                self.assertIn(diagnostic, result.stderr)
                if failure == 'docker exec':
                    self.assertNotIn('docker load', commands)
                if failure in ('docker exec', 'docker load'):
                    self.assertNotIn('docker build', commands)
                if failure == 'kind load':
                    self.assertEqual(
                        3, commands.count('kind load docker-image')
                    )
                self.assertNotIn('prepare_cache_fixtures.py /', commands)

    def test_shell_reuses_preloaded_base_without_network(self):
        script = (
            ROOT / '.github/resources/scripts/build-cache-fixture-image.sh'
        ).read_text()
        dockerfile = (
            ROOT / 'backend/test/v2/resources/cache-image/Dockerfile'
        ).read_text()
        self.assertIn(
            'images export --local --platform "$platform" - "docker.io/library/$base"',
            script,
        )
        self.assertIn(
            'DOCKER_BUILDKIT=0 docker build --network=none --pull=false', script
        )
        self.assertIn(
            'kind load docker-image --name "$cluster_name" "$image"', script
        )
        self.assertIn(
            '--build-arg "PYTHON_BASE=$base" --build-arg "KFP_VERSION=$version"',
            script,
        )
        self.assertIn('FROM ${PYTHON_BASE}', dockerfile)
        self.assertIn(
            'pip install --no-index --no-deps /wheels/kfp-${KFP_VERSION}-*.whl',
            dockerfile,
        )
        self.assertNotIn('sdk/python', script + dockerfile)


if __name__ == '__main__':
    unittest.main()

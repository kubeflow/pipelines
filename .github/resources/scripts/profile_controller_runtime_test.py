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

import json
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
CONTEXT = (
    'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller'
)
COMPONENT = ROOT / CONTEXT
IMAGE = 'ghcr.io/kubeflow/kfp-profile-controller'


class ProfileControllerRuntimeTests(unittest.TestCase):
    def test_recipe_requires_hashed_wheels_from_a_pinned_python_base(self):
        self.assertTrue(
            (COMPONENT / 'Dockerfile').is_file(),
            'Missing dedicated profile runtime recipe',
        )
        recipe = (COMPONENT / 'Dockerfile').read_text().replace('\\\n', ' ')
        base = re.findall(r'^FROM (\S+)', recipe, flags=re.MULTILINE)
        self.assertEqual(len(base), 1)
        self.assertRegex(
            base[0], r'^python:3\.13\.\d+-slim-trixie@sha256:[0-9a-f]{64}$'
        )
        self.assertIn('COPY requirements.txt /tmp/requirements.txt', recipe)
        install = re.search(r'python -m pip install (.*?)&&', recipe, re.DOTALL)
        self.assertIsNotNone(install)
        for argument in (
            '--require-hashes',
            '--only-binary=:all:',
            '--no-cache-dir',
            '-r /tmp/requirements.txt',
        ):
            self.assertIn(argument, install.group(1))
        lock = (COMPONENT / 'requirements.txt').read_text().replace('\\\n', ' ')
        packages = {}
        for line in lock.splitlines():
            if not line.strip() or line.lstrip().startswith('#'):
                continue
            package = re.fullmatch(r'([a-zA-Z0-9_.-]+)==([^\s]+)(.*)', line)
            self.assertIsNotNone(package, line)
            hashes = package.group(3).split()
            self.assertTrue(hashes, package.group(1))
            for digest in hashes:
                self.assertRegex(digest, r'^--hash=sha256:[0-9a-f]{64}$')
            packages[package.group(1)] = package.group(2)
        self.assertIn('botocore', packages)
        self.assertIn('urllib3', packages)
        self.assertNotIn('awscli', packages)
        for line in (COMPONENT / 'requirements.in').read_text().splitlines():
            if line.strip() and not line.startswith('#'):
                name, version = line.split('==')
                self.assertEqual(packages[name], version)

    def test_runtime_identity_and_command_match_the_mounted_controller(self):
        self.assertTrue(
            (COMPONENT / 'Dockerfile').is_file(),
            'Missing dedicated profile runtime recipe',
        )
        recipe = (COMPONENT / 'Dockerfile').read_text()
        deployment = yaml.safe_load((COMPONENT / 'deployment.yaml').read_text())
        pod = deployment['spec']['template']['spec']
        container = next(
            item
            for item in pod['containers']
            if item['name'] == 'profile-controller'
        )
        command = json.loads(
            re.search(r'^CMD (.+)$', recipe, re.MULTILINE).group(1)
        )
        self.assertEqual(command, container['command'])
        self.assertEqual(command, ['python', '/hooks/sync.py'])
        identity = re.search(r'^USER (\d+):(\d+)$', recipe, re.MULTILINE)
        self.assertEqual(
            int(identity.group(1)), container['securityContext']['runAsUser']
        )
        self.assertEqual(
            int(identity.group(2)), container['securityContext']['runAsGroup']
        )
        self.assertTrue(container['securityContext']['runAsNonRoot'])
        self.assertEqual(container['image'].split(':')[0], IMAGE)
        self.assertIn(
            {'name': 'hooks', 'mountPath': '/hooks'}, container['volumeMounts']
        )
        self.assertTrue(
            any(
                item['name'] == 'hooks' and 'configMap' in item
                for item in pod['volumes']
            )
        )

    @unittest.skipUnless(
        shutil.which('yq'), 'yq is required for release manifest rewriting'
    )
    def test_release_preparation_retags_profile_controller(self):
        with tempfile.TemporaryDirectory() as temporary:
            manifests = Path(temporary) / 'manifests'
            shutil.copytree(ROOT / 'manifests/kustomize', manifests)
            subprocess.run(
                [
                    'bash',
                    str(manifests / 'hack/release.sh'),
                    '9.8.7-runtime-test',
                ],
                check=True,
                capture_output=True,
                text=True,
            )
            config = yaml.safe_load(
                (
                    manifests
                    / 'base/installs/multi-user/pipelines-profile-controller/kustomization.yaml'
                ).read_text()
            )
            self.assertIn(
                {'name': IMAGE, 'newTag': '9.8.7-runtime-test'},
                config.get('images', []),
            )


if __name__ == '__main__':
    unittest.main()

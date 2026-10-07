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
"""Keep the profile runtime built, published and deployed by the real
inventories."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import arm64_presubmit_image
import arm64_smoke
import yaml

ROOT = Path(__file__).parents[3]
PROFILE = Path(
    'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller')
IMAGE = 'kfp-profile-controller'
CI_IMAGE = 'profile-controller'


def workflow(name):
    return yaml.safe_load((ROOT / '.github/workflows' / name).read_text())


class ProfileRuntimeTests(unittest.TestCase):

    def test_ci_and_native_publication_build_the_runtime(self):
        ci = workflow('image-builds.yml'
                     )['jobs']['image-build']['strategy']['matrix']['include']
        build = next(item for item in ci if item['image'] == CI_IMAGE)
        self.assertEqual(build['context'], str(PROFILE))
        self.assertEqual(build['dockerfile'], str(PROFILE / 'Dockerfile'))
        for filename, job_name in (('image-builds-master.yml',
                                    'build-images-for-master'),
                                   ('image-builds-release.yml',
                                    'build-images-for-release')):
            jobs = workflow(filename)['jobs']
            matrix = jobs[job_name]['strategy']['matrix']
            build = next(
                item for item in matrix['image'] if item['name'] == IMAGE)
            self.assertEqual(build['context'], str(PROFILE))
            self.assertEqual(build['dockerfile'], str(PROFILE / 'Dockerfile'))
            self.assertEqual({item['arch_name'] for item in matrix['arch']},
                             {'amd64', 'arm64'})
            published = jobs['create-manifests']['strategy']['matrix'][
                'component']
            self.assertIn({'image': IMAGE}, published)

    def test_artifacts_require_the_profile_image_on_both_architectures(self):
        script = ROOT / '.github/resources/scripts/ci-image-artifacts.sh'
        for inventory, expected in (('deploy-image-files', {
                'profile-controller/profile-controller.tar'
        }), ('arm64-image-files',
             {'profile-controller-arm64.tar', 'profile-controller-arm64.json'
             }), ('published-image-files', {'kfp-profile-controller.json'})):
            output = subprocess.check_output(
                ['bash', str(script), inventory], text=True)
            self.assertTrue(expected.issubset(output.splitlines()))
        self.assertIn(IMAGE, arm64_smoke.IMAGES)
        self.assertNotIn(IMAGE, arm64_smoke.CONTROL_IMAGES)
        record = arm64_presubmit_image.image_record(CI_IMAGE, 'a' * 40,
                                                    'kind-registry:5000')
        self.assertEqual(record['reference'],
                         'kind-registry:5000/profile-controller:ci')

    @unittest.skipUnless(
        shutil.which('kubectl'), 'kubectl is required for Kustomize rendering')
    def test_actual_deployment_uses_published_or_loaded_runtime(self):
        for overlay in (None, 'default', 'artifact-proxy', 'postgresql'):
            with self.subTest(overlay=overlay), tempfile.TemporaryDirectory(
            ) as directory:
                specification = {
                    'apiVersion':
                        'kustomize.config.k8s.io/v1beta1',
                    'kind':
                        'Kustomization',
                    'resources': [
                        os.path.relpath(ROOT / PROFILE,
                                        Path(directory).resolve())
                    ],
                }
                expected = 'ghcr.io/kubeflow/kfp-profile-controller:master'
                if overlay:
                    configuration = yaml.safe_load((
                        ROOT /
                        f'.github/resources/manifests/multiuser/{overlay}/kustomization.yaml'
                    ).read_text())
                    specification['images'] = configuration['images']
                    expected = 'kind-registry:5000/profile-controller:ci'
                path = Path(directory) / 'kustomization.yaml'
                path.write_text(json.dumps(specification))
                rendered = subprocess.check_output([
                    'kubectl', 'kustomize', '--load-restrictor',
                    'LoadRestrictionsNone', directory
                ],
                                                   text=True,
                                                   timeout=30)
                deployment = next(
                    item for item in yaml.safe_load_all(rendered)
                    if item['kind'] == 'Deployment')
                container = deployment['spec']['template']['spec'][
                    'containers'][0]
                self.assertEqual(container['image'], expected)
                self.assertEqual(container['imagePullPolicy'], 'IfNotPresent')
                self.assertEqual(container['command'],
                                 ['python', '/hooks/sync.py'])
                self.assertTrue(container['securityContext']['runAsNonRoot'])

    def test_release_generator_updates_the_profile_image_tag(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifests = root / 'manifests'
            for relative in (
                    'hack/release.sh', 'base/pipeline/kustomization.yaml',
                    'base/installs/multi-user/pipelines-profile-controller/kustomization.yaml',
                    'base/installs/generic/pipeline-install-config.yaml',
                    'base/pipeline/ml-pipeline-apiserver-deployment.yaml'):
                target = manifests / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy(ROOT / 'manifests/kustomize' / relative, target)
            bin_dir = root / 'bin'
            bin_dir.mkdir()
            commands = root / 'commands.jsonl'
            recorder = bin_dir / 'yq'
            recorder.write_text(
                '#!/usr/bin/env python3\n'
                'import json, os, pathlib, sys\n'
                'assert pathlib.Path(sys.argv[3]).is_file()\n'
                'with open(os.environ["COMMANDS"], "a") as output:\n'
                '    output.write(json.dumps(sys.argv[1:]) + "\\n")\n')
            recorder.chmod(0o755)
            subprocess.run(
                ['bash', str(manifests / 'hack/release.sh'), '3.0.0'],
                env={
                    **os.environ, 'PATH': f'{bin_dir}:{os.environ["PATH"]}',
                    'COMMANDS': str(commands)
                },
                check=True,
                capture_output=True,
                text=True)
            recorded = [
                json.loads(line) for line in commands.read_text().splitlines()
            ]
            for command in recorded:
                command[2] = str(Path(command[2]).resolve())
            self.assertIn([
                'w', '-i',
                str((
                    manifests /
                    'base/installs/multi-user/pipelines-profile-controller/kustomization.yaml'
                ).resolve()), 'images[*].newTag', '3.0.0'
            ], recorded)


if __name__ == '__main__':
    unittest.main()

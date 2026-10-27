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
"""Test release YAML edits and conversions with the installed yq."""

import copy
import json
from pathlib import Path
import runpy
import shutil
import subprocess
import tempfile
import unittest

HACK = Path(__file__).resolve().parent
REPOSITORY = HACK.parents[2]
TENSORBOARD_UPDATER = REPOSITORY / 'hack/update_tensorboard_image.py'
TENSORBOARD_PATHS = runpy.run_path(str(TENSORBOARD_UPDATER))['REQUIRED_PATHS']
IMAGE_PATHS = (
    'base/cache-deployer/kustomization.yaml',
    'base/cache/kustomization.yaml',
    'base/metadata/base/kustomization.yaml',
    'base/pipeline/metadata-writer/kustomization.yaml',
    'base/pipeline/kustomization.yaml',
    'env/gcp/inverse-proxy/kustomization.yaml',
)


def read_yaml(path):
    return json.loads(
        subprocess.check_output(['yq', 'eval', '-o=json', '.',
                                 str(path)],
                                text=True))


class YqReleaseTest(unittest.TestCase):

    def test_release_updates_only_expected_fields_and_is_repeatable(self):
        paths = (
            *IMAGE_PATHS,
            'base/installs/generic/pipeline-install-config.yaml',
            'base/pipeline/ml-pipeline-apiserver-deployment.yaml',
        )
        with tempfile.TemporaryDirectory() as directory:
            repository = Path(directory)
            root = repository / 'manifests/kustomize'
            (root / 'hack').mkdir(parents=True)
            shutil.copy(HACK / 'release.sh', root / 'hack/release.sh')
            (repository / 'hack').mkdir()
            shutil.copy(TENSORBOARD_UPDATER,
                        repository / 'hack/update_tensorboard_image.py')
            tensorboard_before = {}
            for path in TENSORBOARD_PATHS:
                target = repository / path
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy(REPOSITORY / path, target)
                tensorboard_before[path] = target.read_text()
            current_image = 'ghcr.io/kubeflow/kfp-tensorboard:' + (
                REPOSITORY / 'VERSION').read_text().strip()
            for path in paths:
                (root / path).parent.mkdir(parents=True, exist_ok=True)
                shutil.copy(HACK.parent / path, root / path)
            before = {path: read_yaml(root / path) for path in paths}
            modes = {path: (root / path).stat().st_mode for path in paths}
            # Numeric-looking tags must remain YAML strings.
            for tag in ('123', '3.0.0-rc.1', '3.0.0-rc.1'):
                subprocess.run(
                    ['bash', str(root / 'hack/release.sh'), tag],
                    check=True,
                    capture_output=True,
                    text=True,
                )
                for path, content in tensorboard_before.items():
                    self.assertIn(current_image, content)
                    self.assertEqual(
                        (repository / path).read_text(),
                        content.replace(
                            current_image,
                            f'ghcr.io/kubeflow/kfp-tensorboard:{tag}'),
                    )
                for path in paths:
                    self.assertEqual((root / path).stat().st_mode, modes[path])
                expected = copy.deepcopy(before)
                for path in IMAGE_PATHS:
                    for image in expected[path]['images']:
                        image['newTag'] = tag
                expected[paths[-2]]['data']['appVersion'] = tag
                containers = expected[
                    paths[-1]]['spec']['template']['spec']['containers']
                for container in containers:
                    if container['name'] == 'ml-pipeline-api-server':
                        for env in container['env']:
                            if env['name'] == 'V2_LAUNCHER_IMAGE':
                                env['value'] = (
                                    f'ghcr.io/kubeflow/kfp-launcher:{tag}')
                            if env['name'] == 'V2_DRIVER_IMAGE':
                                env['value'] = (
                                    f'ghcr.io/kubeflow/kfp-driver:{tag}')
                for path in paths:
                    self.assertEqual(read_yaml(root / path), expected[path])

    def test_format_preserves_multiple_documents_and_null_values(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'hack').mkdir()
            shutil.copy(HACK / 'format.sh', root / 'hack/format.sh')
            manifest = root / 'base/pipeline/kustomization.yaml'
            for path in IMAGE_PATHS:
                (root / path).parent.mkdir(parents=True, exist_ok=True)
                (root / path).write_text('images: []\n')
            manifest.write_text(
                'images: [{name: image, newTag: "123"}]\n---\nvalue: null\n')
            before = subprocess.check_output(
                ['yq', 'eval', '-o=json', '.',
                 str(manifest)], text=True)
            mode = manifest.stat().st_mode
            subprocess.run(
                ['bash', str(root / 'hack/format.sh')],
                check=True,
                capture_output=True,
                text=True,
            )
            after = subprocess.check_output(
                ['yq', 'eval', '-o=json', '.',
                 str(manifest)], text=True)
            self.assertEqual(after, before)
            self.assertEqual(manifest.stat().st_mode, mode)

    def test_multidocument_json_preserves_each_resource(self):
        yaml = 'kind: ConfigMap\ndata:\n  version: "123"\n---\nkind: Secret\ndata:\n  value: abc\n'
        encoded = subprocess.check_output(['yq', 'eval', '-o=json', '.', '-'],
                                          input=yaml,
                                          text=True)
        decoder = json.JSONDecoder()
        documents = []
        while encoded.strip():
            document, end = decoder.raw_decode(encoded.lstrip())
            documents.append(document)
            encoded = encoded.lstrip()[end:]
        self.assertEqual(
            documents,
            [
                {
                    'kind': 'ConfigMap',
                    'data': {
                        'version': '123'
                    }
                },
                {
                    'kind': 'Secret',
                    'data': {
                        'value': 'abc'
                    }
                },
            ],
        )


if __name__ == '__main__':
    unittest.main()

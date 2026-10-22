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
"""Exercise release YAML edits and manifest conversion with the installed yq."""

import copy
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

HACK = Path(__file__).resolve().parent
PLUGIN_PATHS = (
    'base/pipeline/ml-pipeline-driver-plugin-cm.yaml',
    'env/cert-manager/platform-agnostic-standalone-tls/patches/ml-pipeline-driver-plugin-cm.yaml',
    'env/openshift/base/patches/driver-agent-plugin.yaml',
)


def read_yaml(path):
    return json.loads(
        subprocess.check_output(
            ['yq', 'eval', '-o=json', '.', str(path)], text=True
        )
    )


class YqReleaseTest(unittest.TestCase):
    def test_release_updates_only_expected_fields_and_is_repeatable(self):
        paths = (
            'base/pipeline/kustomization.yaml',
            'base/installs/generic/pipeline-install-config.yaml',
            'base/pipeline/ml-pipeline-apiserver-deployment.yaml',
            *PLUGIN_PATHS,
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'hack').mkdir()
            shutil.copy(HACK / 'release.sh', root / 'hack/release.sh')
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
                for path in paths:
                    self.assertEqual((root / path).stat().st_mode, modes[path])
                expected = copy.deepcopy(before)
                for image in expected[paths[0]]['images']:
                    image['newTag'] = tag
                expected[paths[1]]['data']['appVersion'] = tag
                containers = expected[paths[2]]['spec']['template']['spec'][
                    'containers'
                ]
                for container in containers:
                    if container['name'] == 'ml-pipeline-api-server':
                        for env in container['env']:
                            if env['name'] == 'V2_LAUNCHER_IMAGE':
                                env['value'] = (
                                    f'ghcr.io/kubeflow/kfp-launcher:{tag}'
                                )
                for path in paths[:3]:
                    self.assertEqual(read_yaml(root / path), expected[path])
                for path in PLUGIN_PATHS:
                    original = before[path]['data']['sidecar.container']
                    actual = read_yaml(root / path)
                    original_image = json.loads(
                        subprocess.check_output(
                            ['yq', 'eval', '-o=json', '.image', '-'],
                            input=original,
                            text=True,
                        )
                    )
                    expected[path]['data']['sidecar.container'] = (
                        original.replace(
                            original_image, f'ghcr.io/kubeflow/kfp-driver:{tag}'
                        )
                    )
                    self.assertEqual(actual, expected[path])
                    # Same nested YAML extraction used by manifest presubmit.
                    image = subprocess.check_output(
                        [
                            'yq',
                            'eval',
                            '.data."sidecar.container" | from_yaml | .image',
                            str(root / path),
                        ],
                        text=True,
                    ).strip()
                    self.assertEqual(
                        image, f'ghcr.io/kubeflow/kfp-driver:{tag}'
                    )

    def test_format_preserves_multiple_documents_and_null_values(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'hack').mkdir()
            shutil.copy(HACK / 'format.sh', root / 'hack/format.sh')
            manifest = root / 'base/pipeline/kustomization.yaml'
            manifest.parent.mkdir(parents=True)
            manifest.write_text(
                'images: [{name: image, newTag: "123"}]\n---\nvalue: null\n'
            )
            before = subprocess.check_output(
                ['yq', 'eval', '-o=json', '.', str(manifest)], text=True
            )
            mode = manifest.stat().st_mode
            subprocess.run(
                ['bash', str(root / 'hack/format.sh')],
                check=True,
                capture_output=True,
                text=True,
            )
            after = subprocess.check_output(
                ['yq', 'eval', '-o=json', '.', str(manifest)], text=True
            )
            self.assertEqual(after, before)
            self.assertEqual(manifest.stat().st_mode, mode)

    def test_multidocument_json_preserves_each_resource(self):
        yaml = 'kind: ConfigMap\ndata:\n  version: "123"\n---\nkind: Secret\ndata:\n  value: abc\n'
        encoded = subprocess.check_output(
            ['yq', 'eval', '-o=json', '.', '-'], input=yaml, text=True
        )
        decoder = json.JSONDecoder()
        documents = []
        while encoded.strip():
            document, end = decoder.raw_decode(encoded.lstrip())
            documents.append(document)
            encoded = encoded.lstrip()[end:]
        self.assertEqual(
            documents,
            [
                {'kind': 'ConfigMap', 'data': {'version': '123'}},
                {'kind': 'Secret', 'data': {'value': 'abc'}},
            ],
        )


if __name__ == '__main__':
    unittest.main()

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

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

REPO_ROOT = Path(__file__).resolve().parents[3]
GUIDE = REPO_ROOT / 'docs/operator-guides/server-config.md'
MULTI_USER = REPO_ROOT / 'manifests/kustomize/base/installs/multi-user'
STANDALONE = REPO_ROOT / 'manifests/kustomize/base/installs/generic'
MULTI_USER_CONFIG = 'pipeline-api-server-config'
ALLOWED_DOMAINS = {
    'name': 'PIPELINE_URL_ALLOWED_DOMAINS',
    'valueFrom': {
        'configMapKeyRef': {
            'name': 'pipeline-install-config',
            'key': 'pipelineUrlAllowedDomains',
            'optional': True
        }
    }
}
CREDENTIALS = [{
    'prefix': 'PIPELINE_URL_CREDENTIAL_ARTIFACTORY_',
    'secretRef': {
        'name': 'artifactory-pipeline-credentials',
        'optional': True
    }
}, {
    'prefix': 'PIPELINE_URL_CREDENTIAL_GITHUB_',
    'secretRef': {
        'name': 'github-pipeline-credentials',
        'optional': True
    }
}]


def render(path):
    yaml = subprocess.check_output(['kustomize', 'build', str(path)], text=True)
    encoded = subprocess.check_output(['yq', 'eval', '-o=json', '.', '-'],
                                      input=yaml,
                                      text=True)
    decoder = json.JSONDecoder()
    resources = []
    while encoded.strip():
        resource, end = decoder.raw_decode(encoded.lstrip())
        resources.append(resource)
        encoded = encoded.lstrip()[end:]
    return resources


def documented_overlay(directory, base):
    section = GUIDE.read_text().split('### Pipeline URL credentials', 1)[1]
    section = section.split('\n## ', 1)[0]
    blocks = [
        block.split('```', 1)[0] for block in section.split('```yaml\n')[1:]
    ]
    patches = next(block for block in blocks if block.startswith('patches:'))
    overlay = Path(directory)
    # Kustomize does not accept an absolute path to another kustomization.
    resource = Path(os.path.relpath(base, overlay)).as_posix()
    config = overlay / 'kustomization.yaml'
    config.write_text(f'resources:\n- {resource}\n{patches}')
    return overlay


def api_server(resources):
    deployment = next(item for item in resources
                      if item['kind'] == 'Deployment' and
                      item['metadata']['name'] == 'ml-pipeline')
    return next(
        container
        for container in deployment['spec']['template']['spec']['containers']
        if container['name'] == 'ml-pipeline-api-server')


class PipelineURLCredentialsGuideTest(unittest.TestCase):

    def test_documented_patches_keep_multi_user_config(self):
        before = render(MULTI_USER)
        with tempfile.TemporaryDirectory() as directory:
            after = render(documented_overlay(directory, MULTI_USER))
        original = api_server(before)
        patched = api_server(after)

        config_refs = [
            source for source in original['envFrom']
            if source.get('configMapRef', {}).get('name', '').startswith(
                MULTI_USER_CONFIG)
        ]
        self.assertEqual(len(config_refs), 1, original['envFrom'])
        self.assertEqual(patched['envFrom'], original['envFrom'] + CREDENTIALS)
        self.assertIn(ALLOWED_DOMAINS, patched['env'])
        self.assertEqual(
            [item for item in patched['env'] if item != ALLOWED_DOMAINS],
            original['env'])

        patched['env'] = original['env']
        patched['envFrom'] = original['envFrom']
        self.assertEqual(after, before,
                         'the patches change only the API server environment')

    def test_documented_patches_stop_when_env_from_is_missing(self):
        # The guide tells standalone installs to create the list instead.
        self.assertNotIn('envFrom', api_server(render(STANDALONE)))
        with tempfile.TemporaryDirectory() as directory:
            overlay = documented_overlay(directory, STANDALONE)
            build = subprocess.run(
                ['kustomize', 'build', str(overlay)],
                capture_output=True,
                text=True)
        self.assertNotEqual(build.returncode, 0, build.stdout[:200])
        self.assertIn('envFrom', build.stderr)


if __name__ == '__main__':
    unittest.main()

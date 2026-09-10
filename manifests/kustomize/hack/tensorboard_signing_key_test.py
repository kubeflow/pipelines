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
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

PIPELINE = Path(__file__).resolve().parents[1] / 'base' / 'pipeline'
SECRET = 'ml-pipeline-ui-tensorboard-proxy'
INIT = 'ml-pipeline-ui-signing-key-init'


def render(path):
    yaml = subprocess.check_output(['kustomize', 'build', str(path)], text=True)
    encoded = subprocess.check_output(['yq', 'r', '-d', '*', '-j', '-'],
                                      input=yaml,
                                      text=True)
    decoder = json.JSONDecoder()
    resources = []
    while encoded.strip():
        resource, end = decoder.raw_decode(encoded.lstrip())
        resources.append(resource)
        encoded = encoded.lstrip()[end:]
    return resources


def resource(resources, kind, prefix):
    exact = [
        item for item in resources
        if item['kind'] == kind and item['metadata']['name'] == prefix
    ]
    if exact:
        return exact[0]
    matches = [
        item for item in resources
        if item['kind'] == kind and item['metadata']['name'].startswith(prefix)
    ]
    assert len(matches) == 1, (kind, prefix, len(matches))
    return matches[0]


class TensorboardSigningKeyTest(unittest.TestCase):

    def test_shared_key_and_least_privilege(self):
        resources = render(PIPELINE)
        secret = resource(resources, 'Secret', SECRET)
        self.assertNotIn('data', secret)
        self.assertNotIn('stringData', secret)
        self.assertNotIn('ownerReferences', secret['metadata'])
        deployment = resource(resources, 'Deployment', 'ml-pipeline-ui')
        self.assertEqual(
            deployment['spec']['strategy'], {
                'type': 'RollingUpdate',
                'rollingUpdate': {
                    'maxUnavailable': 0,
                    'maxSurge': 1
                }
            })
        env = deployment['spec']['template']['spec']['containers'][0]['env']
        signing = next(item for item in env
                       if item['name'] == 'TENSORBOARD_PROXY_SIGNING_SECRET')
        self.assertEqual(signing['valueFrom']['secretKeyRef'], {
            'name': SECRET,
            'key': 'signing-secret'
        })
        role = resource(resources, 'Role', INIT)
        self.assertEqual(role['rules'], [{
            'apiGroups': [''],
            'resources': ['secrets'],
            'resourceNames': [SECRET],
            'verbs': ['get', 'update']
        }])
        ui_role = resource(resources, 'Role', 'ml-pipeline-ui')
        for rule in ui_role['rules']:
            if 'secrets' in rule['resources']:
                self.assertTrue(set(rule['verbs']) <= {'get', 'list', 'watch'})

    def test_job_name_tracks_image_and_template_but_secret_stays_fixed(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory) / 'base'
            shutil.copytree(PIPELINE / 'ui-signing-key', base)
            overlay = Path(directory) / 'overlay'
            overlay.mkdir()
            config = overlay / 'kustomization.yaml'
            template = ('resources: [../base]\n'
                        'namespace: signing-test\n'
                        'namePrefix: custom-\n'
                        'images:\n'
                        '- name: ghcr.io/kubeflow/kfp-frontend\n'
                        '  newTag: {tag}\n')
            config.write_text(template.format(tag='first'))
            first = render(overlay)
            self.assertEqual(first, render(overlay))
            config.write_text(template.format(tag='second'))
            second = render(overlay)
            job_path = base / 'job.yaml'
            job_path.write_text(job_path.read_text().replace(
                'memory: 128Mi', 'memory: 192Mi'))
            third = render(overlay)
            names = []
            for resources in (first, second, third):
                job = resource(resources, 'Job', 'custom-' + INIT)
                names.append(job['metadata']['name'])
                configmap = resource(resources, 'ConfigMap', 'custom-' + INIT)
                self.assertEqual(job['metadata']['name'],
                                 configmap['metadata']['name'])
                self.assertEqual(job['metadata']['namespace'], 'signing-test')
                secret = resource(resources, 'Secret', 'custom-' + SECRET)
                self.assertEqual(secret['metadata']['name'], 'custom-' + SECRET)
                self.assertEqual(configmap['data']['secretName'],
                                 secret['metadata']['name'])
                role = resource(resources, 'Role', 'custom-' + INIT)
                self.assertEqual(role['rules'][0]['resourceNames'],
                                 [secret['metadata']['name']])
                self.assertEqual(
                    job['spec']['template']['metadata']['labels']
                    ['sidecar.istio.io/inject'], 'false')
                pod = job['spec']['template']['spec']
                self.assertEqual(pod['serviceAccountName'], 'custom-' + INIT)
                container = pod['containers'][0]
                self.assertEqual(container['image'], configmap['data']['image'])
                for env in container['env'][1:]:
                    self.assertEqual(
                        env['valueFrom']['configMapKeyRef']['name'],
                        configmap['metadata']['name'])
            self.assertEqual(len(set(names)), 3)


if __name__ == '__main__':
    unittest.main()

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

import shutil
import subprocess
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


@unittest.skipUnless(
    shutil.which('kubectl'), 'kubectl is needed to render overlays'
)
class ApplicationRetirementTests(unittest.TestCase):
    def test_former_consumers_keep_kfp_without_application_controller(self):
        removed = {
            ('Deployment', 'controller-manager'),
            ('Service', 'controller-manager-service'),
            ('Role', 'application-manager-role'),
            ('RoleBinding', 'application-manager-rolebinding'),
            ('ServiceAccount', 'application'),
        }
        for overlay in (
            'env/gcp',
            'env/dev',
            'env/dev/postgresql',
            'env/cert-manager/dev',
            'sample',
        ):
            with self.subTest(overlay=overlay):
                rendered = subprocess.run(
                    [
                        'kubectl',
                        'kustomize',
                        str(ROOT / 'manifests/kustomize' / overlay),
                    ],
                    capture_output=True,
                    text=True,
                    check=True,
                    timeout=120,
                ).stdout
                resources = [
                    item for item in yaml.safe_load_all(rendered) if item
                ]
                identities = {
                    (item['kind'], item['metadata']['name'])
                    for item in resources
                }
                self.assertFalse(identities & removed)
                self.assertFalse(
                    any(item['kind'] == 'Application' for item in resources)
                )
                self.assertNotIn('application-crd-controller', rendered)
                for name in ('ml-pipeline', 'ml-pipeline-ui'):
                    self.assertIn(('Deployment', name), identities)
                    deployment = next(
                        item
                        for item in resources
                        if item['kind'] == 'Deployment'
                        and item['metadata']['name'] == name
                    )
                    self.assertEqual(
                        deployment['spec']['selector']['matchLabels'][
                            'application-crd-id'
                        ],
                        'kubeflow-pipelines',
                    )
                if overlay in ('env/gcp', 'sample'):
                    self.assertIn(('Deployment', 'cloudsqlproxy'), identities)

    def test_shared_application_crd_is_retained(self):
        for overlay in (
            'cluster-scoped-resources',
            'env/cert-manager/cluster-scoped-resources',
        ):
            with self.subTest(overlay=overlay):
                rendered = subprocess.run(
                    [
                        'kubectl',
                        'kustomize',
                        str(ROOT / 'manifests/kustomize' / overlay),
                    ],
                    capture_output=True,
                    text=True,
                    check=True,
                    timeout=120,
                ).stdout
                self.assertTrue(
                    any(
                        item
                        and item['kind'] == 'CustomResourceDefinition'
                        and item['metadata']['name']
                        == 'applications.app.k8s.io'
                        for item in yaml.safe_load_all(rendered)
                    )
                )


if __name__ == '__main__':
    unittest.main()

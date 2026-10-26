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
import subprocess
import tempfile
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[3]
GUIDE = REPO_ROOT / 'docs/operator-guides/high-availability.md'
DEPLOYMENT = 'ml-pipeline-scheduledworkflow'
LEASE = 'kfp-scheduledworkflow'
# The scheduled workflow controller reads LEADER_ELECT from its environment,
# so the setting must reach the container in every install, including the
# standalone TLS install, which overrides the command and arguments.
INSTALLS = [
    'manifests/kustomize/env/platform-agnostic',
    'manifests/kustomize/env/platform-agnostic-multi-user',
    'manifests/kustomize/env/cert-manager/platform-agnostic-standalone-tls',
]


def render(path):
    yaml = subprocess.check_output(['kustomize', 'build', str(path)], text=True)
    encoded = subprocess.check_output(
        ['yq', 'r', '-d', '*', '-j', '-'], input=yaml, text=True
    )
    decoder = json.JSONDecoder()
    resources = []
    while encoded.strip():
        resource, end = decoder.raw_decode(encoded.lstrip())
        resources.append(resource)
        encoded = encoded.lstrip()[end:]
    return resources


def documented_patch():
    guide = GUIDE.read_text()
    blocks = [
        block.split('```', 1)[0] for block in guide.split('```yaml\n')[1:]
    ]
    patches = [block for block in blocks if 'LEADER_ELECT' in block]
    assert len(patches) == 1, len(patches)
    return patches[0]


def render_with_documented_patch(install):
    with tempfile.TemporaryDirectory() as directory:
        overlay = Path(directory).resolve()
        (overlay / 'leader-election.yaml').write_text(documented_patch())
        base = os.path.relpath(REPO_ROOT / install, overlay)
        (overlay / 'kustomization.yaml').write_text(
            f'resources: [{base}]\npatches:\n- path: leader-election.yaml\n'
        )
        return render(overlay)


def controller_container(resources):
    deployment = next(
        item
        for item in resources
        if item['kind'] == 'Deployment'
        and item['metadata']['name'] == DEPLOYMENT
    )
    return next(
        container
        for container in deployment['spec']['template']['spec']['containers']
        if container['name'] == DEPLOYMENT
    )


class ScheduledWorkflowLeaderElectionTest(unittest.TestCase):
    def test_documented_patch_enables_leader_election_in_every_install(self):
        for install in INSTALLS:
            with self.subTest(install=install):
                resources = render_with_documented_patch(install)
                environment = {
                    variable['name']: variable
                    for variable in controller_container(resources).get(
                        'env', []
                    )
                }
                self.assertEqual(
                    environment['LEADER_ELECT'].get('value'), 'true'
                )
                self.assertEqual(
                    environment['POD_NAMESPACE']['valueFrom']['fieldRef'],
                    {'fieldPath': 'metadata.namespace'},
                )

    def test_role_grants_only_the_controller_lease(self):
        resources = render(REPO_ROOT / 'manifests/kustomize/base/pipeline')
        role = next(
            item
            for item in resources
            if item['kind'] == 'Role'
            and item['metadata']['name'] == 'ml-pipeline-scheduledworkflow-role'
        )
        lease_rules = [
            rule for rule in role['rules'] if 'leases' in rule['resources']
        ]
        self.assertEqual(
            lease_rules,
            [
                {
                    'apiGroups': ['coordination.k8s.io'],
                    'resources': ['leases'],
                    'verbs': ['create'],
                },
                {
                    'apiGroups': ['coordination.k8s.io'],
                    'resources': ['leases'],
                    'resourceNames': [LEASE],
                    'verbs': ['get', 'update'],
                },
            ],
        )


if __name__ == '__main__':
    unittest.main()

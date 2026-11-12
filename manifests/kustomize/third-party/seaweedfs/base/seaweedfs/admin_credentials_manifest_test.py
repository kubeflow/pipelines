# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify rendered manifests: set SEAWEEDFS_RENDERED to kubectl kustomize output."""

import os
import unittest
from pathlib import Path

import yaml


class ManifestTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.resources = list(
            yaml.safe_load_all(
                Path(os.environ['SEAWEEDFS_RENDERED']).read_text()
            )
        )

    def resource(self, kind, name):
        return next(
            item
            for item in self.resources
            if item['kind'] == kind and item['metadata']['name'] == name
        )

    def test_no_declarative_legacy_secret_or_generated_credential_data(self):
        for item in self.resources:
            if item['kind'] == 'Secret':
                self.assertNotEqual(
                    'mlpipeline-minio-artifact',
                    item['metadata']['name'],
                    'Default credentials must not be reapplied over runtime credentials',
                )
        state = self.resource('Secret', 'seaweedfs-admin-credentials')
        self.assertFalse(state.get('data'))
        self.assertFalse(state.get('stringData'))

    def test_storage_requires_both_generated_keys_before_start(self):
        container = self.resource('Deployment', 'seaweedfs')['spec'][
            'template'
        ]['spec']['containers'][0]
        env = {item['name']: item for item in container['env']}
        for key in ('accesskey', 'secretkey'):
            reference = env[key]['valueFrom']['secretKeyRef']
            self.assertEqual('seaweedfs-admin-credentials', reference['name'])
            self.assertEqual(key, reference['key'])
            self.assertFalse(reference.get('optional', False))
        self.assertIn(
            '-s3.config=/etc/seaweedfs-auth/s3-auth.json', container['args']
        )
        self.assertIn(
            '/usr/bin/weed shell >/dev/null 2>&1',
            container['lifecycle']['postStart']['exec']['command'][-1],
        )

    def test_both_listeners_require_bootstrap_authentication(self):
        container = self.resource('Deployment', 'seaweedfs')['spec'][
            'template'
        ]['spec']['containers'][0]
        env = {item['name']: item for item in container['env']}
        for variable, key in (
            ('AWS_ACCESS_KEY_ID', 'accesskey'),
            ('AWS_SECRET_ACCESS_KEY', 'secretkey'),
        ):
            reference = env[variable]['valueFrom']['secretKeyRef']
            self.assertEqual('seaweedfs-admin-credentials', reference['name'])
            self.assertEqual(key, reference['key'])
            self.assertFalse(reference.get('optional', False))

    def test_go_bootstrap_replaces_permanent_controller(self):
        self.assertFalse(
            any(
                item['kind'] == 'Deployment'
                and item['metadata']['name'] == 'seaweedfs-credential-rotation'
                for item in self.resources
            )
        )
        pod = self.resource('Deployment', 'seaweedfs')['spec']['template'][
            'spec'
        ]
        self.assertFalse(pod['automountServiceAccountToken'])
        init = next(
            item
            for item in pod['initContainers']
            if item['name'] == 'prepare-storage-credentials'
        )
        self.assertEqual(
            init['command'], ['/bin/credential-rotation', '--mode=bootstrap']
        )
        self.assertNotIn(':dummy', init['image'])
        self.assertEqual('IfNotPresent', init.get('imagePullPolicy'))
        if os.environ.get('OPENSHIFT_RENDERED') == 'true':
            self.assertNotIn('runAsUser', init.get('securityContext', {}))
        else:
            self.assertEqual(1000, init['securityContext']['runAsUser'])
        self.assertTrue(
            any(
                item['name'] == 'POD_NAME'
                and item['valueFrom']['fieldRef']['fieldPath']
                == 'metadata.name'
                for item in init['env']
            )
        )
        self.assertTrue(
            any(
                item['name'] == 'bootstrap-api-access'
                for item in init['volumeMounts']
            )
        )
        for container in pod['containers']:
            self.assertFalse(
                any(
                    item['name'] == 'bootstrap-api-access'
                    for item in container.get('volumeMounts', [])
                )
            )

    def test_atomic_apply_fields_include_api_defaults(self):
        pod = self.resource('Deployment', 'seaweedfs')['spec']['template'][
            'spec'
        ]
        init = next(
            item
            for item in pod['initContainers']
            if item['name'] == 'prepare-storage-credentials'
        )
        for variable in init['env']:
            reference = variable.get('valueFrom', {}).get('fieldRef')
            if reference:
                self.assertEqual('v1', reference.get('apiVersion'))
        volume = next(
            item
            for item in pod['volumes']
            if item['name'] == 'bootstrap-api-access'
        )
        for source in volume['projected']['sources']:
            for item in source.get('downwardAPI', {}).get('items', []):
                self.assertEqual('v1', item['fieldRef'].get('apiVersion'))
        policy = self.resource('NetworkPolicy', 'seaweedfs')
        for rule in policy['spec']['ingress']:
            for port in rule.get('ports', []):
                self.assertEqual('TCP', port.get('protocol'))

    def test_reconciler_has_no_cluster_role_or_pod_exec(self):
        role = self.resource('Role', 'seaweedfs-credential-rotation')
        for rule in role['rules']:
            self.assertNotIn('pods/exec', rule['resources'])
            self.assertNotIn('*', rule['verbs'])
        binding = self.resource('RoleBinding', 'seaweedfs-credential-rotation')
        self.assertEqual('Role', binding['roleRef']['kind'])

    def test_openshift_uses_assigned_identity(self):
        if os.environ.get('OPENSHIFT_RENDERED') != 'true':
            self.skipTest('OpenShift overlay only')
        pod = self.resource('Deployment', 'seaweedfs')['spec']['template'][
            'spec'
        ]
        self.assertNotIn('runAsUser', pod.get('securityContext', {}))
        self.assertNotIn('fsGroup', pod.get('securityContext', {}))


if __name__ == '__main__':
    unittest.main()

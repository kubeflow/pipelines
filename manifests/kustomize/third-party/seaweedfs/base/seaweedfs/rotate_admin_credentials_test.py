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

import copy
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import Mock
from unittest.mock import patch

HERE = Path(__file__).parent
spec = importlib.util.spec_from_file_location(
    'rotation', HERE / 'rotate-admin-credentials.py')
rotation = importlib.util.module_from_spec(spec)
spec.loader.exec_module(rotation)


def secret(name, pair=None):
    return {
        'metadata': {
            'name': name,
            'resourceVersion': '1', 'uid': 'test-secret-uid'
        },
        'data': rotation.encoded(pair) if pair else {},
    }


def deployment(name, credential):
    return {
        'metadata': {
            'name': name,
            'generation': 1,
            'resourceVersion': '1'
        },
        'spec': {
            'replicas': 1,
            'selector': {
                'matchLabels': {
                    'app': name
                }
            },
            'template': {
                'metadata': {
                    'annotations': {}
                },
                'spec': {
                    'containers': [{
                        'envFrom': [{
                            'secretRef': {
                                'name': credential
                            }
                        }]
                    }]
                },
            },
        },
        'status': {
            'observedGeneration': 1,
            'replicas': 1,
            'updatedReplicas': 1,
            'availableReplicas': 1,
        },
    }


def merge(target, changes):
    for key, value in changes.items():
        if isinstance(value, dict):
            merge(target.setdefault(key, {}), value)
        else:
            target[key] = value


class API:

    def __init__(self):
        self.objects = {
            ('secrets', rotation.ADMIN_SECRET):
                secret(rotation.ADMIN_SECRET),
            ('deployments', 'seaweedfs'):
                deployment('seaweedfs', rotation.ADMIN_SECRET),
        }
        self.objects[('deployments', 'seaweedfs')]['spec']['template']['metadata']['annotations'][rotation.REVISION] = 'test-secret-uid'
        self.requests = []
        self.old_storage = False
        self.old_consumer = False

    def get(self, kind, name):
        return copy.deepcopy(self.objects.get((kind, name)))

    def request(self, method, kind, name='', body=None, query=''):
        self.requests.append((method, kind, name, copy.deepcopy(body)))
        if kind == 'pods':
            if 'seaweedfs' in query:
                pod = copy.deepcopy(self.objects[('deployments', 'seaweedfs')]['spec']['template'])
                if self.old_storage:
                    pod['spec']['containers'][0]['envFrom'][0]['secretRef']['name'] = rotation.LEGACY_SECRET
                return {'items': [pod]}
            return {
                'items': [{
                    'metadata': {},
                    'status': {
                        'phase': 'Running'
                    }
                }] if self.old_consumer else []
            }
        if method == 'POST':
            name = body['metadata']['name']
            if (kind, name) in self.objects:
                raise rotation.APIError(409)
            self.objects[(kind, name)] = copy.deepcopy(body)
            self.objects[(kind, name)]['metadata']['resourceVersion'] = '1'
        elif method == 'PATCH':
            obj = self.objects[(kind, name)]
            if (body['metadata'].get('resourceVersion')
                    != obj['metadata']['resourceVersion']):
                raise rotation.APIError(409)
            merge(obj, body)
            obj['metadata']['resourceVersion'] = str(
                int(obj['metadata']['resourceVersion']) + 1)
        return self.get(kind, name)


class RotationTest(unittest.TestCase):

    def setUp(self):
        self.api = API()
        self.storage = Mock()
        self.factory = Mock(return_value=self.storage)
        self.migration = rotation.Migration(self.api, self.factory)

    def test_storage_restarts_after_registration_before_switching_consumers(self):
        self.api.objects[('deployments', 'seaweedfs')]['spec']['template']['metadata']['annotations'] = {}
        with self.assertRaisesRegex(rotation.Retry, 'Rolling deployment seaweedfs'):
            self.migration.reconcile()
        self.storage.verify_registration.assert_called_once()
        self.storage.verify.assert_not_called()
        self.storage.revoke_default.assert_not_called()
        self.assertIsNone(self.api.get('secrets', rotation.LEGACY_SECRET))
        self.migration.reconcile()
        self.storage.verify.assert_called()
        self.storage.revoke_default.assert_called_once()
        self.assertEqual(1, self.factory.call_count)

    def test_new_install_generates_once_and_publishes(self):
        self.migration.reconcile()
        pair = rotation.credentials(
            self.api.get('secrets', rotation.ADMIN_SECRET))
        self.assertFalse(rotation.is_default(pair))
        self.assertEqual(
            pair,
            rotation.credentials(
                self.api.get('secrets', rotation.LEGACY_SECRET)),
        )
        self.assertEqual(
            self.api.get('secrets', rotation.ADMIN_SECRET)['metadata']
            ['annotations'][rotation.PHASE],
            'DefaultRevoked',
        )
        self.migration.reconcile()
        self.assertEqual(
            pair,
            rotation.credentials(
                self.api.get('secrets', rotation.ADMIN_SECRET)),
        )

    def test_nondefault_custom_credentials_preserved(self):
        pair = ('operator-key', 'operator-secret')
        self.api.objects[('secrets', rotation.LEGACY_SECRET)] = secret(
            rotation.LEGACY_SECRET, pair)
        self.migration.reconcile()
        self.assertEqual(
            pair,
            rotation.credentials(
                self.api.get('secrets', rotation.ADMIN_SECRET)),
        )
        self.assertFalse(
            any(request[1:3] == ('secrets', rotation.LEGACY_SECRET)
                for request in self.api.requests))

    def test_public_key_id_with_custom_secret_is_not_revoked(self):
        pair = (rotation.DEFAULT_ACCESS_KEY, 'operator-secret')
        self.api.objects[('secrets', rotation.LEGACY_SECRET)] = secret(
            rotation.LEGACY_SECRET, pair)
        self.migration.reconcile()
        self.storage.revoke_default.assert_called_once()

    def test_historical_customized_key_is_preserved(self):
        storage = rotation.Storage.__new__(rotation.Storage)
        storage.pair = ('replacement', 'new-secret')
        storage.iam = Mock()
        storage.default_accepted = Mock(return_value=False)
        storage.revoke_default()
        storage.iam.delete_access_key.assert_not_called()

    def test_known_default_is_revoked_and_rechecked(self):
        storage = rotation.Storage.__new__(rotation.Storage)
        storage.pair = ('replacement', 'new-secret')
        storage.iam = Mock()
        storage.default_accepted = Mock(side_effect=[True, False])
        storage.revoke_default()
        storage.iam.delete_access_key.assert_called_once_with(
            UserName='kubeflow-admin', AccessKeyId=rotation.DEFAULT_ACCESS_KEY)
        self.assertEqual(2, storage.default_accepted.call_count)

    def test_default_pair_is_rotated(self):
        old = ('synthetic-default', 'synthetic-default-secret')
        self.api.objects[('secrets', rotation.LEGACY_SECRET)] = secret(
            rotation.LEGACY_SECRET, old)
        with patch.object(
                rotation,
                'DEFAULT_FINGERPRINT',
                rotation.hashlib.sha256('\0'.join(old).encode()).hexdigest(),
        ):
            self.migration.reconcile()
        self.assertNotEqual(
            old,
            rotation.credentials(
                self.api.get('secrets', rotation.LEGACY_SECRET)),
        )

    def test_crash_after_prepare_reuses_same_replacement(self):
        state = self.migration.prepare()
        self.storage.verify.side_effect = RuntimeError(
            'simulated unavailable storage')
        with self.assertRaises(RuntimeError):
            self.migration.reconcile()
        self.assertIsNone(self.api.get('secrets', rotation.LEGACY_SECRET))
        self.storage.revoke_default.assert_not_called()
        self.storage.verify.side_effect = None
        self.migration.reconcile()
        self.assertEqual(
            rotation.credentials(state),
            rotation.credentials(
                self.api.get('secrets', rotation.LEGACY_SECRET)),
        )

    def test_old_storage_pod_blocks_revocation_and_publish(self):
        self.api.old_storage = True
        with self.assertRaisesRegex(rotation.Retry, 'old SeaweedFS'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()
        self.assertIsNone(self.api.get('secrets', rotation.LEGACY_SECRET))

    def test_pending_rollout_blocks_revocation(self):
        self.api.objects[('deployments',
                          'ml-pipeline')] = deployment('ml-pipeline',
                                                       rotation.LEGACY_SECRET)
        with self.assertRaisesRegex(rotation.Retry, 'Rolling deployment'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()
        self.api.objects[('deployments',
                          'ml-pipeline')]['status']['availableReplicas'] = 0
        with self.assertRaisesRegex(rotation.Retry, 'finish rolling out'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()
        self.api.objects[('deployments',
                          'ml-pipeline')]['status']['availableReplicas'] = 1
        self.migration.reconcile()
        self.storage.revoke_default.assert_called_once()

    def test_terminating_old_consumer_blocks_revocation(self):
        self.api.objects[('deployments',
                          'ml-pipeline')] = deployment('ml-pipeline',
                                                       rotation.LEGACY_SECRET)
        with self.assertRaises(rotation.Retry):
            self.migration.reconcile()
        self.api.old_consumer = True
        with self.assertRaisesRegex(rotation.Retry, 'old consumer pods'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_unrelated_secrets_and_custom_deployments_untouched(self):
        self.api.objects[('secrets', 'tenant-secret')] = secret(
            'tenant-secret', ('tenant', 'tenant-secret'))
        self.api.objects[('deployments',
                          'ml-pipeline')] = deployment('ml-pipeline',
                                                       'external-s3')
        before = self.api.get('secrets', 'tenant-secret')
        self.migration.reconcile()
        self.assertEqual(before, self.api.get('secrets', 'tenant-secret'))
        self.assertFalse(
            any(request[1] == 'deployments' for request in self.api.requests))

    def test_custom_edit_during_migration_is_not_overwritten(self):
        self.migration.prepare()
        self.api.objects[('secrets', rotation.LEGACY_SECRET)] = secret(
            rotation.LEGACY_SECRET, ('new-operator', 'new-secret'))
        with self.assertRaisesRegex(rotation.Retry, 'custom credentials'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_malformed_secret_blocks_before_generation(self):
        self.api.objects[('secrets', rotation.LEGACY_SECRET)] = secret(
            rotation.LEGACY_SECRET)
        self.api.objects[('secrets', rotation.LEGACY_SECRET)]['data'] = {
            'accesskey': 'not-base64'
        }
        with self.assertRaisesRegex(rotation.Retry, 'incomplete'):
            self.migration.reconcile()
        self.assertIsNone(
            rotation.credentials(
                self.api.get('secrets', rotation.ADMIN_SECRET)))

    def test_declarative_drift_triggers_a_new_consumer_rollout(self):
        self.migration.reconcile()
        first = self.api.get('secrets', rotation.ADMIN_SECRET)
        del self.api.objects[('secrets', rotation.LEGACY_SECRET)]
        self.api.objects[('deployments',
                          'ml-pipeline')] = deployment('ml-pipeline',
                                                       rotation.LEGACY_SECRET)
        self.api.objects[(
            'deployments',
            'ml-pipeline')]['spec']['template']['metadata']['annotations'][
                rotation.REVISION] = rotation.revision(first)
        with self.assertRaisesRegex(rotation.Retry, 'Rolling deployment'):
            self.migration.reconcile()
        second = self.api.get('secrets', rotation.ADMIN_SECRET)
        self.assertEqual(
            rotation.credentials(first), rotation.credentials(second))
        self.assertNotEqual(rotation.revision(first), rotation.revision(second))

    def test_negative_probe_matches_shipped_default_fingerprint(self):
        self.assertTrue(
            rotation.is_default(
                (rotation.DEFAULT_ACCESS_KEY, rotation.DEFAULT_SECRET_KEY)))

    def test_secret_updates_use_resource_version(self):
        self.migration.reconcile()
        for method, kind, _, body in self.api.requests:
            if method == 'PATCH' and kind == 'secrets':
                self.assertIn('resourceVersion', body['metadata'])

    def test_revocation_failure_never_marks_complete(self):
        self.storage.revoke_default.side_effect = RuntimeError(
            'synthetic failure')
        with self.assertRaises(RuntimeError):
            self.migration.reconcile()
        self.assertNotEqual(
            'DefaultRevoked',
            self.api.get('secrets', rotation.ADMIN_SECRET)['metadata']
            ['annotations'][rotation.PHASE],
        )


if __name__ == '__main__':
    unittest.main()

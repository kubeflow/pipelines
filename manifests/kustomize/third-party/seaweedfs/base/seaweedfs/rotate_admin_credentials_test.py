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
import sys
import tempfile
from types import SimpleNamespace
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
            'resourceVersion': '1',
            'uid': 'test-secret-uid',
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
        self.objects[(
            'deployments',
            'seaweedfs')]['spec']['template']['metadata']['annotations'][
                rotation.REVISION] = 'test-secret-uid'
        self.requests = []
        self.old_storage = False
        self.old_consumer = False
        self.drain_pages = [{'metadata': {}, 'items': []}]

    def get(self, kind, name):
        return copy.deepcopy(self.objects.get((kind, name)))

    def request(self, method, kind, name='', body=None, query=''):
        self.requests.append((method, kind, name, copy.deepcopy(body)))
        if kind == 'pods':
            if query.startswith('limit='):
                token = rotation.urllib.parse.parse_qs(query).get(
                    'continue', ['0'])[0]
                return copy.deepcopy(self.drain_pages[int(token)])
            if 'seaweedfs' in query:
                pod = copy.deepcopy(
                    self.objects[('deployments',
                                  'seaweedfs')]['spec']['template'])
                if self.old_storage:
                    pod['spec']['containers'][0]['envFrom'][0]['secretRef'][
                        'name'] = rotation.LEGACY_SECRET
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
        self.storage.default_accepted.return_value = True
        self.factory = Mock(return_value=self.storage)
        self.migration = rotation.Migration(self.api, self.factory)

    def workload(self,
                 name='running-task',
                 phase='Running',
                 labels=None,
                 spec=None):
        return {
            'metadata': {
                'name':
                    name,
                'labels':
                    labels if labels is not None else {
                        'pipeline/runid': 'run-1'
                    },
            },
            'spec': spec or {},
            'status': {
                'phase': phase
            },
        }

    def test_healthy_drain_is_ready_without_completing_migration(self):
        self.api.drain_pages[0]['items'] = [self.workload()]
        with tempfile.TemporaryDirectory() as directory:
            ready = Path(directory) / 'controller-ready'
            with (
                    patch.object(
                        rotation, 'Migration', return_value=self.migration),
                    patch.object(rotation, 'Kubernetes'),
                    patch.dict(rotation.os.environ,
                               {'POD_NAMESPACE': 'fixture'}),
                    patch.object(rotation, 'Path', return_value=ready),
                    patch.object(
                        rotation.time, 'sleep', side_effect=StopIteration),
            ):
                with self.assertRaises(StopIteration):
                    rotation.main()
            self.assertTrue(ready.exists())
        state = self.api.get('secrets', rotation.ADMIN_SECRET)
        self.assertEqual(
            'DrainingWorkloads',
            state['metadata']['annotations'][rotation.PHASE],
        )
        self.storage.revoke_default.assert_not_called()

    def test_readiness_recovers_after_an_actual_failure(self):
        migration = Mock(stage='fixture')
        with tempfile.TemporaryDirectory() as directory:
            ready = Path(directory) / 'controller-ready'
            for failure in (
                    rotation.Retry('bad configuration'),
                    rotation.APIError(503),
            ):
                ready.touch()
                migration.reconcile.side_effect = failure
                rotation.reconcile_once(migration, ready)
                self.assertFalse(ready.exists())
                migration.reconcile.side_effect = rotation.Waiting(
                    'normal rollout wait')
                rotation.reconcile_once(migration, ready)
                self.assertTrue(ready.exists())

    def test_drain_resumes_and_new_work_extends_wait(self):
        self.api.drain_pages[0]['items'] = [self.workload()]
        for _ in range(2):
            with self.assertRaisesRegex(rotation.Retry,
                                        'old credential remains valid'):
                self.migration.reconcile()
            self.migration = rotation.Migration(self.api, self.factory)
        state = self.api.get('secrets', rotation.ADMIN_SECRET)
        self.assertEqual(
            'DrainingWorkloads',
            state['metadata']['annotations'][rotation.PHASE],
        )
        self.assertEqual('1',
                         state['metadata']['annotations'][rotation.DRAIN_COUNT])
        self.api.drain_pages[0]['items'] = [
            self.workload(phase='Succeeded'),
            self.workload('new-task'),
        ]
        with self.assertRaises(rotation.Retry):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()
        self.api.drain_pages[0]['items'][1]['status']['phase'] = 'Failed'
        self.migration.reconcile()
        self.storage.revoke_default.assert_called_once()
        self.assertEqual(
            '0',
            self.api.get('secrets', rotation.ADMIN_SECRET)['metadata']
            ['annotations'][rotation.DRAIN_COUNT],
        )

    def test_drain_detects_secret_references_and_active_phases(self):
        specs = []
        for container_kind in (
                'containers',
                'initContainers',
                'ephemeralContainers',
        ):
            specs.extend([
                {
                    container_kind: [{
                        'envFrom': [{
                            'secretRef': {
                                'name': rotation.LEGACY_SECRET
                            }
                        }]
                    }]
                },
                {
                    container_kind: [{
                        'env': [{
                            'valueFrom': {
                                'secretKeyRef': {
                                    'name': rotation.LEGACY_SECRET
                                }
                            }
                        }]
                    }]
                },
            ])
        specs.extend([
            {
                'volumes': [{
                    'secret': {
                        'secretName': rotation.LEGACY_SECRET
                    }
                }]
            },
            {
                'volumes': [{
                    'projected': {
                        'sources': [{
                            'secret': {
                                'name': rotation.LEGACY_SECRET
                            }
                        }]
                    }
                }]
            },
        ])
        for spec in specs:
            for phase in ('Pending', 'Running', 'Unknown', None):
                with self.subTest(spec=spec, phase=phase):
                    pod = self.workload(labels={}, spec=spec, phase=phase)
                    pod['metadata']['deletionTimestamp'] = (
                        '2026-01-01T00:00:00Z')
                    self.api.drain_pages[0]['items'] = [pod]
                    with self.assertRaises(rotation.Retry):
                        self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_ignores_unrelated_and_verified_managed_pods(self):
        state = self.migration.prepare()
        managed = deployment('ml-pipeline', rotation.LEGACY_SECRET)
        managed['spec']['template']['metadata']['annotations'][
            rotation.REVISION] = rotation.revision(state)
        self.api.objects[('deployments', 'ml-pipeline')] = managed
        pod = self.workload(
            labels={'app': 'ml-pipeline'},
            spec=managed['spec']['template']['spec'],
        )
        pod['metadata']['annotations'] = {
            rotation.REVISION: rotation.revision(state)
        }
        self.api.drain_pages[0]['items'] = [
            self.workload('unrelated', labels={}),
            pod,
        ]
        self.migration.reconcile()
        self.storage.revoke_default.assert_called_once()
        self.storage.revoke_default.reset_mock()
        pod['metadata']['labels']['pipeline/runid'] = 'run-1'
        with self.assertRaises(rotation.Retry):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_does_not_exempt_wrong_consumer_revision(self):
        state = self.migration.prepare()
        managed = deployment('ml-pipeline', rotation.LEGACY_SECRET)
        managed['spec']['template']['metadata']['annotations'][
            rotation.REVISION] = rotation.revision(state)
        self.api.objects[('deployments', 'ml-pipeline')] = managed
        self.api.drain_pages[0]['items'] = [
            self.workload(
                labels={'app': 'ml-pipeline'},
                spec=managed['spec']['template']['spec'],
            )
        ]
        with self.assertRaises(rotation.Retry):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_paginates_with_bounded_sample(self):
        self.api.drain_pages = [
            {
                'metadata': {
                    'continue': '1'
                },
                'items': [self.workload('first')]
            },
            {
                'metadata': {},
                'items': [self.workload('task-' + str(i)) for i in range(8)],
            },
        ]
        with self.assertRaises(rotation.Retry):
            self.migration.reconcile()
        annotations = self.api.get(
            'secrets', rotation.ADMIN_SECRET)['metadata']['annotations']
        self.assertEqual('9', annotations[rotation.DRAIN_COUNT])
        self.assertEqual(
            5, len(rotation.json.loads(annotations[rotation.DRAIN_SAMPLE])))
        self.storage.revoke_default.assert_not_called()

    def test_drain_fails_closed_on_malformed_inventory(self):
        for response in (
            {},
            {
                'metadata': {},
                'items': [None]
            },
            {
                'metadata': {},
                'items': [{}]
            },
            {
                'metadata': {},
                'items': 'bad'
            },
        ):
            with self.subTest(response=response):
                self.api.drain_pages = [response]
                with self.assertRaises(rotation.Retry):
                    self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_api_error_never_revokes(self):
        original = self.api.request

        def request(method, kind, name='', body=None, query=''):
            if kind == 'pods' and query.startswith('limit='):
                raise rotation.APIError(503)
            return original(method, kind, name, body, query)

        self.api.request = request
        with self.assertRaises(rotation.APIError):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_repeated_continuation_never_revokes(self):
        self.api.drain_pages = [
            {
                'metadata': {
                    'continue': '1'
                },
                'items': []
            },
            {
                'metadata': {
                    'continue': '1'
                },
                'items': []
            },
        ]
        with self.assertRaisesRegex(rotation.Retry, 'did not advance'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_scan_limit_never_revokes(self):
        self.api.drain_pages = [{
            'metadata': {
                'continue': str(i + 1)
            },
            'items': []
        } for i in range(100)]
        with self.assertRaisesRegex(rotation.Retry, 'bounded scan'):
            self.migration.reconcile()
        self.storage.revoke_default.assert_not_called()

    def test_drain_skipped_when_default_already_rejected(self):
        self.storage.default_accepted.return_value = False
        self.api.drain_pages[0]['items'] = [self.workload()]
        self.migration.reconcile()
        self.assertEqual(
            'DefaultRevoked',
            self.api.get('secrets', rotation.ADMIN_SECRET)['metadata']
            ['annotations'][rotation.PHASE],
        )

    def test_storage_restarts_after_registration_before_switching_consumers(
        self,):
        self.api.objects[(
            'deployments',
            'seaweedfs')]['spec']['template']['metadata']['annotations'] = {}
        with self.assertRaisesRegex(rotation.Retry,
                                    'Rolling deployment seaweedfs'):
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

    @unittest.skipUnless(
        importlib.util.find_spec('botocore'),
        'SDK model test requires botocore (installed in CI)',
    )
    def test_real_sdk_accepts_short_legacy_id_for_revocation(self):
        from botocore.stub import Stubber

        storage = rotation.Storage(
            ('synthetic-replacement', 'synthetic-secret'))
        storage.default_accepted = Mock(side_effect=[True, False])
        try:
            with Stubber(storage.iam) as stub:
                stub.add_response(
                    'delete_access_key',
                    {},
                    {
                        'UserName': 'kubeflow-admin',
                        'AccessKeyId': rotation.DEFAULT_ACCESS_KEY,
                    },
                )
                storage.revoke_default()
                stub.assert_no_pending_responses()
        finally:
            storage.close()

    def test_stale_iam_cache_blocks_completion_and_retries_deletion(self):

        class ClientError(Exception):

            def __init__(self, status):
                self.response = {'ResponseMetadata': {'HTTPStatusCode': status}}

        storage = rotation.Storage.__new__(rotation.Storage)
        storage.pair = ('replacement', 'new-secret')
        storage.iam = Mock()
        storage.default_s3 = Mock()
        storage.default_s3.head_bucket.side_effect = ClientError(403)
        storage.default_iam = Mock()
        storage.default_iam.list_users.side_effect = [
            {},
            {},
            {},
            ClientError(403),
        ]
        with patch.dict(
                sys.modules,
            {'botocore.exceptions': SimpleNamespace(ClientError=ClientError)},
        ):
            with self.assertRaises(rotation.Retry):
                storage.revoke_default()
            storage.revoke_default()
        self.assertEqual(storage.iam.delete_access_key.call_count, 2)

    def test_iam_probe_server_failure_is_not_revocation(self):

        class ClientError(Exception):

            def __init__(self, status):
                self.response = {'ResponseMetadata': {'HTTPStatusCode': status}}

        storage = rotation.Storage.__new__(rotation.Storage)
        storage.default_s3 = Mock()
        storage.default_s3.head_bucket.side_effect = ClientError(403)
        storage.default_iam = Mock()
        storage.default_iam.list_users.side_effect = ClientError(503)
        with patch.dict(
                sys.modules,
            {'botocore.exceptions': SimpleNamespace(ClientError=ClientError)},
        ):
            with self.assertRaises(ClientError):
                storage.default_accepted()

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

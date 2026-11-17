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
import io
import json
import subprocess
import tempfile
import unittest
from contextlib import nullcontext
from pathlib import Path
from unittest.mock import Mock, patch

import third_party_upgrade_compatibility as compatibility


class CompatibilityTest(unittest.TestCase):
    def test_secret_credentials_string_and_encoded_data(self):
        self.assertEqual(
            compatibility.secret_credentials(
                {'data': {'accesskey': 'bWluaW8=', 'secretkey': 'bWluaW8xMjM='}}
            ),
            ('minio', 'minio123'),
        )
        self.assertEqual(
            compatibility.secret_credentials(
                {
                    'data': {'accesskey': 'b2xk', 'secretkey': 'bWluaW8xMjM='},
                    'stringData': {'accesskey': 'override'},
                }
            ),
            ('override', 'minio123'),
        )

    def test_invalid_secret_credentials_fail_closed(self):
        for secret in (
            {},
            {'data': {'accesskey': 'not base64'}},
            {'stringData': {'accesskey': '', 'secretkey': 'value'}},
            {'stringData': {'accesskey': 123, 'secretkey': 'value'}},
            {'data': {'accesskey': '/w==', 'secretkey': 'bWluaW8='}},
        ):
            with (
                self.subTest(secret=secret),
                self.assertRaisesRegex(ValueError, 'invalid or missing'),
            ):
                compatibility.secret_credentials(secret)

    def test_seed_uses_manifest_credentials_without_overwriting_secret(self):
        storage = [
            {
                'kind': 'Deployment',
                'spec': {
                    'template': {
                        'spec': {'containers': [{'image': 'seaweedfs:new'}]}
                    }
                },
            },
            {
                'kind': 'Secret',
                'metadata': {'name': 'mlpipeline-minio-artifact'},
                'stringData': {
                    'accesskey': 'manifest-user',
                    'secretkey': 'manifest-key',
                },
            },
        ]
        with (
            tempfile.TemporaryDirectory() as directory,
            patch.object(compatibility, 'apply') as apply,
            patch.object(compatibility, 'wait_ready'),
            patch.object(
                compatibility,
                'kubectl',
                return_value='{"metadata":{"uid":"same-pvc"}}',
            ),
            patch.object(
                compatibility,
                'forward',
                side_effect=lambda *args: nullcontext('http://localhost'),
            ),
            patch.object(compatibility, 'client') as client,
            patch.object(compatibility, 'verify_state'),
        ):
            credentials, uid = compatibility.seed_old_store(
                storage, 'seaweedfs:baseline', Path(directory)
            )
        self.assertEqual(credentials, ('manifest-user', 'manifest-key'))
        self.assertEqual(uid, 'same-pvc')
        self.assertEqual(
            storage[0]['spec']['template']['spec']['containers'][0]['image'],
            'seaweedfs:baseline',
        )
        self.assertFalse(
            any(
                obj['kind'] == 'Secret'
                for obj in apply.call_args_list[0].args[0]
            )
        )
        self.assertEqual(apply.call_args_list[1].args[0], storage)
        self.assertEqual(
            [call.args[1:] for call in client.call_args_list],
            [('manifest-user', 'manifest-key')],
        )

    def test_command_failure_surfaces_stderr(self):
        failure = subprocess.CalledProcessError(
            1, ['kubectl'], stderr='forbidden'
        )
        with (
            patch.object(compatibility.subprocess, 'run', side_effect=failure),
            patch('sys.stderr', new_callable=io.StringIO) as errors,
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                compatibility.run('kubectl', 'apply')
        self.assertIn('forbidden', errors.getvalue())

    def test_cleanup_does_not_replace_original_failure(self):
        with (
            patch.object(
                compatibility,
                'run',
                side_effect=subprocess.CalledProcessError(1, ['kind']),
            ),
            patch('sys.stderr', new_callable=io.StringIO) as errors,
        ):
            with self.assertRaisesRegex(RuntimeError, 'original'):
                try:
                    raise RuntimeError('original validation failed')
                finally:
                    compatibility.delete_cluster()
        self.assertIn('Cluster cleanup failed', errors.getvalue())

    def test_baseline_image_is_read_from_rendered_base_ref(self):
        def store(image):
            return [
                {
                    'kind': 'Deployment',
                    'metadata': {'name': 'seaweedfs'},
                    'spec': {
                        'template': {
                            'spec': {
                                'containers': [
                                    {'name': 'seaweedfs', 'image': image}
                                ]
                            }
                        }
                    },
                }
            ]

        with (
            patch.object(
                compatibility,
                'rendered',
                side_effect=[store('candidate:4.50'), store('baseline:4.49')],
            ) as render,
            patch.object(
                compatibility,
                'seed_old_store',
                return_value=(('user', 'key'), 'pvc'),
            ) as seed,
            patch.object(compatibility, 'upgrade_store') as upgrade,
            patch.object(compatibility, 'exercise_profile', return_value={}),
            patch.object(compatibility, 'verify_after_restart'),
            patch.object(compatibility, 'write_result') as result,
        ):
            compatibility.exercise(Path('evidence'), Path('baseline'))
        self.assertEqual(
            render.call_args_list[1].args[0],
            Path('baseline')
            / compatibility.SEAWEED.relative_to(compatibility.ROOT),
        )
        self.assertEqual(seed.call_args.args[1], 'baseline:4.49')
        upgrade.assert_called_once_with('candidate:4.50')
        result.assert_called_once_with(
            Path('evidence'), 'baseline:4.49', 'candidate:4.50', 'pvc'
        )

    def test_profile_uses_candidate_build_and_preserves_credentials_after_restart(
        self,
    ):
        profile = [
            {
                'kind': 'Deployment',
                'metadata': {'name': 'kubeflow-pipelines-profile-controller'},
                'spec': {
                    'template': {
                        'spec': {
                            'containers': [
                                {
                                    'name': 'profile-controller',
                                    'image': 'unpublished:tag',
                                }
                            ]
                        }
                    }
                },
            }
        ]
        secret = {'data': {'accesskey': 'dXNlcg==', 'secretkey': 'a2V5'}}
        receipt = {'image': 'kfp-profile-controller:compat-source'}
        with (
            tempfile.TemporaryDirectory() as directory,
            patch.object(compatibility, 'rendered', return_value=profile),
            patch.object(
                compatibility,
                'image_reference',
                return_value='ghcr.io/kubeflow/kfp-profile-controller:master',
            ),
            patch.object(
                compatibility, 'build_profile_image', return_value=receipt
            ),
            patch.object(
                compatibility,
                'run',
                side_effect=['a' * 40, '', '[{"Id":"sha256:built"}]'],
            ) as run,
            patch.object(compatibility, 'apply'),
            patch.object(compatibility, 'wait_ready'),
            patch.object(compatibility, 'kubectl') as kubectl,
            patch.object(
                compatibility,
                'forward',
                side_effect=lambda *args: nullcontext('http://localhost'),
            ),
            patch.object(
                compatibility, 'sync_profile', return_value=secret
            ) as sync,
            patch.object(compatibility, 'client'),
            patch.object(compatibility, 'verify_payload'),
            patch.object(compatibility, 'verify_state'),
            patch.object(
                compatibility, 'exercise_profile_rollback'
            ) as rollback,
        ):
            self.assertEqual(
                compatibility.exercise_profile(
                    ('admin', 'key'), Path(directory)
                ),
                secret,
            )
            self.assertTrue((Path(directory) / 'profile-runtime.json').exists())
        container = profile[0]['spec']['template']['spec']['containers'][0]
        self.assertEqual(container['image'], receipt['image'])
        self.assertEqual(container['imagePullPolicy'], 'Never')
        self.assertIn(
            (
                'kind',
                'load',
                'docker-image',
                '--name',
                compatibility.CLUSTER,
                receipt['image'],
            ),
            [call.args for call in run.call_args_list],
        )
        kubectl.assert_called_once_with(
            'rollout',
            'restart',
            'deployment/kubeflow-pipelines-profile-controller',
        )
        rollback.assert_called_once_with(
            receipt['image'], secret, ('admin', 'key'), Path(directory)
        )
        self.assertEqual(sync.call_count, 3)
        self.assertEqual(sync.call_args.args[-1], secret)

    def test_profile_rollback_preserves_credentials_and_objects_both_ways(
        self,
    ) -> None:
        secret = {'data': {'accesskey': 'dXNlcg==', 'secretkey': 'a2V5'}}
        objects = {
            'private-artifacts/compat-user/after-upgrade': compatibility.PAYLOAD
        }
        user = Mock()

        def put_object(*, Bucket: str, Key: str, Body: bytes) -> None:
            objects[Key] = Body

        def get_object(*, Bucket: str, Key: str) -> dict:
            return {'Body': io.BytesIO(objects[Key])}

        user.put_object.side_effect = put_object
        user.get_object.side_effect = get_object
        with (
            tempfile.TemporaryDirectory() as directory,
            patch.object(compatibility, 'run') as run,
            patch.object(compatibility, 'kubectl') as kubectl,
            patch.object(compatibility, 'wait_ready'),
            patch.object(
                compatibility,
                'profile_running_image',
                side_effect=lambda image: {
                    'image': image,
                    'image_id': 'sha256:test',
                },
            ) as running,
            patch.object(
                compatibility,
                'forward',
                side_effect=lambda *args: nullcontext('http://localhost'),
            ),
            patch.object(
                compatibility, 'sync_profile', return_value=secret
            ) as sync,
            patch.object(compatibility, 'client', return_value=user) as client,
            patch.object(compatibility, 'verify_state'),
        ):
            compatibility.exercise_profile_rollback(
                'candidate:tested', secret, ('admin', 'key'), Path(directory)
            )
            receipts = json.loads(
                (Path(directory) / 'profile-rollback.json').read_text()
            )
        self.assertEqual(
            [call.args[-1] for call in kubectl.call_args_list],
            [
                f'profile-controller={compatibility.PROFILE_ROLLBACK_IMAGE}',
                'profile-controller=candidate:tested',
            ],
        )
        self.assertEqual(
            [call.args[0] for call in running.call_args_list],
            [compatibility.PROFILE_ROLLBACK_IMAGE, 'candidate:tested'],
        )
        self.assertEqual(
            [item['phase'] for item in receipts], ['rollback', 'restore']
        )
        self.assertEqual(user.close.call_count, 4)
        self.assertEqual(sync.call_count, 4)
        self.assertTrue(
            all(call.args[-1] == secret for call in sync.call_args_list)
        )
        self.assertEqual(
            [call.args[1:] for call in client.call_args_list],
            [('admin', 'key'), ('user', 'key')] * 2,
        )
        self.assertEqual(
            [call.kwargs['Key'] for call in user.get_object.call_args_list],
            [
                'private-artifacts/compat-user/after-upgrade',
                'private-artifacts/compat-user/after-rollback',
                'private-artifacts/compat-user/after-upgrade',
                'private-artifacts/compat-user/after-rollback',
                'private-artifacts/compat-user/after-restore',
            ],
        )
        self.assertEqual(
            run.call_args_list[0].args[-1], compatibility.PROFILE_ROLLBACK_IMAGE
        )
        self.assertEqual(
            run.call_args_list[1].args,
            (
                'kind',
                'load',
                'docker-image',
                '--name',
                compatibility.CLUSTER,
                compatibility.PROFILE_ROLLBACK_IMAGE,
            ),
        )

    def test_profile_rollback_fails_on_credential_rotation_in_either_direction(
        self,
    ) -> None:
        secret = {'data': {'accesskey': 'dXNlcg==', 'secretkey': 'a2V5'}}
        for responses in ([{'data': {}}], [secret, secret, {'data': {}}]):
            with (
                self.subTest(responses=responses),
                tempfile.TemporaryDirectory() as directory,
                patch.object(compatibility, 'run'),
                patch.object(compatibility, 'kubectl'),
                patch.object(compatibility, 'wait_ready'),
                patch.object(
                    compatibility, 'profile_running_image', return_value={}
                ),
                patch.object(
                    compatibility,
                    'forward',
                    side_effect=lambda *args: nullcontext('http://localhost'),
                ),
                patch.object(
                    compatibility, 'sync_profile', side_effect=responses
                ),
                patch.object(compatibility, 'client'),
                patch.object(compatibility, 'verify_state'),
                patch.object(compatibility, 'verify_payload'),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'replaced existing credentials'
                ):
                    compatibility.exercise_profile_rollback(
                        'candidate:tested',
                        secret,
                        ('admin', 'key'),
                        Path(directory),
                    )
                self.assertFalse(
                    (Path(directory) / 'profile-rollback.json').exists()
                )

    def test_profile_rollback_failures_do_not_write_success_receipt(
        self,
    ) -> None:
        secret = {'data': {'accesskey': 'dXNlcg==', 'secretkey': 'a2V5'}}
        for failed_step in (
            'wait_ready',
            'profile_running_image',
            'forward',
            'verify_payload',
        ):
            with (
                self.subTest(failed_step=failed_step),
                tempfile.TemporaryDirectory() as directory,
                patch.object(compatibility, 'run'),
                patch.object(compatibility, 'kubectl'),
                patch.object(compatibility, 'wait_ready'),
                patch.object(
                    compatibility, 'profile_running_image', return_value={}
                ),
                patch.object(
                    compatibility,
                    'forward',
                    side_effect=lambda *args: nullcontext('http://localhost'),
                ),
                patch.object(
                    compatibility, 'sync_profile', return_value=secret
                ),
                patch.object(compatibility, 'client'),
                patch.object(compatibility, 'verify_state'),
                patch.object(compatibility, 'verify_payload'),
            ):
                getattr(compatibility, failed_step).side_effect = RuntimeError(
                    'smoke failed'
                )
                with self.assertRaisesRegex(RuntimeError, 'smoke failed'):
                    compatibility.exercise_profile_rollback(
                        'candidate:tested',
                        secret,
                        ('admin', 'key'),
                        Path(directory),
                    )
                self.assertFalse(
                    (Path(directory) / 'profile-rollback.json').exists()
                )

    def test_profile_running_image_requires_requested_ready_runtime(
        self,
    ) -> None:
        pod = {
            'metadata': {'uid': 'pod'},
            'spec': {
                'containers': [
                    {'name': 'profile-controller', 'image': 'candidate:tested'}
                ]
            },
            'status': {
                'containerStatuses': [
                    {
                        'name': 'profile-controller',
                        'ready': True,
                        'state': {'running': {}},
                        'imageID': 'sha256:test',
                    }
                ]
            },
        }
        with patch.object(
            compatibility, 'kubectl', return_value=json.dumps({'items': [pod]})
        ):
            self.assertEqual(
                compatibility.profile_running_image('candidate:tested')[
                    'image_id'
                ],
                'sha256:test',
            )
            with self.assertRaisesRegex(AssertionError, 'requested image'):
                compatibility.profile_running_image('prior:tested')
        pod['status']['containerStatuses'][0]['ready'] = False
        with patch.object(
            compatibility, 'kubectl', return_value=json.dumps({'items': [pod]})
        ):
            with self.assertRaisesRegex(AssertionError, 'requested image'):
                compatibility.profile_running_image('candidate:tested')

    def test_ambiguous_baseline_image_fails(self):
        with self.assertRaisesRegex(ValueError, 'Expected one'):
            compatibility.storage_image([])

    def test_corrupted_persistent_object_fails(self):
        s3 = Mock()
        s3.get_object.return_value = {'Body': io.BytesIO(b'changed')}
        with self.assertRaisesRegex(AssertionError, 'changed during upgrade'):
            compatibility.verify_payload(s3, 'object')

    def test_missing_lifecycle_rule_fails(self):
        s3 = Mock()
        s3.get_object.return_value = {'Body': io.BytesIO(compatibility.PAYLOAD)}
        s3.get_bucket_lifecycle_configuration.return_value = {'Rules': []}
        with self.assertRaisesRegex(AssertionError, 'Lifecycle configuration'):
            compatibility.verify_state(s3)

    def test_persistent_data_and_lifecycle_pass(self):
        s3 = Mock()
        s3.get_object.return_value = {'Body': io.BytesIO(compatibility.PAYLOAD)}
        s3.get_bucket_lifecycle_configuration.return_value = {
            'Rules': [{'Status': 'Enabled', 'Expiration': {'Days': 30}}]
        }
        compatibility.verify_state(s3)

    def test_reconciliation_sends_existing_credentials(self):
        secret = {
            'kind': 'Secret',
            'metadata': {'name': 'mlpipeline-minio-artifact'},
            'data': {'accesskey': 'test'},
        }
        token = {
            'kind': 'Secret',
            'metadata': {'name': 'pipeline-runner-token'},
            'type': 'kubernetes.io/service-account-token',
        }
        response = io.BytesIO(
            json.dumps({'attachments': [token, secret]}).encode()
        )
        with patch.object(
            compatibility.urllib.request, 'urlopen', return_value=response
        ) as request:
            self.assertEqual(
                compatibility.sync_profile('http://localhost', 'user', secret),
                secret,
            )
        observed = json.loads(request.call_args.args[0].data)
        self.assertEqual(
            observed['attachments']['Secret.v1'],
            {'user/mlpipeline-minio-artifact': secret},
        )

    def test_reconciliation_rejects_missing_or_ambiguous_artifact_secret(self):
        secret = {
            'kind': 'Secret',
            'metadata': {'name': 'mlpipeline-minio-artifact'},
        }
        for attachments in (
            [],
            [{'kind': 'Secret', 'metadata': {'name': 'token'}}],
            [secret, secret],
        ):
            with self.subTest(attachments=attachments):
                response = io.BytesIO(
                    json.dumps({'attachments': attachments}).encode()
                )
                with patch.object(
                    compatibility.urllib.request,
                    'urlopen',
                    return_value=response,
                ):
                    with self.assertRaisesRegex(
                        ValueError, 'Expected one artifact'
                    ):
                        compatibility.sync_profile('http://localhost', 'user')


if __name__ == '__main__':
    unittest.main()

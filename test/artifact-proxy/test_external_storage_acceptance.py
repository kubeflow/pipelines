# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Validate the external-storage fixture's broker and adoption boundaries."""

import base64
import json
import stat
import tempfile
import unittest
from contextlib import ExitStack
from io import BytesIO
from pathlib import Path
from unittest.mock import Mock, patch
from urllib.error import URLError

import artifact_authorization_acceptance as managed
import external_storage_acceptance as acceptance
import external_storage_broker as broker
from botocore.exceptions import ClientError

RESPONSE = b"""<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithWebIdentityResult><Credentials>
<AccessKeyId>fixture-access</AccessKeyId><SecretAccessKey>fixture-secret</SecretAccessKey>
<SessionToken>fixture-session</SessionToken><Expiration>2030-01-01T00:00:00Z</Expiration>
</Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>"""


class BrokerTest(unittest.TestCase):
    def test_real_sts_xml_writes_atomic_named_profile_and_safe_status(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            token = root / 'token'
            token.write_text('fixture-projected-token')
            with patch.object(
                broker.urllib.request, 'urlopen', return_value=BytesIO(RESPONSE)
            ) as request:
                broker.refresh(
                    'http://storage:8333',
                    'arn:aws:iam::role/team-a',
                    token,
                    root,
                    3,
                )
            posted = broker.urllib.parse.parse_qs(
                request.call_args.args[0].data.decode()
            )
            self.assertEqual(
                posted['WebIdentityToken'], ['fixture-projected-token']
            )
            self.assertEqual(posted['RoleArn'], ['arn:aws:iam::role/team-a'])
            self.assertNotIn('DurationSeconds', posted)
            credentials = (root / 'credentials').read_text()
            self.assertIn('[artifact-reader]', credentials)
            self.assertIn('aws_session_token = fixture-session', credentials)
            self.assertEqual(
                stat.S_IMODE((root / 'credentials').stat().st_mode), 0o600
            )
            self.assertFalse((root / 'credentials.tmp').exists())
            status = (root / 'status.json').read_text()
            self.assertEqual(json.loads(status)['generation'], 3)
            self.assertNotIn('fixture-access', status)
            self.assertNotIn('fixture-secret', status)
            self.assertNotIn('fixture-session', status)

    def test_broker_failure_preserves_previous_profile(self):
        for response in [
            URLError('unavailable'),
            RESPONSE.replace(
                b'<SessionToken>fixture-session</SessionToken>', b''
            ),
        ]:
            with (
                self.subTest(response_type=type(response).__name__),
                tempfile.TemporaryDirectory() as directory,
            ):
                root = Path(directory)
                token = root / 'token'
                token.write_text('fixture-token')
                (root / 'credentials').write_text('previous complete profile')
                kwargs = (
                    {'side_effect': response}
                    if isinstance(response, Exception)
                    else {'return_value': BytesIO(response)}
                )
                with patch.object(broker.urllib.request, 'urlopen', **kwargs):
                    with self.assertRaises(Exception):
                        broker.refresh(
                            'http://storage:8333',
                            'arn:aws:iam::role/team-a',
                            token,
                            root,
                            2,
                        )
                self.assertEqual(
                    (root / 'credentials').read_text(),
                    'previous complete profile',
                )
                self.assertFalse((root / 'status.json').exists())


class ApprovalTest(unittest.TestCase):
    def test_approved_template_uses_projected_identity_and_read_only_receiver_credentials(
        self,
    ):
        fixture = acceptance.ExternalAcceptance.__new__(
            acceptance.ExternalAcceptance
        )
        fixture.ui_image = 'kind-registry:5000/frontend:latest'
        fixture.external_endpoint = 'http://storage.fixture.svc:8333'
        with patch.object(
            acceptance,
            'get',
            return_value={'metadata': {'uid': 'actual-namespace-uid'}},
        ):
            approval = fixture.approval('team-a')
        self.assertEqual(approval['namespaceUID'], 'actual-namespace-uid')
        self.assertEqual(approval['storage']['bucket'], 'external.artifacts')
        self.assertEqual(
            approval['storage']['endpoint'], fixture.external_endpoint
        )
        self.assertEqual(
            approval['identity']['principal'], 'arn:aws:iam::role/team-a'
        )
        pod = approval['podTemplate']['spec']
        self.assertEqual(pod['serviceAccountName'], 'artifact-broker')
        projected = next(
            volume for volume in pod['volumes'] if volume['name'] == 'identity'
        )
        self.assertEqual(
            projected['projected']['sources'][0]['serviceAccountToken'][
                'audience'
            ],
            'artifact-storage',
        )
        receiver = next(
            container
            for container in pod['containers']
            if container['name'] == acceptance.RECEIVER
        )
        self.assertEqual(receiver['image'], fixture.ui_image)
        self.assertEqual(receiver['imagePullPolicy'], 'IfNotPresent')
        self.assertEqual(
            receiver['volumeMounts'],
            [
                {
                    'name': 'credentials',
                    'mountPath': '/var/run/kfp-artifacts',
                    'readOnly': True,
                }
            ],
        )
        self.assertFalse(
            any(
                mount['name'] == 'identity'
                for mount in receiver['volumeMounts']
            )
        )
        self.assertEqual(receiver['securityContext']['runAsUser'], 1000)

    def test_diagnostics_capture_pending_receiver_pull_failure(self):
        fixture = acceptance.ExternalAcceptance.__new__(
            acceptance.ExternalAcceptance
        )
        fixture.gateway_created = False
        fixture.namespace = 'kubeflow'
        fixture.tenants = ['team-a']
        pod = {
            'metadata': {'name': 'receiver-pending'},
            'spec': {
                'containers': [
                    {'name': acceptance.RECEIVER, 'imagePullPolicy': 'Always'}
                ]
            },
            'status': {
                'phase': 'Pending',
                'containerStatuses': [
                    {
                        'name': acceptance.RECEIVER,
                        'ready': False,
                        'state': {
                            'waiting': {
                                'reason': 'ImagePullBackOff',
                                'message': 'Registry unavailable: '
                                + 'x' * 2000,
                            }
                        },
                    }
                ],
            },
        }
        observed = {
            'namespace': {},
            'pods': {
                'items': [pod, {**pod, 'metadata': {'name': 'receiver-new'}}]
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            fixture.output = Path(directory)
            with (
                patch.object(acceptance.Acceptance, 'diagnostics'),
                patch.object(
                    acceptance,
                    'get',
                    side_effect=lambda resource, *args, **kwargs: observed.get(
                        resource
                    ),
                ),
                patch.object(acceptance, 'kubectl') as command,
            ):
                command.return_value.stdout = ''
                command.return_value.returncode = 0
                self.assertIsNone(fixture.pod('team-a'))
                fixture.diagnostics()
            result = json.loads(
                (fixture.output / 'external-diagnostics.json').read_text()
            )
        self.assertEqual(len(result['receiverStartup']['team-a']), 2)
        status = result['receiverStartup']['team-a'][0]
        self.assertEqual(status['pod'], 'receiver-pending')
        self.assertEqual(status['waitingReason'], 'ImagePullBackOff')
        self.assertEqual(status['imagePullPolicy'], 'Always')
        self.assertEqual(len(status['waitingMessage']), 1000)


class PendingDiagnosticsTest(unittest.TestCase):
    def test_ownership_shape_is_bounded_and_contains_no_names_or_uid_values(
        self,
    ):
        resource = {
            'metadata': {
                'name': 'private-resource-name',
                'annotations': {'private-annotation': 'secret-value'},
                'ownerReferences': [
                    {
                        'kind': 'Namespace',
                        'name': 'tenant',
                        'uid': 'current-uid',
                        'controller': True,
                    },
                    {
                        'kind': 'Namespace',
                        'name': 'other-tenant',
                        'uid': 'private-foreign-uid',
                    },
                    {
                        'kind': 'secret-value',
                        'name': 'private-owner',
                        'uid': 'private-uid',
                    },
                    *[
                        {'kind': 'Profile', 'uid': 'private-' + str(index)}
                        for index in range(5)
                    ],
                ],
            }
        }
        shape = acceptance.receiver_ownership_shape(resource, 'current-uid')
        self.assertTrue(shape['exists'])
        self.assertEqual(shape['ownerCount'], 8)
        self.assertEqual(len(shape['owners']), 5)
        self.assertEqual(
            shape['owners'][0],
            {'kind': 'Namespace', 'matchesNamespace': True, 'controller': True},
        )
        self.assertFalse(shape['owners'][1]['matchesNamespace'])
        self.assertEqual(shape['owners'][2]['kind'], 'Other')
        encoded = json.dumps(shape)
        for value in ['private', 'secret-value', 'current-uid', 'tenant']:
            self.assertNotIn(value, encoded)
        self.assertEqual(
            acceptance.receiver_ownership_shape(None, 'current-uid'),
            {'exists': False, 'ownerCount': 0, 'owners': []},
        )
        self.assertFalse(
            acceptance.receiver_ownership_shape(
                {'metadata': {'ownerReferences': [{'kind': 'Namespace'}]}}, None
            )['owners'][0]['matchesNamespace']
        )

    def test_keeps_only_authored_tenant_reasons_and_redacts_values(self):
        prefix = 'External artifact storage readiness for team-a is pending: '
        known = 'Storage policy source is missing or lacks its discovery label'
        lines = [
            'ordinary log containing secret-value',
            'External artifact storage authorization for team-a is pending: '
            + known
            + '.',
            'External artifact storage authorization for team-a is pending: secret-value.',
            'External artifact storage authorization for other-tenant is pending: '
            + known
            + '.',
            prefix
            + json.dumps(
                {
                    'stage': 'credential-wiring',
                    'paths': [
                        'Deployment.spec.template.spec.containers[1].env[2].valueFrom.fieldRef.apiVersion',
                        'metadata.annotations.secret-value',
                        'secret-value',
                        'x' * 193,
                        'Deployment.spec.containers[1].env',
                        'secret-value',
                    ],
                    'raw': 'secret-value',
                }
            ),
            prefix + '{invalid JSON with secret-value',
            prefix + json.dumps({'stage': [], 'paths': []}),
            prefix + json.dumps({'stage': {}, 'paths': []}),
            prefix + json.dumps({'stage': 'secret-value', 'paths': ['spec']}),
        ]
        result = acceptance.controller_pending_reasons(
            '\n'.join(lines), ['team-a']
        )
        self.assertEqual(result['team-a'][0], {'reason': known})
        self.assertEqual(
            result['team-a'][1],
            {
                'stage': 'credential-wiring',
                'paths': [
                    'Deployment.spec.template.spec.containers[1].env[2].valueFrom.fieldRef.apiVersion',
                    '<redacted-path>',
                    '<redacted-path>',
                    '<redacted-path>',
                    'Deployment.spec.containers[1].env',
                ],
            },
        )
        self.assertNotIn('secret-value', json.dumps(result))
        self.assertNotIn('other-tenant', json.dumps(result))

    def test_accepts_authored_resource_roots_and_redacted_map_key_markers(self):
        for path in [
            'Deployment.metadata.deletionTimestamp',
            'Deployment.spec.template.metadata.annotations[key]',
            'Deployment.spec.template.spec.containers[0].lifecycle.postStart',
            'Deployment.spec.template.spec.containers[1].securityContext.procMount',
            'Deployment.status.unavailableReplicas',
            'Service.spec.externalName',
            'NetworkPolicy.spec.ingress[0].from[0].namespaceSelector.matchLabels[key]',
        ]:
            self.assertEqual(acceptance.safe_readiness_path(path), path)
        self.assertEqual(
            acceptance.safe_readiness_path('Unknown.spec'), '<redacted-path>'
        )

    def test_bounds_input_and_retained_reasons(self):
        prefix = 'External artifact storage readiness for team-a is pending: '
        old = prefix + json.dumps({'stage': 'resource-deleting', 'paths': []})
        lines = [old, 'x' * acceptance.CONTROLLER_LOG_LIMIT]
        for index in range(20):
            lines.append(
                prefix
                + json.dumps(
                    {
                        'stage': 'credential-wiring',
                        'paths': [
                            'Deployment.spec.containers[' + str(index) + '].env'
                        ],
                    }
                )
            )
        result = acceptance.controller_pending_reasons(
            '\n'.join(lines), ['team-a']
        )['team-a']
        self.assertEqual(len(result), acceptance.PENDING_REASON_LIMIT)
        self.assertTrue(
            all(entry['stage'] == 'credential-wiring' for entry in result)
        )
        self.assertEqual(
            result[0]['paths'], ['Deployment.spec.containers[10].env']
        )
        self.assertEqual(
            result[-1]['paths'], ['Deployment.spec.containers[19].env']
        )


class AdoptionPolicyTest(unittest.TestCase):
    @staticmethod
    def fixture():
        fixture = acceptance.ExternalAcceptance.__new__(
            acceptance.ExternalAcceptance
        )
        fixture.namespace = 'kubeflow'
        fixture.run_id = 'fixture-policy'
        fixture.binding_source = 'fixture-bindings'
        fixture.template_source = 'fixture-template'
        fixture.external_endpoint = 'http://storage.fixture.svc:8333'
        fixture.ui_image = 'kind-registry:5000/frontend:latest'
        fixture.tenants = ['team-a', 'team-b']
        return fixture

    def test_policy_pins_central_sources_and_external_destination(self):
        fixture = self.fixture()
        policy = fixture.adoption_policy()
        self.assertEqual(
            policy['namespaceSelector'],
            {'matchLabels': {acceptance.CLASS_LABEL: fixture.run_id}},
        )
        self.assertEqual(
            policy['bindingsSource'],
            {
                'namespace': 'kubeflow',
                'configMap': 'fixture-bindings',
                'key': 'bindings.json',
            },
        )
        self.assertEqual(
            policy['receiverTemplateRef'],
            {
                'namespace': 'kubeflow',
                'configMap': 'fixture-template',
                'key': 'receiver.json',
            },
        )
        self.assertEqual(
            policy['allowedStorage'],
            {
                'source': 's3',
                'bucket': 'external.artifacts',
                'endpoint': fixture.external_endpoint,
                'region': 'us-east-1',
            },
        )
        self.assertTrue(policy['adoptExistingNamespaces'])
        self.assertTrue(policy['enrollNewNamespaces'])
        source = fixture.source_configmap(
            'fixture-bindings', 'bindings.json', {}
        )
        self.assertEqual(source['metadata']['namespace'], 'kubeflow')
        self.assertEqual(
            source['metadata']['labels'], {acceptance.SOURCE_LABEL: 'true'}
        )

    def test_shared_template_uses_real_pod_namespace_not_a_fixed_tenant(self):
        fixture = self.fixture()
        with patch.object(
            acceptance, 'get', return_value={'metadata': {'uid': 'team-a-uid'}}
        ):
            template = fixture.shared_template('team-a')
        self.assertNotIn('principal', template['identity'])
        self.assertEqual(template['identity']['profile'], 'artifact-reader')
        broker = next(
            container
            for container in template['podTemplate']['spec']['containers']
            if container['name'] == 'broker'
        )
        self.assertEqual(
            broker['env'],
            [
                {'name': 'STS_ENDPOINT', 'value': fixture.external_endpoint},
                {
                    'name': 'POD_NAMESPACE',
                    'valueFrom': {
                        'fieldRef': {'fieldPath': 'metadata.namespace'}
                    },
                },
                {
                    'name': 'STORAGE_ROLE',
                    'value': 'arn:aws:iam::role/$(POD_NAMESPACE)',
                },
            ],
        )
        self.assertNotIn('team-a', json.dumps(template))

    def test_publication_preserves_manual_approvals_and_existing_policy_data(
        self,
    ):
        fixture = self.fixture()
        original = {
            'metadata': {'labels': {'existing': 'label'}},
            'data': {
                'approvals.json': '{"unrelated":{"namespaceUID":"existing"}}',
                'unrelated': 'preserve',
                acceptance.POLICIES_KEY: json.dumps(
                    {'version': 1, 'policies': [{'name': 'existing'}]}
                ),
            },
        }
        fixture.approval_snapshot = original
        with (
            patch.object(acceptance, 'get') as read,
            patch.object(acceptance, 'apply') as write,
        ):
            read.side_effect = lambda *args, **kwargs: (
                write.call_args.args[0] if write.called else original
            )
            fixture.publish_adoption_policy()
        published = write.call_args.args[0]
        self.assertEqual(
            published['data']['approvals.json'],
            original['data']['approvals.json'],
        )
        self.assertEqual(published['data']['unrelated'], 'preserve')
        self.assertEqual(
            published['metadata']['labels'],
            {'existing': 'label', acceptance.SOURCE_LABEL: 'true'},
        )
        policies = json.loads(published['data'][acceptance.POLICIES_KEY])[
            'policies'
        ]
        self.assertEqual(policies[0], {'name': 'existing'})
        self.assertEqual(policies[1], fixture.adoption_policy())

    def test_manual_fixture_approval_is_rejected_even_if_preexisting(self):
        fixture = self.fixture()
        fixture.approval_snapshot = {
            'data': {'approvals.json': '{"team-a": {}}'}
        }
        with patch.object(
            acceptance, 'get', return_value=fixture.approval_snapshot
        ):
            with self.assertRaisesRegex(AssertionError, 'manual approval'):
                fixture.assert_no_manual_approvals()

    def test_forgery_probe_requires_actual_forbidden_central_writes(self):
        for returncode, stderr, succeeds in [
            (0, '', False),
            (1, 'connection refused', False),
            (1, 'Error from server (Forbidden): denied', True),
        ]:
            with self.subTest(returncode=returncode, stderr=stderr):
                fixture = self.fixture()
                fixture.users = {'team-a': 'alice@example.com'}
                fixture.trusted_bindings = {
                    'team-a': {'namespaceUID': 'uid-a', 'principal': 'role-a'},
                    'team-b': {'namespaceUID': 'uid-b', 'principal': 'role-b'},
                }
                fixture.receiver_template = {'identity': {}, 'podTemplate': {}}
                with ExitStack() as patches:
                    for name in [
                        'publish_bindings',
                        'wait_unavailable',
                        'wait_enrolled',
                        'check_external_access',
                    ]:
                        patches.enter_context(patch.object(fixture, name))
                    passed = patches.enter_context(
                        patch.object(fixture, 'passed')
                    )
                    read = patches.enter_context(
                        patch.object(acceptance, 'get')
                    )
                    read.side_effect = lambda *args, **kwargs: {
                        'data': {
                            'bindings.json': json.dumps(
                                fixture.trusted_bindings
                            )
                        }
                    }
                    command = patches.enter_context(
                        patch.object(acceptance, 'kubectl')
                    )
                    command.return_value.returncode = returncode
                    command.return_value.stderr = stderr
                    if succeeds:
                        fixture.check_adoption_forgery()
                        fixture.check_external_access.assert_called_once_with(
                            recovering=True
                        )
                        passed.assert_called_once()
                        self.assertEqual(
                            fixture.trusted_bindings['team-a']['principal'],
                            'role-a',
                        )
                        self.assertEqual(command.call_count, 4)
                    else:
                        with self.assertRaisesRegex(
                            AssertionError, 'Tenant could change'
                        ):
                            fixture.check_adoption_forgery()
                        passed.assert_not_called()

    def test_pending_record_cannot_hide_successful_artifact_response(self):
        fixture = self.fixture()
        fixture.timeout = 1
        with (
            patch.object(fixture, 'wait_state'),
            patch.object(
                fixture, 'external_request', return_value={'status': 200}
            ),
        ):
            with self.assertRaisesRegex(
                AssertionError, 'Unapproved external identity'
            ):
                fixture.wait_unavailable('team-a')

    def test_enrollment_drains_superseded_pods_and_requires_exact_traffic(self):
        exact = {'status': 200, 'body': 'team-a historical artifact'}
        for response, succeeds in [
            (exact, True),
            ({'status': 200, 'body': '<html>wrong page</html>'}, False),
            ({'status': 500, 'body': 'provider error'}, False),
        ]:
            with self.subTest(response=response):
                fixture = self.fixture()
                fixture.timeout = 2
                with ExitStack() as patches:
                    patches.enter_context(
                        patch.object(
                            fixture,
                            'wait_state',
                            return_value={
                                'adoptionPolicy': {'name': fixture.run_id}
                            },
                        )
                    )
                    patches.enter_context(
                        patch.object(
                            fixture, 'broker_status', return_value=True
                        )
                    )
                    drain = patches.enter_context(
                        patch.object(fixture, 'wait_receiver_pods')
                    )
                    patches.enter_context(
                        patch.object(fixture, 'capture_images')
                    )
                    ownership = patches.enter_context(
                        patch.object(fixture, 'assert_receiver_ownership')
                    )
                    patches.enter_context(
                        patch.object(fixture, 'assert_no_manual_approvals')
                    )
                    patches.enter_context(
                        patch.object(acceptance.time, 'sleep')
                    )
                    request = patches.enter_context(
                        patch.object(
                            fixture,
                            'external_request',
                            side_effect=[
                                {'status': 503, 'retryAfter': '5'},
                                response,
                            ],
                        )
                    )
                    if succeeds:
                        fixture.wait_enrolled('team-a', {'previous-stock-pod'})
                    else:
                        with self.assertRaisesRegex(
                            AssertionError, 'exact historical artifact'
                        ):
                            fixture.wait_enrolled(
                                'team-a', {'previous-stock-pod'}
                            )
                    drain.assert_called_once_with(
                        'team-a', fixture.ui_image, {'previous-stock-pod'}
                    )
                    self.assertEqual(request.call_count, 2)
                    ownership.assert_called_once_with('team-a')

    def test_enrollment_rejects_orphaned_or_foreign_receiver_resources(self):
        valid = {
            'metadata': {
                'ownerReferences': [
                    {
                        'kind': 'Namespace',
                        'uid': 'current-uid',
                        'controller': True,
                    }
                ]
            }
        }
        invalid = [
            None,
            {'metadata': {}},
            {
                'metadata': {
                    'ownerReferences': [
                        {'kind': 'Namespace', 'uid': 'foreign-uid'}
                    ]
                }
            },
            {
                'metadata': {
                    'ownerReferences': [
                        {'kind': 'Deployment', 'uid': 'current-uid'}
                    ]
                }
            },
        ]
        invalid.extend(
            [
                {
                    'metadata': {
                        'ownerReferences': [
                            {'kind': 'Namespace', 'uid': 'current-uid'}
                        ]
                    }
                },
                {
                    'metadata': {
                        'ownerReferences': [
                            {
                                'kind': 'Namespace',
                                'uid': 'current-uid',
                                'controller': False,
                            }
                        ]
                    }
                },
            ]
        )
        fixture = self.fixture()
        for kind in ['deployment', 'service', 'networkpolicy']:
            for bad_resource in invalid:
                with self.subTest(kind=kind, resource=bad_resource):

                    def read(resource, *args, **kwargs):
                        if resource == 'namespace':
                            return {'metadata': {'uid': 'current-uid'}}
                        return bad_resource if resource == kind else valid

                    with patch.object(acceptance, 'get', side_effect=read):
                        with self.assertRaisesRegex(
                            AssertionError, 'lost its Namespace ownership'
                        ):
                            fixture.assert_receiver_ownership('team-a')
        with patch.object(
            acceptance,
            'get',
            side_effect=lambda resource, *args, **kwargs: (
                {'metadata': {'uid': 'current-uid'}}
                if resource == 'namespace'
                else valid
            ),
        ) as read:
            fixture.assert_receiver_ownership('team-a')
        self.assertEqual(
            [call.args[0] for call in read.call_args_list],
            ['namespace', 'deployment', 'service', 'networkpolicy'],
        )


class ManagedSharedRootTest(unittest.TestCase):
    def fixture(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.tenants = ['team-a', 'team-b']
        fixture.run_id = 'fixture'
        fixture.shared_viewer = 'fixture-viewer@example.com'
        fixture.timeout = 1
        fixture.objects = []
        fixture.results = []
        fixture.s3 = Mock()
        fixture.namespace_storage_client = Mock()
        fixture.seed_shared_root()
        fixture.namespace_storage_client.return_value.list_objects_v2.return_value = {
            'Contents': [{'Key': fixture.shared}],
            'IsTruncated': False,
        }
        fixture.keys = {
            namespace: {
                'private': 'private-artifacts/' + namespace + '/model.txt'
            }
            for namespace in fixture.tenants
        }

        def request(namespace, key, **kwargs):
            if key == fixture.shared:
                return {'status': 200, 'body': 'tenant-written shared artifact'}
            if key == fixture.shared.rsplit('/', 1)[0] + '/':
                return {
                    'status': 200,
                    'body': '',
                }
            if (
                namespace == 'team-a'
                and key == fixture.keys['team-a']['private']
            ):
                return {'status': 200, 'body': 'team-a:private'}
            if namespace == 'team-a':
                return {'status': 500, 'body': 'Access Denied'}
            return {
                'status': 403,
                'body': f'User is not authorized to GET VIEWERS in namespace {namespace}: Forbidden',
            }

        fixture.request = Mock(side_effect=request)
        return fixture

    @staticmethod
    def review(**kwargs):
        attributes = kwargs['payload']['spec']['resourceAttributes']
        allowed = (
            attributes['namespace'] == 'team-a'
            and attributes['resource'] == 'viewers'
        )
        return Mock(stdout=json.dumps({'status': {'allowed': allowed}}))

    def test_shared_object_is_written_by_tenant_credentials_and_cleaned_up(
        self,
    ):
        fixture = self.fixture()
        fixture.namespace_storage_client.assert_called_once_with('team-b')
        fixture.namespace_storage_client.return_value.put_object.assert_called_once_with(
            Bucket='mlpipeline',
            Key=fixture.shared,
            Body=b'tenant-written shared artifact',
        )
        fixture.s3.put_object.assert_not_called()
        self.assertEqual(fixture.objects, [fixture.shared])

    def test_dedicated_reader_binding_only_uses_the_first_namespace_viewer_role(
        self,
    ):
        fixture = self.fixture()
        with patch.object(managed, 'apply') as apply:
            fixture.configure_shared_viewer()
        binding = apply.call_args.args[0]
        self.assertEqual(binding['kind'], 'RoleBinding')
        self.assertEqual(binding['metadata']['namespace'], 'team-a')
        self.assertEqual(binding['roleRef']['kind'], 'Role')
        self.assertEqual(
            binding['roleRef']['name'], 'artifact-acceptance-viewer'
        )
        self.assertEqual(
            binding['subjects'],
            [
                {
                    'apiGroup': 'rbac.authorization.k8s.io',
                    'kind': 'User',
                    'name': fixture.shared_viewer,
                }
            ],
        )

    def test_shared_viewer_read_and_real_storage_listing_have_distinct_proofs(
        self,
    ):
        fixture = self.fixture()
        with patch.object(
            managed,
            'kubectl',
            side_effect=lambda *args, **kwargs: self.review(**kwargs),
        ) as review:
            fixture.check_shared_access()
        self.assertEqual(review.call_count, 3)
        attributes = []
        for call in review.call_args_list:
            self.assertEqual(call.args, ('create', '-f', '-', '-o', 'json'))
            payload = call.kwargs['payload']
            self.assertEqual(payload['kind'], 'SubjectAccessReview')
            self.assertEqual(payload['spec']['user'], fixture.shared_viewer)
            attributes.append(payload['spec']['resourceAttributes'])
        self.assertEqual(
            attributes,
            [
                {
                    'namespace': 'team-a',
                    'group': 'kubeflow.org',
                    'resource': 'viewers',
                    'verb': 'get',
                },
                {
                    'namespace': 'team-a',
                    'group': '',
                    'resource': 'pods',
                    'verb': 'create',
                },
                {
                    'namespace': 'team-b',
                    'group': 'kubeflow.org',
                    'resource': 'viewers',
                    'verb': 'get',
                },
            ],
        )
        fixture.request.assert_any_call(
            'team-a', fixture.shared, user=fixture.shared_viewer, raw=False
        )
        fixture.request.assert_any_call(
            'team-a', fixture.shared, user=fixture.shared_viewer, raw=True
        )
        fixture.namespace_storage_client.assert_called_with('team-a')
        fixture.namespace_storage_client.return_value.list_objects_v2.assert_called_once_with(
            Bucket='mlpipeline', Prefix='shared/fixture/'
        )
        self.assertEqual(fixture.request.call_count, 5)
        for target in ['team-a', 'team-b']:
            fixture.request.assert_any_call(
                'team-a',
                fixture.keys[target]['private'],
                user=fixture.shared_viewer,
            )
        fixture.request.assert_any_call(
            'team-b',
            fixture.keys['team-b']['private'],
            user=fixture.shared_viewer,
        )
        self.assertEqual(len(fixture.results), 2)

    def test_privileged_viewer_cannot_satisfy_shared_access_proof(self):
        for checks in [[True, True], [True, False, True]]:
            fixture = self.fixture()
            results = [
                Mock(stdout=json.dumps({'status': {'allowed': allowed}}))
                for allowed in checks
            ]
            with (
                self.subTest(checks=checks),
                patch.object(managed, 'kubectl', side_effect=results),
            ):
                with self.assertRaisesRegex(AssertionError, 'unexpectedly has'):
                    fixture.check_shared_access()
            fixture.request.assert_not_called()
            self.assertEqual(fixture.results, [])

    def test_spa_failed_read_private_leak_or_namespace_claim_cannot_pass(
        self,
    ):
        for failed_call, response in [
            (
                4,
                {
                    'status': 403,
                    'body': 'User is not authorized in namespace team-b: Forbidden team-b:private',
                },
            ),
            (0, {'status': 200, 'body': '<html>SPA</html>'}),
            (2, {'status': 503, 'body': 'retry shortly'}),
            (3, {'status': 500, 'body': 'Access Denied team-b:private'}),
            (4, {'status': 200, 'body': 'team-b:private'}),
        ]:
            fixture = self.fixture()
            normal = fixture.request.side_effect
            calls = 0

            def request(*args, **kwargs):
                nonlocal calls
                current = calls
                calls += 1
                return (
                    response
                    if current == failed_call
                    else normal(*args, **kwargs)
                )

            fixture.request.side_effect = request
            with (
                self.subTest(failed_call=failed_call),
                patch.object(
                    managed,
                    'kubectl',
                    side_effect=lambda *args, **kwargs: self.review(**kwargs),
                ),
            ):
                with self.assertRaises(AssertionError):
                    fixture.check_shared_access()
            self.assertEqual(fixture.results, [])

    def test_empty_foreign_or_truncated_listing_cannot_satisfy_storage_proof(
        self,
    ):
        for listing in [
            {'Contents': [], 'IsTruncated': False},
            {
                'Contents': [{'Key': 'private-artifacts/team-b/secret.txt'}],
                'IsTruncated': False,
            },
            {
                'Contents': [
                    {'Key': 'shared/fixture/model.txt'},
                    {'Key': 'private-artifacts/team-b/secret.txt'},
                ],
                'IsTruncated': False,
            },
            {
                'Contents': [{'Key': 'shared/fixture/model.txt'}],
                'IsTruncated': True,
            },
        ]:
            fixture = self.fixture()
            fixture.namespace_storage_client.return_value.list_objects_v2.return_value = listing
            with (
                self.subTest(listing=listing),
                patch.object(
                    managed,
                    'kubectl',
                    side_effect=lambda *args, **kwargs: self.review(**kwargs),
                ),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'Shared storage listing'
                ):
                    fixture.check_shared_access()
            self.assertEqual(fixture.results, [])

    def test_denied_listing_cannot_satisfy_storage_proof(self):
        fixture = self.fixture()
        error = ClientError(
            {'Error': {'Code': 'AccessDenied'}}, 'ListObjectsV2'
        )
        fixture.namespace_storage_client.return_value.list_objects_v2.side_effect = error
        with patch.object(
            managed,
            'kubectl',
            side_effect=lambda *args, **kwargs: self.review(**kwargs),
        ):
            with self.assertRaises(ClientError) as raised:
                fixture.check_shared_access()
        self.assertEqual(
            raised.exception.response['Error']['Code'], 'AccessDenied'
        )
        self.assertEqual(fixture.results, [])


class ManagedExplicitStorageTest(unittest.TestCase):
    mode_name = 'KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE'

    def fixture(self):
        fixture, _ = ManagedLegacyRootTest().fixture()
        fixture.s3.delete_object.return_value = {}
        fixture.namespace = 'kubeflow'
        fixture.timeout = 1
        fixture.users = {
            name: name + '@example.com' for name in fixture.tenants
        }
        fixture.ui = 'http://default-ui'
        fixture.ui_container = 'ml-pipeline-ui'
        fixture.ui_image = 'candidate/frontend:ci'
        fixture.ui_template = {
            'spec': {
                'serviceAccountName': 'shared-ui-account',
                'containers': [
                    {
                        'name': fixture.ui_container,
                        'image': fixture.ui_image,
                        'env': [
                            {'name': 'ENABLE_AUTHZ', 'value': 'true'},
                            {
                                'name': 'API_SERVER_ADDRESS',
                                'value': 'http://real-api',
                            },
                            {'name': self.mode_name, 'value': 'auto'},
                        ],
                    }
                ],
            }
        }
        fixture.keys = {
            name: {'private': 'private-artifacts/' + name + '/model'}
            for name in fixture.tenants
        }
        fixture.probes = []
        fixture.forwarders = []
        fixture.controller_paused = False
        fixture.controller_replicas = None
        fixture.recovery_bucket_policy = None
        fixture.iam = None
        fixture.capture_images = Mock()
        fixture.port_forward = Mock(return_value='http://storage-probe')
        fixture.request = Mock(
            side_effect=lambda *args, **kwargs: self.response(
                fixture, *args, **kwargs
            )
        )
        return fixture

    def response(self, fixture, namespace, key, **kwargs):
        self.assertEqual(kwargs.get('base_url'), 'http://storage-probe')
        if key.startswith('artifacts'):
            return {'status': 403, 'body': 'Forbidden'}
        if (
            kwargs.get('user') == fixture.users['team-a']
            and namespace == 'team-b'
        ):
            return {
                'status': 403,
                'body': 'Denied in namespace team-b: viewers/get',
            }
        if namespace == 'team-a' and key == fixture.keys['team-b']['private']:
            return {'status': 500, 'body': 'Access Denied'}
        return {'status': 200, 'body': namespace + ':private'}

    def patches(self, stack, fixture, mutate_pod=None):
        applied = []
        stack.enter_context(
            patch.object(
                managed, 'apply', side_effect=lambda pod: applied.append(pod)
            )
        )

        def observed(kind, *args, **kwargs):
            if kind == 'namespace':
                return None
            self.assertEqual(kind, 'pod')
            pod = json.loads(json.dumps(applied[-1]))
            pod['status'] = {
                'containerStatuses': [
                    {'name': fixture.ui_container, 'state': {'running': {}}}
                ]
            }
            if mutate_pod:
                mutate_pod(pod)
            return pod

        stack.enter_context(patch.object(managed, 'get', side_effect=observed))
        stack.enter_context(
            patch.object(
                managed,
                'wait_for',
                side_effect=lambda description, predicate, timeout: predicate(),
            )
        )
        command = stack.enter_context(
            patch.object(managed, 'kubectl', return_value=Mock(returncode=0))
        )
        return applied, command

    def test_probe_uses_storage_mode_same_identity_and_separate_endpoint(self):
        fixture = self.fixture()
        original = json.loads(json.dumps(fixture.ui_template))
        with ExitStack() as stack:
            applied, command = self.patches(stack, fixture)
            fixture.check_explicit_storage()
        self.assertEqual(fixture.ui_template, original)
        pod = applied[0]
        self.assertEqual(pod['spec']['serviceAccountName'], 'shared-ui-account')
        self.assertEqual(pod['metadata']['labels']['app'], 'ml-pipeline-ui')
        self.assertEqual(pod['spec']['restartPolicy'], 'Never')
        self.assertEqual(
            pod['spec']['readinessGates'],
            [{'conditionType': 'artifact-acceptance/not-a-service-endpoint'}],
        )
        container = pod['spec']['containers'][0]
        self.assertEqual(container['image'], fixture.ui_image)
        self.assertEqual(container['imagePullPolicy'], 'IfNotPresent')
        self.assertEqual(
            [
                item
                for item in container['env']
                if item['name'] == self.mode_name
            ],
            [{'name': self.mode_name, 'value': 'storage'}],
        )
        self.assertEqual(
            [
                item
                for item in container['env']
                if item['name'] != self.mode_name
            ],
            original['spec']['containers'][0]['env'][:-1],
        )
        name = fixture.run_id + '-storage-ui'
        self.assertIn((fixture.namespace, name), fixture.probes)
        fixture.port_forward.assert_called_once_with('pod/' + name, 3000)
        self.assertEqual(fixture.request.call_count, 23)
        command.assert_any_call(
            'delete', 'pod', name, '-n', fixture.namespace, '--wait=false'
        )
        self.assertEqual(
            [r['name'] for r in fixture.results],
            [
                'explicit storage UI preserves private access and cross-tenant namespace isolation',
                'stock namespace credentials retain shared legacy workflow object and listing access',
                'explicit storage blocks ambiguous legacy workflow previews, downloads and directory listings',
            ],
        )

    def test_probe_rejects_default_endpoint_or_observed_auto_mode(self):
        for fault in [
            'default-url',
            'auto-mode',
            'different-image',
            'different-account',
        ]:
            with self.subTest(fault=fault), ExitStack() as stack:
                fixture = self.fixture()
                if fault == 'default-url':
                    fixture.port_forward.return_value = fixture.ui

                def mutate(pod):
                    if fault == 'different-image':
                        pod['spec']['containers'][0]['image'] = 'wrong-image'
                    if fault == 'different-account':
                        pod['spec']['serviceAccountName'] = 'wrong-account'
                    if fault == 'auto-mode':
                        for item in pod['spec']['containers'][0]['env']:
                            if item['name'] == self.mode_name:
                                item['value'] = 'auto'

                self.patches(stack, fixture, mutate)
                with self.assertRaises(AssertionError):
                    fixture.check_explicit_storage()
                fixture.request.assert_not_called()
                self.assertEqual(fixture.results, [])

    def test_probe_rejects_outage_wrong_bytes_and_denied_payload_leaks(self):
        for fault in [
            'own-503',
            'own-bytes',
            'foreign-503',
            'foreign-leak',
            'sar-leak',
            'sar-success',
            'legacy-root-leak',
            'legacy-503',
            'legacy-leak',
        ]:
            with self.subTest(fault=fault), ExitStack() as stack:
                fixture = self.fixture()

                def response(namespace, key, **kwargs):
                    normal = self.response(fixture, namespace, key, **kwargs)
                    if key.startswith('artifacts'):
                        if fault == 'legacy-root-leak' and key in (
                            'artifacts',
                            'artifacts/',
                        ):
                            return {
                                'status': 403,
                                'body': 'team-a:legacy-shared-root',
                            }
                        if fault == 'legacy-503':
                            return {'status': 503, 'body': 'pending'}
                        if fault == 'legacy-leak':
                            return {
                                'status': 403,
                                'body': 'team-a:legacy-shared-root',
                            }
                    elif kwargs.get('user'):
                        if fault == 'sar-leak':
                            return {'status': 403, 'body': 'team-b:private'}
                        if fault == 'sar-success':
                            return {'status': 200, 'body': 'allowed'}
                    elif key == fixture.keys['team-b']['private']:
                        if fault == 'foreign-503':
                            return {'status': 503, 'body': 'Access Denied'}
                        if fault == 'foreign-leak':
                            return {
                                'status': 500,
                                'body': 'Access Denied team-b:private',
                            }
                    else:
                        if fault == 'own-503':
                            return {'status': 503, 'body': 'team-a:private'}
                        if fault == 'own-bytes':
                            return {'status': 200, 'body': 'wrong'}
                    return normal

                fixture.request.side_effect = response
                self.patches(stack, fixture)
                with self.assertRaises(AssertionError):
                    fixture.check_explicit_storage()
                self.assertFalse(
                    any(
                        r['name'].startswith('explicit storage blocks')
                        for r in fixture.results
                    )
                )

    def test_failed_probe_is_registered_before_creation_and_cleaned_up(self):
        fixture = self.fixture()
        with patch.object(managed, 'apply', side_effect=RuntimeError('failed')):
            with self.assertRaises(RuntimeError):
                fixture.check_explicit_storage()
        self.assertEqual(fixture.probes, [('kubeflow', 'fixture-storage-ui')])
        with (
            patch.object(
                managed, 'kubectl', return_value=Mock(returncode=0)
            ) as command,
            patch.object(managed, 'get', return_value=None),
        ):
            self.assertEqual(fixture.cleanup(), [])
        command.assert_any_call(
            'delete',
            'pod',
            'fixture-storage-ui',
            '--ignore-not-found',
            '--wait=false',
            '-n',
            'kubeflow',
            optional=True,
            timeout=20,
        )

    def test_run_executes_storage_probe_after_first_access_and_keeps_failure(
        self,
    ):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture()
            fixture.output = Path(directory)
            fixture.phase = 'initialization'
            fixture.source_head = 'fixture-head'
            fixture.runtime_images = []
            fixture.diagnostics = Mock()
            fixture.cleanup = Mock(return_value=[])
            phases = []

            def operation(name):
                def run():
                    phases.append(name)
                    if name == 'check_explicit_storage':
                        raise AssertionError('storage guard failed')

                run.__name__ = name
                return run

            with ExitStack() as stack:
                for name in [
                    'prepare',
                    'pending_rollout',
                    'seed_existing_roots',
                    'check_access',
                    'check_explicit_storage',
                    'check_protocol',
                    'check_network_policy',
                    'mixed_versions',
                    'recover_bucket_policy',
                    'rotate_credentials',
                ]:
                    stack.enter_context(
                        patch.object(fixture, name, operation(name))
                    )
                self.assertEqual(fixture.run(), 1)
            self.assertEqual(
                phases,
                [
                    'prepare',
                    'pending_rollout',
                    'seed_existing_roots',
                    'check_access',
                    'check_explicit_storage',
                ],
            )
            fixture.cleanup.assert_called_once()
            report = json.loads((fixture.output / 'results.json').read_text())
            self.assertFalse(report['passed'])
            self.assertEqual(report['phase'], 'check_explicit_storage')
            self.assertEqual(report['failure'], 'storage guard failed')


class ManagedLegacyRootTest(unittest.TestCase):
    def fixture(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.tenants = ['team-a', 'team-b']
        fixture.run_id = 'fixture'
        fixture.objects = []
        fixture.results = []
        fixture.s3 = Mock()
        fixture.s3.meta.endpoint_url = 'http://127.0.0.1:8333'
        fixture.record = Mock(return_value={'version': 1, 'state': 'ready'})
        fixture.request = Mock(
            return_value={'status': 403, 'body': 'Forbidden'}
        )
        fixture.seed_legacy_shared_root()
        client = Mock()

        def read(**kwargs):
            namespace = next(
                owner
                for owner, key in fixture.legacy_keys.items()
                if key == kwargs['Key']
            )
            return {
                'Body': BytesIO((namespace + ':legacy-shared-root').encode())
            }

        client.get_object.side_effect = read
        client.list_objects_v2.side_effect = lambda **kwargs: {
            'Contents': [
                {'Key': key}
                for key in fixture.legacy_keys.values()
                if key.startswith(kwargs['Prefix'])
            ]
        }
        fixture.namespace_storage_client = Mock(return_value=client)
        return fixture, client

    def test_seeds_real_pre_namespace_workflow_paths_and_tracks_cleanup(self):
        fixture, _ = self.fixture()
        self.assertEqual(len(fixture.objects), 2)
        for namespace, key in fixture.legacy_keys.items():
            self.assertRegex(
                key,
                r'^artifacts/fixture-workflow-[ab]/2025/08/19/fixture-pod-[ab]/model.txt$',
            )
            self.assertIn(key, fixture.objects)
            fixture.s3.put_object.assert_any_call(
                Bucket='mlpipeline',
                Key=key,
                Body=(namespace + ':legacy-shared-root').encode(),
            )

    def test_proves_store_compatibility_and_denies_all_authenticated_routes(
        self,
    ):
        fixture, client = self.fixture()
        fixture.check_legacy_shared_root()
        self.assertEqual(client.get_object.call_count, 4)
        self.assertEqual(client.list_objects_v2.call_count, 4)
        self.assertEqual(fixture.request.call_count, 20)
        for namespace in fixture.tenants:
            fixture.namespace_storage_client.assert_any_call(namespace)
            for key in fixture.legacy_keys.values():
                fixture.request.assert_any_call(namespace, key)
                fixture.request.assert_any_call(namespace, key, raw=True)
                directory = key.rsplit('/', 1)[0] + '/'
                fixture.request.assert_any_call(namespace, directory, peek=4096)
                fixture.request.assert_any_call(namespace, directory, raw=True)
        self.assertEqual(len(fixture.results), 2)

    def test_legacy_guard_rejects_success_outage_and_content_leak(self):
        for response in [
            {'status': 200, 'body': 'team-b:legacy-shared-root'},
            {'status': 200, 'body': '<html>SPA</html>'},
            {'status': 404, 'body': 'missing'},
            {'status': 503, 'body': 'pending'},
            {'status': 403, 'body': 'team-a:legacy-shared-root'},
        ]:
            with self.subTest(response=response):
                fixture, _ = self.fixture()
                fixture.request.return_value = response
                with self.assertRaisesRegex(
                    AssertionError, 'shared legacy workflow'
                ):
                    fixture.check_legacy_shared_root()
                self.assertEqual(len(fixture.results), 1)

    def test_legacy_guard_requires_real_store_read_and_listing(self):
        for fault in ['wrong-body', 'missing-listing']:
            with self.subTest(fault=fault):
                fixture, client = self.fixture()
                if fault == 'wrong-body':
                    client.get_object.side_effect = lambda **kwargs: {
                        'Body': BytesIO(b'wrong')
                    }
                else:
                    client.list_objects_v2.side_effect = lambda **kwargs: {
                        'Contents': []
                    }
                with self.assertRaisesRegex(
                    AssertionError, 'real legacy fixture object'
                ):
                    fixture.check_legacy_shared_root()
                fixture.request.assert_not_called()
                self.assertEqual(fixture.results, [])

    def test_legacy_guard_does_not_mistake_pending_or_external_record_for_managed_ready(
        self,
    ):
        for record in [
            None,
            {'version': 1, 'state': 'pending'},
            {'version': 2, 'state': 'ready'},
        ]:
            with self.subTest(record=record):
                fixture, _ = self.fixture()
                fixture.record.return_value = record
                with self.assertRaisesRegex(
                    AssertionError, 'actually ready managed-v1'
                ):
                    fixture.check_legacy_shared_root()
                fixture.namespace_storage_client.assert_not_called()
                fixture.request.assert_not_called()

    def test_namespace_store_probe_uses_tenant_secret_not_admin_session(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.s3 = Mock()
        fixture.s3.meta.endpoint_url = 'http://127.0.0.1:8333'
        secret = {
            'data': {
                'accesskey': base64.b64encode(b'tenant-key').decode(),
                'secretkey': base64.b64encode(b'tenant-secret').decode(),
            }
        }
        with (
            patch.object(managed, 'get', return_value=secret) as read,
            patch.object(managed.boto3, 'Session') as session,
        ):
            fixture.namespace_storage_client('team-a')
        read.assert_called_once_with(
            'secret', managed.CREDENTIAL_SECRET, 'team-a'
        )
        session.assert_called_once_with(
            aws_access_key_id='tenant-key',
            aws_secret_access_key='tenant-secret',
            region_name='us-east-1',
        )
        self.assertEqual(
            session.return_value.client.call_args.kwargs['endpoint_url'],
            'http://127.0.0.1:8333',
        )

    def test_authenticated_request_builds_real_preview_and_raw_routes(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.ui = 'http://ui'
        fixture.users = {'team-a': 'alice'}
        for raw in [False, True]:
            response = BytesIO(b'Forbidden')
            response.status = 403
            response.headers = {}
            with (
                self.subTest(raw=raw),
                patch.object(
                    managed.urllib.request, 'urlopen', return_value=response
                ) as fetch,
            ):
                result = fixture.request(
                    'team-a', 'artifacts/workflow/model.txt', raw=raw, peek=4096
                )
            sent = fetch.call_args.args[0]
            parsed = managed.urllib.parse.urlparse(sent.full_url)
            query = managed.urllib.parse.parse_qs(parsed.query)
            self.assertEqual(sent.get_header('Kubeflow-userid'), 'alice')
            self.assertEqual(query['namespace'], ['team-a'])
            self.assertEqual(query['peek'], ['4096'])
            self.assertEqual(
                parsed.path,
                '/artifacts/minio/mlpipeline/artifacts/workflow/model.txt'
                if raw
                else '/artifacts/get',
            )
            if not raw:
                self.assertEqual(query['key'], ['artifacts/workflow/model.txt'])
            self.assertEqual(result['status'], 403)


class SharedConfigurationRecoveryTest(unittest.TestCase):
    def fixture(self):
        fixture = AdoptionPolicyTest.fixture()
        fixture.timeout = 6
        fixture.objects = []
        fixture.bundled_history = {
            'team-a': {
                'key': 'private-artifacts/team-a/before-external.txt',
                'body': 'team-a bundled history before external adoption',
            },
        }
        fixture.s3 = Mock()
        fixture.approval_snapshot = {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'metadata': {
                'name': acceptance.APPROVALS,
                'namespace': 'kubeflow',
                'resourceVersion': 'old-version',
                'uid': 'old-config-uid',
            },
            'data': {'unrelated': 'preserved'},
        }
        return fixture

    @staticmethod
    def record_secret(uid='record-uid', version=1, state='ready'):
        return {
            'metadata': {
                'uid': uid,
                'ownerReferences': [
                    {'kind': 'Namespace', 'uid': 'namespace-uid'}
                ],
            },
            'data': {
                'authorization.json': base64.b64encode(
                    json.dumps(
                        {
                            'namespace': 'team-a',
                            'namespaceUID': 'namespace-uid',
                            'version': version,
                            'state': state,
                        }
                    ).encode()
                ).decode()
            },
        }

    @staticmethod
    def managed_resources():
        resources = []
        for kind, name in [
            ('Secret', 'mlpipeline-minio-artifact'),
            ('ConfigMap', 'kfp-launcher'),
            ('ConfigMap', 'artifact-repositories'),
            ('Deployment', acceptance.RECEIVER),
            ('Service', acceptance.RECEIVER),
            ('NetworkPolicy', acceptance.RECEIVER),
            ('ServiceAccount', 'executor'),
            ('Secret', 'executor-token'),
        ]:
            resource = {
                'kind': kind,
                'metadata': {
                    'name': name,
                    'namespace': 'team-a',
                    'uid': kind + '-' + name,
                    'ownerReferences': [
                        {
                            'apiVersion': 'v1',
                            'kind': 'Namespace',
                            'name': 'team-a',
                            'uid': 'namespace-uid',
                            'controller': True,
                        }
                    ],
                },
            }
            if kind == 'Secret':
                resource['data'] = {'token': 'fixture-credential-bytes'}
            resources.append(resource)
        return resources

    def managed_snapshot(self, fixture, resources):
        with patch.object(
            acceptance,
            'get',
            side_effect=[
                {'metadata': {'uid': 'namespace-uid'}},
                {'items': resources},
            ],
        ):
            return fixture.managed_children_snapshot('team-a')

    def test_managed_snapshot_covers_optional_bootstrap_without_secret_bytes(
        self,
    ):
        fixture = self.fixture()
        resources = self.managed_resources()
        snapshot = self.managed_snapshot(fixture, resources)
        self.assertEqual(len(snapshot), len(resources))
        self.assertIn(('ServiceAccount', 'executor'), snapshot)
        self.assertIn(('Secret', 'executor-token'), snapshot)
        self.assertNotIn('fixture-credential-bytes', repr(snapshot))
        self.assertEqual(
            len(snapshot[('Secret', 'executor-token')]['secretDataHash']), 64
        )
        with patch.object(acceptance, 'get', return_value={'items': resources}):
            fixture.assert_managed_children('team-a', snapshot)

    def test_managed_snapshot_rejects_missing_or_foreign_baseline_ownership(
        self,
    ):
        fixture = self.fixture()
        for missing in [True, False]:
            resources = self.managed_resources()
            if missing:
                resources[0]['metadata']['ownerReferences'] = []
            else:
                resources[0]['metadata']['ownerReferences'].append(
                    {
                        'kind': 'Namespace',
                        'uid': 'another-namespace',
                    }
                )
            with (
                self.subTest(missing=missing),
                self.assertRaisesRegex(
                    AssertionError,
                    'namespace-owned children|exclusive current Namespace',
                ),
            ):
                self.managed_snapshot(fixture, resources)

    def test_managed_recovery_rejects_loss_replacement_and_secret_mutation_immediately(
        self,
    ):
        fixture = self.fixture()
        for fault in [
            'deleted',
            'replaced',
            'detached',
            'foreign',
            'secret-data',
        ]:
            resources = self.managed_resources()
            snapshot = self.managed_snapshot(fixture, resources)
            resource = resources[-1]
            if fault == 'deleted':
                resources.pop()
            elif fault == 'replaced':
                resource['metadata']['uid'] = 'replacement-uid'
            elif fault == 'detached':
                resource['metadata']['ownerReferences'] = []
            elif fault == 'foreign':
                resource['metadata']['ownerReferences'][0]['uid'] = (
                    'foreign-uid'
                )
            else:
                resource['data']['token'] = 'changed-credential-bytes'
            with (
                self.subTest(fault=fault),
                patch.object(
                    acceptance, 'get', return_value={'items': resources}
                ) as read,
                self.assertRaisesRegex(
                    AssertionError,
                    'removed, replaced or detached|Secret data changed',
                ),
            ):
                fixture.assert_managed_children('team-a', snapshot)
            read.assert_called_once()

    def test_bundled_recovery_does_not_wait_through_child_ownership_loss(self):
        fixture = self.fixture()
        with (
            patch.object(
                fixture,
                'assert_managed_children',
                side_effect=AssertionError('detached child'),
            ) as ownership,
            patch.object(fixture, 'authorization_snapshot') as record,
            patch.object(managed.time, 'sleep') as sleep,
        ):
            with self.assertRaisesRegex(AssertionError, 'detached child'):
                fixture.wait_bundled_history(
                    'team-a', 'record-uid', {'baseline': {}}
                )
        ownership.assert_called_once_with('team-a', {'baseline': {}})
        record.assert_not_called()
        sleep.assert_not_called()

    def test_recovery_rejects_deleted_or_recreated_record_secret(self):
        fixture = self.fixture()
        for secret in [None, self.record_secret('replacement-uid')]:
            with (
                self.subTest(secret=secret),
                patch.object(acceptance, 'get', return_value=secret),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'deleted or replaced'
                ):
                    fixture.authorization_snapshot('team-a', 'record-uid')
        original = self.record_secret()
        with patch.object(
            acceptance,
            'get',
            side_effect=[original, {'metadata': {'uid': 'namespace-uid'}}],
        ):
            self.assertEqual(
                fixture.authorization_snapshot('team-a', 'record-uid')[0],
                original,
            )

    def test_recovery_record_requires_current_namespace_owner(self):
        fixture = self.fixture()
        secret = self.record_secret()
        secret['metadata']['ownerReferences'] = []
        with patch.object(
            acceptance,
            'get',
            side_effect=[secret, {'metadata': {'uid': 'namespace-uid'}}],
        ):
            with self.assertRaisesRegex(AssertionError, 'actual namespace'):
                fixture.authorization_snapshot('team-a', 'record-uid')

    def test_bundled_recovery_requires_exact_bytes_and_retry_protocol(self):
        for response, message in [
            (
                {'status': 200, 'body': '<html>not artifact</html>'},
                'exact bundled historical artifact',
            ),
            (
                {'status': 403, 'body': 'denied'},
                'exact bundled historical artifact',
            ),
            ({'status': 503, 'retryAfter': '60'}, 'not retryable'),
        ]:
            with self.subTest(response=response), ExitStack() as patches:
                fixture = self.fixture()
                patches.enter_context(
                    patch.object(
                        fixture,
                        'authorization_snapshot',
                        return_value=(
                            self.record_secret(),
                            {'version': 1, 'state': 'ready'},
                        ),
                    )
                )
                request = patches.enter_context(
                    patch.object(fixture, 'request', return_value=response)
                )
                with self.assertRaisesRegex(AssertionError, message):
                    fixture.wait_bundled_history('team-a', 'record-uid')
                request.assert_called_once()

    def test_bundled_recovery_waits_for_managed_record_and_only_retries_503(
        self,
    ):
        fixture = self.fixture()
        secret = self.record_secret()
        with (
            patch.object(
                fixture,
                'authorization_snapshot',
                side_effect=[
                    (secret, {'version': 2, 'state': 'pending'}),
                    (secret, {'version': 1, 'state': 'ready'}),
                    (secret, {'version': 1, 'state': 'ready'}),
                ],
            ) as record,
            patch.object(
                fixture,
                'request',
                side_effect=[
                    {'status': 503, 'retryAfter': '5'},
                    {
                        'status': 200,
                        'body': fixture.bundled_history['team-a']['body'],
                    },
                ],
            ) as request,
            patch.object(managed.time, 'sleep'),
        ):
            self.assertEqual(
                fixture.wait_bundled_history('team-a', 'record-uid'), secret
            )
        self.assertEqual(record.call_count, 3)
        self.assertTrue(
            all(
                call.args == ('team-a', 'record-uid')
                for call in record.call_args_list
            )
        )
        self.assertEqual(request.call_count, 2)

    def test_malformed_config_must_deny_real_request_and_never_delete_secret(
        self,
    ):
        for status in [200, 503]:
            with self.subTest(status=status), ExitStack() as patches:
                fixture = self.fixture()
                children = {('Secret', 'fixture'): {'uid': 'fixture-uid'}}
                patches.enter_context(
                    patch.object(
                        fixture,
                        'managed_children_snapshot',
                        return_value=children,
                    )
                )
                ownership = patches.enter_context(
                    patch.object(fixture, 'assert_managed_children')
                )
                patches.enter_context(
                    patch.object(
                        fixture,
                        'wait_bundled_history',
                        return_value=self.record_secret(),
                    )
                )
                pending = patches.enter_context(
                    patch.object(fixture, 'wait_authorization')
                )
                request = patches.enter_context(
                    patch.object(
                        fixture,
                        'request',
                        return_value={'status': status, 'retryAfter': '5'},
                    )
                )
                passed = patches.enter_context(patch.object(fixture, 'passed'))
                write = patches.enter_context(patch.object(acceptance, 'apply'))
                command = patches.enter_context(
                    patch.object(acceptance, 'kubectl')
                )
                if status == 200:
                    with self.assertRaisesRegex(
                        AssertionError,
                        'did not block the real bundled artifact',
                    ):
                        fixture.check_unadopted_config_recovery('team-a')
                    passed.assert_not_called()
                else:
                    fixture.check_unadopted_config_recovery('team-a')
                    self.assertEqual(request.call_count, 2)
                    self.assertEqual(passed.call_count, 2)
                    self.assertEqual(ownership.call_count, 3)
                    self.assertTrue(
                        all(
                            call.args == ('team-a', children)
                            for call in ownership.call_args_list
                        )
                    )
                    self.assertEqual(pending.call_count, 2)
                    self.assertTrue(
                        all(
                            call.args == ('team-a', 2, 'pending', 'record-uid')
                            for call in pending.call_args_list
                        )
                    )
                    self.assertEqual(
                        write.call_args.args[0]['data'],
                        fixture.approval_snapshot['data'],
                    )
                    self.assertNotIn('uid', write.call_args.args[0]['metadata'])
                    command.assert_called_once_with(
                        'delete',
                        'configmap',
                        acceptance.APPROVALS,
                        '-n',
                        'kubeflow',
                        '--ignore-not-found',
                        timeout=20,
                    )
                self.assertTrue(fixture.approvals_changed)
                self.assertEqual(
                    fixture.s3.put_object.call_args.kwargs['Bucket'],
                    'mlpipeline',
                )
                self.assertEqual(
                    fixture.objects, [fixture.bundled_history['team-a']['key']]
                )

    def test_external_cutover_requires_bundled_denial_and_external_exact_bytes(
        self,
    ):
        for status, external_body, succeeds in [
            (403, 'team-a historical artifact', True),
            (200, 'team-a historical artifact', False),
            (503, 'team-a historical artifact', False),
            (403, '<html>wrong response</html>', False),
        ]:
            with (
                self.subTest(status=status, external_body=external_body),
                ExitStack() as patches,
            ):
                fixture = self.fixture()
                patches.enter_context(
                    patch.object(
                        fixture,
                        'request',
                        return_value={'status': status, 'body': 'denied'},
                    )
                )
                patches.enter_context(
                    patch.object(
                        fixture,
                        'external_request',
                        return_value={'status': 200, 'body': external_body},
                    )
                )
                passed = patches.enter_context(patch.object(fixture, 'passed'))
                if succeeds:
                    fixture.check_bundled_history_cutover('team-a')
                    passed.assert_called_once()
                else:
                    with self.assertRaises(AssertionError):
                        fixture.check_bundled_history_cutover('team-a')
                    passed.assert_not_called()

    def test_cutover_rechecks_both_destinations_after_retryable_pending(self):
        fixture = self.fixture()
        with (
            patch.object(
                fixture,
                'request',
                side_effect=[
                    {'status': 503, 'retryAfter': '5'},
                    {'status': 403, 'body': 'denied'},
                    {'status': 403, 'body': 'denied'},
                ],
            ) as bundled,
            patch.object(
                fixture,
                'external_request',
                side_effect=[
                    {'status': 503, 'retryAfter': '5'},
                    {'status': 200, 'body': 'team-a historical artifact'},
                ],
            ) as external,
            patch.object(fixture, 'passed') as passed,
            patch.object(managed.time, 'sleep'),
        ):
            fixture.check_bundled_history_cutover('team-a')
        self.assertEqual(bundled.call_count, 3)
        self.assertEqual(external.call_count, 2)
        passed.assert_called_once()

    def test_adopted_recovery_preserves_secret_identity_and_rechecks_isolation(
        self,
    ):
        fixture = self.fixture()
        original = fixture.approval_snapshot
        with ExitStack() as patches:
            patches.enter_context(
                patch.object(acceptance, 'get', return_value=original)
            )
            patches.enter_context(
                patch.object(
                    fixture,
                    'authorization_snapshot',
                    side_effect=lambda namespace: (
                        self.record_secret(namespace + '-record'),
                        {},
                    ),
                )
            )
            write = patches.enter_context(
                patch.object(fixture, 'apply_approval_configuration')
            )
            record = patches.enter_context(
                patch.object(fixture, 'wait_authorization')
            )
            deny = patches.enter_context(
                patch.object(fixture, 'wait_unavailable')
            )
            enroll = patches.enter_context(
                patch.object(fixture, 'wait_enrolled')
            )
            access = patches.enter_context(
                patch.object(fixture, 'check_external_access')
            )
            passed = patches.enter_context(patch.object(fixture, 'passed'))
            fixture.check_adopted_config_recovery()
        self.assertEqual(
            write.call_args_list[0].args[0]['data'][acceptance.POLICIES_KEY],
            '{malformed',
        )
        self.assertEqual(write.call_args_list[1].args[0], original)
        self.assertEqual(
            [call.args for call in record.call_args_list],
            [
                ('team-a', 2, 'pending', 'team-a-record'),
                ('team-b', 2, 'pending', 'team-b-record'),
                ('team-a', 2, 'ready', 'team-a-record'),
                ('team-b', 2, 'ready', 'team-b-record'),
            ],
        )
        self.assertEqual(deny.call_count, 2)
        self.assertEqual(enroll.call_count, 2)
        access.assert_called_once_with(recovering=True)
        passed.assert_called_once()


class ExternalAccessRecoveryTest(unittest.TestCase):
    def setUp(self):
        self.fixture = AdoptionPolicyTest.fixture()
        self.fixture.timeout = 6
        self.fixture.users = {
            'team-a': 'alice@example.com',
            'team-b': 'bob@example.com',
        }
        self.fixture.prefixes = {
            'team-a': 'historical/a/',
            'team-b': 'historical/b/',
        }

    @staticmethod
    def responses():
        return [
            response
            for namespace in ['team-a', 'team-b']
            for response in [
                {'status': 200, 'body': namespace + ' historical artifact'},
                {'status': 500, 'body': 'provider denied'},
                {'status': 403, 'body': 'SAR denied'},
            ]
        ]

    def listing(self, namespace, prefix):
        if prefix == self.fixture.prefixes[namespace] + 'directory/':
            return {'objects': [prefix + 'part.txt']}
        return {'denied': True, 'code': 'AccessDenied'}

    def test_policy_recovery_replays_full_probe_after_ready_then_pending(self):
        responses = [self.responses()[0], {'status': 503, 'retryAfter': '5'}]
        responses += self.responses()
        with (
            patch.object(
                self.fixture, 'external_request', side_effect=responses
            ) as request,
            patch.object(
                self.fixture, 'list_prefix', side_effect=self.listing
            ) as listing,
            patch.object(self.fixture, 'passed') as passed,
            patch.object(managed.time, 'sleep') as sleep,
            patch.object(managed.time, 'monotonic', side_effect=range(4)),
        ):
            self.fixture.check_external_access(recovering=True)
        self.assertEqual(request.call_count, 8)
        self.assertEqual(listing.call_count, 6)
        sleep.assert_called_once_with(2)
        passed.assert_called_once()

    def test_policy_recovery_rejects_security_and_protocol_failures_immediately(
        self,
    ):
        own, denied, _ = self.responses()[:3]
        cases = [
            (
                [own, {'status': 200, 'body': 'team-b historical artifact'}],
                AssertionError,
                'did not deny another tenant object',
            ),
            (
                [own, {'status': 503, 'retryAfter': None}],
                AssertionError,
                'not retryable',
            ),
            (
                [{'status': 503, 'retryAfter': '1'}],
                AssertionError,
                'not retryable',
            ),
            (
                [{'status': 200, 'body': 'incorrect artifact'}],
                AssertionError,
                'historical external artifact',
            ),
            (
                [own, denied, {'status': 200, 'body': 'impersonated'}],
                AssertionError,
                'SAR allowed namespace impersonation',
            ),
            ([URLError('connection reset')], URLError, 'connection reset'),
        ]
        for responses, error, message in cases:
            with (
                self.subTest(message=message),
                patch.object(
                    self.fixture, 'external_request', side_effect=responses
                ) as request,
                patch.object(self.fixture, 'passed') as passed,
                patch.object(managed.time, 'sleep') as sleep,
            ):
                with self.assertRaisesRegex(error, message):
                    self.fixture.check_external_access(recovering=True)
                self.assertEqual(request.call_count, len(responses))
                sleep.assert_not_called()
                passed.assert_not_called()

    def test_policy_recovery_keeps_directory_denials_strict(self):
        with (
            patch.object(
                self.fixture, 'external_request', side_effect=self.responses()
            ),
            patch.object(
                self.fixture,
                'list_prefix',
                side_effect=[
                    self.listing('team-a', 'historical/a/directory/'),
                    {'objects': ['historical/b/model.txt']},
                ],
            ),
            patch.object(self.fixture, 'passed') as passed,
            patch.object(managed.time, 'sleep') as sleep,
        ):
            with self.assertRaisesRegex(
                AssertionError, 'foreign or bucket-wide listing'
            ):
                self.fixture.check_external_access(recovering=True)
            sleep.assert_not_called()
            passed.assert_not_called()

    def test_policy_recovery_uses_one_bounded_deadline(self):
        self.fixture.timeout = 3
        with (
            patch.object(
                self.fixture,
                'external_request',
                return_value={'status': 503, 'retryAfter': '5'},
            ) as request,
            patch.object(self.fixture, 'passed') as passed,
            patch.object(managed.time, 'sleep'),
            patch.object(
                managed.time, 'monotonic', side_effect=[0, 0, 1, 2, 3]
            ),
        ):
            with self.assertRaisesRegex(
                AssertionError,
                'Timed out waiting for external access after policy restoration',
            ):
                self.fixture.check_external_access(recovering=True)
        self.assertEqual(request.call_count, 3)
        passed.assert_not_called()

    def test_steady_state_does_not_retry_pending(self):
        with (
            patch.object(
                self.fixture,
                'external_request',
                return_value={'status': 503, 'retryAfter': '5'},
            ) as request,
            patch.object(self.fixture, 'passed') as passed,
            patch.object(managed.time, 'sleep') as sleep,
        ):
            with self.assertRaisesRegex(
                AssertionError, 'historical external artifact'
            ):
                self.fixture.check_external_access()
        request.assert_called_once()
        passed.assert_not_called()
        sleep.assert_not_called()

    def test_uid_and_source_restoration_enable_recovery(self):
        self.fixture.trusted_bindings = {
            'team-a': {'namespaceUID': 'current-uid'}
        }
        for operation, expected_calls in [
            ('check_adoption_uid_mismatch', 1),
            ('check_adoption_revocation', 2),
        ]:
            with self.subTest(operation=operation), ExitStack() as patches:
                for method in [
                    'publish_bindings',
                    'publish_template',
                    'wait_unavailable',
                    'wait_enrolled',
                    'assert_no_manual_approvals',
                    'check_adopted_config_recovery',
                    'passed',
                ]:
                    patches.enter_context(patch.object(self.fixture, method))
                patches.enter_context(patch.object(acceptance, 'kubectl'))
                access = patches.enter_context(
                    patch.object(self.fixture, 'check_external_access')
                )
                getattr(self.fixture, operation)()
                self.assertEqual(access.call_count, expected_calls)
                for call in access.call_args_list:
                    self.assertEqual(call.args, ())
                    self.assertEqual(call.kwargs, {'recovering': True})
        self.assertEqual(
            self.fixture.trusted_bindings['team-a']['namespaceUID'],
            'current-uid',
        )


class AcceptanceSequenceTest(unittest.TestCase):
    def test_run_requires_foreign_role_assumption_denial_and_cleans_up(self):
        for sts_response, expected_exit in [('403', 0), ('ALLOWED', 1)]:
            with (
                self.subTest(sts_response=sts_response),
                tempfile.TemporaryDirectory() as directory,
            ):
                fixture = acceptance.ExternalAcceptance.__new__(
                    acceptance.ExternalAcceptance
                )
                fixture.output = Path(directory)
                fixture.source_head = 'fixture'
                fixture.phase = 'initialization'
                fixture.results = []
                fixture.runtime_images = []
                fixture.tenants = ['team-a', 'team-b']
                fixture.external_endpoint = 'http://storage.fixture.svc:8333'
                with ExitStack() as patches:
                    patches.enter_context(patch('builtins.print'))
                    for operation in [
                        'prepare',
                        'prepare_external_store',
                        'prepare_profiles',
                        'check_external_access',
                        'check_rotation',
                        'check_expiry_and_recovery',
                        'check_adoption_uid_mismatch',
                        'check_adoption_forgery',
                        'check_adoption_revocation',
                    ]:
                        patches.enter_context(
                            patch.object(fixture, operation, lambda: None)
                        )
                    patches.enter_context(patch.object(fixture, 'diagnostics'))
                    cleanup = patches.enter_context(
                        patch.object(fixture, 'cleanup', return_value=[])
                    )
                    patches.enter_context(
                        patch.object(
                            fixture,
                            'pod',
                            return_value={'metadata': {'name': 'receiver'}},
                        )
                    )
                    command = patches.enter_context(
                        patch.object(acceptance, 'kubectl')
                    )
                    command.return_value.stdout = sts_response + '\n'
                    result = fixture.run()
                self.assertEqual(result, expected_exit)
                command.assert_called_once()
                self.assertEqual(
                    command.call_args.args[-1], 'arn:aws:iam::role/team-b'
                )
                cleanup.assert_called_once()
                report = json.loads(
                    (fixture.output / 'results.json').read_text()
                )
                self.assertEqual(report['passed'], expected_exit == 0)
                self.assertEqual(report['cleanupErrors'], [])
                if expected_exit:
                    self.assertEqual(report['phase'], 'check_role_trust')
                    self.assertIn(
                        'could assume another tenant storage role',
                        report['failure'],
                    )
                else:
                    self.assertTrue(
                        any(
                            'cannot assume the other tenant storage role'
                            in check['name']
                            for check in report['checks']
                        )
                    )

    def test_run_executes_each_adoption_security_probe_and_preserves_failure(
        self,
    ):
        phases = [
            'prepare',
            'prepare_external_store',
            'prepare_profiles',
            'check_external_access',
            'check_role_trust',
            'check_rotation',
            'check_expiry_and_recovery',
            'check_adoption_uid_mismatch',
            'check_adoption_forgery',
            'check_adoption_revocation',
        ]
        for failed_phase in phases[-3:]:
            with (
                self.subTest(failed_phase=failed_phase),
                tempfile.TemporaryDirectory() as directory,
            ):
                fixture = acceptance.ExternalAcceptance.__new__(
                    acceptance.ExternalAcceptance
                )
                fixture.output = Path(directory)
                fixture.source_head = 'fixture'
                fixture.phase = 'initialization'
                fixture.results = []
                fixture.runtime_images = []
                executed = []

                def operation(name):

                    def execute():
                        executed.append(name)
                        if name == failed_phase:
                            raise AssertionError(
                                'Adoption security probe rejected unsafe state'
                            )

                    execute.__name__ = name
                    return execute

                with ExitStack() as patches:
                    patches.enter_context(patch('builtins.print'))
                    for phase in phases:
                        patches.enter_context(
                            patch.object(fixture, phase, operation(phase))
                        )
                    patches.enter_context(patch.object(fixture, 'diagnostics'))
                    cleanup = patches.enter_context(
                        patch.object(fixture, 'cleanup', return_value=[])
                    )
                    self.assertEqual(fixture.run(), 1)
                self.assertEqual(
                    executed, phases[: phases.index(failed_phase) + 1]
                )
                cleanup.assert_called_once()
                report = json.loads(
                    (fixture.output / 'results.json').read_text()
                )
                self.assertFalse(report['passed'])
                self.assertEqual(report['phase'], failed_phase)
                self.assertEqual(
                    report['failure'],
                    'Adoption security probe rejected unsafe state',
                )
                self.assertEqual(report['cleanupErrors'], [])


class RolloutConvergenceTest(unittest.TestCase):
    @staticmethod
    def pod(uid='replacement', image='current', ready=True, deleting=False):
        return {
            'metadata': {
                'uid': uid,
                **(
                    {'deletionTimestamp': '2026-10-20T01:34:35Z'}
                    if deleting
                    else {}
                ),
            },
            'spec': {
                'containers': [{'name': managed.RECEIVER, 'image': image}]
            },
            'status': {
                'containerStatuses': [
                    {'name': managed.RECEIVER, 'ready': ready}
                ]
            },
        }

    def test_rejects_superseded_or_draining_receivers_even_with_same_image(
        self,
    ):
        ready = self.pod()
        self.assertTrue(
            managed.receiver_pods_converged([ready], 'current', {'previous'})
        )
        cases = [
            [],
            [ready, self.pod(uid='previous')],
            [ready, self.pod(uid='draining', deleting=True)],
            [self.pod(image='legacy')],
            [self.pod(ready=False)],
        ]
        for pods in cases:
            with self.subTest(pods=pods):
                self.assertFalse(
                    managed.receiver_pods_converged(
                        pods, 'current', {'previous'}
                    )
                )

    def test_waits_for_all_previous_generation_pods_to_disappear(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 6
        current = self.pod()
        observations = [
            {'items': [current, self.pod(uid='previous')]},
            {'items': [current, self.pod(uid='previous', deleting=True)]},
            {'items': [current]},
        ]
        with (
            patch.object(managed, 'get', side_effect=observations) as read,
            patch.object(managed.time, 'sleep'),
            patch.object(managed.time, 'monotonic', side_effect=range(8)),
        ):
            fixture.wait_receiver_pods('team-a', 'current', {'previous'})
        self.assertEqual(read.call_count, 3)

    @staticmethod
    def upgrade_response():
        return {
            'status': 503,
            'body': managed.RECEIVER_UPGRADE_MESSAGE,
            'retryAfter': '5',
            'contentType': 'text/plain; charset=utf-8',
            'contentDisposition': 'attachment',
            'contentTypeOptions': 'nosniff',
            'proxyError': 'receiver-upgrade-required',
        }

    def test_retries_safe_old_receiver_transition_until_exact_protocol_error(
        self,
    ):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 6
        expected = self.upgrade_response()
        responses = [
            URLError('connection reset'),
            dict(expected, status=200, body='team-a:historical'),
            dict(expected, body='<html>private application state</html>'),
            expected,
        ]
        with (
            patch.object(fixture, 'request', side_effect=responses) as request,
            patch.object(managed.time, 'sleep'),
            patch.object(managed.time, 'monotonic', side_effect=range(8)),
        ):
            actual = fixture.wait_receiver_upgrade_required('team-a', 'key')
        self.assertEqual(actual, expected)
        self.assertEqual(request.call_count, 4)

    def test_html_wrong_body_and_missing_retry_cannot_satisfy_upgrade_rejection(
        self,
    ):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 1
        expected = self.upgrade_response()
        for response in [
            dict(expected, body='<html>private application state</html>'),
            dict(expected, body='upstream error with sensitive details'),
            dict(expected, retryAfter=None),
        ]:
            with (
                self.subTest(response=response),
                patch.object(fixture, 'request', return_value=response),
                patch.object(managed.time, 'sleep'),
                patch.object(managed.time, 'monotonic', side_effect=[0, 0, 2]),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'Timed out waiting for shared UI'
                ):
                    fixture.wait_receiver_upgrade_required('team-a', 'key')

    def test_upgrade_rejection_requires_plain_text_and_hardening_headers(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 1
        for field in [
            'contentType',
            'contentDisposition',
            'contentTypeOptions',
            'proxyError',
        ]:
            response = dict(self.upgrade_response(), **{field: None})
            with (
                self.subTest(field=field),
                patch.object(fixture, 'request', return_value=response),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'hardened plain-text upgrade-required'
                ):
                    fixture.wait_receiver_upgrade_required('team-a', 'key')

    def test_retries_recovery_until_exact_historical_content(self):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 6
        expected = {
            'status': 200,
            'body': 'team-a:historical',
            'retryAfter': None,
        }
        responses = [
            {'status': 503, 'body': 'upgrade required', 'retryAfter': '5'},
            expected,
        ]
        with (
            patch.object(fixture, 'request', side_effect=responses) as request,
            patch.object(managed.time, 'sleep'),
            patch.object(managed.time, 'monotonic', side_effect=range(8)),
        ):
            actual = fixture.wait_artifact_response(
                'team-a',
                'key',
                'receiver recovery',
                status=200,
                body='team-a:historical',
            )
        self.assertEqual(actual, expected)
        self.assertEqual(request.call_count, 2)

    def test_network_failure_spa_and_wrong_content_cannot_satisfy_recovery(
        self,
    ):
        fixture = managed.Acceptance.__new__(managed.Acceptance)
        fixture.timeout = 1
        for response in [
            URLError('offline'),
            {
                'status': 200,
                'body': '<html>application</html>',
                'retryAfter': None,
            },
            {'status': 200, 'body': 'other:historical', 'retryAfter': None},
        ]:
            kwargs = (
                {'side_effect': response}
                if isinstance(response, Exception)
                else {'return_value': response}
            )
            with (
                self.subTest(response_type=type(response).__name__),
                patch.object(fixture, 'request', **kwargs),
                patch.object(managed.time, 'sleep'),
                patch.object(managed.time, 'monotonic', side_effect=[0, 0, 2]),
            ):
                with self.assertRaisesRegex(
                    AssertionError, 'Timed out waiting for receiver recovery'
                ):
                    fixture.wait_artifact_response(
                        'team-a',
                        'key',
                        'receiver recovery',
                        status=200,
                        body='team-a:historical',
                    )


if __name__ == '__main__':
    unittest.main()

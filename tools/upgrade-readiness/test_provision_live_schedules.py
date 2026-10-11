# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Safety and API contract tests for the mutating disposable CI helper."""

from contextlib import redirect_stderr
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import provision_live_schedules as fixture


class ProvisionTests(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name)
        self.state = dict(
            context=fixture.CONTEXT,
            namespace=fixture.NAMESPACE,
            owner_marker='a' * 32,
            rbac_ready=True,
            schedules=[])

    def test_context_and_consent_required_before_any_io(self):
        for context, consent in [('production', True),
                                 (fixture.CONTEXT, False)]:
            args = [
                'fixture', '--context', context, '--phase', 'rbac',
                '--state-dir',
                str(self.path)
            ]
            if consent:
                args.append('--allow-test-cluster-mutations')
            with mock.patch('sys.argv', args), mock.patch.object(
                    fixture, 'provision_rbac') as provision, redirect_stderr(
                        io.StringIO()):
                with self.assertRaises(SystemExit) as caught:
                    fixture.main()
            self.assertEqual(caught.exception.code, 1)
            provision.assert_not_called()

    def test_namespace_ownership_must_match_state(self):
        for label in (None, 'b' * 32):
            obj = dict(metadata=dict(labels={fixture.LABEL: label}))
            with mock.patch.object(
                    fixture, 'kubectl_get', return_value=(obj, None)):
                with self.assertRaisesRegex(fixture.FixtureError, 'ownership'):
                    fixture.verify_state(fixture.CONTEXT, self.state)
        obj = dict(metadata=dict(labels={fixture.LABEL: 'a' * 32}))
        with mock.patch.object(
                fixture, 'kubectl_get', return_value=(obj, None)):
            fixture.verify_state(fixture.CONTEXT, self.state)

    def test_existing_namespace_create_failure_does_not_adopt(self):
        with mock.patch.object(
                fixture, 'infrastructure_rbac',
                return_value=[]), mock.patch.object(
                    fixture, 'fixture_runtime_config',
                    return_value=[]), mock.patch.object(
                        fixture,
                        'kubectl_get',
                        return_value=({
                            'rules': [{
                                'verbs': ['get']
                            }]
                        }, None)), mock.patch.object(
                            fixture,
                            'kubectl_create',
                            side_effect=fixture.FixtureError(
                                'already_exists')) as create:
            with self.assertRaises(fixture.FixtureError):
                fixture.provision_rbac(fixture.CONTEXT, self.path)
            self.assertEqual(create.call_count, 1)
            self.assertEqual(create.call_args.args[1][0]['kind'], 'Namespace')
        self.assertFalse(
            json.loads(
                (self.path / 'state.json').read_text()).get('rbac_ready'))

    def test_infrastructure_roles_remain_namespaced_except_authentication(self):

        def get(context, namespace, resource):
            if resource == 'deployment/workflow-controller':
                return {
                    'spec': {
                        'template': {
                            'spec': {
                                'serviceAccountName': 'argo'
                            }
                        }
                    }
                }, None
            if resource == 'rolebindings':
                return {
                    'items': [{
                        'subjects': [{
                            'kind': 'ServiceAccount',
                            'name': 'argo',
                            'namespace': 'kubeflow'
                        }],
                        'roleRef': {
                            'apiGroup': 'rbac.authorization.k8s.io',
                            'kind': 'Role',
                            'name': 'argo'
                        }
                    }]
                }, None
            return {
                'rules': [{
                    'apiGroups': ['argoproj.io'],
                    'resources': ['workflows'],
                    'verbs': ['get', 'list', 'watch']
                }]
            }, None

        with mock.patch.object(fixture, 'kubectl_get', side_effect=get):
            objects = fixture.infrastructure_rbac(fixture.CONTEXT)
        roles = [o for o in objects if o['kind'] == 'Role']
        self.assertEqual(len(roles), 4)
        self.assertTrue(
            all(o['metadata']['namespace'] == fixture.NAMESPACE for o in roles))
        cluster_roles = [o for o in objects if o['kind'] == 'ClusterRole']
        self.assertEqual(len(cluster_roles), 1)
        self.assertEqual([r['resources'] for r in cluster_roles[0]['rules']],
                         [['tokenreviews'], ['subjectaccessreviews']])
        cluster_binding = [
            o for o in objects if o['kind'] == 'ClusterRoleBinding'
        ][0]
        self.assertEqual(cluster_binding['subjects'], [{
            'kind': 'ServiceAccount',
            'name': 'ml-pipeline',
            'namespace': 'kubeflow'
        }])

    def test_copied_infrastructure_cannot_grant_custom_account_use(self):
        for resources, verbs in [(['serviceaccounts'], ['use']),
                                 (['*'], ['*'])]:
            obj = {
                'rules': [{
                    'apiGroups': [''],
                    'resources': resources,
                    'verbs': verbs
                }]
            }
            with mock.patch.object(
                    fixture, 'kubectl_get', return_value=(obj, None)):
                with self.assertRaisesRegex(fixture.FixtureError,
                                            'grants_account_use'):
                    fixture.source_role(fixture.CONTEXT, 'source')

    def test_runtime_config_copies_only_required_data_to_fixture_namespace(
            self):
        values = [({
            'metadata': {
                'uid': 'never-copy',
                'namespace': 'kubeflow'
            },
            'data': {
                'accesskey': 'YQ==',
                'secretkey': 'Yg=='
            },
            'type': 'Opaque'
        }, None),
                  ({
                      'metadata': {
                          'uid': 'never-copy'
                      },
                      'data': {
                          'defaultPipelineRoot': 's3://mlpipeline'
                      }
                  }, None)]
        with mock.patch.object(fixture, 'kubectl_get', side_effect=values):
            objects = fixture.fixture_runtime_config(fixture.CONTEXT)
        self.assertEqual([o['kind'] for o in objects], ['Secret', 'ConfigMap'])
        self.assertTrue(
            all(o['metadata']['namespace'] == fixture.NAMESPACE and
                'uid' not in o['metadata'] for o in objects))
        self.assertEqual(objects[0]['data']['secretkey'], 'Yg==')

    def test_runtime_config_rejects_oversize_or_unexpected_secret(self):
        for value in ({
                'data': {
                    'accesskey': 'a',
                    'secretkey': 'x' * 100
                }
        }, {
                'type': 'kubernetes.io/service-account-token',
                'data': {
                    'accesskey': 'a',
                    'secretkey': 'b'
                }
        }):
            with mock.patch.object(
                    fixture, 'kubectl_get',
                    return_value=(value, None)), mock.patch.object(
                        fixture, 'MAX_BYTES', 80):
                with self.assertRaises(fixture.FixtureError):
                    fixture.fixture_runtime_config(fixture.CONTEXT)

    def test_existing_state_is_not_reused_for_rbac(self):
        fixture.write_object(self.path / 'state.json', self.state)
        with mock.patch.object(fixture, 'kubectl_get') as get:
            with self.assertRaises(fixture.FixtureError):
                fixture.provision_rbac(fixture.CONTEXT, self.path)
            get.assert_not_called()

    def test_controller_grant_is_scoped_to_one_custom_account(self):
        objects = fixture.fixture_rbac(
            [dict(apiGroups=[''], resources=['pods'], verbs=['get'])])
        roles = {
            o['metadata']['name']: o for o in objects if o['kind'] == 'Role'
        }
        controller_rules = roles['fixture-controller']['rules']
        use = [r for r in controller_rules if 'use' in r['verbs']]
        self.assertEqual(use, [
            dict(
                apiGroups=[''],
                resources=['serviceaccounts'],
                resourceNames=['readiness-granted'],
                verbs=['use'])
        ])
        self.assertNotIn('ClusterRole', [o['kind'] for o in objects])
        self.assertTrue(
            all(o['metadata']['namespace'] == fixture.NAMESPACE
                for o in objects))

    def test_prepare_creates_disabled_default_and_custom_schedules(self):
        client = mock.Mock()
        client.post.side_effect = [{
            'experiment_id': 'experiment'
        }] + [{
            'recurring_run_id': 'uid-' + str(i)
        } for i in range(3)]
        pipeline = dict(
            pipelineInfo={'name': 'fixture'},
            root={'dag': {}},
            deploymentSpec={'executors': {}})
        with mock.patch.object(
                fixture,
                'schedule_identity',
                side_effect=['default', 'scoped', 'denied']):
            fixture.prepare(fixture.CONTEXT, self.path, self.state, client,
                            pipeline)
        bodies = [call.args[1] for call in client.post.call_args_list[1:]]
        self.assertNotIn('service_account', bodies[0])
        self.assertEqual([b.get('service_account') for b in bodies[1:]],
                         ['readiness-granted', 'readiness-denied'])
        self.assertTrue(
            all(b['mode'] == 'DISABLE' and b['no_catchup'] for b in bodies))
        cases = json.loads((self.path / 'cases.json').read_text())['cases']
        self.assertEqual([c['expected_outcome'] for c in cases],
                         ['run_created', 'run_created', 'blocked'])
        self.assertTrue(
            json.loads((self.path / 'state.json').read_text())['prepared'])

    def test_prepare_requires_matching_disabled_schedule(self):
        obj = dict(
            metadata=dict(
                uid='uid-one', name='fixture', namespace=fixture.NAMESPACE),
            spec=dict(enabled=True))
        with mock.patch.object(
                fixture, 'kubectl_get', return_value=({
                    'items': [obj]
                }, None)):
            with self.assertRaisesRegex(fixture.FixtureError, 'not_disabled'):
                fixture.schedule_identity(fixture.CONTEXT, 'uid-one')
            obj['spec']['enabled'] = False
            self.assertEqual(
                fixture.schedule_identity(fixture.CONTEXT, 'uid-one'),
                'fixture')
            obj['metadata']['namespace'] = 'other'
            with self.assertRaisesRegex(fixture.FixtureError,
                                        'namespace_mismatch'):
                fixture.schedule_identity(fixture.CONTEXT, 'uid-one')

    def test_partial_prepare_preserves_id_and_refuses_recreation(self):
        client = mock.Mock()
        client.post.side_effect = [{
            'experiment_id': 'experiment'
        }, {
            'recurring_run_id': 'uid-1'
        }]
        pipeline = dict(
            pipelineInfo={'name': 'fixture'},
            root={'dag': {}},
            deploymentSpec={'executors': {}})
        with mock.patch.object(
                fixture,
                'schedule_identity',
                side_effect=fixture.FixtureError('not_found')):
            with self.assertRaises(fixture.FixtureError):
                fixture.prepare(fixture.CONTEXT, self.path, self.state, client,
                                pipeline)
        persisted = json.loads((self.path / 'state.json').read_text())
        self.assertEqual(persisted['schedules'][0]['schedule_uid'], 'uid-1')
        with self.assertRaises(fixture.FixtureError):
            fixture.prepare(fixture.CONTEXT, self.path, persisted, client,
                            pipeline)
        self.assertEqual(client.post.call_count, 2)

    def test_enable_failure_attempts_disable_for_every_fixture(self):
        self.state.update(
            prepared=True,
            schedules=[dict(schedule_uid='one'),
                       dict(schedule_uid='two')])
        client = mock.Mock()
        client.post.side_effect = [{},
                                   fixture.FixtureError('denied'),
                                   fixture.FixtureError('failure'), {}]
        with self.assertRaises(fixture.FixtureError):
            fixture.set_enabled(self.path, self.state, client, True)
        self.assertEqual([c.args[0] for c in client.post.call_args_list], [
            '/apis/v2beta1/recurringruns/one:enable',
            '/apis/v2beta1/recurringruns/two:enable',
            '/apis/v2beta1/recurringruns/one:disable',
            '/apis/v2beta1/recurringruns/two:disable'
        ])
        self.assertFalse((self.path / 'activation-start.txt').exists())

    def test_disable_continues_after_one_failure(self):
        self.state['schedules'] = [
            dict(schedule_uid='one'),
            dict(schedule_uid='two')
        ]
        client = mock.Mock()
        client.post.side_effect = [fixture.FixtureError('failure'), {}]
        with self.assertRaises(fixture.FixtureError):
            fixture.set_enabled(self.path, self.state, client, False)
        self.assertEqual(client.post.call_count, 2)

    def test_enable_records_start_before_requests(self):
        self.state.update(prepared=True, schedules=[dict(schedule_uid='one')])
        client = mock.Mock()
        fixture.set_enabled(self.path, self.state, client, True)
        saved = json.loads((self.path / 'state.json').read_text())
        self.assertTrue(saved['enabled'])
        self.assertEqual(
            (self.path / 'activation-start.txt').read_text().strip(),
            saved['activation_start'])

    def test_post_rejects_nonloopback_and_unsafe_endpoints(self):
        token = self.path / 'token'
        token.write_text('secret-token')
        for endpoint in [
                'https://127.0.0.1:9000', 'http://localhost:9000',
                'http://192.0.2.1:9000', 'http://127.0.0.1:9000/path',
                'http://user@127.0.0.1:9000', 'http://127.0.0.1:9000?x=1'
        ]:
            with self.assertRaises(fixture.FixtureError):
                fixture.FixtureClient(endpoint, token)
        fixture.FixtureClient('http://127.0.0.1:9000', token)
        fixture.FixtureClient('http://[::1]:9000', token)

    def test_post_rejects_redirect_without_following_or_error_leak(self):
        token = self.path / 'token'
        token.write_text('secret-token')
        client = fixture.FixtureClient('http://127.0.0.1:9000', token)
        connection = mock.Mock()
        connection.getresponse.return_value.status = 302
        with mock.patch.object(
                fixture.http.client, 'HTTPConnection', return_value=connection):
            with self.assertRaisesRegex(fixture.FixtureError,
                                        '^fixture_api_request_failed$'):
                client.post('/apis/v2beta1/experiments', {})
        connection.close.assert_called_once()
        self.assertEqual(connection.request.call_count, 1)

    def test_post_is_bounded_and_sanitizes_backend_errors(self):
        token = self.path / 'token'
        token.write_text('secret-token')
        client = fixture.FixtureClient('http://127.0.0.1:9000', token)
        for body in (b'x' * 20, b'{"error":"secret-backend-detail"}'):
            connection = mock.Mock()
            response = connection.getresponse.return_value
            response.status = 200
            response.read1.side_effect = [body, b'']
            with mock.patch.object(
                    fixture.http.client, 'HTTPConnection',
                    return_value=connection), mock.patch.object(
                        fixture, 'MAX_BYTES', 10):
                with self.assertRaisesRegex(fixture.FixtureError,
                                            '^fixture_api_request_failed$'):
                    client.post('/apis/v2beta1/experiments', {})
            connection.close.assert_called_once()


if __name__ == '__main__':
    unittest.main()

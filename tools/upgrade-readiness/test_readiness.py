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

import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import readiness


def obj(kind, name, namespace='team-a', **kwargs):
    return dict(
        kind=kind, metadata=dict(name=name, namespace=namespace), **kwargs)


def assess(items):
    return readiness.analyze(
        dict(items=items), 'kubeflow', ['kubeflow', 'team-a'], 'ml-pipeline-ui',
        'cache-server', '2.17.2')


def role_items(verbs):
    return [
        obj('Role',
            'reader',
            rules=[
                dict(
                    apiGroups=['pipelines.kubeflow.org'],
                    resources=['runs'],
                    verbs=verbs)
            ]),
        obj('RoleBinding', 'reader', roleRef=dict(kind='Role', name='reader'))
    ]


class ReadinessTest(unittest.TestCase):

    def test_empty_inventory_never_green(self):
        report = assess([])
        self.assertEqual(report['assessment'], 'incomplete')
        self.assertGreater(report['counts']['unknown'], 0)
        self.assertTrue(
            any(f['rule'] == 'coverage.unassessed' for f in report['findings']))

    def test_role_deficiency_is_not_effective_denial(self):
        report = assess(role_items(['get', 'list']))
        result = next(
            f for f in report['findings'] if f['rule'] == 'rbac.readLog')
        self.assertEqual(result['status'], 'unknown')
        self.assertIn('Other grants', result['evidence'])

    def test_readlog_and_wildcards_are_recognized(self):
        for verbs in [['get', 'readLog'], ['*']]:
            with self.subTest(verbs=verbs):
                result = next(
                    f for f in assess(role_items(verbs))['findings']
                    if f['rule'] == 'rbac.readLog')
                self.assertEqual(result['status'], 'no_issue_detected')
                self.assertIn('not assessed', result['evidence'])

    def test_wrong_group_or_subresource_does_not_grant_readlog(self):
        items = role_items(['get'])
        items[0]['rules'] += [
            dict(apiGroups=[''], resources=['runs'], verbs=['readLog']),
            dict(
                apiGroups=['pipelines.kubeflow.org'],
                resources=['runs/log'],
                verbs=['get'])
        ]
        result = next(
            f for f in assess(items)['findings'] if f['rule'] == 'rbac.readLog')
        self.assertEqual(result['status'], 'unknown')

    def test_unresolved_role_and_aggregate_are_unknown(self):
        items = [
            obj('RoleBinding',
                'viewer',
                roleRef=dict(kind='ClusterRole', name='view'))
        ]
        self.assertTrue(
            any(f['rule'] == 'rbac.coverage'
                for f in assess(items)['findings']))
        items.append(
            obj('ClusterRole',
                'view',
                namespace='',
                aggregationRule={'clusterRoleSelectors': []},
                rules=[]))
        self.assertTrue(
            any(f['rule'] == 'rbac.coverage'
                for f in assess(items)['findings']))

    def test_scope_is_respected(self):
        items = role_items(['get'])
        items[1]['metadata']['namespace'] = 'outside'
        self.assertFalse(
            any(f['rule'].startswith('rbac.')
                for f in assess(items)['findings']))

    def test_secret_values_and_subjects_never_reported(self):
        ui = obj(
            'Deployment',
            'ml-pipeline-ui',
            'kubeflow',
            spec=dict(
                template=dict(
                    spec=dict(containers=[
                        dict(
                            name='ml-pipeline-ui',
                            env=[
                                dict(
                                    name='TENSORBOARD_PROXY_SIGNING_SECRET',
                                    value='SENSITIVE_KEY'),
                                dict(name='PASSWORD', value='OTHER_SECRET')
                            ])
                    ]))))
        items = [ui] + role_items(['get'])
        items[-1]['subjects'] = [dict(kind='User', name='PRIVATE_IDENTITY')]
        report = assess(items)
        for output in [json.dumps(report), readiness.markdown(report)]:
            for private in [
                    'SENSITIVE_KEY', 'OTHER_SECRET', 'PRIVATE_IDENTITY'
            ]:
                self.assertNotIn(private, output)
        key = next(
            f for f in report['findings'] if f['rule'] == 'tensorboard.key')
        self.assertEqual(key['status'], 'unknown')

    def test_imported_or_sidecar_key_does_not_prove_shared_key(self):
        ui = obj(
            'Deployment',
            'ml-pipeline-ui',
            'kubeflow',
            spec=dict(
                template=dict(
                    spec=dict(containers=[
                        dict(
                            name='ml-pipeline-ui',
                            envFrom=[dict(secretRef=dict(name='key'))]),
                        dict(
                            name='sidecar',
                            env=[
                                dict(
                                    name='TENSORBOARD_PROXY_SIGNING_SECRET',
                                    value='sidecar')
                            ])
                    ]))))
        key = next(
            f for f in assess([ui])['findings']
            if f['rule'] == 'tensorboard.key')
        self.assertIn('imports', key['evidence'])
        self.assertEqual(key['status'], 'unknown')

    def test_viewer_roles_do_not_prompt_blanket_edit_grants(self):
        items = [
            obj('Role',
                'view',
                rules=[
                    dict(
                        apiGroups=['kubeflow.org'],
                        resources=['viewers'],
                        verbs=['get'])
                ]),
            obj('RoleBinding', 'view', roleRef=dict(kind='Role', name='view'))
        ]
        f = next(
            f for f in assess(items)['findings']
            if f['rule'] == 'rbac.tensorboard')
        self.assertEqual(f['status'], 'unknown')
        self.assertIn('Keep readers read-only', f['action'])

    def test_collector_only_gets_scoped_objects_and_referenced_clusterrole(
            self):
        calls = []

        def get(context, namespace, resource):
            calls.append((context, namespace, resource))
            if resource == 'roles.rbac.authorization.k8s.io':
                return None, 'collection_failed'
            if resource == 'rolebindings.rbac.authorization.k8s.io':
                return dict(items=[
                    obj('RoleBinding',
                        'view',
                        roleRef=dict(kind='ClusterRole', name='view'))
                ]), None
            if resource.startswith('clusterroles.'):
                return obj('ClusterRole', 'view', namespace='', rules=[]), None
            return dict(items=[]), None

        with mock.patch.object(readiness, 'kubectl_get', side_effect=get):
            inventory, failures = readiness.collect('chosen-context',
                                                    ['team-a'])
        self.assertEqual(len(calls), 4)
        self.assertEqual(calls[-1],
                         ('chosen-context', None,
                          'clusterroles.rbac.authorization.k8s.io/view'))
        self.assertEqual(len(failures), 1)
        report = readiness.analyze(inventory, 'team-a', ['team-a'], 'ui',
                                   'cache', '2.17.2', failures)
        self.assertTrue(
            any(f['rule'] == 'inventory.collection'
                for f in report['findings']))

    def test_subprocess_limit_and_error_redaction(self):
        original = subprocess.Popen
        calls = []

        def run_script(script):

            def launch(command, **kwargs):
                calls.append(command)
                return original([sys.executable, '-c', script], **kwargs)

            with mock.patch.object(
                    readiness.subprocess, 'Popen', side_effect=launch):
                return readiness.kubectl_get('context', 'team-a',
                                             'deployments.apps')

        with mock.patch.object(readiness, 'MAX_BYTES', 128):
            data, error = run_script("import sys;sys.stdout.write('x'*1024)")
            self.assertIsNone(data)
            self.assertEqual(error, 'collection_exceeded_16_mib')
        data, error = run_script(
            "import sys;sys.stderr.write('PRIVATE_ERROR');sys.exit(1)")
        self.assertEqual(error, 'collection_failed')
        data, error = run_script("print('{\"items\": []}')")
        self.assertIsNone(error)
        self.assertEqual(data, {'items': []})
        self.assertIn('get', calls[0])
        self.assertNotIn('secrets', calls[0])

    def test_cli_json_and_exit_codes(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'inventory.json'
            path.write_text(json.dumps(dict(items=[])))
            args = [
                '--inventory',
                str(path), '--source-version', '2.17.2', '--system-namespace',
                'kubeflow', '--format', 'json'
            ]
            with contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertEqual(readiness.main(args), 2)
            self.assertEqual(
                json.loads(output.getvalue())['assessment'], 'incomplete')
            path.write_text(
                '{"items": [{"kind": "Secret", "data": "PRIVATE"}]}')
            with contextlib.redirect_stderr(io.StringIO()) as output:
                self.assertEqual(readiness.main(args), 1)
            self.assertNotIn('PRIVATE', output.getvalue())

    def test_malformed_role_arrays_cannot_produce_positive_findings(self):
        for field in ['apiGroups', 'resources', 'verbs', 'resourceNames']:
            items = role_items(['get', 'readLog'])
            items[0]['rules'][0][field] = 'forget readLog'
            with self.subTest(field=field), self.assertRaises(ValueError):
                assess(items)

    def test_collection_has_cumulative_inventory_budget(self):
        first = obj('Deployment', 'one', spec={'padding': 'x' * 100})
        second = obj('Role', 'two', rules=[], padding='x' * 100)
        limit = max(len(json.dumps(first)), len(json.dumps(second))) + 1
        with mock.patch.object(readiness, 'MAX_BYTES', limit):
            with mock.patch.object(
                    readiness,
                    'kubectl_get',
                    side_effect=[({
                        'items': [first]
                    }, None), ({
                        'items': [second]
                    }, None)]) as get:
                inventory, failures = readiness.collect('ctx', ['team-a'])
        self.assertEqual(len(inventory['items']), 1)
        self.assertEqual(get.call_count, 2)
        self.assertIn('remaining_scope_not_collected', failures[0]['reason'])

    def test_subprocess_deadline(self):
        original = subprocess.Popen

        def launch(command, **kwargs):
            return original(
                [sys.executable, '-c', 'import time; time.sleep(10)'], **kwargs)

        with mock.patch.object(
                readiness.subprocess, 'Popen', side_effect=launch):
            with mock.patch.object(
                    readiness.time, 'monotonic', side_effect=[0, 31]):
                data, error = readiness.kubectl_get('context', 'team-a',
                                                    'deployments.apps')
        self.assertIsNone(data)
        self.assertEqual(error, 'collection_timed_out')

    def test_oversized_offline_inventory(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'big.json'
            path.write_text(' ' * 129)
            with mock.patch.object(readiness, 'MAX_BYTES', 128):
                with self.assertRaises(ValueError):
                    readiness.read_json(path)


if __name__ == '__main__':
    unittest.main()

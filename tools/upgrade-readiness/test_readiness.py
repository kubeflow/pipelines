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
import time
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

    def test_controller_hints_do_not_expose_identity_values(self):
        deployment = obj(
            'Deployment',
            'ml-pipeline-scheduledworkflow',
            'kubeflow',
            spec={
                'template': {
                    'spec': {
                        'serviceAccountName':
                            'private-account',
                        'containers': [{
                            'args': ['--userIdentityValue=private-user']
                        }]
                    }
                }
            })
        report = readiness.analyze({'items': [deployment]},
                                   'kubeflow', ['kubeflow'],
                                   'ui',
                                   'cache',
                                   '2.17.2',
                                   include_schedules=True)
        text = json.dumps(report)
        self.assertNotIn('private-account', text)
        self.assertNotIn('private-user', text)
        self.assertTrue(
            any(f['rule'] == 'schedule.controllerIdentity' and
                f['status'] == 'unknown' for f in report['findings']))

    def test_schedule_cli_opt_in(self):
        schedule = obj(
            'ScheduledWorkflow',
            'nightly',
            spec={
                'enabled': True,
                'serviceAccount': 'runner'
            })
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'inventory.json'
            path.write_text(json.dumps({'items': [schedule]}))
            base = [
                '--inventory',
                str(path), '--source-version', '2.17.2', '--system-namespace',
                'team-a', '--format', 'json'
            ]
            for include in (False, True):
                with contextlib.redirect_stdout(io.StringIO()) as output:
                    self.assertEqual(
                        readiness.main(base + (
                            ['--include-schedules'] if include else [])), 2)
                report = json.loads(output.getvalue())
                self.assertEqual(report['source']['schedules_requested'],
                                 include)
                self.assertEqual(
                    any(f['rule'] == 'schedule.serviceAccount'
                        for f in report['findings']), include)

    def test_schedule_collection_requires_opt_in_and_preserves_scope(self):
        for include in (False, True):
            with mock.patch.object(
                    readiness, 'kubectl_get', return_value=({
                        'items': []
                    }, None)) as get:
                readiness.collect(
                    'ctx', ['team-a', 'team-b'], include_schedules=include)
            schedule_calls = [
                call.args
                for call in get.call_args_list
                if call.args[2] == 'scheduledworkflows.kubeflow.org'
            ]
            self.assertEqual(
                schedule_calls,
                [('ctx', namespace, 'scheduledworkflows.kubeflow.org')
                 for namespace in ['team-a', 'team-b']] if include else [])

    def test_schedule_accounts_are_evidence_not_access_decisions(self):
        schedules = [
            obj('ScheduledWorkflow',
                'custom',
                spec={
                    'enabled': True,
                    'serviceAccount': 'custom-runner'
                }),
            obj('ScheduledWorkflow', 'default', spec={'enabled': False}),
            obj('ScheduledWorkflow',
                'embedded',
                spec={
                    'enabled': True,
                    'serviceAccount': 'WRONG_PATH',
                    'workflow': {
                        'spec': {
                            'serviceAccountName': 'EMBEDDED_ACCOUNT',
                            'secret': 'PRIVATE_SPEC'
                        }
                    }
                }),
            obj('ScheduledWorkflow',
                'outside',
                namespace='outside',
                spec={'serviceAccount': 'OUTSIDE_ACCOUNT'}),
        ]
        self.assertFalse(
            any(f['rule'].startswith('schedule.')
                for f in assess(schedules)['findings']))
        report = readiness.analyze({'items': schedules},
                                   'kubeflow', ['kubeflow', 'team-a'],
                                   'ui',
                                   'cache',
                                   '2.17.2',
                                   include_schedules=True)
        results = {
            f['resource'].split('/')[-1]: f
            for f in report['findings']
            if f['rule'] == 'schedule.serviceAccount'
        }
        self.assertEqual(set(results), {'custom', 'default', 'embedded'})
        self.assertTrue(all(f['status'] == 'unknown' for f in results.values()))
        self.assertIn('account name custom-runner',
                      results['custom']['evidence'])
        self.assertIn('target run namespace', results['custom']['evidence'])
        self.assertIn('disabled', results['default']['evidence'])
        self.assertIn('default was not resolved',
                      results['default']['evidence'])
        self.assertIn('Embedded workflow path', results['embedded']['evidence'])
        for output in (json.dumps(report), readiness.markdown(report)):
            for private in [
                    'WRONG_PATH', 'EMBEDDED_ACCOUNT', 'PRIVATE_SPEC',
                    'OUTSIDE_ACCOUNT'
            ]:
                self.assertNotIn(private, output)

    def test_schedule_failures_and_zero_counts_are_unknown(self):

        def get(context, namespace, resource):
            if resource == 'scheduledworkflows.kubeflow.org':
                return None, 'collection_failed'
            return {'items': []}, None

        with mock.patch.object(readiness, 'kubectl_get', side_effect=get):
            inventory, failures = readiness.collect(
                'ctx', ['team-a'], include_schedules=True)
        report = readiness.analyze(
            inventory,
            'team-a', ['team-a'],
            'ui',
            'cache',
            '2.17.2',
            failures,
            mode='live',
            include_schedules=True)
        self.assertTrue(
            any(f['rule'] == 'inventory.collection' and
                'scheduledworkflows' in f['resource']
                for f in report['findings']))
        coverage = next(
            f for f in report['findings'] if f['rule'] == 'schedule.coverage')
        self.assertEqual(coverage['status'], 'unknown')
        self.assertIn('0 ScheduledWorkflow', coverage['evidence'])

    def test_malformed_schedule_identity_is_unresolved_and_redacted(self):
        for spec in [
                None, [], {
                    'serviceAccount': ['PRIVATE']
                }, {
                    'serviceAccount': 'PRIVATE/BAD'
                }, {
                    'workflow': 'PRIVATE'
                }
        ]:
            results = readiness.analyze_schedules(
                [obj('ScheduledWorkflow', 'bad', spec=spec)])
            self.assertEqual(results[-1]['status'], 'unknown')
            self.assertNotIn('PRIVATE', json.dumps(results))

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

    def test_namespace_scoped_inventory_requires_namespace(self):
        for kind in ['Deployment', 'Role', 'RoleBinding']:
            for value in [None, '', 42, 'INVALID', ' ']:
                item = obj(kind, 'test', namespace=value)
                with self.subTest(
                        kind=kind,
                        namespace=value), self.assertRaises(ValueError):
                    assess([item])
            item = obj(kind, 'test')
            del item['metadata']['namespace']
            with self.assertRaises(ValueError):
                assess([item])
        readiness.validate_inventory({
            'items': [{
                'kind': 'ClusterRole',
                'metadata': {
                    'name': 'view'
                },
                'rules': []
            }]
        })

    def test_missing_ui_reports_both_rules_and_offline_completeness(self):
        report = assess([])
        for rule in ['tensorboard.key', 'tensorboard.rollout']:
            result = next(f for f in report['findings'] if f['rule'] == rule)
            self.assertEqual(result['status'], 'unknown')
        self.assertTrue(
            any(f['rule'] == 'inventory.collection' and
                f['evidence'] == 'offline_inventory_completeness_unverified'
                for f in report['findings']))
        live = readiness.analyze({'items': []},
                                 'kubeflow', ['kubeflow'],
                                 'ui',
                                 'cache',
                                 '2.17.2',
                                 mode='live')
        self.assertFalse(
            any(f['evidence'] == 'offline_inventory_completeness_unverified'
                for f in live['findings']))

    def test_usage_errors_are_not_report_exit_codes(self):
        base = [
            '--inventory', 'unused.json', '--source-version', '2.17.2',
            '--system-namespace', 'kubeflow'
        ]
        for args in [[], base + ['--format', 'invalid'],
                     base + ['--namespace', 'INVALID'],
                     base + ['--source-version', 'invalid']]:
            with self.subTest(args=args), contextlib.redirect_stderr(
                    io.StringIO()):
                with self.assertRaises(SystemExit) as result:
                    readiness.main(args)
                self.assertEqual(result.exception.code, 1)
        with contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(SystemExit) as result:
                readiness.main(['--help'])
            self.assertEqual(result.exception.code, 0)

    def test_collection_limit_kills_inherited_plugin(self):
        original = subprocess.Popen
        for cause in ['timeout', 'size']:
            with self.subTest(
                    cause=cause), tempfile.TemporaryDirectory() as directory:
                marker = Path(directory) / 'survived'
                child = "import time; from pathlib import Path; time.sleep(0.4); Path(%r).touch()" % str(
                    marker)
                script = (
                    "import subprocess,sys,time; subprocess.Popen([sys.executable,'-c',%r]); "
                    "print(%r,flush=True); time.sleep(10)") % (
                        child, 'x' * 1024 if cause == 'size' else 'ready')

                def launch(command, **kwargs):
                    self.assertTrue(kwargs['start_new_session'])
                    return original([sys.executable, '-c', script], **kwargs)

                with mock.patch.object(
                        readiness.subprocess, 'Popen', side_effect=launch):
                    if cause == 'timeout':
                        with mock.patch.object(
                                readiness.time, 'monotonic',
                                side_effect=[0, 0, 31]):
                            _, error = readiness.kubectl_get(
                                'ctx', 'ns', 'deployments.apps')
                        self.assertEqual(error, 'collection_timed_out')
                    else:
                        with mock.patch.object(readiness, 'MAX_BYTES', 128):
                            _, error = readiness.kubectl_get(
                                'ctx', 'ns', 'deployments.apps')
                        self.assertEqual(error, 'collection_exceeded_16_mib')
                time.sleep(0.6)
                self.assertFalse(
                    marker.exists(),
                    'Inherited credential plugin survived collection limit')

    def test_oversized_offline_inventory(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'big.json'
            path.write_text(' ' * 129)
            with mock.patch.object(readiness, 'MAX_BYTES', 128):
                with self.assertRaises(ValueError):
                    readiness.read_json(path)


if __name__ == '__main__':
    unittest.main()

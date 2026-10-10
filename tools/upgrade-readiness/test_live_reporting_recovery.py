# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Fail-closed evidence checks for the live persistence recovery fixture."""
import copy
import io
from pathlib import Path
import unittest
from unittest import mock

import live_reporting_recovery as recovery


def workflow():
    return dict(
        metadata=dict(
            name='source',
            uid='immutable',
            namespace=recovery.NAMESPACE,
            labels={'pipeline/runid': 'run-id'}),
        spec=dict(suspend=True),
        status=dict(phase='Succeeded'))


def run():
    return dict(
        run_id='run-id',
        experiment_id='experiment',
        recurring_run_id='schedule',
        state='RUNNING')


class ReportingRecoveryTest(unittest.TestCase):

    def test_timeout_diagnostics_never_include_raw_specs_or_env_values(self):
        obj = dict(
            spec=dict(
                template=dict(
                    spec=dict(containers=[
                        dict(
                            name='ml-pipeline-scheduledworkflow',
                            env=[
                                dict(name='TOKEN', value='secret-value'),
                                dict(name='NAMESPACE', value='secret-value')
                            ])
                    ]))))
        with mock.patch.object(
                recovery,
                'get',
                side_effect=[
                    dict(items=[
                        dict(
                            status=dict(phase='Running'),
                            spec=dict(secret='secret-value'))
                    ]), obj, obj
                ]):
            report = recovery.timeout_diagnostics()
        self.assertNotIn('secret-value', str(report))
        self.assertEqual(report['workflows'], {'Running': 1})
        self.assertEqual(report['controllers']['ml-pipeline-scheduledworkflow'],
                         ['other'])

    def test_timeout_emits_phase_and_preserves_failure_if_diagnostics_fail(
            self):
        output = io.StringIO()
        with mock.patch.object(recovery.time, 'monotonic', side_effect=[0, 2]), \
             mock.patch.object(recovery, 'timeout_diagnostics', side_effect=RuntimeError('secret')), \
             mock.patch.object(recovery.sys, 'stderr', output):
            with self.assertRaisesRegex(ValueError,
                                        'source_recurring_run_deadline'):
                recovery.wait_for(
                    lambda: False, 1, phase='source_recurring_run')
        self.assertIn('source_recurring_run', output.getvalue())
        self.assertNotIn('secret', output.getvalue())

    def test_partial_ownership_evidence_survives_wait_failure(self):
        resource = 'runs/' + recovery.NAMESPACE + '/source-run'
        findings = [
            dict(
                resource=resource,
                status='review_required',
                rule='ownership.stored_identity_unreadable')
        ]
        coverage = dict(completed_scopes=['runs/' + recovery.NAMESPACE])
        with mock.patch('ownership_diagnostics.collect', return_value=(findings, coverage)), \
             mock.patch.object(recovery, 'write_object') as write:
            self.assertIsNone(
                recovery.ownership_evidence(mock.Mock(),
                                            [dict(run_id='source-run')],
                                            Path('/evidence')))
        self.assertEqual(write.call_args.args[0].name,
                         'reporting-ownership.json')
        self.assertEqual(write.call_args.args[1]['findings'], findings)

    def test_incomplete_collection_fails_without_claiming_identity_missing(
            self):
        failure = dict(
            resource='runs/' + recovery.NAMESPACE,
            status='unknown',
            rule='ownership.collection_request_budget_exceeded')
        with mock.patch('ownership_diagnostics.collect', return_value=([failure], dict(completed_scopes=[]))), \
             mock.patch.object(recovery, 'write_object') as write:
            with self.assertRaisesRegex(ValueError, 'collection_incomplete'):
                recovery.ownership_evidence(mock.Mock(),
                                            [dict(run_id='source-run')],
                                            Path('/evidence'))
        self.assertEqual(write.call_count, 1)
        self.assertEqual(write.call_args.args[1]['findings'], [failure])

    def test_fault_preserves_other_permissions(self):
        rules = [
            dict(
                apiGroups=['argoproj.io'],
                resources=['workflows'],
                verbs=['get', 'list', 'watch', 'patch']),
            dict(apiGroups=[''], resources=['pods'], verbs=['get'])
        ]
        original = copy.deepcopy(rules)
        result = recovery.without_workflow_get(rules)
        self.assertEqual(rules, original)
        self.assertEqual(result[0]['verbs'], ['list', 'watch', 'patch'])
        self.assertEqual(result[1], rules[1])

    def test_refuses_broad_or_missing_rule(self):
        for rules in ([], [
                dict(
                    apiGroups=['argoproj.io'],
                    resources=['workflows', 'other'],
                    verbs=['get'])
        ]):
            with self.assertRaises(ValueError):
                recovery.without_workflow_get(rules)

    def test_source_requires_suspended_nonterminal_run(self):
        client = mock.Mock()
        client.get.return_value = run()
        wf = workflow()
        with mock.patch.object(recovery, 'get', return_value=dict(items=[wf])):
            self.assertEqual(
                recovery.capture(client, ['run-id'])['runs'][0]['workflow_uid'],
                'immutable')
            wf['spec']['suspend'] = False
            with self.assertRaisesRegex(ValueError, 'not_suspended'):
                recovery.capture(client, ['run-id'])

    def test_source_refuses_duplicate_or_missing_run(self):
        with self.assertRaisesRegex(ValueError, 'source_run_count'):
            recovery.capture(mock.Mock(), ['a', 'a'])
        with mock.patch.object(recovery, 'get', return_value=dict(items=[])):
            with self.assertRaises(ValueError):
                recovery.capture(mock.Mock(), ['a'])

    def test_does_not_claim_blockage_when_api_already_finished(self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        obj['state'] = 'SUCCEEDED'
        client = mock.Mock()
        client.get.return_value = obj
        with mock.patch.object(recovery, 'get', return_value=wf):
            with self.assertRaisesRegex(ValueError, 'did_not_block'):
                recovery.observe(client, [record])
            self.assertTrue(recovery.observe(client, [record], recovered=True))

    def test_recovery_refuses_replacement_workflow_or_changed_owner(self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        client = mock.Mock()
        client.get.return_value = obj
        with mock.patch.object(recovery, 'get', return_value=wf):
            wf['metadata']['uid'] = 'replacement'
            with self.assertRaisesRegex(ValueError, 'identity_changed'):
                recovery.observe(client, [record], recovered=True)
            wf['metadata']['uid'] = 'immutable'
            obj['recurring_run_id'] = 'different-schedule'
            with self.assertRaisesRegex(ValueError, 'identity_changed'):
                recovery.observe(client, [record], recovered=True)

    def test_recovery_waits_for_api_and_rejects_wrong_final_state(self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        client = mock.Mock()
        client.get.return_value = obj
        with mock.patch.object(recovery, 'get', return_value=wf):
            self.assertIsNone(
                recovery.observe(client, [record], recovered=True))
            obj['state'] = 'FAILED'
            with self.assertRaisesRegex(ValueError, 'wrong_terminal'):
                recovery.observe(client, [record], recovered=True)

    def test_prepare_disables_job_when_source_run_never_appears(self):
        client = mock.Mock()
        client.post.side_effect = [
            dict(id='experiment'),
            dict(run=dict(id='immediate')),
            dict(id='schedule'), {}, {}
        ]
        with mock.patch.object(recovery, 'write_object'), \
             mock.patch.object(Path, 'exists', return_value=False), \
             mock.patch.object(recovery, 'wait_for', side_effect=ValueError('timeout')):
            with self.assertRaisesRegex(ValueError, 'timeout'):
                recovery.prepare_pair(client, mock.Mock(), Path('/unused'))
        self.assertEqual(client.post.call_args.args[0],
                         '/apis/v1beta1/jobs/schedule/disable')
        immediate = client.post.call_args_list[1].args[1]
        self.assertIn('"suspend": true',
                      immediate['pipeline_spec']['workflow_manifest'])

    def test_proxy_evidence_uses_allowed_readiness_transport_path(self):
        proxy = recovery.Client('http://127.0.0.1:18888')
        response = io.BytesIO(b'{"namespace":"kfp-readiness-test","runs":[]}')
        response.status = 200
        with mock.patch.object(proxy._opener, 'open', return_value=response) as opened, \
             mock.patch.object(recovery, 'get', return_value=dict(items=[])):
            recovery.deleted_evidence(mock.Mock(), proxy, [])
        self.assertEqual(
            opened.call_args.args[0].full_url,
            'http://127.0.0.1:18888/apis/v2beta1/reporting-fixture-evidence')

    def test_deleted_report_requires_api_success_and_exact_captured_identity(
            self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        item = dict(
            record,
            deleted=True,
            upstream_code='NotFound',
            deletion_failed=False)
        client, proxy = mock.Mock(), mock.Mock()
        client.get.return_value = obj
        proxy.get.return_value = dict(namespace=recovery.NAMESPACE, runs=[item])
        with mock.patch.object(recovery, 'get', return_value=dict(items=[])):
            self.assertIsNone(
                recovery.deleted_evidence(client, proxy, [record]))
            obj['state'] = 'SUCCEEDED'
            self.assertTrue(recovery.deleted_evidence(client, proxy, [record]))
            item['workflow_uid'] = 'replacement'
            with self.assertRaisesRegex(ValueError, 'proxy_identity'):
                recovery.deleted_evidence(client, proxy, [record])

    def test_deleted_report_cannot_pass_without_actual_deletion(self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        obj['state'] = 'SUCCEEDED'
        item = dict(record, deleted=True, upstream_code='OK')
        client, proxy = mock.Mock(), mock.Mock()
        client.get.return_value = obj
        proxy.get.return_value = dict(namespace=recovery.NAMESPACE, runs=[item])
        with mock.patch.object(recovery, 'get', return_value=dict(items=[wf])):
            with self.assertRaisesRegex(ValueError, 'still_exists'):
                recovery.deleted_evidence(client, proxy, [record])
        item['deleted'] = False
        with mock.patch.object(recovery, 'get', return_value=dict(items=[])):
            self.assertIsNone(
                recovery.deleted_evidence(client, proxy, [record]))

    def test_deleted_report_rejects_security_errors(self):
        obj, wf = run(), workflow()
        record = recovery.project(wf, obj)
        client, proxy = mock.Mock(), mock.Mock()
        proxy.get.return_value = dict(
            namespace=recovery.NAMESPACE,
            runs=[dict(record, deleted=True, upstream_code='PermissionDenied')])
        with mock.patch.object(recovery, 'get', return_value=dict(items=[])):
            with self.assertRaisesRegex(ValueError, 'captured_report_rejected'):
                recovery.deleted_evidence(client, proxy, [record])

    def test_restores_rbac_when_resume_fails(self):
        rules = [
            dict(
                apiGroups=['argoproj.io'],
                resources=['workflows'],
                verbs=['get', 'list'])
        ]
        role = dict(rules=rules)
        with mock.patch.object(recovery, 'get', side_effect=[role, RuntimeError()]), \
             mock.patch.object(recovery, 'worker_pods', return_value=['worker']), \
             mock.patch.object(recovery, 'permitted', return_value=True), \
             mock.patch.object(recovery, 'wait_for'), \
             mock.patch.object(recovery, 'write_object'), \
             mock.patch.object(recovery, 'kube') as kube:
            with self.assertRaises(RuntimeError):
                recovery.recover(mock.Mock(),
                                 dict(runs=[dict(workflow_name='a')]),
                                 mock.MagicMock())
        self.assertEqual(kube.call_count, 2)
        self.assertIn('"get", "list"', kube.call_args.args[-1])


if __name__ == '__main__':
    unittest.main()

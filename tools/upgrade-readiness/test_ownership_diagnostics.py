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
import unittest
from unittest import mock

from kfp_http import CollectionError
import ownership_diagnostics as ownership
import readiness


class OwnershipTest(unittest.TestCase):

    def assess(self, run=None, detail=None, experiment=None, schedule=None):
        run = run if run is not None else dict(
            run_id='run', experiment_id='exp', state='RUNNING')
        manifest = dict(
            kind='Workflow',
            metadata=dict(name='wf', namespace='team', uid='uid'),
            spec=dict(secret='never-print-this'))
        default_detail = dict(
            run=dict(
                id='run',
                resource_references=[
                    dict(
                        relationship='OWNER',
                        key=dict(type='NAMESPACE', id='team'))
                ]),
            pipeline_runtime=dict(workflow_manifest=json.dumps(manifest)))
        responses = {
            '/apis/v2beta1/runs':
                dict(runs=[run]),
            '/apis/v2beta1/recurringruns':
                dict(recurring_runs=[schedule] if schedule else []),
            '/apis/v2beta1/experiments/exp':
                experiment if experiment is not None else dict(
                    experiment_id='exp', namespace='team'),
            '/apis/v1beta1/runs/run':
                detail if detail is not None else default_detail,
        }
        client = mock.Mock()

        def get(path, params=None):
            response = responses[path]
            if isinstance(response, Exception):
                raise response
            return response

        client.get.side_effect = get
        findings, coverage = ownership.collect(client, ['team'])
        return findings, coverage, client

    def reasons(self, findings):
        return {f['evidence']: f['status'] for f in findings}

    def test_present_evidence_does_not_claim_recovery_or_emit_manifest(self):
        findings, coverage, _ = self.assess()
        self.assertEqual(set(self.reasons(findings).values()), {'observed'})
        self.assertIn('stored_identity_present', self.reasons(findings))
        self.assertNotIn('never-print-this', json.dumps(findings))
        self.assertEqual(coverage['completed_scopes'],
                         ['runs/team', 'recurringruns/team'])

    def test_missing_namespace_and_identity_are_distinct(self):
        findings, _, _ = self.assess(detail=dict(run=dict(id='run')))
        self.assertEqual(
            self.reasons(findings)['missing_namespace_reference'],
            'review_required')
        self.assertEqual(
            self.reasons(findings)['missing_stored_identity'],
            'review_required')
        self.assertIn('experiment_namespace_present', self.reasons(findings))

    def test_missing_experiment_namespace(self):
        findings, _, _ = self.assess(experiment=dict(experiment_id='exp'))
        self.assertIn('missing_experiment_namespace', self.reasons(findings))

    def test_missing_experiment_reference(self):
        findings, _, _ = self.assess(run=dict(run_id='run', state='RUNNING'))
        self.assertIn('missing_experiment_reference', self.reasons(findings))

    def test_denied_or_not_found_is_unknown_not_missing(self):
        for reason in ('http_403', 'http_404', 'request_timeout'):
            with self.subTest(reason=reason):
                findings, _, _ = self.assess(
                    experiment=CollectionError(reason),
                    detail=CollectionError(reason))
                self.assertEqual(
                    set(self.reasons(findings).values()), {'unknown'})
                self.assertNotIn('missing_stored_identity',
                                 self.reasons(findings))

    def test_stored_uid_is_required(self):
        findings, _, _ = self.assess(
            detail=dict(
                run=dict(id='run'),
                pipeline_runtime=dict(
                    pipeline_manifest=json.dumps(
                        dict(kind='Workflow', metadata=dict(name='wf'))))))
        self.assertIn('incomplete_stored_identity', self.reasons(findings))

    def test_pipeline_manifest_fallback(self):
        findings, _, _ = self.assess(
            detail=dict(
                run=dict(id='run'),
                pipeline_runtime=dict(
                    pipelineManifest=json.dumps(
                        dict(
                            kind='Workflow',
                            metadata=dict(name='wf', uid='uid'))))))
        self.assertIn('stored_identity_present', self.reasons(findings))

    def test_stored_identity_allows_omitted_type_metadata(self):
        # Typed informer objects can omit TypeMeta when persisted. The server
        # decodes these into Workflow without requiring kind/apiVersion.
        for kind in (None, '', 'Workflow'):
            with self.subTest(kind=kind):
                manifest = dict(
                    metadata=dict(name='wf', uid='uid', namespace='team'))
                if kind is not None:
                    manifest['kind'] = kind
                findings, _, _ = self.assess(
                    detail=dict(
                        run=dict(id='run'),
                        pipeline_runtime=dict(
                            workflow_manifest=json.dumps(manifest))))
                self.assertEqual(
                    self.reasons(findings)['stored_identity_present'],
                    'observed')

    def test_explicit_wrong_kind_is_unknown(self):
        findings, _, _ = self.assess(
            detail=dict(
                run=dict(id='run'),
                pipeline_runtime=dict(
                    workflow_manifest=json.dumps(
                        dict(kind='Pod', metadata=dict(name='wf',
                                                       uid='uid'))))))
        self.assertEqual(
            self.reasons(findings)['stored_identity_unreadable'], 'unknown')

    def test_invalid_manifest_is_unknown(self):
        findings, _, _ = self.assess(
            detail=dict(
                run=dict(id='run'),
                pipeline_runtime=dict(workflow_manifest='bad')))
        self.assertEqual(
            self.reasons(findings)['stored_identity_unreadable'], 'unknown')

    def test_terminal_skipped_but_canceling_remains_in_scope(self):
        for state in ('SUCCEEDED', 'FAILED', 'CANCELED', 'SKIPPED', 3, 4, 5, 7):
            findings, _, client = self.assess(
                run=dict(run_id='run', state=state))
            self.assertEqual(findings, [])
            self.assertEqual(client.get.call_count, 2)
        for state in ('CANCELING', 6, 'PAUSED', 8, None):
            findings, _, _ = self.assess(run=dict(run_id='run', state=state))
            self.assertIn('stored_identity_present', self.reasons(findings))

    def test_schedule_identity_not_exposed_even_when_disabled(self):
        findings, _, _ = self.assess(
            schedule=dict(
                recurring_run_id='job',
                experiment_id='exp',
                mode='DISABLE',
                namespace='team'))
        self.assertEqual(
            self.reasons(findings)['schedule_stored_identity_not_exposed'],
            'unknown')

    def test_list_failure_and_repeated_token_incomplete(self):
        client = mock.Mock()
        client.get.return_value = dict(next_page_token='same')
        findings, coverage = ownership.collect(client, ['team'])
        self.assertEqual(coverage['completed_scopes'], [])
        self.assertEqual(len(findings), 2)
        self.assertIn('collection_repeated_page_token', self.reasons(findings))

    def test_record_budget_preserves_findings_and_marks_scope_incomplete(self):
        with mock.patch.object(ownership, 'MAX_RECORDS', 1):
            findings, coverage, _ = self.assess(
                schedule=dict(recurring_run_id='job'))
        self.assertIn('stored_identity_present', self.reasons(findings))
        self.assertIn('collection_record_limit', self.reasons(findings))
        self.assertEqual(coverage['completed_scopes'], ['runs/team'])
        self.assertEqual(coverage['records_examined'], 1)

    def test_request_budget_is_unknown_not_empty_success(self):
        client = mock.Mock()
        client.get.side_effect = CollectionError('request_budget_exceeded')
        findings, coverage = ownership.collect(client, ['team'])
        self.assertEqual(len(findings), 2)
        self.assertEqual(coverage['completed_scopes'], [])
        self.assertEqual(set(self.reasons(findings).values()), {'unknown'})

    def test_malformed_empty_runtime_and_references_are_unknown(self):
        for detail in (dict(run=dict(id='run'), pipeline_runtime=[]),
                       dict(run=dict(id='run', resource_references={}))):
            with self.subTest(detail=detail):
                findings, _, _ = self.assess(detail=detail)
                reasons = self.reasons(findings)
                self.assertNotIn('missing_stored_identity', reasons)
                self.assertTrue(
                    any(
                        reason.startswith('invalid_') and status == 'unknown'
                        for reason, status in reasons.items()))

    def test_cli_without_schedule_policy(self):
        client = mock.Mock()
        client.get.side_effect = CollectionError('access_denied')
        with mock.patch.object(
                readiness, 'Client',
                return_value=client), contextlib.redirect_stdout(
                    io.StringIO()) as output:
            code = readiness.main([
                '--inventory',
                str(Path(__file__).parent / 'examples/inventory.json'),
                '--system-namespace', 'kubeflow', '--source-version', '2.17.2',
                '--include-ownership', '--kfp-endpoint', 'https://kfp.example',
                '--format', 'json'
            ])
        self.assertEqual(code, 2)
        report = json.loads(output.getvalue())
        self.assertIn('ownership_collection', report['source'])
        self.assertTrue(
            any(f['rule'] == 'ownership.collection_access_denied'
                for f in report['findings']))


if __name__ == '__main__':
    unittest.main()

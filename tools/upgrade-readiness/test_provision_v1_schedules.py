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
"""V1 fixture API, identity and partial-activation cleanup contracts."""

import copy
from datetime import datetime
from datetime import timezone
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import uuid

import fixture_http
import live_schedule_check as live
import provision_live_schedules as common
import provision_v1_schedules as v1


class V1Tests(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)
        self.parent = dict(
            context=common.CONTEXT,
            namespace=common.NAMESPACE,
            owner_marker='a' * 32,
            rbac_ready=True,
            enabled=False)

    def prepare(self, client):
        with mock.patch.object(common, 'verify_state'), mock.patch.object(
                common,
                'schedule_identity',
                side_effect=lambda c, uid: 'schedule-' + uid):
            v1.prepare(common.CONTEXT, self.path, self.parent, client)

    def test_real_v1_workflow_payload_and_disabled_jobs(self):
        client = mock.Mock()
        client.post.side_effect = [dict(
            id='experiment')] + [dict(id='job-' + str(i)) for i in range(3)]
        self.prepare(client)
        calls = client.post.call_args_list
        self.assertEqual(calls[0].args[0], '/apis/v1beta1/experiments')
        self.assertEqual(calls[0].args[1]['resource_references'],
                         [v1.reference('NAMESPACE', common.NAMESPACE)])
        for i, call in enumerate(calls[1:]):
            path, payload = call.args
            self.assertEqual(path, '/apis/v1beta1/jobs')
            self.assertIs(payload['enabled'], False)
            self.assertEqual(payload['resource_references'][0],
                             v1.reference('EXPERIMENT', 'experiment'))
            self.assertNotIn('pipeline_spec', payload['pipeline_spec'])
            manifest = json.loads(payload['pipeline_spec']['workflow_manifest'])
            self.assertEqual(manifest['kind'], 'Workflow')
            self.assertNotIn('serviceAccountName', manifest['spec'])
            if i == 0:
                self.assertNotIn('service_account', payload)
            else:
                self.assertEqual(payload['service_account'], common.ACCOUNTS[i])
        state = common.read_object(self.path / 'state.json')
        self.assertTrue(state['prepared'])
        self.assertEqual(state['api_version'], 'v1beta1')
        self.assertEqual(len(state['schedules']), 3)

    def test_partial_creation_keeps_disabled_ids_for_cleanup(self):
        client = mock.Mock()
        client.post.side_effect = [
            dict(id='experiment'),
            dict(id='job-0'),
            common.FixtureError('failure')
        ]
        with self.assertRaises(common.FixtureError):
            self.prepare(client)
        state = common.read_object(self.path / 'state.json')
        self.assertNotIn('prepared', state)
        self.assertEqual(state['schedules'][0]['schedule_uid'], 'job-0')
        client.post.reset_mock(side_effect=True)
        common.set_enabled(self.path, state, v1.ActivationClient(client), False)
        client.post.assert_called_once_with('/apis/v1beta1/jobs/job-0/disable',
                                            {})

    def test_partial_enable_rolls_back_every_v1_job(self):
        state = dict(
            prepared=True,
            schedules=[dict(schedule_uid='job-' + str(i)) for i in range(3)])
        client = mock.Mock()
        client.post.side_effect = [
            dict(),
            common.FixtureError('failed'),
            dict(),
            dict(),
            dict()
        ]
        with self.assertRaises(common.FixtureError):
            common.set_enabled(self.path, state, v1.ActivationClient(client),
                               True)
        self.assertEqual([c.args[0] for c in client.post.call_args_list], [
            '/apis/v1beta1/jobs/job-0/enable',
            '/apis/v1beta1/jobs/job-1/enable',
            '/apis/v1beta1/jobs/job-0/disable',
            '/apis/v1beta1/jobs/job-1/disable',
            '/apis/v1beta1/jobs/job-2/disable'
        ])

    def test_parent_ownership_and_disabled_state_required(self):
        client = mock.Mock()
        with mock.patch.object(
                common,
                'verify_state',
                side_effect=common.FixtureError('ownership')):
            with self.assertRaises(common.FixtureError):
                v1.prepare(common.CONTEXT, self.path, self.parent, client)
        with mock.patch.object(common, 'verify_state'):
            self.parent['enabled'] = True
            with self.assertRaises(common.FixtureError):
                v1.prepare(common.CONTEXT, self.path, self.parent, client)
        client.post.assert_not_called()

    def test_transport_allows_only_exact_v1_mutation_paths(self):
        token = self.path / 'token'
        token.write_text('token')
        client = fixture_http.FixtureClient('http://127.0.0.1:8888', str(token))
        with mock.patch.object(
                fixture_http.http.client,
                'HTTPConnection',
                side_effect=RuntimeError('transport')) as connection:
            for path in ('/apis/v1beta1/jobs', '/apis/v1beta1/experiments',
                         '/apis/v1beta1/jobs/id/enable',
                         '/apis/v1beta1/jobs/id/disable'):
                with self.assertRaisesRegex(RuntimeError, 'transport'):
                    client.post(path, {})
            connection.reset_mock()
            for path in ('/apis/v1beta1/jobs/id/delete',
                         '/apis/v1beta1/jobs/../enable',
                         '/apis/v1beta1/jobs/id/enable?x=1',
                         '/apis/v1beta1/runs',
                         '/apis/v1beta1/experiments/id/enable'):
                with self.assertRaises(fixture_http.FixtureError):
                    client.post(path, {})
            connection.assert_not_called()

    def test_v1_run_read_view_still_requires_exact_account_and_job(self):
        case = dict(
            schedule_uid='v1-job',
            service_account='readiness-granted',
            baseline_run_ids=[])
        record = dict(
            run_id='run',
            recurring_run_id='v1-job',
            experiment_id='experiment',
            service_account='readiness-granted',
            created_at='2026-01-01T00:01:00Z',
            state='SUCCEEDED')
        client = mock.Mock()
        client.get.side_effect = lambda path, *args: dict(
            experiment_id='experiment', namespace=common.NAMESPACE
        ) if '/experiments/' in path else dict(runs=[record])
        start = datetime(2026, 1, 1, tzinfo=timezone.utc)
        self.assertEqual(
            live.run_evidence(client, common.NAMESPACE, case, start),
            [dict(run_id='run', state='SUCCEEDED')])
        for key, value in [('service_account', 'other'),
                           ('recurring_run_id', 'v2-job')]:
            changed = copy.deepcopy(record)
            record[key] = value
            with self.assertRaises(ValueError):
                live.run_evidence(client, common.NAMESPACE, case, start)
            record.clear()
            record.update(changed)

    def test_workflow_evidence_must_match_strict_successful_api_runs(self):
        state = dict(
            enabled=False,
            prepared=True,
            activation_start='2026-01-01T00:00:00Z',
            schedules=[dict(scenario='default', schedule_uid='v1-job')])
        run = dict(run_id='run', state='SUCCEEDED')
        with mock.patch.object(
                live, 'run_evidence', return_value=[run]), mock.patch.object(
                    v1,
                    'target_workflow_evidence',
                    return_value=[dict(run,
                                       workflow_uid='workflow')]) as workflow:
            report = v1.verify_execution(state, mock.Mock())
            self.assertEqual(report['outcome'], 'passed')
            self.assertFalse(report['pre_upgrade_prediction_validated'])
            workflow.return_value = [dict(run, run_id='other')]
            with self.assertRaises(common.FixtureError):
                v1.verify_execution(state, mock.Mock())
        for scenario, runs in [('default', []),
                               ('default', [dict(run, state='RUNNING')]),
                               ('denied', [run])]:
            state['schedules'][0]['scenario'] = scenario
            with mock.patch.object(
                    live, 'run_evidence', return_value=runs), mock.patch.object(
                        v1, 'target_workflow_evidence', return_value=runs):
                with self.assertRaises(common.FixtureError):
                    v1.verify_execution(state, mock.Mock())

    def test_target_workflow_uses_run_uuid_not_api_display_name(self):
        run_id = '3ec3b3f2-9ec4-5d4b-af55-2186719bcdec'
        expected_name = 'run-' + str(
            uuid.uuid5(
                uuid.UUID('c2f3a9d4-1e6b-4c8a-9f7d-0b5e3a1c2d4f'), run_id))
        case = dict(
            schedule_uid='job',
            schedule_name='schedule',
            service_account='readiness-granted',
            baseline_run_ids=[])
        run = dict(
            run_id=run_id,
            display_name='schedule-123',
            created_at='2026-01-01T00:01:00Z',
            service_account='readiness-granted',
            state='SUCCEEDED')
        workflow = dict(
            metadata=dict(
                name=expected_name,
                namespace=common.NAMESPACE,
                uid='workflow-uid',
                creationTimestamp='2026-01-01T00:01:01Z',
                labels={'pipeline/runid': run_id},
                ownerReferences=[
                    dict(
                        uid='job',
                        name='schedule',
                        kind='ScheduledWorkflow',
                        controller=True)
                ]),
            spec=dict(serviceAccountName='readiness-granted'),
            status=dict(phase='Succeeded'))
        start = datetime(2026, 1, 1, tzinfo=timezone.utc)
        with mock.patch.object(live, 'list_runs', return_value=[run]):
            get = mock.Mock(return_value=(dict(items=[workflow]), None))
            result = v1.target_workflow_evidence(
                mock.Mock(), common.NAMESPACE, case, start, get=get)
            self.assertEqual(result, [
                dict(
                    run_id=run_id,
                    workflow_uid='workflow-uid',
                    workflow_name=expected_name,
                    state='SUCCEEDED')
            ])
            self.assertNotEqual(expected_name, run['display_name'])
            # A matching run label alone is insufficient. Require deterministic
            # name, fresh live UID, namespace, exact owner, account and success.
            mutations = [
                ('metadata', 'name', run['display_name']),
                ('metadata', 'namespace', 'other'),
                ('metadata', 'uid', ''),
                ('metadata', 'creationTimestamp', '2025-01-01T00:00:00Z'),
                ('metadata', 'labels', {
                    'pipeline/runid': 'other'
                }),
                ('metadata', 'ownerReferences', [
                    dict(
                        uid='other',
                        name='schedule',
                        kind='ScheduledWorkflow',
                        controller=True)
                ]),
                ('metadata', 'ownerReferences', [
                    dict(
                        uid='job',
                        name='other',
                        kind='ScheduledWorkflow',
                        controller=True)
                ]),
                ('spec', 'serviceAccountName', 'other'),
                ('status', 'phase', 'Running'),
            ]
            for section, key, value in mutations:
                changed = copy.deepcopy(workflow)
                changed[section][key] = value
                get.return_value = (dict(items=[changed]), None)
                with self.subTest(
                        section=section,
                        key=key), self.assertRaises(ValueError):
                    v1.target_workflow_evidence(
                        mock.Mock(), common.NAMESPACE, case, start, get=get)
            get.return_value = (dict(items=[workflow, workflow]), None)
            with self.assertRaisesRegex(ValueError, 'identity_unavailable'):
                v1.target_workflow_evidence(
                    mock.Mock(), common.NAMESPACE, case, start, get=get)
            get.return_value = (dict(items=[workflow]), None)
            run['service_account'] = 'other'
            with self.assertRaisesRegex(ValueError, 'account_mismatch'):
                v1.target_workflow_evidence(
                    mock.Mock(), common.NAMESPACE, case, start, get=get)

    def test_v1_denial_requires_fresh_exact_controller_event(self):
        case = dict(
            schedule_uid='v1-job',
            schedule_name='v1-name',
            service_account='readiness-denied',
            baseline_event_counts={'event': 1})
        event = dict(
            metadata=dict(uid='event'),
            count=2,
            type='Warning',
            reason='Failed',
            involvedObject=dict(
                uid='v1-job',
                name='v1-name',
                namespace=common.NAMESPACE,
                kind='ScheduledWorkflow'),
            source=dict(component='scheduled-workflow-controller'),
            lastTimestamp='2026-01-01T00:01:00Z',
            message='code = PermissionDenied service account authorization error '
            'Namespace:' + common.NAMESPACE +
            ',Verb:use,Group:,Resource:serviceaccounts,Subresource:,Name:readiness-denied,'
        )
        start = datetime(2026, 1, 1, tzinfo=timezone.utc)
        self.assertTrue(live.denied([event], common.NAMESPACE, case, start))
        for key, value in [
            ('count', 1), ('lastTimestamp', '2025-01-01T00:00:00Z'),
            ('message', 'code = PermissionDenied unrelated authorization error')
        ]:
            self.assertFalse(
                live.denied([dict(event, **{key: value})], common.NAMESPACE,
                            case, start))


if __name__ == '__main__':
    unittest.main()

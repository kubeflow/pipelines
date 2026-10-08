# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Regression checks for populated adoption evidence, without a cluster."""

import copy
import json
from pathlib import Path
import subprocess
import unittest
from unittest import mock

from live_adoption_check import AdoptionError
from live_adoption_check import command_diagnostics
from live_adoption_check import command_failure
from live_adoption_check import kube
from live_adoption_check import offline_job
from live_adoption_check import require_stopped
from live_adoption_check import snapshot
from live_adoption_check import sql
from live_adoption_check import validate_adoption
from live_adoption_check import validate_continuation
from live_adoption_check import validate_idempotent


def inventory():
    fixture = {'schedules': [{'schedule_uid': str(i)} for i in range(3)]}
    before = dict(jobs=[], runs=[], schedules=[], workflows=[])
    after = dict(
        states=[],
        receipts=[
            dict(
                ID='legacy-2.18',
                Ready=1,
                AdoptedCount=3,
                CompletedAt=200,
                JobIDs='["0","1","2"]')
        ])
    for i in range(3):
        uid = str(i)
        before['jobs'].append(
            dict(
                UUID=uid,
                Enabled=int(i == 0),
                PipelineSpecManifest='ir',
                RuntimeParameters='{}'))
        before['runs'].append(
            dict(
                UUID='r' + uid,
                Name='w' + uid,
                JobUUID=uid,
                ScheduledAtInSec=100,
                CreatedAtInSec=100,
                State='RUNNING' if i == 0 else 'SUCCEEDED',
                Conditions=''))
        before['schedules'].append(
            dict(
                uid=uid,
                enabled=i == 0,
                trigger=dict(
                    lastWorkflowIndex=1,
                    lastTriggeredTime='1970-01-01T00:01:40Z')))
        before['workflows'].append(
            dict(
                uid='wuid' + uid,
                name='w' + uid,
                owners=[uid],
                run_id='r' + uid,
                index=1,
                suspended=i == 0))
        after['states'].append(
            dict(
                JobUUID=uid,
                LastRunUUID='r' + uid,
                LastRunIndex=1,
                LastScheduledAtInSec=100,
                Pending=0))
    after.update(copy.deepcopy(before))
    return fixture, before, after


class AdoptionTests(unittest.TestCase):

    def test_offline_environment_setup_never_waits_for_live_api(self):
        root = Path(__file__).resolve().parents[2]
        shared = (
            root /
            '.github/resources/scripts/readiness-schedules.sh').read_text()
        helper = shared.split('set_api_env() {',
                              1)[1].split('configure_controllers() {', 1)[0]
        script = (
            root /
            '.github/resources/scripts/readiness-adoption.sh').read_text()
        offline = script.split('else\n',
                               1)[1].split('  for attempt in first repeat;',
                                           1)[0]
        # A live configure call fails this execution. Offline setup must first
        # prove stopped writers, then only mutate deployment environment.
        harness = """set -eu
stopped=false
check() { [[ "$1" == stopped ]]; stopped=true; }
configure_api() { exit 91; }
configure_controllers() { :; }
kube() {
  [[ "$stopped" == true ]] || exit 92
  [[ "$*" == '-n kubeflow set env deployment/ml-pipeline '* ]] || exit 93
}
set_api_env() {""" + helper + offline
        subprocess.run(['bash', '-c', harness], check=True, timeout=5)
        resumed = script.split('scale deployment/ml-pipeline --replicas=1',
                               1)[1]
        self.assertLess(
            resumed.index('configure_api enforce'),
            resumed.index('for controller in'))

    def test_offline_fence_rejects_live_and_terminating_writers(self):
        deployment = dict(
            spec=dict(replicas=0, selector=dict(matchLabels={'app': 'api'})),
            status=dict(replicas=0))
        with mock.patch('live_adoption_check.get', return_value=deployment), \
                mock.patch('live_adoption_check.kube', return_value='{"items":[]}'):
            require_stopped()
        for pods in [{
                'items': [{}]
        }, {
                'items': [{
                    'metadata': {
                        'deletionTimestamp': 'now'
                    }
                }]
        }]:
            with mock.patch('live_adoption_check.get', return_value=deployment), \
                    mock.patch('live_adoption_check.kube', return_value=json.dumps(pods)):
                with self.assertRaisesRegex(AdoptionError,
                                            'writer_pods_not_terminated'):
                    require_stopped()
        deployment['spec']['replicas'] = 1
        with mock.patch('live_adoption_check.get', return_value=deployment):
            with self.assertRaisesRegex(AdoptionError, 'writers_not_stopped'):
                require_stopped()

    def test_unknown_mysql_code_and_exec_failure_keep_only_structural_data(
            self):
        payload = 'SECRET-server-query-identity'
        info = command_diagnostics(
            mock.Mock(
                returncode=1,
                stdout='',
                stderr='ERROR 2002 (hy000): Cannot connect ' + payload))
        self.assertEqual(info['mysql_error'], 2002)
        self.assertNotIn(payload, json.dumps(info))
        info = command_diagnostics(
            mock.Mock(
                returncode=127,
                stdout='',
                stderr='OCI runtime exec failed: executable file not found in $PATH '
                + payload))
        self.assertIsNone(info['mysql_error'])
        self.assertEqual(info['exit_code'], 127)
        self.assertNotIn(payload, json.dumps(info))

    def test_sql_uses_working_loopback_tcp_instead_of_source_default_socket(
            self):
        with mock.patch(
                'live_adoption_check.subprocess.run',
                return_value=mock.Mock(
                    returncode=0, stdout='{"UUID":"fixture"}\n',
                    stderr='')) as run:
            self.assertEqual(sql('jobs', ('UUID',)), [{'UUID': 'fixture'}])
        command = run.call_args.args[0]
        client = command[command.index('--') + 1:]
        self.assertEqual(
            client[:4],
            ['mysql', '-uroot', '--protocol=TCP', '--host=127.0.0.1'])
        self.assertIn('deployment/mysql', command)
        self.assertIn('mlpipeline', client)
        self.assertEqual(run.call_args.kwargs['timeout'], 45)
        self.assertTrue(client[-1].endswith('ORDER BY `UUID`'))
        self.assertNotIn('ORDER BY 1', client[-1])

    def test_sql_diagnostics_identify_stage_without_echoing_payload(self):
        self.assertEqual(
            command_failure(
                'jobs',
                "ERROR 1054 (42S22): unknown column secret-raw-payload"),
            'fixture_sql_jobs_column_missing_failed')
        self.assertEqual(
            command_failure(
                'run_details',
                "ERROR 9999 (HY000): unrecognized secret-raw-payload"),
            'fixture_sql_run_details_command_failed')
        self.assertEqual(
            command_failure('jobs', 'secret token authentication failure'),
            'fixture_sql_jobs_command_failed')
        with mock.patch(
                'live_adoption_check.subprocess.run',
                return_value=mock.Mock(
                    returncode=1,
                    stdout='raw payload',
                    stderr='ERROR 3144 (22032): secret-raw-payload')):
            with self.assertRaisesRegex(
                    AdoptionError,
                    '^fixture_sql_jobs_json_character_set_failed$'):
                kube('exec', operation='jobs')

    def test_reads_real_workflow_label_and_progress_field(self):
        objects = [
            dict(items=[
                dict(
                    metadata=dict(uid='s', name='schedule'),
                    spec=dict(enabled=True),
                    status=dict(
                        trigger=dict(
                            lastWorkflowIndex=7,
                            lastTriggeredTime='2026-10-08T12:00:00Z')))
            ]),
            dict(items=[
                dict(
                    metadata=dict(
                        uid='w',
                        name='workflow',
                        labels={
                            'scheduledworkflows.kubeflow.org/workflowIndex':
                                '7',
                            'pipeline/runid':
                                'r'
                        },
                        ownerReferences=[
                            dict(
                                kind='ScheduledWorkflow',
                                controller=True,
                                uid='s')
                        ]),
                    spec=dict(suspend=True))
            ])
        ]
        with mock.patch('live_adoption_check.sql', return_value=[]), \
                mock.patch('live_adoption_check.get', side_effect=objects):
            actual = snapshot()
        self.assertEqual(actual['workflows'][0]['index'], 7)
        self.assertEqual(actual['schedules'][0]['trigger']['lastWorkflowIndex'],
                         7)

    def test_rerun_cannot_reseed_or_rewrite_receipt(self):
        _, _, first = inventory()
        validate_idempotent(first, copy.deepcopy(first))
        for collection, key, value in [('receipts', 'CompletedAt', 300),
                                       ('states', 'LastRunIndex', 0)]:
            repeated = copy.deepcopy(first)
            repeated[collection][0][key] = value
            with self.assertRaisesRegex(AdoptionError,
                                        'rerun_changed_adoption'):
                validate_idempotent(first, repeated)

    def test_preserved_populated_inventory_passes(self):
        fixture, before, after = inventory()
        self.assertEqual(
            validate_adoption(before, after, fixture)['AdoptedCount'], 3)
        validate_continuation(before, after, fixture, held=True)

    def test_rejects_reset_or_mutated_adoption(self):
        changes = [
            ('states', 'LastRunIndex', 0, 'historical_progress_reset'),
            ('states', 'LastRunUUID', 'other', 'last_run_identity_changed'),
            ('states', 'LastScheduledAtInSec', 99, 'historical_time_changed'),
            ('jobs', 'PipelineSpecManifest', 'different',
             'stored_definition_changed'),
            ('jobs', 'RuntimeParameters', 'different',
             'stored_definition_changed'),
            ('schedules', 'enabled', False, 'enablement_changed'),
            ('receipts', 'Ready', 0, 'invalid_receipt'),
        ]
        for collection, key, value, reason in changes:
            with self.subTest(collection=collection, key=key):
                fixture, before, after = inventory()
                after[collection][0][key] = value
                with self.assertRaisesRegex(AdoptionError, reason):
                    validate_adoption(before, after, fixture)

    def add_tick(self, value, job='0', index=2):
        value['runs'].append(
            dict(
                UUID='new',
                Name='new-workflow',
                JobUUID=job,
                ScheduledAtInSec=130,
                CreatedAtInSec=131,
                State='SUCCEEDED',
                Conditions=''))
        value['workflows'].append(
            dict(
                uid='new-uid',
                name='new-workflow',
                owners=[job],
                run_id='new',
                index=index,
                suspended=False))

    def test_active_run_must_occupy_concurrency_slot(self):
        fixture, before, after = inventory()
        self.add_tick(after)
        with self.assertRaisesRegex(AdoptionError, 'active_run_escaped'):
            validate_continuation(before, after, fixture, held=True)

    def test_duplicate_tick_rejected(self):
        fixture, before, after = inventory()
        self.add_tick(after, index=1)
        with self.assertRaisesRegex(AdoptionError, 'duplicate_tick'):
            validate_continuation(before, after, fixture)

    def test_disabled_schedule_must_not_fire(self):
        fixture, before, after = inventory()
        self.add_tick(after, job='1')
        with self.assertRaisesRegex(AdoptionError, 'disabled_schedule_fired'):
            validate_continuation(before, after, fixture)

    def test_source_active_and_candidate_runs_must_succeed(self):
        fixture, before, after = inventory()
        self.add_tick(after)
        with self.assertRaisesRegex(AdoptionError,
                                    'source_active_run_not_completed'):
            validate_continuation(before, after, fixture)
        after['runs'][0]['State'] = 'SUCCEEDED'
        validate_continuation(before, after, fixture)
        after['runs'][-1]['State'] = 'FAILED'
        with self.assertRaisesRegex(AdoptionError,
                                    'candidate_tick_not_successful'):
            validate_continuation(before, after, fixture)

    def test_skipped_tick_rejected(self):
        fixture, before, after = inventory()
        self.add_tick(after, index=3)
        after['runs'][0]['State'] = 'SUCCEEDED'
        with self.assertRaisesRegex(AdoptionError,
                                    'candidate_tick_skipped_or_replayed'):
            validate_continuation(before, after, fixture)

    def test_job_preserves_credentials_but_removes_server_probes(self):
        pod = dict(
            serviceAccountName='ml-pipeline',
            volumes=[{
                'name': 'config'
            }],
            containers=[
                dict(
                    name='ml-pipeline',
                    image='candidate',
                    env=[dict(name='MULTIUSER', value='true')],
                    volumeMounts=[dict(name='config', mountPath='/config')],
                    readinessProbe={'httpGet': {}},
                    livenessProbe={'httpGet': {}},
                    startupProbe={'httpGet': {}})
            ])
        original = copy.deepcopy(pod)
        job = offline_job({'spec': {
            'template': {
                'spec': pod
            }
        }}, 'readiness-adopt-first')
        self.assertEqual(pod, original)
        spec = job['spec']['template']['spec']
        self.assertEqual(spec['volumes'], pod['volumes'])
        self.assertEqual(spec['serviceAccountName'], 'ml-pipeline')
        container = spec['containers'][0]
        self.assertIn('--adopt-legacy-recurring-runs', container['args'])
        self.assertEqual(container['env'], pod['containers'][0]['env'])
        self.assertEqual(container['volumeMounts'],
                         pod['containers'][0]['volumeMounts'])
        self.assertFalse(
            set(container)
            & {'readinessProbe', 'livenessProbe', 'startupProbe'})
        self.assertEqual(job['spec']['backoffLimit'], 0)
        self.assertEqual(spec['restartPolicy'], 'Never')


if __name__ == '__main__':
    unittest.main()

# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Regression checks for populated adoption evidence, without a cluster."""

import copy
from datetime import datetime
from datetime import timezone
import json
from pathlib import Path
import subprocess
import unittest
from unittest import mock

from live_adoption_check import adoption_job_diagnostics
from live_adoption_check import adoption_log_categories
from live_adoption_check import adoption_stack_frames
from live_adoption_check import adoption_startup_milestones
from live_adoption_check import AdoptionError
from live_adoption_check import command_diagnostics
from live_adoption_check import command_failure
from live_adoption_check import exception_evidence
from live_adoption_check import kube
from live_adoption_check import offline_job
from live_adoption_check import prepare_active
from live_adoption_check import require_stopped
from live_adoption_check import snapshot
from live_adoption_check import source_progress_shapes
from live_adoption_check import sql
from live_adoption_check import validate_adoption
from live_adoption_check import validate_continuation
from live_adoption_check import validate_idempotent
from live_adoption_check import wait_adoption_job


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
                RuntimeParameters='{}',
                NoCatchup=1,
                IntervalSecond=30))
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
                epoch=100,
                suspended=i == 0))
        after['states'].append(
            dict(
                JobUUID=uid,
                LastRunUUID='r' + uid,
                LastRunIndex=1,
                LastScheduledAtInSec=100,
                LastCreatedAtInSec=100,
                Pending=0))
    after.update(copy.deepcopy(before))
    return fixture, before, after


class AdoptionTests(unittest.TestCase):

    def test_failed_job_stops_without_waiting_for_completion(self):
        with mock.patch('live_adoption_check.get', return_value=dict(status=dict(failed=1))), \
                mock.patch('live_adoption_check.adoption_job_diagnostics', return_value=dict(receipt_logged=False)), \
                mock.patch('live_adoption_check.time.sleep') as sleep:
            with self.assertRaises(AdoptionError) as failure:
                wait_adoption_job('readiness-adopt-first')
        self.assertEqual(failure.exception.job_evidence['job_outcome'],
                         'failed')
        sleep.assert_not_called()

    def test_completed_job_requires_expected_receipt_and_timeout_collects_diagnostics(
            self):
        for receipt in (True, False):
            with mock.patch('live_adoption_check.get', return_value=dict(status=dict(
                    conditions=[dict(type='Complete', status='True')]))), \
                    mock.patch('live_adoption_check.adoption_job_diagnostics', return_value=dict(receipt_logged=receipt)):
                if receipt:
                    self.assertEqual(
                        wait_adoption_job('readiness-adopt-first')
                        ['job_outcome'], 'complete')
                else:
                    with self.assertRaises(AdoptionError):
                        wait_adoption_job('readiness-adopt-first')
        with mock.patch('live_adoption_check.time.monotonic', side_effect=[0, 331]), \
                mock.patch('live_adoption_check.adoption_job_diagnostics', return_value=dict(receipt_logged=False)) as diagnostics:
            with self.assertRaises(AdoptionError) as failure:
                wait_adoption_job('readiness-adopt-repeat')
        self.assertEqual(failure.exception.job_evidence['job_outcome'],
                         'timeout')
        diagnostics.assert_called_once()

    def test_job_diagnostics_keep_states_and_categories_without_raw_logs(self):
        pods = dict(items=[
            dict(
                metadata=dict(name='fixture-pod'),
                status=dict(
                    initContainerStatuses=[
                        dict(
                            name='init',
                            state=dict(waiting=dict(reason='PodInitializing')))
                    ],
                    containerStatuses=[
                        dict(
                            name='api',
                            state=dict(
                                terminated=dict(reason='Error', exitCode=255)))
                    ]))
        ])
        raw = 'Legacy recurring-run adoption failed: workflow PRIVATE has no persisted run; TOKEN'
        with mock.patch(
                'live_adoption_check.kube',
                side_effect=[json.dumps(pods), '', raw]):
            evidence = adoption_job_diagnostics('readiness-adopt-first')
        self.assertEqual(evidence['containers'][0]['reason'], 'PodInitializing')
        self.assertEqual(evidence['containers'][1]['exit_code'], 255)
        self.assertEqual(evidence['containers'][1]['log_categories'],
                         ['adoption_failed', 'unpersisted_workflow'])
        self.assertNotIn('PRIVATE', json.dumps(evidence))
        self.assertNotIn('TOKEN', json.dumps(evidence))
        self.assertEqual(
            adoption_log_categories('PRIVATE arbitrary error TOKEN'), [])

    def test_startup_diagnostics_classify_fixed_messages_without_payload(self):
        cases = {
            'Failed to initialize ClientManager: PRIVATE TOKEN':
                'initialization',
            'failed to detect schema version: PRIVATE TOKEN':
                'database_initialization',
            'failed to initialize object store: PRIVATE TOKEN':
                'object_store_initialization',
            'ERROR: Timed out waiting for PRIVATE after 60 attempts.':
                'init_dependency_timeout',
            'ERROR: WAIT_HOST or WAIT_PORT is not set.':
                'init_dependency_configuration',
        }
        for message, category in cases.items():
            with self.subTest(category=category):
                self.assertEqual(adoption_log_categories(message), [category])
        partial = 'Initializing DB client...\nDetected legacy schema. Running upgrade flow.\nPRIVATE TOKEN'
        self.assertEqual(
            adoption_startup_milestones(partial),
            ['database_started', 'legacy_schema_detected'])
        self.assertEqual(adoption_startup_milestones('PRIVATE TOKEN'), [])
        self.assertEqual(
            adoption_startup_milestones(
                'DB client initialized successfully\nInitializing Object store client...'
            ), ['database_ready', 'object_store_started'])

    def test_panic_evidence_excludes_messages_arguments_and_unknown_paths(self):
        self.assertEqual(
            adoption_log_categories(
                'workflow PRIVATE conflicts with its persisted run TOKEN'),
            ['execution_mismatch', 'workflow_run_mismatch'])
        self.assertEqual(
            adoption_log_categories(
                'run PRIVATE has no valid controller index: TOKEN'),
            ['execution_index'])
        logs = ('panic: PRIVATE TOKEN\n'
                'main.main(PRIVATE, TOKEN)\n'
                '\t/build/private/backend/src/apiserver/main.go:205 +0x123\n'
                'main.PRIVATE(TOKEN)\n'
                '\t/build/backend/src/apiserver/main.go:210 +0xabc\n'
                '\t/build/backend/PRIVATE.go:1 +0x1\n'
                '\t/private/TOKEN.go:15 +0x1\n'
                '\t/build/backend/../PRIVATE.go:1 +0x1\n')
        evidence = dict(
            categories=adoption_log_categories(logs),
            frames=adoption_stack_frames(logs))
        self.assertEqual(evidence['categories'], ['panic'])
        self.assertEqual(evidence['frames'], [
            dict(file='backend/src/apiserver/main.go', line=205, symbol='main'),
            dict(file='backend/src/apiserver/main.go', line=210)
        ])
        self.assertNotIn('PRIVATE', json.dumps(evidence))
        self.assertNotIn('TOKEN', json.dumps(evidence))
        self.assertNotIn('/build', json.dumps(evidence))
        self.assertEqual(
            adoption_stack_frames('\tbackend/src/apiserver/main.go:999999'), [])

    def test_source_progress_shapes_omit_identifiers(self):
        current = dict(
            schedules=[
                dict(
                    uid='PRIVATE',
                    name='TOKEN',
                    trigger=dict(
                        lastWorkflowIndex=3,
                        lastTriggeredTime='1970-01-01T00:01:40Z'))
            ],
            runs=[
                dict(
                    UUID='RUN_SECRET',
                    Name='NAME_SECRET',
                    DisplayName='NAME_SECRET',
                    JobUUID='PRIVATE',
                    CreatedAtInSec=125,
                    ScheduledAtInSec=120)
            ],
            workflows=[
                dict(
                    run_id='RUN_SECRET',
                    index=3,
                    epoch=120,
                    created='1970-01-01T00:02:05Z',
                    phase='Succeeded')
            ])
        result = source_progress_shapes(current)[0]
        self.assertEqual(result['created'], 125)
        self.assertEqual(result['scheduled'], 120)
        self.assertEqual(result['acknowledged_time'], 100)
        self.assertTrue(result['display_equals_workflow_name'])
        self.assertIsNone(result['request_index'])
        self.assertEqual(result['workflow_created'], 125)
        for private in ('PRIVATE', 'TOKEN', 'SECRET'):
            self.assertNotIn(private, json.dumps(result))
        current['schedules'][0]['name'] = 'WORKFLOW_NAME'
        current['runs'][0]['DisplayName'] = 'WORKFLOW_NAME-51-2626342551'
        self.assertEqual(
            source_progress_shapes(current)[0]['request_index'], 51)
        current['runs'][0]['DisplayName'] = 'WORKFLOW_NAME-51-0'
        self.assertIsNone(source_progress_shapes(current)[0]['request_index'])

    def test_active_source_accepts_persisted_unacknowledged_submission(self):
        fixture = {'schedules': [dict(scenario='default', schedule_uid='0')]}
        current = dict(
            jobs=[dict(UUID='0', Enabled=1)],
            schedules=[
                dict(uid='0', enabled=True, trigger=dict(lastWorkflowIndex=1))
            ],
            runs=[dict(UUID='r', State='UNKNOWN', Conditions='')],
            workflows=[
                dict(
                    uid='w',
                    name='held',
                    run_id='r',
                    index=2,
                    suspended=True,
                    phase='')
            ])
        with mock.patch('live_adoption_check.FixtureClient'), \
                mock.patch('live_adoption_check.snapshot', side_effect=[dict(workflows=[]), current]), \
                mock.patch('live_adoption_check.kube') as command, \
                mock.patch('live_adoption_check.write_object'), \
                mock.patch('live_adoption_check.time.monotonic', side_effect=[0, 0]):
            observation = prepare_active(Path('/unused'), fixture)
        self.assertTrue(observation['recoverable_unacknowledged_submission'])
        self.assertTrue(observation['persisted_run_nonterminal'])
        self.assertFalse(observation['controller_acknowledged'])
        self.assertFalse(
            any('scale' in call.args for call in command.call_args_list))

    def test_exception_evidence_never_includes_payload(self):
        try:
            raise KeyError('PRIVATE TOKEN')
        except KeyError as error:
            evidence = exception_evidence(error)
        self.assertEqual(evidence['type'], 'KeyError')
        self.assertTrue(evidence['locations'])
        self.assertTrue(
            all(location['file'].startswith('tools/upgrade-readiness/')
                for location in evidence['locations']))
        self.assertNotIn('PRIVATE', json.dumps(evidence))
        self.assertNotIn('TOKEN', json.dumps(evidence))

    def test_adoption_accepts_first_unacknowledged_disabled_tick(self):
        for trigger in ({}, {'lastWorkflowIndex': 0}):
            fixture, before, after = inventory()
            before['schedules'][1]['trigger'] = trigger
            before['runs'][1]['DisplayName'] = before['runs'][1]['Name']
            after['runs'] = copy.deepcopy(before['runs'])
            validate_adoption(before, after, fixture)
            for invalid in ({
                    'lastWorkflowIndex': 1
            }, {
                    'lastWorkflowIndex': 0,
                    'lastTriggeredTime': '1970-01-01T00:01:40Z'
            }):
                before['schedules'][1]['trigger'] = invalid
                with self.assertRaisesRegex(AdoptionError,
                                            'source_trigger_incomplete'):
                    validate_adoption(before, after, fixture)

    def test_adoption_recovers_one_persisted_submission_without_resetting_history(
            self):
        for epoch, created, expected, embedded in ((130, 131, 130, False),
                                                   (200, 201, 200, False),
                                                   (135, 136, 135, False),
                                                   (135, 135, 130, False),
                                                   (135, 135, 135, True)):
            with self.subTest(epoch=epoch, created=created):
                fixture, before, after = inventory()
                before['runs'][0]['State'] = 'SUCCEEDED'
                before['workflows'][0]['suspended'] = False
                self.add_tick(before)
                before['runs'][-1].update(
                    ScheduledAtInSec=epoch,
                    CreatedAtInSec=created,
                    State='RUNNING')
                if embedded:
                    before['runs'][-1]['DisplayName'] = before['runs'][-1][
                        'Name']
                before['workflows'][-1].update(epoch=epoch, suspended=True)
                after.update(copy.deepcopy(before))
                after['states'][0].update(
                    LastRunUUID='new',
                    LastRunIndex=2,
                    LastScheduledAtInSec=expected,
                    LastCreatedAtInSec=created)
                after['schedules'][0]['trigger'].update(
                    lastWorkflowIndex=2,
                    lastTriggeredTime=datetime.fromtimestamp(
                        expected, timezone.utc).isoformat())
                validate_adoption(before, after, fixture)
                self.assertEqual(
                    before['schedules'][0]['trigger']['lastWorkflowIndex'], 1)
                after['states'][0]['LastRunUUID'] = 'r0'
                with self.assertRaisesRegex(AdoptionError,
                                            'last_run_identity_changed'):
                    validate_adoption(before, after, fixture)
                after['states'][0]['LastRunUUID'] = 'new'
                before['workflows'][-1]['index'] = 3
                after['workflows'][-1]['index'] = 3
                with self.assertRaisesRegex(AdoptionError,
                                            'source_progress_gap'):
                    validate_adoption(before, after, fixture)

    def test_active_source_timeout_identifies_missing_evidence(self):
        fixture = {'schedules': [dict(scenario='default', schedule_uid='0')]}
        baseline = dict(workflows=[])
        for missing in ('workflow', 'run', 'acknowledgement'):
            with self.subTest(missing=missing):
                current = dict(
                    jobs=[dict(UUID='0', Enabled=1)],
                    schedules=[
                        dict(
                            uid='0',
                            enabled=True,
                            trigger=dict(lastWorkflowIndex=3 if missing ==
                                         'acknowledgement' else 1))
                    ],
                    runs=[] if missing == 'run' else
                    [dict(UUID='r', State='RUNNING', Conditions='')],
                    workflows=[] if missing == 'workflow' else [
                        dict(
                            uid='w',
                            name='held',
                            run_id='r',
                            index=1,
                            suspended=True,
                            phase='')
                    ])
                with mock.patch('live_adoption_check.FixtureClient'), \
                        mock.patch('live_adoption_check.snapshot', side_effect=[baseline, current]), \
                        mock.patch('live_adoption_check.kube'), \
                        mock.patch('live_adoption_check.time.sleep'), \
                        mock.patch('live_adoption_check.time.monotonic', side_effect=[0, 0, 181]):
                    with self.assertRaisesRegex(
                            AdoptionError,
                            'source_active_run_not_persisted') as error:
                        prepare_active(Path('/unused'), fixture)
                evidence = error.exception.source_observation
                self.assertEqual(evidence['fresh_workflow_count'],
                                 0 if missing == 'workflow' else 1)
                self.assertEqual(evidence['persisted_run_found'],
                                 missing == 'acknowledgement')
                self.assertEqual(evidence['controller_acknowledged'],
                                 missing == 'run')
                self.assertTrue(evidence['schedule_enabled'])
                self.assertTrue(evidence['job_enabled'])

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
                epoch=130,
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

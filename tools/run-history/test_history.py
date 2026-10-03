# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Transaction and graph tests without external services or optional
packages."""

import copy
import json
import os
import sqlite3
import unittest
import uuid

import history
import mlmd

SCHEMA = '''
        CREATE TABLE experiments (UUID TEXT PRIMARY KEY, Name TEXT NOT NULL,
          Namespace TEXT NOT NULL, LastRunCreatedAtInSec INTEGER,
          UNIQUE(Name, Namespace));
        CREATE TABLE pipelines (UUID TEXT PRIMARY KEY, Name TEXT, Namespace TEXT,
          UNIQUE(Name, Namespace));
        CREATE TABLE pipeline_versions (UUID TEXT PRIMARY KEY, Name TEXT,
          PipelineId TEXT REFERENCES pipelines(UUID), UNIQUE(PipelineId, Name));
        CREATE TABLE jobs (UUID TEXT PRIMARY KEY, Name TEXT, Enabled BOOL,
          ExperimentUUID TEXT, PipelineId TEXT, PipelineVersionId TEXT);
        CREATE TABLE run_details (UUID TEXT PRIMARY KEY, ExperimentUUID TEXT REFERENCES experiments(UUID),
          Namespace TEXT, State TEXT, FinishedAtInSec INTEGER, CreatedAtInSec INTEGER,
          PipelineId TEXT, PipelineVersionId TEXT, JobUUID TEXT,
          PipelineContextId INTEGER, PipelineRunContextId INTEGER, RetryClaimedAtInSec INTEGER,
          ImportedFrom TEXT, ImportDigest TEXT);
        CREATE TABLE tasks (UUID TEXT PRIMARY KEY, RunUUID TEXT REFERENCES run_details(UUID),
          MLMDExecutionID TEXT, Fingerprint TEXT, MLMDInputs TEXT, MLMDOutputs TEXT, Payload TEXT);
        CREATE TABLE run_metrics (RunUUID TEXT REFERENCES run_details(UUID), NodeID TEXT,
          Name TEXT, NumberValue REAL, PRIMARY KEY(RunUUID,NodeID,Name));
        CREATE TABLE resource_references (ResourceUUID TEXT, ResourceType TEXT,
          ReferenceType TEXT, ReferenceUUID TEXT, Payload TEXT,
          PRIMARY KEY(ResourceUUID,ResourceType,ReferenceType));
    '''


def database():
    connection = sqlite3.connect(':memory:')
    connection.execute('PRAGMA foreign_keys=ON')
    connection.executescript(SCHEMA)
    db = history.Database(connection)
    db.integrity_error = sqlite3.IntegrityError
    return db


def mysql_database():
    import mysql.connector
    config = json.loads(os.environ['KFP_HISTORY_MYSQL_CONFIG'])
    config.pop('driver', None)
    connection = mysql.connector.connect(**config)
    db = history.Database(connection, 'mysql')
    db.integrity_error = mysql.connector.IntegrityError
    name = 'history_test_' + uuid.uuid4().hex
    db.query('CREATE DATABASE ' + db.quote(name))
    db.query('USE ' + db.quote(name))
    db.test_database_name = name
    schema = SCHEMA.replace('TEXT',
                            'VARCHAR(191)').replace('Payload VARCHAR(191)',
                                                    'Payload TEXT')
    for statement in schema.split(';'):
        if statement.strip():
            db.query(statement)
    return db


def source_database(factory=database):
    db = factory()
    db.insert(
        'experiments', {
            'UUID': 'exp-a',
            'Name': 'Default',
            'Namespace': 'team',
            'LastRunCreatedAtInSec': 10
        })
    db.insert('pipelines', {
        'UUID': 'pipeline-a',
        'Name': 'training',
        'Namespace': 'team'
    })
    db.insert('pipeline_versions', {
        'UUID': 'version-a',
        'Name': 'v1',
        'PipelineId': 'pipeline-a'
    })
    db.insert(
        'jobs', {
            'UUID': 'job-a',
            'Name': 'daily',
            'Enabled': True,
            'ExperimentUUID': 'exp-a',
            'PipelineId': 'pipeline-a',
            'PipelineVersionId': 'version-a'
        })
    db.insert(
        'run_details', {
            'UUID': 'run-a',
            'ExperimentUUID': 'exp-a',
            'Namespace': 'team',
            'State': 'SUCCEEDED',
            'CreatedAtInSec': 1,
            'FinishedAtInSec': 10,
            'PipelineId': 'pipeline-a',
            'PipelineVersionId': 'version-a',
            'JobUUID': 'job-a',
            'PipelineContextId': 1,
            'PipelineRunContextId': 2,
            'RetryClaimedAtInSec': 0,
            'ImportedFrom': None,
            'ImportDigest': None
        })
    outputs = history.canonical({'model': {'artifact_ids': [20]}})
    db.insert(
        'tasks', {
            'UUID':
                'task-a',
            'RunUUID':
                'run-a',
            'MLMDExecutionID':
                '11',
            'Fingerprint':
                'source-cache-key',
            'MLMDInputs':
                '{}',
            'MLMDOutputs':
                outputs,
            'Payload':
                history.canonical({
                    'MLMDExecutionID': '11',
                    'Fingerprint': 'source-cache-key',
                    'MLMDOutputs': outputs
                })
        })
    db.insert(
        'run_metrics', {
            'RunUUID': 'run-a',
            'NodeID': 'node',
            'Name': 'accuracy',
            'NumberValue': 0.9
        })
    db.insert(
        'resource_references', {
            'ResourceUUID': 'run-a',
            'ResourceType': 'Run',
            'ReferenceType': 'Experiment',
            'ReferenceUUID': 'exp-a',
            'Payload': history.canonical({'ReferenceUUID': 'exp-a'})
        })
    db.connection.commit()
    return db


def graph():
    return {
        'context_types': [{
            'id': '1',
            'name': 'system.Pipeline'
        }, {
            'id': '2',
            'name': 'system.PipelineRun'
        }],
        'execution_types': [{
            'id': '3',
            'name': 'system.ContainerExecution'
        }],
        'artifact_types': [{
            'id': '4',
            'name': 'system.Model'
        }],
        'contexts': [{
            'id': '1',
            'type_id': '1',
            'name': 'training'
        }, {
            'id': '2',
            'type_id': '2',
            'name': 'run-a'
        }],
        'executions': [{
            'id': '10',
            'type_id': '3',
            'name': 'parent',
            'last_known_state': 'COMPLETE',
            'create_time_since_epoch': '1000'
        }, {
            'id': '11',
            'type_id': '3',
            'last_known_state': 'CACHED',
            'custom_properties': {
                'parent_dag_id': {
                    'int_value': '10'
                },
                'cached_execution_id': {
                    'string_value': '10'
                },
                'cache_fingerprint': {
                    'string_value': 'fingerprint'
                }
            }
        }],
        'artifacts': [{
            'id': '20',
            'type_id': '4',
            'uri': 's3://retained/model',
            'state': 'LIVE'
        }],
        'events': [{
            'execution_id': '10',
            'artifact_id': '20',
            'type': 'OUTPUT',
            'path': {
                'steps': [{
                    'key': 'model'
                }]
            },
            'milliseconds_since_epoch': '1000'
        }],
        'associations': [{
            'context_id': '2',
            'execution_id': '10'
        }, {
            'context_id': '2',
            'execution_id': '11'
        }],
        'attributions': [{
            'context_id': '2',
            'artifact_id': '20'
        }],
        'parents': [{
            'child_id': '2',
            'parent_id': '1'
        }],
        'run_context_id': '2',
    }


class FakeRPC:
    """MLMD-shaped in-memory service with independent destination integer
    IDs."""

    def __init__(self, initial=None):
        self.graph = copy.deepcopy(
            initial or {key: [] for key in graph() if key != 'run_context_id'})
        self.next_id = 1000
        self.calls = []
        self.fail_on = None

    def call(self, method, **fields):
        self.calls.append((method, copy.deepcopy(fields)))
        if method == self.fail_on:
            self.fail_on = None
            raise RuntimeError('injected MLMD connection failure')
        for collection, kind in mlmd.KINDS.items():
            singular = kind.lower()
            types = singular + '_types'
            if method == 'Get' + kind + 'Type':
                node = next(
                    (row for row in self.graph[types]
                     if row['name'] == fields['type_name'] and
                     row.get('version', '') == fields.get('type_version', '')),
                    None)
                return {singular + '_type': copy.deepcopy(node)} if node else {}
            if method == 'Get' + kind + 'TypesByID':
                return {
                    types:
                        copy.deepcopy([
                            row for row in self.graph[types]
                            if row['id'] in fields['type_ids']
                        ])
                }
            if method == 'Put' + kind + 'Type':
                node = copy.deepcopy(fields[singular + '_type'])
                self.next_id += 1
                node['id'] = str(self.next_id)
                self.graph[types].append(node)
                return {'type_id': node['id']}
            if method == 'Get' + kind + 'ByTypeAndName':
                type_ids = {
                    row['id']
                    for row in self.graph[types]
                    if row['name'] == fields['type_name']
                }
                node = next((row for row in self.graph[collection]
                             if row['type_id'] in type_ids and
                             row.get('name') == fields[singular + '_name']),
                            None)
                return {singular: copy.deepcopy(node)} if node else {}
            if method == 'Get' + kind + 'sByID':
                return {
                    collection:
                        copy.deepcopy([
                            row for row in self.graph[collection]
                            if row['id'] in fields[singular + '_ids']
                        ])
                }
            if method == 'Put' + kind + 's':
                ids = []
                for original in fields[collection]:
                    node = copy.deepcopy(original)
                    if 'id' not in node:
                        self.next_id += 1
                        node['id'] = str(self.next_id)
                    self.graph[collection] = [
                        row for row in self.graph[collection]
                        if row['id'] != node['id']
                    ]
                    self.graph[collection].append(node)
                    ids.append(node['id'])
                return {singular + '_ids': ids}
        if method in ('GetEventsByExecutionIDs', 'GetEventsByArtifactIDs'):
            key = 'execution_id' if 'Execution' in method else 'artifact_id'
            return {
                'events':
                    copy.deepcopy([
                        row for row in self.graph['events']
                        if row[key] in fields[key + 's']
                    ])
            }
        if method == 'PutEvents':
            self.graph['events'].extend(copy.deepcopy(fields['events']))
            return {}
        if method == 'PutAttributionsAndAssociations':
            for collection in ('attributions', 'associations'):
                self.graph[collection] = mlmd.unique(self.graph[collection] +
                                                     fields[collection])
            return {}
        if method == 'PutParentContexts':
            self.graph['parents'] = mlmd.unique(self.graph['parents'] +
                                                fields['parent_contexts'])
            return {}
        if method == 'GetExecutionsByContext':
            ids = {
                row['execution_id']
                for row in self.graph['associations']
                if row['context_id'] == fields['context_id']
            }
            return {
                'executions':
                    copy.deepcopy([
                        row for row in self.graph['executions']
                        if row['id'] in ids
                    ])
            }
        if method == 'GetArtifactsByContext':
            ids = {
                row['artifact_id']
                for row in self.graph['attributions']
                if row['context_id'] == fields['context_id']
            }
            return {
                'artifacts':
                    copy.deepcopy([
                        row for row in self.graph['artifacts']
                        if row['id'] in ids
                    ])
            }
        if method in ('GetContextsByExecution', 'GetContextsByArtifact'):
            collection, key = ('associations',
                               'execution_id') if 'Execution' in method else (
                                   'attributions', 'artifact_id')
            ids = {
                row['context_id']
                for row in self.graph[collection]
                if row[key] == fields[key]
            }
            return {
                'contexts':
                    copy.deepcopy([
                        row for row in self.graph['contexts']
                        if row['id'] in ids
                    ])
            }
        if method == 'GetParentContextsByContext':
            ids = {
                row['parent_id']
                for row in self.graph['parents']
                if row['child_id'] == fields['context_id']
            }
            return {
                'contexts':
                    copy.deepcopy([
                        row for row in self.graph['contexts']
                        if row['id'] in ids
                    ])
            }
        raise AssertionError('Unimplemented RPC ' + method)


class TransferTest(unittest.TestCase):
    make_database = staticmethod(database)

    def setUp(self):
        self.source = source_database(self.make_database)
        self.archive = history.export_run(self.source,
                                          mlmd.Metadata(FakeRPC(graph())),
                                          'run-a', 'cluster-a')
        self.destination = self.make_database()
        self.rpc = FakeRPC()
        self.metadata = mlmd.Metadata(self.rpc)

    def tearDown(self):
        for db in (self.source, self.destination):
            if hasattr(db, 'test_database_name'):
                db.query('DROP DATABASE ' + db.quote(db.test_database_name))
            db.connection.close()

    def test_roundtrip_preserves_history_and_remaps_mlmd(self):
        self.assertEqual(
            history.import_run(self.destination, self.metadata, self.archive),
            'imported')
        run = self.destination.select('run_details', UUID='run-a')[0]
        task = self.destination.select('tasks', UUID='task-a')[0]
        self.assertEqual(run['CreatedAtInSec'], 1)
        self.assertEqual(run['FinishedAtInSec'], 10)
        self.assertEqual(run['ImportedFrom'], 'cluster-a')
        self.assertEqual(run['ImportDigest'], self.archive['digest'])
        self.assertNotEqual(run['PipelineRunContextId'], 2)
        self.assertNotEqual(task['MLMDExecutionID'], '11')
        self.assertEqual(task['Fingerprint'], '')
        self.assertEqual(json.loads(task['Payload'])['Fingerprint'], '')
        artifact_id = int(self.rpc.graph['artifacts'][0]['id'])
        self.assertEqual(
            json.loads(task['MLMDOutputs'])['model']['artifact_ids'],
            [artifact_id])
        self.assertEqual(
            json.loads(task['Payload'])['MLMDExecutionID'],
            task['MLMDExecutionID'])
        self.assertEqual(self.destination.select('jobs'), [])
        self.assertIsNone(run['JobUUID'])
        self.assertEqual(self.rpc.graph['artifacts'][0]['uri'],
                         's3://retained/model')
        context = next(
            row for row in self.rpc.graph['contexts'] if row['name'] == 'run-a')
        self.assertEqual(int(context['id']), run['PipelineRunContextId'])
        cached = next(row for row in self.rpc.graph['executions']
                      if 'cached_execution_id' in row['custom_properties'])
        parent_id = cached['custom_properties']['parent_dag_id']['int_value']
        self.assertEqual(
            cached['custom_properties']['cached_execution_id']['string_value'],
            parent_id)
        self.assertNotEqual(parent_id, '10')
        self.assertTrue(cached['custom_properties']['cache_fingerprint']
                        ['string_value'].startswith('history:'))
        self.assertEqual(len(self.rpc.graph['events']), 1)

    def test_repeat_import_performs_no_metadata_writes(self):
        history.import_run(self.destination, self.metadata, self.archive)
        before = copy.deepcopy(self.rpc.graph)
        self.rpc.calls.clear()
        self.assertEqual(
            history.import_run(self.destination, self.metadata, self.archive),
            'already imported')
        self.assertEqual(before, self.rpc.graph)
        self.assertEqual(self.rpc.calls, [])

    def test_populated_destination_explicit_experiment_mapping(self):
        self.destination.insert(
            'experiments', {
                'UUID': 'exp-b',
                'Name': 'Default',
                'Namespace': 'team',
                'LastRunCreatedAtInSec': 99
            })
        self.destination.connection.commit()
        with self.assertRaises(self.destination.integrity_error):
            history.import_run(self.destination, self.metadata, self.archive)
        self.assertFalse(
            any(method.startswith('Put') for method, _ in self.rpc.calls))
        history.import_run(
            self.destination,
            self.metadata,
            self.archive,
            experiment_id='exp-b')
        self.assertEqual(
            self.destination.select('run_details')[0]['ExperimentUUID'],
            'exp-b')
        self.assertEqual(
            self.destination.select('experiments')[0]['LastRunCreatedAtInSec'],
            99)
        reference = self.destination.select('resource_references')[0]
        self.assertEqual(reference['ReferenceUUID'], 'exp-b')
        self.assertEqual(
            json.loads(reference['Payload'])['ReferenceUUID'], 'exp-b')

    def test_dry_run_rolls_back_all_reserved_sql_and_never_puts_mlmd(self):
        self.assertEqual(
            history.import_run(
                self.destination, self.metadata, self.archive, dry_run=True),
            'validated (no changes)')
        for table in history.ORDER:
            if table in self.destination.tables():
                self.assertEqual(self.destination.select(table), [])
        self.assertFalse(
            any(method.startswith('Put') for method, _ in self.rpc.calls))

    def test_staging_failure_rolls_back_sql_and_retry_reuses_metadata(self):
        self.rpc.fail_on = 'PutEvents'
        with self.assertRaises(RuntimeError):
            history.import_run(self.destination, self.metadata, self.archive)
        self.assertEqual(self.destination.select('run_details'), [])
        self.assertEqual(self.destination.select('experiments'), [])
        self.assertTrue(self.rpc.graph['executions'])
        node_ids = {row['id'] for row in self.rpc.graph['executions']}
        history.import_run(self.destination, self.metadata, self.archive)
        self.assertEqual(node_ids,
                         {row['id'] for row in self.rpc.graph['executions']})
        self.assertEqual(len(self.rpc.graph['events']), 1)

    def test_existing_run_is_never_overwritten(self):
        self.destination.insert(
            'experiments', {
                'UUID': 'exp-a',
                'Name': 'Default',
                'Namespace': 'team',
                'LastRunCreatedAtInSec': 0
            })
        row = copy.deepcopy(self.archive['tables']['run_details'][0])
        self.destination.insert('run_details', row)
        self.destination.connection.commit()
        with self.assertRaisesRegex(history.HistoryError,
                                    'run ID already exists'):
            history.import_run(self.destination, self.metadata, self.archive)
        self.assertEqual(self.rpc.calls, [])
        self.assertIsNone(
            self.destination.select('run_details')[0]['ImportedFrom'])

    def test_reject_cross_generation_running_tampered_and_dangling_graph(self):
        for change in ('generation', 'running', 'digest', 'graph'):
            with self.subTest(change=change):
                value = copy.deepcopy(self.archive)
                if change == 'generation':
                    value['format'] = 'kfp-history-native/v1'
                elif change == 'running':
                    value['tables']['run_details'][0]['State'] = 'RUNNING'
                elif change == 'digest':
                    value['tables']['run_details'][0]['CreatedAtInSec'] = 9
                else:
                    value['mlmd']['events'][0]['artifact_id'] = '99999'
                    value['digest'] = history.archive_digest(value)
                with self.assertRaises(history.HistoryError):
                    history.import_run(self.destination, self.metadata, value)
        self.assertEqual(self.rpc.calls, [])

    def test_export_rejects_unfinished_run_and_unfinished_ancestor(self):
        self.source.query(
            'UPDATE run_details SET State=' + self.source.placeholder,
            ('RUNNING',))
        self.source.connection.commit()
        with self.assertRaises(history.HistoryError):
            history.export_run(self.source, self.metadata, 'run-a', 'cluster-a')
        value = graph()
        value['executions'][0]['last_known_state'] = 'RUNNING'
        with self.assertRaises(history.HistoryError):
            mlmd.validate_graph(value)

    def test_metadata_conflict_does_not_publish_sql(self):
        self.rpc.graph['execution_types'] = [{
            'id': '900',
            'name': 'system.ContainerExecution',
            'properties': {
                'changed': 'INT'
            }
        }]
        with self.assertRaisesRegex(history.HistoryError, 'type differs'):
            history.import_run(self.destination, self.metadata, self.archive)
        self.assertEqual(self.destination.select('run_details'), [])
        self.assertFalse(
            any(method.startswith('Put') for method, _ in self.rpc.calls))

    def test_same_source_pipeline_node_is_reused_by_another_import(self):
        first = self.metadata.import_graph(graph(), 'cluster-a')
        second = self.metadata.import_graph(graph(), 'cluster-a')
        self.assertEqual(first, second)
        self.assertEqual(len(self.rpc.graph['contexts']), 2)
        self.assertEqual(len(self.rpc.graph['events']), 1)

    def test_source_id_prevents_unrelated_import_reusing_run_context(self):
        self.metadata.import_graph(graph(), 'cluster-a')
        with self.assertRaisesRegex(history.HistoryError,
                                    'conflicting history provenance'):
            self.metadata.import_graph(graph(), 'cluster-b')

    def test_sql_to_mlmd_dangling_references_fail_before_staging(self):
        for field in ('MLMDExecutionID', 'MLMDInputs'):
            with self.subTest(field=field):
                value = copy.deepcopy(self.archive)
                value['tables']['tasks'][0][
                    field] = '999' if field == 'MLMDExecutionID' else '{"x":{"artifact_ids":[999]}}'
                value['digest'] = history.archive_digest(value)
                with self.assertRaisesRegex(history.HistoryError,
                                            'absent MLMD'):
                    history.import_run(
                        self.destination, self.metadata, value, dry_run=True)
                self.assertEqual(self.rpc.calls, [])

    def test_resource_references_cannot_target_unrelated_records(self):
        value = copy.deepcopy(self.archive)
        value['tables']['resource_references'][0][
            'ResourceUUID'] = 'unrelated-run'
        value['digest'] = history.archive_digest(value)
        with self.assertRaisesRegex(history.HistoryError, 'owner is outside'):
            history.import_run(
                self.destination, self.metadata, value, dry_run=True)
        self.assertEqual(self.rpc.calls, [])

    def test_graph_cycles_fail_before_staging(self):
        value = copy.deepcopy(self.archive)
        value['mlmd']['parents'].append({'parent_id': '2', 'child_id': '1'})
        value['digest'] = history.archive_digest(value)
        with self.assertRaisesRegex(history.HistoryError, 'cycle'):
            history.import_run(
                self.destination, self.metadata, value, dry_run=True)
        self.assertEqual(self.rpc.calls, [])

    def test_export_retains_lowercase_pipeline_references(self):
        self.source.insert(
            'resource_references', {
                'ResourceUUID': 'pipeline-a',
                'ResourceType': 'pipeline',
                'ReferenceType': 'Namespace',
                'ReferenceUUID': 'team',
                'Payload': '{}'
            })
        self.source.connection.commit()
        archive = history.export_run(self.source,
                                     mlmd.Metadata(FakeRPC(graph())), 'run-a',
                                     'cluster-a')
        self.assertTrue(
            any(row['ResourceType'] == 'pipeline'
                for row in archive['tables']['resource_references']))

    def test_schema_narrowing_is_rejected_before_metadata_calls(self):
        value = copy.deepcopy(self.archive)
        value['schema']['tasks'][0]['type'] = 'VARCHAR(1)'
        value['digest'] = history.archive_digest(value)
        with self.assertRaisesRegex(history.HistoryError,
                                    'SQL schema mismatch'):
            history.import_run(
                self.destination, self.metadata, value, dry_run=True)
        self.assertEqual(self.rpc.calls, [])

    def test_no_destination_schedule_or_job_reference_is_imported(self):
        self.destination.insert(
            'jobs', {
                'UUID': 'job-b',
                'Name': 'daily',
                'Enabled': True,
                'ExperimentUUID': None,
                'PipelineId': None,
                'PipelineVersionId': None
            })
        self.destination.connection.commit()
        value = copy.deepcopy(self.archive)
        for resource_type in ('Job', 'RecurringRun'):
            value['tables']['resource_references'].append({
                'ResourceUUID': 'run-a',
                'ResourceType': 'Run',
                'ReferenceType': resource_type,
                'ReferenceUUID': 'job-a',
                'Payload': '{}'
            })
        value['digest'] = history.archive_digest(value)
        history.import_run(self.destination, self.metadata, value)
        jobs = self.destination.select('jobs')
        self.assertEqual(len(jobs), 1)
        self.assertEqual(jobs[0]['UUID'], 'job-b')
        self.assertTrue(jobs[0]['Enabled'])
        self.assertFalse(
            any(row['ReferenceType'] in ('Job', 'RecurringRun')
                for row in self.destination.select('resource_references')))


@unittest.skipUnless(
    os.environ.get('KFP_HISTORY_MYSQL_CONFIG'),
    'MySQL integration config not supplied')
class MySQLTransferTest(TransferTest):
    make_database = staticmethod(mysql_database)


if __name__ == '__main__':
    unittest.main()

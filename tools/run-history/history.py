#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Administrative, completed-run transfer for the KFP 2.18 MLMD generation."""

import argparse
import copy
import hashlib
import json
import os
import re
import sys

FORMAT = 'kfp-history-mlmd-2.18/v1'
MAX_ARCHIVE_BYTES = 256 * 1024 * 1024
TERMINAL = {'SUCCEEDED', 'FAILED', 'ERROR', 'CANCELED', 'CANCELLED', 'SKIPPED'}
KEYS = {
    'experiments': ('UUID',),
    'pipelines': ('UUID',),
    'pipeline_versions': ('UUID',),
    'pipeline_tags': ('PipelineId', 'TagKey'),
    'pipeline_version_tags': ('PipelineVersionId', 'TagKey'),
    'jobs': ('UUID',),
    'run_details': ('UUID',),
    'tasks': ('UUID',),
    'run_metrics': ('RunUUID', 'NodeID', 'Name'),
    'resource_references': ('ResourceUUID', 'ResourceType', 'ReferenceType'),
}
ORDER = list(KEYS)


class HistoryError(ValueError):
    """An archive or destination cannot safely participate in the transfer."""


def canonical(value):
    return json.dumps(
        value, sort_keys=True, separators=(',', ':'), ensure_ascii=False)


def digest(value):
    return hashlib.sha256(canonical(value).encode()).hexdigest()


def row_key(table, row):
    try:
        return tuple(row[key] for key in KEYS[table])
    except KeyError as exc:
        raise HistoryError('Missing primary key in ' + table) from exc


class Database:
    """Small DB-API adapter.

    All identifiers come from inspected, allowed tables.
    """

    def __init__(self, connection, driver='sqlite'):
        self.connection = connection
        self.driver = driver
        self.placeholder = '?' if driver == 'sqlite' else '%s'
        self._columns = {}

    def quote(self, identifier):
        if not re.fullmatch(r'[A-Za-z][A-Za-z0-9_]*', identifier):
            raise HistoryError('Invalid SQL identifier')
        quote = '`' if self.driver == 'mysql' else '"'
        return quote + identifier + quote

    def query(self, sql, values=()):
        cursor = self.connection.cursor()
        try:
            cursor.execute(sql, values)
            if not cursor.description:
                return []
            names = [item[0] for item in cursor.description]
            return [dict(zip(names, row)) for row in cursor.fetchall()]
        finally:
            cursor.close()

    def tables(self):
        if self.driver == 'sqlite':
            rows = self.query(
                "SELECT name FROM sqlite_master WHERE type='table'")
            return {row['name'] for row in rows}
        if self.driver == 'mysql':
            rows = self.query(
                'SELECT TABLE_NAME AS name FROM information_schema.tables '
                'WHERE table_schema=DATABASE()')
        else:
            rows = self.query(
                'SELECT table_name AS name FROM information_schema.tables '
                'WHERE table_schema=current_schema()')
        return {row['name'] for row in rows}

    def columns(self, table):
        if table not in KEYS:
            raise HistoryError('Unsupported table: ' + table)
        if table not in self._columns:
            cursor = self.connection.cursor()
            try:
                cursor.execute('SELECT * FROM ' + self.quote(table) +
                               ' WHERE 1=0')
                self._columns[table] = {item[0] for item in cursor.description}
            finally:
                cursor.close()
        return self._columns[table]

    def select(self, table, **where):
        self.columns(table)
        if not set(where) <= self.columns(table):
            raise HistoryError('Unknown column in ' + table)
        predicates = [self.quote(key) + '=' + self.placeholder for key in where]
        sql = 'SELECT * FROM ' + self.quote(table)
        if predicates:
            sql += ' WHERE ' + ' AND '.join(predicates)
        return self.query(sql, tuple(where.values()))

    def schema(self, table):
        self.columns(table)
        if self.driver == 'mysql':
            rows = self.query(
                'SELECT COLUMN_NAME AS name, COLUMN_TYPE AS type, '
                'IS_NULLABLE AS nullable, COLLATION_NAME AS collation_name '
                'FROM information_schema.columns WHERE table_schema=DATABASE() '
                'AND table_name=%s ORDER BY ORDINAL_POSITION', (table,))
        elif self.driver == 'sqlite':
            rows = [{
                'name': row['name'],
                'type': row['type'],
                'nullable': not row['notnull']
            } for row in self.query('PRAGMA table_info(' + self.quote(table) +
                                    ')')]
        else:
            raise HistoryError('Unsupported database schema inspection')
        return sorted([
            row for row in rows
            if row['name'] not in ('ImportedFrom', 'ImportDigest')
        ],
                      key=lambda row: row['name'])

    def insert(self, table, row):
        if not set(row) <= self.columns(table):
            raise HistoryError('Schema mismatch for ' + table +
                               '; use matching 2.18 schemas')
        columns = list(row)
        sql = 'INSERT INTO ' + self.quote(table) + ' ('
        sql += ','.join(self.quote(key) for key in columns) + ') VALUES ('
        sql += ','.join(self.placeholder for _ in columns) + ')'
        self.query(sql, tuple(row[key] for key in columns))

    def update(self, table, row):
        keys = KEYS[table]
        fields = [key for key in row if key not in keys]
        sql = 'UPDATE ' + self.quote(table) + ' SET '
        sql += ','.join(
            self.quote(key) + '=' + self.placeholder for key in fields)
        sql += ' WHERE ' + ' AND '.join(
            self.quote(key) + '=' + self.placeholder for key in keys)
        self.query(sql, tuple(row[key] for key in fields + list(keys)))

    def begin(self, readonly=False):
        # Each invocation owns its connection. End metadata inspection's implicit txn.
        self.connection.rollback()
        if self.driver == 'sqlite':
            self.query('BEGIN' if readonly else 'BEGIN IMMEDIATE')
        elif self.driver == 'mysql':
            self.query('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ')
            self.query('START TRANSACTION WITH CONSISTENT SNAPSHOT'
                       if readonly else 'START TRANSACTION')
        else:
            self.query('BEGIN ISOLATION LEVEL REPEATABLE READ' +
                       (' READ ONLY' if readonly else ''))

    def check_generation(self, importing=False):
        if 'MLMDExecutionID' not in self.columns('tasks'):
            raise HistoryError(
                'Expected KFP 2.18 MLMD schema; native-history import is unsupported'
            )
        if importing and not {'ImportedFrom', 'ImportDigest'
                             } <= self.columns('run_details'):
            raise HistoryError(
                'Upgrade destination API server and run its migrations before import'
            )


def connect(path):
    with open(path, encoding='utf-8') as stream:
        config = json.load(stream)
    driver = config.pop('driver')
    if driver == 'mysql':
        import mysql.connector  # Optional operator dependency.
        connection = mysql.connector.connect(**config)
    else:
        raise HistoryError(
            'Release 2.18 history transfer supports the MySQL deployment only')
    return Database(connection, driver)


def require_completed(run):
    state = str(run.get('State') or run.get('Conditions') or '').upper()
    if state not in TERMINAL or int(run.get('FinishedAtInSec') or 0) <= 0:
        raise HistoryError('Only finished terminal runs can be transferred')
    if run.get('ImportedFrom'):
        raise HistoryError(
            'Export from the original installation, not an imported copy')


def archive_digest(archive):
    value = copy.deepcopy(archive)
    value.pop('digest', None)
    # An experiment's activity changes when another unrelated run starts.
    for row in value.get('tables', {}).get('experiments', []):
        row.pop('LastRunCreatedAtInSec', None)
    return digest(value)


def export_run(db, mlmd, run_id, source):
    if not source or len(source) > 191:
        raise HistoryError(
            'source-id must be a stable installation ID of 1–191 characters')
    db.check_generation()
    db.begin(readonly=True)
    try:
        runs = db.select('run_details', UUID=run_id)
        if len(runs) != 1:
            raise HistoryError('Run not found: ' + run_id)
        run = runs[0]
        require_completed(run)
        present = db.tables()
        tables = {table: [] for table in ORDER if table in present}
        tables['run_details'] = runs
        tables['tasks'] = db.select('tasks', RunUUID=run_id)
        tables['run_metrics'] = db.select('run_metrics', RunUUID=run_id)

        def include(table, key, value):
            if not value:
                return
            rows = db.select(table, **{key: value})
            if len(rows) != 1:
                raise HistoryError('Missing dependency ' + table + ':' +
                                   str(value))
            if rows[0] not in tables[table]:
                tables[table].append(rows[0])

        include('experiments', 'UUID', run.get('ExperimentUUID'))
        include('jobs', 'UUID', run.get('JobUUID'))
        for owner in runs + tables.get('jobs', []):
            include('pipeline_versions', 'UUID', owner.get('PipelineVersionId'))
            include('pipelines', 'UUID', owner.get('PipelineId'))
            include('experiments', 'UUID', owner.get('ExperimentUUID'))
        for version in tables.get('pipeline_versions', []):
            include('pipelines', 'UUID', version['PipelineId'])
        for table, target, key in [
            ('pipelines', 'pipeline_tags', 'PipelineId'),
            ('pipeline_versions', 'pipeline_version_tags', 'PipelineVersionId')
        ]:
            if target in tables:
                for row in tables[table]:
                    tables[target].extend(
                        db.select(target, **{key: row['UUID']}))
        for table, resource_type in [('run_details', 'Run'), ('jobs', 'Job'),
                                     ('experiments', 'Experiment'),
                                     ('pipelines', 'pipeline'),
                                     ('pipeline_versions', 'PipelineVersion')]:
            for row in tables[table]:
                tables['resource_references'].extend(
                    db.select(
                        'resource_references',
                        ResourceUUID=row['UUID'],
                        ResourceType=resource_type))
        for table, rows in tables.items():
            rows.sort(key=lambda row: canonical(row_key(table, row)))
        graph = mlmd.export_graph(run, tables['tasks'])
        archive = {
            'format': FORMAT,
            'source': source,
            'tables': tables,
            'mlmd': graph,
            'engine': db.driver,
            'schema': {
                table: db.schema(table) for table in tables
            }
        }
        archive['digest'] = archive_digest(archive)
        validate_archive(archive)
        return archive
    finally:
        db.connection.rollback()


def validate_archive(archive):
    if archive.get('format') != FORMAT:
        raise HistoryError(
            'Archive is not KFP 2.18 MLMD history; cross-generation imports are unsupported'
        )
    source = archive.get('source')
    if not isinstance(source, str) or not source or len(source) > 191:
        raise HistoryError('Invalid archive source installation ID')
    tables = archive.get('tables', {})
    if set(tables) - set(KEYS) or not {'run_details', 'tasks'} <= set(tables):
        raise HistoryError('Unexpected or missing archive tables')
    if set(archive.get('schema', {})) != set(tables):
        raise HistoryError('Archive is missing table schema metadata')
    runs = tables['run_details']
    if len(runs) != 1:
        raise HistoryError('An archive must contain exactly one completed run')
    run = runs[0]
    require_completed(run)
    for table, rows in tables.items():
        keys = [row_key(table, row) for row in rows]
        if len(keys) != len(set(keys)):
            raise HistoryError('Duplicate archive row in ' + table)
    for table in ('tasks', 'run_metrics'):
        for row in tables.get(table, []):
            if row['RunUUID'] != run['UUID']:
                raise HistoryError(
                    'Archive contains a row belonging to another run')
    task_ids = {row['UUID'] for row in tables['tasks']}
    parents = {row['UUID']: row.get('ParentTaskUUID') for row in tables['tasks']}
    for parent in parents.values():
        if parent and parent not in task_ids:
            raise HistoryError('Task parent is missing from the archive')
    if archive.get('digest') != archive_digest(archive):
        raise HistoryError('Archive digest does not match its contents')
    from mlmd import require_acyclic
    from mlmd import validate_graph
    require_acyclic(
        [(parent, child) for child, parent in parents.items() if parent],
        'SQL task parent graph')
    validate_graph(archive.get('mlmd', {}))
    graph = archive['mlmd']
    node_ids = {
        kind: {str(row['id']) for row in graph.get(kind, [])}
        for kind in ('contexts', 'executions', 'artifacts')
    }
    for key in ('PipelineContextId', 'PipelineRunContextId'):
        if run.get(key) and str(run[key]) not in node_ids['contexts']:
            raise HistoryError('Run references an absent MLMD context')
    for task in tables['tasks']:
        execution = str(task.get('MLMDExecutionID') or '0')
        if execution != '0' and execution not in node_ids['executions']:
            raise HistoryError('Task references an absent MLMD execution')
        for key in ('MLMDInputs', 'MLMDOutputs'):
            # Validate every typed artifact reference before staging any metadata.
            map_artifact_lists(
                task.get(key),
                {value: value for value in node_ids['artifacts']})
    for table, key in [('experiments', 'ExperimentUUID'),
                       ('pipelines', 'PipelineId'),
                       ('pipeline_versions', 'PipelineVersionId'),
                       ('jobs', 'JobUUID')]:
        if run.get(key) and run[key] not in {
                row['UUID'] for row in tables.get(table, [])
        }:
            raise HistoryError('Run dependency is missing from archive: ' +
                               table)
    for version in tables.get('pipeline_versions', []):
        if version['PipelineId'] not in {
                row['UUID'] for row in tables.get('pipelines', [])
        }:
            raise HistoryError('Pipeline version has an absent parent pipeline')
    for job in tables.get('jobs', []):
        for table, key in [('experiments', 'ExperimentUUID'),
                           ('pipelines', 'PipelineId'),
                           ('pipeline_versions', 'PipelineVersionId')]:
            if job.get(key) and job[key] not in {
                    row['UUID'] for row in tables.get(table, [])
            }:
                raise HistoryError('Recurring run has a missing dependency')
    resources = {
        resource: {
            row['UUID']: row for row in tables.get(table, [])
        } for resource, table in [(
            'Run', 'run_details'), ('Experiment', 'experiments'), (
                'Job',
                'jobs'), ('RecurringRun', 'jobs'), (
                    'pipeline',
                    'pipelines'), ('PipelineVersion', 'pipeline_versions')]
    }
    for row in tables.get('resource_references', []):
        if row['ResourceUUID'] not in resources.get(row['ResourceType'], {}):
            raise HistoryError(
                'Resource reference owner is outside the archive')
        if row['ReferenceType'] != 'Namespace' and row[
                'ReferenceUUID'] not in resources.get(row['ReferenceType'], {}):
            raise HistoryError(
                'Resource reference target is outside the archive')
        if row.get('Payload'):
            payload = json.loads(row['Payload'])
            for key in ('ResourceUUID', 'ResourceType', 'ReferenceUUID',
                        'ReferenceType'):
                if key in payload and payload[key] != row[key]:
                    raise HistoryError(
                        'Resource reference payload disagrees with its columns')
    experiment = resources['Experiment'].get(run.get('ExperimentUUID'))
    if experiment and run.get('Namespace') and experiment.get(
            'Namespace') != run['Namespace']:
        raise HistoryError('Run and experiment namespaces disagree')
    return run


def comparable(table, row):
    value = dict(row)
    if table == 'experiments':
        value.pop('LastRunCreatedAtInSec', None)
    return value


def insert_or_match(db, table, row):
    found = db.select(table, **dict(zip(KEYS[table], row_key(table, row))))
    if found:
        if comparable(table, found[0]) != comparable(table, row):
            raise HistoryError('Destination conflicts with imported ' + table +
                               ' identity')
    else:
        db.insert(table, row)


def map_artifact_lists(text, mapping):
    if not text:
        return text
    data = json.loads(text)

    def visit(value):
        if isinstance(value, dict):
            for key, child in value.items():
                if key in ('artifact_ids', 'artifactIds'):
                    try:
                        value[key] = [int(mapping[str(item)]) for item in child]
                    except (KeyError, TypeError) as error:
                        raise HistoryError(
                            'Task references an absent MLMD artifact'
                        ) from error
                else:
                    visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)

    visit(data)
    return canonical(data)


def import_run(db, mlmd, archive, experiment_id=None, dry_run=False):
    original = validate_archive(archive)
    db.check_generation(importing=True)
    if archive.get('engine') != db.driver:
        raise HistoryError('Cross-engine history import is unsupported')
    for table in archive['tables']:
        if archive['schema'][table] != db.schema(table):
            raise HistoryError('SQL schema mismatch for ' + table +
                               '; use matching 2.18 schemas')
    source, checksum = archive['source'], archive['digest']
    tables = copy.deepcopy(archive['tables'])
    run = tables['run_details'][0]
    db.begin()
    try:
        existing = db.select('run_details', UUID=run['UUID'])
        if existing:
            prior = existing[0]
            if prior.get('ImportedFrom') == source and prior.get(
                    'ImportDigest') == checksum:
                if experiment_id and prior['ExperimentUUID'] != experiment_id:
                    raise HistoryError(
                        'This run was already imported into a different experiment'
                    )
                db.connection.rollback()
                return 'already imported'
            raise HistoryError(
                'Destination run ID already exists with different provenance or contents'
            )
        if experiment_id:
            targets = db.select('experiments', UUID=experiment_id)
            source_experiment = next((row for row in tables['experiments']
                                      if row['UUID'] == run['ExperimentUUID']),
                                     None)
            if len(targets) != 1 or not source_experiment:
                raise HistoryError(
                    'Source and target experiment must exist for explicit mapping'
                )
            if targets[0].get('Namespace',
                              '') != source_experiment.get('Namespace', ''):
                raise HistoryError(
                    'Experiment mapping must preserve namespace authorization')
            old_experiment = run['ExperimentUUID']
            tables['experiments'] = [
                row for row in tables['experiments']
                if row['UUID'] != old_experiment
            ]
            for row in [run] + tables.get('jobs', []):
                if row.get('ExperimentUUID') == old_experiment:
                    row['ExperimentUUID'] = experiment_id
            for row in tables.get('resource_references', []):
                if row['ReferenceType'] == 'Experiment' and row[
                        'ReferenceUUID'] == old_experiment:
                    row['ReferenceUUID'] = experiment_id
                    if row.get('Payload'):
                        payload = json.loads(row['Payload'])
                        payload['ReferenceUUID'] = experiment_id
                        row['Payload'] = canonical(payload)
        run['ImportedFrom'], run['ImportDigest'] = source, checksum
        run['RetryClaimedAtInSec'] = 0
        # Even disabled job rows participate in local ScheduledWorkflow
        # reconciliation. Keep schedule provenance in the archive only.
        run['JobUUID'] = None
        tables['jobs'] = []
        tables['resource_references'] = [
            row for row in tables.get('resource_references', [])
            if row['ResourceType'] not in ('Job', 'RecurringRun') and
            row['ReferenceType'] not in ('Job', 'RecurringRun')
        ]
        # Reserve SQL identities/unique names before any MLMD side effects. These
        # rows remain invisible to other SQL connections until the final commit.
        for table in ORDER:
            if table in ('tasks', 'run_metrics', 'resource_references'):
                continue
            for row in tables.get(table, []):
                insert_or_match(db, table, row)
        for task in tables['tasks']:
            task['Fingerprint'] = ''
            # Initial reservation is rolled back if graph staging fails.
            insert_or_match(db, 'tasks', task)
        for table in ('run_metrics', 'resource_references'):
            for row in tables.get(table, []):
                insert_or_match(db, table, row)
        mlmd.preflight(archive['mlmd'], source)
        if dry_run:
            db.connection.rollback()
            return 'validated (no changes)'
        mapping = mlmd.import_graph(archive['mlmd'], source)
        for field in ('PipelineContextId', 'PipelineRunContextId'):
            value = original.get(field)
            if value:
                run[field] = int(mapping['contexts'][str(value)])
        # Some 2.18 rows never persisted the context IDs; discover by run UUID.
        context_id = archive['mlmd'].get('run_context_id')
        if context_id:
            run['PipelineRunContextId'] = int(
                mapping['contexts'][str(context_id)])
        db.update('run_details', run)
        for task in tables['tasks']:
            old_execution = task.get('MLMDExecutionID')
            if old_execution and old_execution != '0':
                task['MLMDExecutionID'] = str(
                    mapping['executions'][str(old_execution)])
            for field in ('MLMDInputs', 'MLMDOutputs'):
                if field in task:
                    task[field] = map_artifact_lists(task[field],
                                                     mapping['artifacts'])
            if task.get('Payload'):
                payload = json.loads(task['Payload'])
                payload['MLMDExecutionID'] = task.get('MLMDExecutionID', '')
                payload['Fingerprint'] = ''
                for field in ('MLMDInputs', 'MLMDOutputs'):
                    if field in task:
                        payload[field] = task[field]
                task['Payload'] = canonical(payload)
            db.update('tasks', task)
        db.connection.commit()
        return 'imported'
    except BaseException:
        db.connection.rollback()
        raise


def read_archive(path):
    with open(path, 'rb') as stream:
        data = stream.read(MAX_ARCHIVE_BYTES + 1)
    if len(data) > MAX_ARCHIVE_BYTES:
        raise HistoryError('Archive exceeds 256 MiB; export a smaller run')
    return json.loads(data)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        '--db-config', required=True, help='private JSON DB connection config')
    parser.add_argument(
        '--mlmd-config',
        required=True,
        help='private JSON MLMD connection config')
    commands = parser.add_subparsers(dest='command', required=True)
    export = commands.add_parser('export')
    export.add_argument('--run-id', required=True)
    export.add_argument('--source-id', required=True)
    export.add_argument('--output', required=True)
    restore = commands.add_parser('import')
    restore.add_argument('--archive', required=True)
    restore.add_argument('--experiment-id')
    restore.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()
    from mlmd import Metadata
    from mlmd import RPC
    db = connect(args.db_config)
    try:
        with open(args.mlmd_config, encoding='utf-8') as stream:
            metadata = Metadata(RPC(json.load(stream)))
        if args.command == 'export':
            archive = export_run(db, metadata, args.run_id, args.source_id)
            # Archives may contain parameters and URIs; never create them world-readable.
            descriptor = os.open(args.output,
                                 os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, 'w', encoding='utf-8') as stream:
                stream.write(canonical(archive) + '\n')
            print('Exported completed run ' + args.run_id)
        else:
            archive = read_archive(args.archive)
            print(
                import_run(db, metadata, archive, args.experiment_id,
                           args.dry_run))
    finally:
        db.connection.close()


if __name__ == '__main__':
    # mlmd imports these shared types; keep one module identity when run as a script.
    sys.modules.setdefault('history', sys.modules[__name__])
    try:
        main()
    except Exception as error:
        # Driver error messages may contain SQL values or credentials.
        if isinstance(error, HistoryError):
            print(str(error), file=sys.stderr)
        else:
            print(
                'Transfer did not report successful completion (' +
                type(error).__name__ + '). '
                'Inspect destination conflicts and connection configuration, then retry the same '
                'archive. A disconnected commit may have succeeded; MLMD staging may remain.',
                file=sys.stderr)
        sys.exit(1)

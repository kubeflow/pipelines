"""Build the PR 1 source contract from the released MySQL snapshot.

Expected fields are deliberately limited to mappings settled by the migration
design. Unsettled fields remain visible in ``unresolved``; this script must not
silently invent a target representation for them.
"""

import base64
import hashlib
import json
import re
import sys
import uuid
from collections import defaultdict
from functools import cache
from pathlib import Path

import yaml
from google.protobuf.json_format import MessageToDict, ParseDict
from google.protobuf.struct_pb2 import Struct, Value

FIXTURE_DIR = Path(__file__).resolve().parents[1] / 'testdata' / 'release_218'
TABLES = {
    'run_details': ('kfp', 'run_details'),
    'tasks': ('kfp', 'tasks'),
    'Type': ('mlmd', 'types'),
    'Context': ('mlmd', 'contexts'),
    'ContextProperty': ('mlmd', 'context_properties'),
    'Execution': ('mlmd', 'executions'),
    'ExecutionProperty': ('mlmd', 'execution_properties'),
    'Association': ('mlmd', 'associations'),
    'Artifact': ('mlmd', 'artifacts'),
    'ArtifactProperty': ('mlmd', 'artifact_properties'),
    'Attribution': ('mlmd', 'attributions'),
    'Event': ('mlmd', 'events'),
    'EventPath': ('mlmd', 'event_paths'),
    'ParentContext': ('mlmd', 'parent_contexts'),
}
SCHEMAS = re.compile(
    r'^CREATE TABLE `([^`]+)` \(\n(.*?)^\) ENGINE=', re.M | re.S
)
INSERTS = re.compile(r'^INSERT INTO `([^`]+)` VALUES (.*);$', re.M)
COLUMN = re.compile(r'^  `([^`]+)` ', re.M)
ESCAPES = {'0': '\0', 'b': '\b', 'n': '\n', 'r': '\r', 't': '\t', 'Z': '\x1a'}


def sql_values(values: str) -> list[list[object]]:
    """Read mysqldump VALUES tuples without executing source SQL."""
    rows = []
    index = 0
    while index < len(values):
        assert values[index] == '(', values[index : index + 30]
        index += 1
        row = []
        while True:
            if values[index] == "'":
                index += 1
                chars = []
                while True:
                    char = values[index]
                    index += 1
                    if char == "'":
                        break
                    if char == '\\':
                        char = values[index]
                        index += 1
                        char = ESCAPES.get(char, char)
                    chars.append(char)
                value = ''.join(chars)
            else:
                end = index
                while end < len(values) and values[end] not in ',)':
                    end += 1
                token = values[index:end]
                if token == 'NULL':
                    value = None
                elif re.fullmatch(r'-?\d+', token):
                    value = int(token)
                else:
                    value = float(token)
                index = end
            row.append(value)
            delimiter = values[index]
            index += 1
            if delimiter == ')':
                break
            assert delimiter == ',', delimiter
        rows.append(row)
        if index < len(values):
            assert values[index] == ',', values[index]
            index += 1
    return rows


def source_from_sql() -> dict:
    sql = (FIXTURE_DIR / 'source.sql').read_text()
    columns = {
        name: COLUMN.findall(body)
        for name, body in SCHEMAS.findall(sql)
        if name in TABLES
    }
    assert set(columns) == set(TABLES), set(TABLES) - set(columns)
    source = {'kfp': {}, 'mlmd': {}}
    for section, key in TABLES.values():
        source[section][key] = []
    for name, values in INSERTS.findall(sql):
        if name not in TABLES:
            continue
        section, key = TABLES[name]
        for fields in sql_values(values):
            assert len(fields) == len(columns[name]), name
            source[section][key].append(dict(zip(columns[name], fields)))
    for section in source.values():
        for rows in section.values():
            rows.sort(
                key=lambda row: (
                    str(row.get('id', row.get('UUID', ''))),
                    json.dumps(row, sort_keys=True),
                )
            )
    return source


def identity(kind: str, source_id: int | str) -> str:
    return str(uuid.uuid5(uuid.NAMESPACE_DNS, f'{kind}:{source_id}'))


def typed_property(row: dict) -> dict:
    for field, kind in (
        ('int_value', 'int'),
        ('double_value', 'double'),
        ('string_value', 'string'),
        ('bool_value', 'bool'),
    ):
        if row[field] is not None:
            return {'kind': kind, 'value': row[field]}
    return {'kind': 'unset', 'value': None}


def property_index(rows: list[dict], owner: str) -> dict[int, dict]:
    result = defaultdict(dict)
    for row in rows:
        assert row['name'] not in result[row[owner]], (owner, row)
        result[row[owner]][row['name']] = typed_property(row)
    return result


ScalarPropertyValue = str | int | float | bool | None


def property_value(
    props: dict, name: str, default: ScalarPropertyValue = None
) -> ScalarPropertyValue:
    return props[name]['value'] if name in props else default


PARAMETER_COLUMNS = {'inputs': 'InputParameters', 'outputs': 'OutputParameters'}


def project_parameters(runs: dict, tasks: list, exec_props: dict) -> None:
    """Add captured values and prove parameter origins within each saved
    DAG."""
    specs = {
        run_id: yaml.safe_load(run['PipelineSpecManifest']) or {}
        for run_id, run in runs.items()
    }
    tasks_by_uuid = {task['UUID']: task for task in tasks}
    siblings = defaultdict(list)
    for task in tasks:
        siblings[task['RunUUID'], task['ParentTaskUUID'], task['Name']].append(
            task
        )

    @cache
    def parameters(task_id: str, direction: str) -> dict[str, Value] | None:
        encoded = property_value(exec_props[task_id], direction)
        if encoded is None:
            return None
        prefix, marker, payload = encoded.partition('::')
        if prefix != 'mlmd-struct' or not marker:
            raise ValueError(
                f'Unexpected {direction} encoding for task {task_id}'
            )
        return Struct.FromString(
            base64.b64decode(payload, validate=True)
        ).fields

    @cache
    def task_spec(task_id: str) -> tuple[dict, dict]:
        task = tasks_by_uuid[task_id]
        spec = specs[task['RunUUID']]
        parent = tasks_by_uuid.get(task['ParentTaskUUID'])
        if parent is None:
            return {}, spec.get('root', {})
        if parent['RunUUID'] != task['RunUUID']:
            return {}, {}
        parent_spec, component = task_spec(parent['UUID'])
        if (
            task['Name'] == parent['Name']
            and 'iterationIndex' in task['TypeAttrs']
            and 'iterationCount' in parent['TypeAttrs']
            and 'parameterIterator' in parent_spec
        ):
            return parent_spec, component
        spec_task = (
            component.get('dag', {}).get('tasks', {}).get(task['Name'], {})
        )
        component_name = spec_task.get('componentRef', {}).get('name')
        return spec_task, spec.get('components', {}).get(component_name, {})

    def input_origin(task: dict, key: str, value: Value) -> dict:
        spec_task, component = task_spec(task['UUID'])
        if task['Type'] == 1:
            runtime = json.loads(
                runs[task['RunUUID']]['RuntimeParameters'] or '{}'
            )
            if key in runtime and ParseDict(runtime[key], Value()) == value:
                return {'type': 'RUNTIME_VALUE_INPUT'}
            return {}

        bindings = spec_task.get('inputs', {}).get('parameters', {})
        binding = bindings.get(key, {})
        constant = binding.get('runtimeValue', {})
        if (
            'constant' in constant
            and ParseDict(constant['constant'], Value()) == value
        ):
            return {'type': 'RUNTIME_VALUE_INPUT'}
        parent_id = task['ParentTaskUUID']
        parent_key = binding.get('componentInputParameter')
        if (
            parent_key
            and (parameters(parent_id, 'inputs') or {}).get(parent_key) == value
        ):
            return {'type': 'COMPONENT_INPUT'}
        output = binding.get('taskOutputParameter')
        if output:
            producers = siblings[
                task['RunUUID'], parent_id, output['producerTask']
            ]
            if len(producers) == 1:
                producer = producers[0]
                values = parameters(producer['UUID'], 'outputs') or {}
                if (
                    producer['Type'] == 2
                    and 'iterationIndex' not in producer['TypeAttrs']
                    and values.get(output['outputParameterKey']) == value
                ):
                    return {
                        'type': 'TASK_OUTPUT_INPUT',
                        'producer': {'taskName': producer['Name']},
                    }

        iterator = spec_task.get('parameterIterator', {})
        if key == iterator.get('itemInput'):
            index = task['TypeAttrs'].get('iterationIndex')
            raw = iterator.get('items', {}).get('raw')
            if index is not None and raw is not None:
                items = json.loads(raw)
                if (
                    0 <= int(index) < len(items)
                    and ParseDict(items[int(index)], Value()) == value
                ):
                    return {
                        'type': 'ITERATOR_INPUT',
                        'producer': {
                            'taskName': task['Name'],
                            'iteration': index,
                        },
                    }
            return {}

        definitions = component.get('inputDefinitions', {}).get(
            'parameters', {}
        )
        default = definitions.get(key, {})
        if (
            key not in bindings
            and 'defaultValue' in default
            and ParseDict(default['defaultValue'], Value()) == value
        ):
            return {'type': 'COMPONENT_DEFAULT_INPUT'}
        return {}

    for task in tasks:
        for direction, column in PARAMETER_COLUMNS.items():
            values = parameters(task['UUID'], direction)
            if values is None:
                continue
            entries = []
            for key, value in sorted(values.items()):
                entry = {'parameterKey': key, 'value': MessageToDict(value)}
                if direction == 'inputs':
                    entry.update(input_origin(task, key, value))
                elif task['Type'] == 2 and task['Name']:
                    producer = {'taskName': task['Name']}
                    index = task['TypeAttrs'].get('iterationIndex')
                    entry.update(type='OUTPUT', producer=producer)
                    if index is not None:
                        entry['type'] = 'ITERATOR_OUTPUT'
                        producer['iteration'] = index
                entries.append(entry)
            task[column] = entries


def verify_legacy_payload(rows: list[dict]) -> None:
    fields = {
        'UUID': 'UUID',
        'Namespace': 'Namespace',
        'PipelineName': 'PipelineName',
        'RunID': 'RunUUID',
        'PodName': 'PodName',
        'MLMDExecutionID': 'MLMDExecutionID',
        'CreatedTimestamp': 'CreatedTimestamp',
        'StartedTimestamp': 'StartedTimestamp',
        'FinishedTimestamp': 'FinishedTimestamp',
        'Fingerprint': 'Fingerprint',
        'Name': 'Name',
        'ParentTaskId': 'ParentTaskUUID',
        'State': 'State',
        'MLMDInputs': 'MLMDInputs',
        'MLMDOutputs': 'MLMDOutputs',
    }
    for row in rows:
        payload = json.loads(row['Payload'])
        for payload_field, row_field in fields.items():
            assert payload[payload_field] == row[row_field], (
                row['UUID'],
                payload_field,
            )
        for field in ('StateHistory', 'ChildrenPods'):
            saved = json.loads(row[field]) if row[field] else None
            assert payload[field] == saved, (row['UUID'], field)


def build_expected(source: dict) -> tuple[dict, list[dict]]:
    kfp, mlmd = source['kfp'], source['mlmd']
    verify_legacy_payload(kfp['tasks'])
    types = {row['id']: row['name'] for row in mlmd['types']}
    contexts = {row['id']: row for row in mlmd['contexts']}
    runs = {row['UUID']: row for row in kfp['run_details']}
    exec_props = property_index(mlmd['execution_properties'], 'execution_id')
    artifact_props = property_index(mlmd['artifact_properties'], 'artifact_id')
    cache_rows = defaultdict(list)
    for task in kfp['tasks']:
        if task['MLMDExecutionID']:
            cache_rows[int(task['MLMDExecutionID']), task['RunUUID']].append(
                task
            )
    exec_runs = defaultdict(set)
    for association in mlmd['associations']:
        context = contexts[association['context_id']]
        if types[context['type_id']] == 'system.PipelineRun':
            exec_runs[association['execution_id']].add(context['name'])
    paths = defaultdict(list)
    for path in mlmd['event_paths']:
        paths[path['event_id']].append(path)

    expected = {
        'run_details': [
            {'source': f'kfp.run_details[UUID={row["UUID"]}]', 'fields': row}
            for row in kfp['run_details']
        ],
        'tasks': [],
        'artifacts': [],
        'artifact_tasks': [],
        'api': [],
        'old_task_aliases': [],
        'skips': [],
        'diagnostics': [],
    }
    for context in mlmd['contexts']:
        if (
            types[context['type_id']] == 'system.PipelineRun'
            and context['name'] not in runs
        ):
            expected['skips'].append(
                {
                    'source': f'mlmd.contexts[id={context["id"]}]',
                    'reason': 'run context has no matching run_details row',
                    'severity': 'warning',
                }
            )
    unresolved = []
    execution_rows = {row['id']: row for row in mlmd['executions']}
    execution_skips = {}

    def execution_skip(
        execution_id: int, visiting: set[int] | None = None
    ) -> tuple[str, str] | None:
        if execution_id in execution_skips:
            return execution_skips[execution_id]
        if visiting is None:
            visiting = set()
        if execution_id in visiting:
            return 'execution parent cycle', 'error'
        row = execution_rows[execution_id]
        owned_runs = exec_runs[execution_id]
        if len(owned_runs) != 1 or not owned_runs <= runs.keys():
            result = ('run context has no matching run_details row', 'warning')
        elif types[row['type_id']] not in (
            'system.DAGExecution',
            'system.ContainerExecution',
            'system.ImporterExecution',
        ):
            result = ('unknown execution type', 'warning')
        else:
            parent = property_value(exec_props[execution_id], 'parent_dag_id')
            if parent and parent not in execution_rows:
                result = (f'missing parent execution:{parent}', 'error')
            elif parent:
                parent_result = execution_skip(
                    parent, visiting | {execution_id}
                )
                if parent_result:
                    result = (f'parent execution:{parent} was skipped', 'error')
                elif exec_runs[parent] != owned_runs:
                    result = (
                        f'parent execution:{parent} belongs to a different run',
                        'error',
                    )
                else:
                    result = None
            else:
                result = None
        execution_skips[execution_id] = result
        return result

    for row in mlmd['executions']:
        source_ref = f'mlmd.executions[id={row["id"]}]'
        skip = execution_skip(row['id'])
        if skip:
            reason, severity = skip
            expected['skips'].append(
                {
                    'source': source_ref,
                    'target_uuid': identity('execution', row['id']),
                    'reason': reason,
                    'severity': severity,
                }
            )
            continue
        props = exec_props[row['id']]
        owned_runs = exec_runs[row['id']]
        (run_id,) = owned_runs
        if 'display_name' not in props:
            expected['diagnostics'].append(
                {
                    'source': source_ref,
                    'reason': 'missing optional display_name; default to empty',
                    'severity': 'warning',
                }
            )
        kind = types[row['type_id']]
        parent = property_value(props, 'parent_dag_id')
        name = property_value(props, 'task_name', '')
        if kind == 'system.ContainerExecution':
            task_type = 2
        elif kind == 'system.ImporterExecution':
            task_type = 7
        elif not parent:
            task_type = 1
        elif name.startswith('condition-branches-'):
            task_type = 3
        elif name.startswith('condition-'):
            task_type = 4
        elif name.startswith(('exit-handler-', 'on-exit-', 'onexit-')):
            task_type = 6
        elif 'iteration_count' in props:
            task_type = 5
        else:
            task_type = 8
        target_id = identity('execution', row['id'])
        linked_cache_rows = cache_rows[row['id'], run_id]
        cache_row = linked_cache_rows[0] if len(linked_cache_rows) == 1 else {}
        mlmd_pod_name = property_value(props, 'pod_name')
        pod_name = cache_row.get('PodName') or mlmd_pod_name
        pod = {'name': pod_name} if pod_name else None
        if (
            pod
            and pod_name == mlmd_pod_name
            and property_value(props, 'pod_uid')
        ):
            pod['uid'] = property_value(props, 'pod_uid')
        type_attrs = {
            target: str(property_value(props, source))
            for source, target in (
                ('iteration_index', 'iterationIndex'),
                ('iteration_count', 'iterationCount'),
            )
            if property_value(props, source) is not None
        }
        cached_id = property_value(props, 'cached_execution_id')
        cached_execution_id = int(cached_id) if cached_id else None
        cached_task_id = (
            identity('execution', cached_execution_id)
            if cached_execution_id in execution_rows
            and not execution_skip(cached_execution_id)
            else None
        )
        expected['tasks'].append(
            {
                'source': source_ref,
                'target_uuid': target_id,
                'disposition': 'mlmd_execution',
                'fields': {
                    'UUID': target_id,
                    'RunUUID': run_id,
                    'Namespace': property_value(
                        props, 'namespace', runs[run_id]['Namespace']
                    ),
                    'Name': name,
                    'DisplayName': property_value(props, 'display_name', ''),
                    'CreatedAtInSec': cache_row.get('CreatedTimestamp')
                    or row['create_time_since_epoch'] // 1000,
                    'StartedInSec': cache_row.get('StartedTimestamp') or 0,
                    'FinishedInSec': cache_row.get('FinishedTimestamp') or 0,
                    'Pods': [pod] if pod else [],
                    'TypeAttrs': type_attrs,
                    **(
                        {
                            'StatusMetadata': {
                                'customProperties': {
                                    'cached_task_id': cached_task_id
                                }
                            }
                        }
                        if cached_task_id
                        else {}
                    ),
                    'Type': task_type,
                    'State': {1: 0, 2: 1, 3: 2, 4: 4, 5: 5, 6: 4}[
                        row['last_known_state']
                    ],
                    'ParentTaskUUID': identity('execution', parent)
                    if parent
                    else None,
                    'Fingerprint': property_value(
                        props, 'cache_fingerprint', ''
                    )
                    if row['last_known_state'] in (3, 5)
                    else '',
                },
            }
        )
        unresolved.append(
            {
                'source': source_ref,
                'fields': [
                    'create_time_since_epoch',
                    'last_update_time_since_epoch',
                    'execution_properties.image',
                    *(
                        ['execution_properties.cached_execution_id']
                        if cached_id and not cached_task_id
                        else []
                    ),
                    'execution_properties.total_dag_tasks',
                ],
                'reason': 'Exact native storage for remaining execution metadata and millisecond times is not specified',
            }
        )

    project_parameters(
        runs,
        [row['fields'] for row in expected['tasks']],
        {
            identity('execution', key): props
            for key, props in exec_props.items()
        },
    )
    for row in expected['tasks']:
        pending = [
            f'execution_properties.{direction}'
            for direction, column in PARAMETER_COLUMNS.items()
            if any(
                'type' not in entry for entry in row['fields'].get(column, [])
            )
        ]
        if pending:
            unresolved.append(
                {
                    'source': row['source'],
                    'fields': pending,
                    'reason': 'Parameter values are preserved but some I/O types or producer links are unverified',
                }
            )

    tasks_by_uuid = {
        row['target_uuid']: row['fields'] for row in expected['tasks']
    }
    for row in kfp['tasks']:
        source_ref = f'kfp.tasks[UUID={row["UUID"]}]'
        if not row['MLMDExecutionID']:
            expected['skips'].append(
                {
                    'source': source_ref,
                    'run_uuid': row['RunUUID'],
                    'reason': 'Workflow report stays in the saved run manifest',
                    'severity': 'info',
                }
            )
            continue
        target_id = identity('execution', int(row['MLMDExecutionID']))
        task = tasks_by_uuid.get(target_id)
        if task is None or task['RunUUID'] != row['RunUUID']:
            expected['skips'].append(
                {
                    'source': source_ref,
                    'run_uuid': row['RunUUID'],
                    'reason': 'cache row has no eligible execution in the same run',
                    'severity': 'warning',
                }
            )
            continue
        expected['old_task_aliases'].append(
            {
                'source': source_ref,
                'target_uuid': target_id,
                'disposition': 'covered_by_mlmd_execution',
            }
        )

    artifact_runs = defaultdict(set)
    for attribution in mlmd['attributions']:
        context = contexts[attribution['context_id']]
        if types[context['type_id']] == 'system.PipelineRun':
            artifact_runs[attribution['artifact_id']].add(context['name'])
    for row in mlmd['artifacts']:
        source_ref = f'mlmd.artifacts[id={row["id"]}]'
        props = artifact_props[row['id']]
        owned_runs = artifact_runs[row['id']] & runs.keys()
        if not owned_runs:
            expected['skips'].append(
                {
                    'source': source_ref,
                    'reason': 'artifact has no matching KFP run',
                    'severity': 'warning',
                }
            )
            continue
        namespaces = {runs[run_id]['Namespace'] for run_id in owned_runs}
        assert len(namespaces) == 1, (source_ref, namespaces)
        (namespace,) = namespaces
        kind = types[row['type_id']]
        if kind not in ('system.Dataset', 'system.Artifact', 'system.Metrics'):
            expected['skips'].append(
                {
                    'source': source_ref,
                    'reason': f'unknown artifact type: {kind}',
                    'severity': 'warning',
                }
            )
            continue
        metrics = sorted(
            name
            for name, prop in props.items()
            if kind == 'system.Metrics' and prop['kind'] in ('int', 'double')
        )
        if kind == 'system.Metrics' and not metrics:
            expected['skips'].append(
                {
                    'source': source_ref,
                    'reason': 'metrics artifact has no scalar numeric property',
                    'severity': 'warning',
                }
            )
            continue
        targets = metrics if kind == 'system.Metrics' else [None]
        for index, metric in enumerate(targets):
            target_id = (
                identity('artifact', row['id'])
                if index == 0
                else identity('artifact', f'{row["id"]}:metric:{metric}')
            )
            metadata = {
                name: value['value']
                for name, value in props.items()
                if name not in ('display_name', *metrics)
            }
            expected['artifacts'].append(
                {
                    'source': source_ref,
                    'target_uuid': target_id,
                    'metric_name': metric,
                    'fields': {
                        'UUID': target_id,
                        'Namespace': namespace,
                        'Type': 6
                        if metric
                        else {'system.Dataset': 3, 'system.Artifact': 1}[kind],
                        'URI': row['uri'],
                        'CreatedAtInSec': row['create_time_since_epoch']
                        // 1000,
                        'LastUpdateInSec': row['last_update_time_since_epoch']
                        // 1000,
                        'Name': metric
                        or property_value(props, 'display_name', '')
                        or row['name']
                        or '',
                        'Metadata': metadata,
                        'NumberValue': property_value(props, metric)
                        if metric
                        else None,
                    },
                }
            )
        unresolved.append(
            {
                'source': source_ref,
                'fields': [
                    'create_time_since_epoch',
                    'last_update_time_since_epoch',
                    'name',
                    'external_id',
                    'state',
                ],
                'reason': 'Exact native storage for original MLMD artifact fields and millisecond times is not specified',
            }
        )

    skipped_executions = {
        int(item['source'][len('mlmd.executions[id=') : -1]): item['severity']
        for item in expected['skips']
        if item['source'].startswith('mlmd.executions[id=')
    }
    skipped_artifacts = {
        int(item['source'][len('mlmd.artifacts[id=') : -1])
        for item in expected['skips']
        if item['source'].startswith('mlmd.artifacts[id=')
    }
    for row in mlmd['events']:
        source_ref = f'mlmd.events[id={row["id"]}]'
        if (
            row['execution_id'] in skipped_executions
            or row['artifact_id'] in skipped_artifacts
        ):
            expected['skips'].append(
                {
                    'source': source_ref,
                    'target_uuid': identity('event', row['id']),
                    'reason': 'dependent execution or artifact was skipped',
                    'severity': skipped_executions.get(
                        row['execution_id'], 'warning'
                    ),
                }
            )
            continue
        (run_id,) = exec_runs[row['execution_id']]
        target_id = identity('event', row['id'])
        steps = paths[row['id']]
        key = None
        if len(steps) == 1 and not steps[0]['is_index_step']:
            key = steps[0]['step_key'] or None
        expected['artifact_tasks'].append(
            {
                'source': source_ref,
                'target_uuid': target_id,
                'fields': {
                    'UUID': target_id,
                    'TaskID': identity('execution', row['execution_id']),
                    'ArtifactID': identity('artifact', row['artifact_id']),
                    'RunUUID': run_id,
                    'Type': 9 if row['type'] == 4 else None,
                    'ArtifactKey': key,
                },
            }
        )
        unresolved.append(
            {
                'source': source_ref,
                'fields': [
                    'milliseconds_since_epoch',
                    *(
                        ['event_paths indexed or composite path']
                        if key is None
                        else []
                    ),
                    *(['type'] if row['type'] != 4 else []),
                ],
                'reason': 'Native event time and indexed-path representation are not specified',
            }
        )

    for run_id in sorted(runs):
        rows = [
            row
            for row in expected['tasks']
            if row['fields'] and row['fields']['RunUUID'] == run_id
        ]
        expected['api'].append(
            {
                'run_id': run_id,
                'task_ids': sorted(row['target_uuid'] for row in rows),
                'source_api_file': 'api_runs.json',
                'status': 'pending_field_mapping',
            }
        )
    for name in ('tasks', 'artifacts', 'artifact_tasks'):
        expected[name].sort(key=lambda row: (row['target_uuid'], row['source']))
    return expected, unresolved


def dispositions(source: dict, expected: dict) -> list[dict]:
    result = []
    skip_reasons = {
        item['source']: item['reason'] for item in expected['skips']
    }
    type_names = {row['id']: row['name'] for row in source['mlmd']['types']}
    artifact_types = {
        row['id']: type_names[row['type_id']]
        for row in source['mlmd']['artifacts']
    }
    artifact_props = property_index(
        source['mlmd']['artifact_properties'], 'artifact_id'
    )
    execution_states = {
        row['id']: row['last_known_state']
        for row in source['mlmd']['executions']
    }
    task_aliases = {
        row['source']: row['target_uuid']
        for row in expected['old_task_aliases']
    }
    task_fields = {
        row['target_uuid']: row['fields'] for row in expected['tasks']
    }
    event_fields = {
        row['target_uuid']: row['fields'] for row in expected['artifact_tasks']
    }
    for section, tables in source.items():
        for table, rows in tables.items():
            for index, row in enumerate(rows):
                key = row.get('id', row.get('UUID', index))
                owner = f'{section}.{table}[{key}]'
                for field in row:
                    path = f'{owner}.{field}'
                    targets = []
                    reason = ''
                    partial = False
                    if table == 'run_details':
                        targets = [
                            f'expected.run_details[UUID={key}].fields.{field}'
                        ]
                    elif table == 'tasks':
                        task_source = f'kfp.tasks[UUID={key}]'
                        task_id = task_aliases.get(task_source)
                        target_field = {
                            'RunUUID': 'RunUUID',
                            'Namespace': 'Namespace',
                            'Fingerprint': 'Fingerprint',
                            'CreatedTimestamp': 'CreatedAtInSec',
                            'StartedTimestamp': 'StartedInSec',
                            'FinishedTimestamp': 'FinishedInSec',
                        }.get(field)
                        if field == 'Payload':
                            reason = 'Checked duplicate of the other captured historical task fields'
                        elif task_id and field in ('UUID', 'MLMDExecutionID'):
                            targets = [
                                f'expected.old_task_aliases[source={task_source}].target_uuid'
                            ]
                        elif (
                            task_id
                            and target_field
                            and task_fields[task_id].get(target_field)
                            == row[field]
                        ):
                            targets = [
                                f'expected.tasks[UUID={task_id}].fields.{target_field}'
                            ]
                        elif row[field] in ('', 'null', None) or (
                            field == 'State'
                            and row[field] == 'RUNTIME_STATE_UNSPECIFIED'
                        ):
                            reason = 'Legacy cache row carries no value for this field'
                        else:
                            reason = 'Legacy cache field needs an explicit native/API representation'
                    elif table == 'executions':
                        mapped = {
                            'id': 'UUID',
                            'type_id': 'Type',
                            'last_known_state': 'State',
                        }
                        if field in mapped:
                            targets = [
                                f'expected.tasks[UUID={identity("execution", key)}].fields.{mapped[field]}'
                            ]
                        elif (
                            field == 'create_time_since_epoch'
                            and task_fields.get(
                                identity('execution', key), {}
                            ).get('CreatedAtInSec')
                            == row[field] // 1000
                        ):
                            targets = [
                                f'expected.tasks[UUID={identity("execution", key)}].fields.CreatedAtInSec'
                            ]
                            partial = True
                            reason = 'Native creation time is seconds; original millisecond precision has no settled destination'
                        elif row[field] is None:
                            reason = 'No source execution value'
                        else:
                            reason = 'Execution field needs an exact native/API representation'
                    elif table == 'execution_properties':
                        target_id = identity('execution', row['execution_id'])
                        mapped = {
                            'task_name': 'Name',
                            'display_name': 'DisplayName',
                            'namespace': 'Namespace',
                            'parent_dag_id': 'ParentTaskUUID',
                            'cache_fingerprint': 'Fingerprint',
                            'pod_name': 'Pods[0].name',
                            'pod_uid': 'Pods[0].uid',
                            'cached_execution_id': 'StatusMetadata.customProperties.cached_task_id',
                            'iteration_index': 'TypeAttrs.iterationIndex',
                            'iteration_count': 'TypeAttrs.iterationCount',
                            **PARAMETER_COLUMNS,
                        }
                        native = task_fields.get(target_id, {})
                        present = {
                            **{
                                name: column in native
                                for name, column in PARAMETER_COLUMNS.items()
                            },
                            'pod_name': bool(native.get('Pods'))
                            and native['Pods'][0]['name']
                            == typed_property(row)['value'],
                            'pod_uid': bool(
                                native.get('Pods')
                                and native['Pods'][0].get('uid')
                            ),
                            'cached_execution_id': bool(
                                native.get('StatusMetadata')
                            ),
                            'iteration_index': 'iterationIndex'
                            in native.get('TypeAttrs', {}),
                            'iteration_count': 'iterationCount'
                            in native.get('TypeAttrs', {}),
                        }
                        if row[
                            'name'
                        ] == 'cache_fingerprint' and execution_states[
                            row['execution_id']
                        ] not in (3, 5):
                            reason = 'Fingerprint excluded because execution is neither COMPLETE nor CACHED'
                        elif field == 'execution_id':
                            targets = [
                                f'expected.tasks[UUID={target_id}].source'
                            ]
                        elif (
                            field
                            in (
                                'name',
                                'int_value',
                                'double_value',
                                'string_value',
                                'bool_value',
                            )
                            and row['name'] in mapped
                            and present.get(row['name'], True)
                            and row[field] is not None
                        ):
                            column = mapped[row['name']]
                            targets = [
                                f'expected.tasks[UUID={target_id}].fields.{column}'
                            ]
                            if row['name'] in PARAMETER_COLUMNS and any(
                                'type' not in entry for entry in native[column]
                            ):
                                partial = True
                                reason = 'Parameter values are preserved but some I/O types or producer links are unverified'
                        elif row[field] is None:
                            reason = 'Unused slot in the typed execution-property record'
                        else:
                            reason = 'Typed execution property needs an exact native/API representation'
                    elif table == 'artifacts':
                        mapped = {
                            'id': 'UUID',
                            'type_id': 'Type',
                            'uri': 'URI',
                            'create_time_since_epoch': 'CreatedAtInSec',
                            'last_update_time_since_epoch': 'LastUpdateInSec',
                        }
                        if field in mapped:
                            targets = [
                                f'expected.artifacts[source={owner}].fields.{mapped[field]}'
                            ]
                            if field.endswith('_since_epoch'):
                                partial = True
                                reason = 'Native artifact times are seconds; original millisecond precision has no settled destination'
                        elif (
                            field == 'name'
                            and row[field]
                            and artifact_types[key] != 'system.Metrics'
                            and not property_value(
                                artifact_props[key], 'display_name'
                            )
                        ):
                            targets = [
                                f'expected.artifacts[source={owner}].fields.Name'
                            ]
                        elif row[field] is None:
                            reason = 'No source artifact value'
                        else:
                            reason = 'Artifact field needs an exact native/API representation'
                    elif table == 'artifact_properties':
                        prop = row['name']
                        artifact_ref = (
                            f'mlmd.artifacts[id={row["artifact_id"]}]'
                        )
                        if field == 'artifact_id':
                            targets = [
                                f'expected.artifacts[source={artifact_ref}].source'
                            ]
                        elif prop == 'display_name' and field in (
                            'name',
                            'string_value',
                        ):
                            targets = [
                                f'expected.artifacts[source={artifact_ref}].fields.Name'
                            ]
                        elif (
                            artifact_types[row['artifact_id']]
                            == 'system.Metrics'
                            and typed_property(row)['kind'] in ('int', 'double')
                            and field in ('name', 'int_value', 'double_value')
                            and row[field] is not None
                        ):
                            targets = [
                                f'expected.artifacts[source={artifact_ref},metric={prop}].fields.NumberValue'
                            ]
                        elif (
                            prop != 'display_name'
                            and (
                                field == 'name'
                                or field
                                in (
                                    'int_value',
                                    'double_value',
                                    'string_value',
                                    'bool_value',
                                )
                                and row[field] is not None
                            )
                            and typed_property(row)['kind'] != 'unset'
                        ):
                            targets = [
                                f'expected.artifacts[source={artifact_ref}].fields.Metadata.{prop}'
                            ]
                        elif row[field] is None:
                            reason = 'Unused slot in the typed artifact-property record'
                        else:
                            reason = 'Typed artifact property needs an exact native/API representation'
                    elif table == 'events':
                        event_id = identity('event', key)
                        mapped = {
                            'id': 'UUID',
                            'execution_id': 'TaskID',
                            'artifact_id': 'ArtifactID',
                            'type': 'Type',
                        }
                        if (
                            field == 'type'
                            and event_fields.get(event_id, {}).get('Type')
                            is None
                        ):
                            reason = 'Event type needs a native representation'
                        elif field in mapped:
                            targets = [
                                f'expected.artifact_tasks[UUID={event_id}].fields.{mapped[field]}'
                            ]
                        else:
                            reason = 'Event time needs a native representation'
                    elif table == 'event_paths':
                        event_id = identity('event', row['event_id'])
                        artifact_key = event_fields.get(event_id, {}).get(
                            'ArtifactKey'
                        )
                        if field == 'event_id':
                            targets = [
                                f'expected.artifact_tasks[UUID={event_id}].source'
                            ]
                        elif artifact_key and artifact_key == row['step_key']:
                            if field == 'step_key':
                                targets = [
                                    f'expected.artifact_tasks[UUID={event_id}].fields.ArtifactKey'
                                ]
                            else:
                                reason = 'Named path component is represented by ArtifactKey'
                        else:
                            reason = 'Indexed or composite event path needs a native representation'
                    elif table in (
                        'types',
                        'contexts',
                        'context_properties',
                        'associations',
                        'attributions',
                        'parent_contexts',
                    ):
                        if table == 'context_properties' and row['name'] in (
                            'pipeline_root',
                            'store_session_info',
                            'resource_name',
                        ):
                            reason = 'Historical context property has no settled native destination'
                        else:
                            reason = {
                                'types': 'Resolve MLMD type ID to source entity kind',
                                'contexts': 'Resolve run and pipeline context ownership',
                                'context_properties': 'Resolve context namespace',
                                'associations': 'Resolve execution to run',
                                'attributions': 'Resolve artifact to namespace and run',
                                'parent_contexts': 'Validate pipeline/run context hierarchy',
                            }[table]
                    else:
                        raise AssertionError(path)
                    source_ref = f'{section}.{table}[id={key}]'
                    if table in (
                        'execution_properties',
                        'artifact_properties',
                        'event_paths',
                    ):
                        owner_table, owner_id = {
                            'execution_properties': (
                                'executions',
                                row.get('execution_id'),
                            ),
                            'artifact_properties': (
                                'artifacts',
                                row.get('artifact_id'),
                            ),
                            'event_paths': ('events', row.get('event_id')),
                        }[table]
                        source_ref = f'mlmd.{owner_table}[id={owner_id}]'
                    elif table in (
                        'context_properties',
                        'associations',
                        'attributions',
                        'parent_contexts',
                    ):
                        source_ref = f'mlmd.contexts[id={row["context_id"]}]'
                    elif table == 'tasks':
                        source_ref = task_source
                    if source_ref in skip_reasons:
                        status = 'skip'
                        targets = []
                        reason = skip_reasons[source_ref]
                    else:
                        status = (
                            'redundant'
                            if table == 'tasks' and field == 'Payload'
                            else 'skip'
                            if table == 'execution_properties'
                            and row['name'] == 'cache_fingerprint'
                            and execution_states[row['execution_id']]
                            not in (3, 5)
                            else 'mapped_pending_design'
                            if partial
                            else 'mapped'
                            if targets
                            else 'pending_design'
                            if 'needs' in reason or 'no settled' in reason
                            else 'lookup_only'
                        )
                    result.append(
                        {
                            'source': path,
                            'status': status,
                            'targets': targets,
                            'reason': reason,
                        }
                    )
    return result


def write_or_check(name: str, content: str, check: bool) -> None:
    path = FIXTURE_DIR / name
    if check:
        assert path.read_text() == content, f'{name} differs from source.sql'
    else:
        path.write_text(content)


def main() -> None:
    check = sys.argv[1:] == ['--check']
    assert not sys.argv[1:] or check, (
        'usage: build_migration_fixture.py [--check]'
    )
    source = source_from_sql()
    expected, unresolved = build_expected(source)
    contract = {
        'format_version': 1,
        'contract_status': 'provisional_due_to_unresolved_fields',
        'design_conflicts': [],
        'source': source,
        'expected': expected,
        'dispositions': dispositions(source, expected),
        'unresolved': unresolved,
        'assertions': {
            'target_counts_status': 'provisional',
            'counts': {
                'source': {
                    f'{section}.{table}': len(rows)
                    for section, tables in source.items()
                    for table, rows in tables.items()
                },
                'target': {
                    'tasks': len(expected['tasks']),
                    'artifacts': len(expected['artifacts']),
                    'artifact_tasks': len(expected['artifact_tasks']),
                },
            },
            'id_mappings': [
                {'source': row['source'], 'target_uuid': row['target_uuid']}
                for row in expected['tasks'] + expected['old_task_aliases']
            ],
        },
    }
    write_or_check(
        'migration_fixture.json',
        json.dumps(contract, indent=2, sort_keys=True, ensure_ascii=False)
        + '\n',
        check,
    )
    files = (
        'source.sql',
        'workflows.json',
        'api_runs.json',
        'deployment-images.json',
        'install-config.json',
    )
    schema_definitions = sorted(
        (name, body)
        for name, body in SCHEMAS.findall(
            (FIXTURE_DIR / 'source.sql').read_text()
        )
    )
    provenance = {
        'kfp_release': '2.18.0',
        'kfp_commit': '555bfa60a70da88c594960bdacfd78fa7c061043',
        'source_databases': ['mlpipeline', 'metadb'],
        'source_schema_sha256': hashlib.sha256(
            json.dumps(schema_definitions, separators=(',', ':')).encode()
        ).hexdigest(),
        'checksums_sha256': {
            name: hashlib.sha256((FIXTURE_DIR / name).read_bytes()).hexdigest()
            for name in files
        },
        'counts': contract['assertions']['counts'],
        'capture_commands': ['capture_api.py', 'capture_fixture.sh'],
    }
    write_or_check(
        'provenance.json',
        json.dumps(provenance, indent=2, sort_keys=True) + '\n',
        check,
    )


if __name__ == '__main__':
    main()

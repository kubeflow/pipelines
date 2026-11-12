#!/usr/bin/env python3
"""Export KFP/MLMD relationship columns from a disposable release cluster."""

import argparse
import json
import subprocess
from pathlib import Path


QUERIES = {
    'runs': (
        "SELECT JSON_OBJECT('id', UUID, 'state', State, "
        "'namespace', Namespace, 'display_name', DisplayName, "
        "'experiment_id', ExperimentUUID, 'service_account', ServiceAccount, "
        "'pipeline_name', PipelineName, 'pipeline_id', PipelineId, "
        "'pipeline_version_id', PipelineVersionId, "
        "'pipeline_spec_manifest', PipelineSpecManifest, "
        "'runtime_parameters', RuntimeParameters, "
        "'pipeline_root', PipelineRoot) "
        'FROM mlpipeline.run_details ORDER BY UUID'
    ),
    'tasks': (
        "SELECT JSON_OBJECT('id', UUID, 'run_id', RunUUID, 'name', Name, "
        "'namespace', Namespace, 'state', State, "
        "'execution_id', MLMDExecutionID, 'parent_id', ParentTaskUUID, "
        "'pod_name', PodName, 'fingerprint', Fingerprint, "
        "'created_timestamp', CreatedTimestamp, "
        "'started_timestamp', StartedTimestamp, "
        "'finished_timestamp', FinishedTimestamp, "
        "'state_history', StateHistory) FROM mlpipeline.tasks "
        'ORDER BY RunUUID, CreatedTimestamp, UUID'
    ),
    'types': (
        "SELECT JSON_OBJECT('id', id, 'name', name, 'kind', type_kind) "
        'FROM metadb.Type ORDER BY id'
    ),
    'contexts': (
        "SELECT JSON_OBJECT('id', id, 'name', name, 'type_id', type_id) "
        'FROM metadb.Context ORDER BY id'
    ),
    'executions': (
        "SELECT JSON_OBJECT('id', id, 'name', name, 'type_id', type_id, "
        "'state', last_known_state, "
        "'created_ms', create_time_since_epoch, "
        "'updated_ms', last_update_time_since_epoch) "
        'FROM metadb.Execution ORDER BY id'
    ),
    'associations': (
        "SELECT JSON_OBJECT('context_id', context_id, "
        "'execution_id', execution_id) "
        'FROM metadb.Association ORDER BY context_id, execution_id'
    ),
    'execution_properties': (
        "SELECT JSON_OBJECT('execution_id', execution_id, 'name', name, "
        "'custom', is_custom_property, 'int_value', int_value, "
        "'double_value', double_value, 'string_value', string_value, "
        "'bool_value', bool_value) FROM metadb.ExecutionProperty "
        'ORDER BY execution_id, name, is_custom_property'
    ),
    'artifacts': (
        "SELECT JSON_OBJECT('id', id, 'type_id', type_id, 'name', name, "
        "'uri', uri, 'state', state, "
        "'created_ms', create_time_since_epoch, "
        "'updated_ms', last_update_time_since_epoch) "
        'FROM metadb.Artifact ORDER BY id'
    ),
    'artifact_properties': (
        "SELECT JSON_OBJECT('artifact_id', artifact_id, 'name', name, "
        "'custom', is_custom_property, 'int_value', int_value, "
        "'double_value', double_value, 'string_value', string_value, "
        "'bool_value', bool_value) FROM metadb.ArtifactProperty "
        'ORDER BY artifact_id, name, is_custom_property'
    ),
    'events': (
        "SELECT JSON_OBJECT('id', id, 'execution_id', execution_id, "
        "'artifact_id', artifact_id, 'type', type) "
        'FROM metadb.Event ORDER BY id'
    ),
    'event_paths': (
        "SELECT JSON_OBJECT('event_id', event_id, "
        "'is_index_step', is_index_step, 'step_index', step_index, "
        "'step_key', step_key) FROM metadb.EventPath "
        'ORDER BY event_id, is_index_step, step_index, step_key'
    ),
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--kubeconfig')
    parser.add_argument('--context')
    parser.add_argument('--container')
    parser.add_argument('--namespace', default='kubeflow')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if bool(args.container) == bool(args.kubeconfig and args.context):
        parser.error('provide either --container or --kubeconfig and --context')

    graph = {}
    for name, query in QUERIES.items():
        if args.container:
            command = ['podman', 'exec', args.container]
        else:
            command = [
                'kubectl', '--kubeconfig', args.kubeconfig,
                '--context', args.context, 'exec', '-n', args.namespace,
                'deployment/mysql', '--',
            ]
        command += [
            'sh', '-c',
            'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot --batch --raw --skip-column-names -e "$1"',
            'sh', query,
        ]
        result = subprocess.run(
            command, check=True, capture_output=True, text=True
        )
        graph[name] = [json.loads(line) for line in result.stdout.splitlines()]

    args.output.write_text(json.dumps(graph, indent=2, sort_keys=True) + '\n')


if __name__ == '__main__':
    main()

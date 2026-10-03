# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Bounded, restartable MLMD graph export and source-isolated import."""

from collections import deque
import copy
import json

from history import canonical
from history import digest
from history import HistoryError

KINDS = {
    'contexts': 'Context',
    'executions': 'Execution',
    'artifacts': 'Artifact'
}
MAX_NODES = 100000
EXECUTION_REFERENCES = {
    'parent_dag_id': 'int_value',
    'cached_execution_id': 'string_value'
}
PROVENANCE = 'kfp_history_'
TERMINAL = {'COMPLETE', 'FAILED', 'CACHED', 'CANCELED'}


class RPC:
    """Wire adapter uses the installed MLMD version's generated protobufs."""

    def __init__(self, config):
        from google.protobuf import json_format
        import grpc
        from ml_metadata.proto import metadata_store_service_pb2 as messages
        from ml_metadata.proto import \
            metadata_store_service_pb2_grpc as services
        self.grpc, self.json_format, self.messages = grpc, json_format, messages
        self.timeout = config.get('timeout_seconds', 60)
        target = config['target']
        options = [('grpc.max_receive_message_length', 256 * 1024 * 1024),
                   ('grpc.max_send_message_length', 256 * 1024 * 1024)]
        if config.get('insecure', False):
            channel = grpc.insecure_channel(target, options=options)
        else:

            def read(path):
                if not path:
                    return None
                with open(path, 'rb') as stream:
                    return stream.read()

            credentials = grpc.ssl_channel_credentials(
                read(config.get('ca_file')), read(config.get('key_file')),
                read(config.get('cert_file')))
            channel = grpc.secure_channel(target, credentials, options=options)
        self.stub = services.MetadataStoreServiceStub(channel)

    def call(self, method, **fields):
        request = getattr(self.messages, method + 'Request')()
        self.json_format.ParseDict(fields, request)
        try:
            response = getattr(self.stub, method)(request, timeout=self.timeout)
        except self.grpc.RpcError as error:
            if error.code(
            ) == self.grpc.StatusCode.NOT_FOUND and method.startswith('Get'):
                return {}
            raise
        return self.json_format.MessageToDict(
            response, preserving_proto_field_name=True)


def node_digest(node):
    value = copy.deepcopy(node)
    # Shared pipeline contexts may be touched by unrelated later runs.
    for key in ('create_time_since_epoch', 'last_update_time_since_epoch'):
        value.pop(key, None)
    return digest(value)


def unique(rows):
    return [json.loads(key) for key in sorted({canonical(row) for row in rows})]


def require_acyclic(edges, label):
    children, incoming = {}, {}
    for parent, child in set(edges):
        children.setdefault(parent, []).append(child)
        incoming.setdefault(parent, 0)
        incoming[child] = incoming.get(child, 0) + 1
    ready = deque(node for node, count in incoming.items() if count == 0)
    visited = 0
    while ready:
        node = ready.popleft()
        visited += 1
        for child in children.get(node, []):
            incoming[child] -= 1
            if incoming[child] == 0:
                ready.append(child)
    if visited != len(incoming):
        raise HistoryError(label + ' contains a cycle')


def validate_graph(graph):
    if not isinstance(graph, dict):
        raise HistoryError('Missing MLMD graph')
    allowed = set(KINDS) | {kind[:-1] + '_types' for kind in KINDS}
    allowed |= {
        'events', 'associations', 'attributions', 'parents', 'run_context_id'
    }
    if set(graph) - allowed:
        raise HistoryError('Unknown MLMD graph collection')
    ids = {}
    for collection in KINDS:
        rows = graph.get(collection, [])
        keys = [str(row.get('id', '')) for row in rows]
        if any(not key.isdigit() or int(key) <= 0
               for key in keys) or len(keys) != len(set(keys)):
            raise HistoryError('Invalid or duplicate MLMD ' + collection +
                               ' IDs')
        ids[collection] = set(keys)
        types = graph.get(collection[:-1] + '_types', [])
        type_ids = {str(row['id']) for row in types}
        if len(type_ids) != len(types):
            raise HistoryError('Duplicate MLMD type ID')
        for row in rows:
            if str(row.get('type_id')) not in type_ids:
                raise HistoryError('MLMD node references a missing type')
            if any(
                    key.startswith(PROVENANCE)
                    for key in row.get('custom_properties', {})):
                raise HistoryError(
                    'MLMD graph already contains imported history')
    if sum(len(value) for value in ids.values()) > MAX_NODES:
        raise HistoryError('MLMD graph exceeds the 100000-node transfer limit')

    def check(collection, value):
        if str(value) not in ids[collection]:
            raise HistoryError('MLMD graph has a dangling ' + collection +
                               ' reference')

    if graph.get('run_context_id'):
        check('contexts', graph['run_context_id'])
    for execution in graph.get('executions', []):
        if execution.get('last_known_state') not in TERMINAL:
            raise HistoryError(
                'MLMD graph includes an unfinished execution; retry export after completion'
            )
        for key, field in EXECUTION_REFERENCES.items():
            value = execution.get('custom_properties', {}).get(key,
                                                               {}).get(field)
            if value and str(value) != '0':
                check('executions', value)
    for event in graph.get('events', []):
        check('executions', event.get('execution_id'))
        check('artifacts', event.get('artifact_id'))
    for row in graph.get('associations', []):
        check('contexts', row.get('context_id'))
        check('executions', row.get('execution_id'))
    for row in graph.get('attributions', []):
        check('contexts', row.get('context_id'))
        check('artifacts', row.get('artifact_id'))
    for row in graph.get('parents', []):
        check('contexts', row.get('parent_id'))
        check('contexts', row.get('child_id'))
    require_acyclic([(str(row['parent_id']), str(row['child_id']))
                     for row in graph.get('parents', [])], 'MLMD context graph')
    execution_edges = []
    for node in graph.get('executions', []):
        for key, field in EXECUTION_REFERENCES.items():
            value = node.get('custom_properties', {}).get(key, {}).get(field)
            if value and str(value) != '0':
                execution_edges.append((str(value), str(node['id'])))
    require_acyclic(execution_edges, 'MLMD execution graph')


class Metadata:

    def __init__(self, rpc):
        self.rpc = rpc

    def call(self, method, **fields):
        return self.rpc.call(method, **fields)

    def fetch(self, collection, ids):
        requested = sorted(
            {str(value) for value in ids if value and str(value) != '0'})
        if not requested:
            return []
        plural = KINDS[collection] + 's'
        result = []
        for start in range(0, len(requested), 100):
            part = requested[start:start + 100]
            rows = self.call('Get' + plural + 'ByID', **{
                collection[:-1] + '_ids': part
            }).get(collection, [])
            if {str(row['id']) for row in rows} != set(part):
                raise HistoryError('Source MLMD is missing referenced ' +
                                   collection)
            result.extend(rows)
        return result

    def context_nodes(self, collection, context_id):
        rows, token = [], ''
        while True:
            options = {'max_result_size': 100}
            if token:
                options['next_page_token'] = token
            response = self.call(
                'Get' + KINDS[collection] + 'sByContext',
                context_id=str(context_id),
                options=options)
            rows.extend(response.get(collection, []))
            if len(rows) > MAX_NODES:
                raise HistoryError('MLMD context exceeds transfer node limit')
            token = response.get('next_page_token', '')
            if not token:
                return rows

    def export_graph(self, run, tasks):
        graph = {
            key: [] for key in list(KINDS) + [
                'context_types', 'execution_types', 'artifact_types', 'events',
                'associations', 'attributions', 'parents'
            ]
        }
        contexts, executions, artifacts = {}, {}, {}
        run_context = self.call(
            'GetContextByTypeAndName',
            type_name='system.PipelineRun',
            context_name=run['UUID']).get('context')
        if run_context:
            contexts[str(run_context['id'])] = run_context
            graph['run_context_id'] = str(run_context['id'])
            for execution in self.context_nodes('executions',
                                                run_context['id']):
                executions[str(execution['id'])] = execution
            for artifact in self.context_nodes('artifacts', run_context['id']):
                artifacts[str(artifact['id'])] = artifact
        elif run.get('PipelineRuntimeManifest'):
            raise HistoryError(
                'Completed v2 run has no MLMD run context; cannot export complete history'
            )
        context_ids = [
            run.get('PipelineContextId'),
            run.get('PipelineRunContextId')
        ]
        for row in self.fetch('contexts', context_ids):
            contexts[str(row['id'])] = row
        for row in self.fetch('executions',
                              [task.get('MLMDExecutionID') for task in tasks]):
            executions[str(row['id'])] = row
        processed_exec, processed_artifact = set(), set()
        while set(executions) - processed_exec or set(
                artifacts) - processed_artifact:
            if len(contexts) + len(executions) + len(artifacts) > MAX_NODES:
                raise HistoryError(
                    'MLMD ancestor graph exceeds transfer node limit')
            for execution_id in sorted(set(executions) - processed_exec):
                execution = executions[execution_id]
                processed_exec.add(execution_id)
                if execution.get('last_known_state') not in TERMINAL:
                    raise HistoryError(
                        'Source MLMD execution is unfinished; retry export after completion'
                    )
                references = [
                    execution.get('custom_properties', {}).get(key,
                                                               {}).get(field)
                    for key, field in EXECUTION_REFERENCES.items()
                ]
                for row in self.fetch(
                        'executions',
                        set(str(value) for value in references if value) -
                        set(executions)):
                    executions[str(row['id'])] = row
                events = self.call(
                    'GetEventsByExecutionIDs',
                    execution_ids=[execution_id]).get('events', [])
                graph['events'].extend(events)
                for row in self.fetch(
                        'artifacts',
                    {str(event['artifact_id']) for event in events} -
                        set(artifacts)):
                    artifacts[str(row['id'])] = row
                for context in self.call(
                        'GetContextsByExecution',
                        execution_id=execution_id).get('contexts', []):
                    contexts[str(context['id'])] = context
                    graph['associations'].append({
                        'execution_id': execution_id,
                        'context_id': str(context['id'])
                    })
            for artifact_id in sorted(set(artifacts) - processed_artifact):
                processed_artifact.add(artifact_id)
                # Follow producer ancestry only, never unrelated consumers.
                events = self.call(
                    'GetEventsByArtifactIDs',
                    artifact_ids=[artifact_id]).get('events', [])
                producers = {
                    str(event['execution_id'])
                    for event in events
                    if event.get('type') in ('OUTPUT', 'DECLARED_OUTPUT',
                                             'INTERNAL_OUTPUT')
                }
                for row in self.fetch('executions',
                                      producers - set(executions)):
                    executions[str(row['id'])] = row
                for context in self.call(
                        'GetContextsByArtifact',
                        artifact_id=artifact_id).get('contexts', []):
                    contexts[str(context['id'])] = context
                    graph['attributions'].append({
                        'artifact_id': artifact_id,
                        'context_id': str(context['id'])
                    })
        processed_contexts = set()
        while set(contexts) - processed_contexts:
            if len(contexts) > MAX_NODES:
                raise HistoryError(
                    'MLMD parent context graph exceeds transfer node limit')
            for context_id in sorted(set(contexts) - processed_contexts):
                processed_contexts.add(context_id)
                for parent in self.call(
                        'GetParentContextsByContext',
                        context_id=context_id).get('contexts', []):
                    contexts[str(parent['id'])] = parent
                    graph['parents'].append({
                        'child_id': context_id,
                        'parent_id': str(parent['id'])
                    })
        for collection, nodes in [('contexts', contexts),
                                  ('executions', executions),
                                  ('artifacts', artifacts)]:
            graph[collection] = [nodes[key] for key in sorted(nodes)]
            type_ids = sorted({str(row['type_id']) for row in nodes.values()})
            if type_ids:
                response = self.call(
                    'Get' + KINDS[collection] + 'TypesByID', type_ids=type_ids)
                graph[collection[:-1] + '_types'] = response.get(
                    collection[:-1] + '_types', [])
        for key in graph:
            if isinstance(graph[key], list):
                graph[key] = unique(graph[key])
        validate_graph(graph)
        return graph

    def type_by_name(self, kind, definition):
        fields = {'type_name': definition['name']}
        if definition.get('version'):
            fields['type_version'] = definition['version']
        return self.call('Get' + kind + 'Type',
                         **fields).get(kind.lower() + '_type')

    @staticmethod
    def type_compatible(source, target):
        # Servers differ in whether optional default values appear on reads.
        def schema(value):
            return {
                key: value.get(key, default)
                for key, default in (('name', ''), ('version', ''),
                                     ('properties', {}), ('base_type', 'UNSET'),
                                     ('input_type', {}), ('output_type', {}))
            }

        return schema(source) == schema(target)

    @staticmethod
    def imported_name(collection, node, definition, source):
        # The 2.18 UI locates system.PipelineRun by its unchanged run UUID.
        if collection == 'contexts' and definition[
                'name'] == 'system.PipelineRun':
            return node['name']
        return 'kfp-history-' + digest(
            source)[:24] + '-' + collection + '-' + str(node['id'])

    def find_node(self, collection, name, definition):
        kind = KINDS[collection]
        fields = {'type_name': definition['name'], kind.lower() + '_name': name}
        if definition.get('version'):
            fields['type_version'] = definition['version']
        return self.call('Get' + kind + 'ByTypeAndName',
                         **fields).get(kind.lower())

    def preflight(self, graph, source):
        validate_graph(graph)
        for collection, kind in KINDS.items():
            definitions = {
                str(row['id']): row
                for row in graph.get(collection[:-1] + '_types', [])
            }
            for definition in definitions.values():
                current = self.type_by_name(kind, definition)
                if current and not self.type_compatible(definition, current):
                    raise HistoryError('Destination MLMD type differs: ' +
                                       definition['name'])
            for node in graph.get(collection, []):
                definition = definitions[str(node['type_id'])]
                name = self.imported_name(collection, node, definition, source)
                current = self.find_node(collection, name, definition)
                if current:
                    props = current.get('custom_properties', {})
                    if props.get(PROVENANCE + 'source', {}).get('string_value') != source or \
                       props.get(PROVENANCE + 'digest', {}).get('string_value') != node_digest(node):
                        raise HistoryError(
                            'Destination MLMD node has conflicting history provenance'
                        )

    def import_graph(self, graph, source):
        self.preflight(graph, source)
        mapping = {collection: {} for collection in KINDS}
        definitions_by_kind, imported_nodes = {}, {}
        for collection, kind in KINDS.items():
            definitions = {
                str(row['id']): row
                for row in graph.get(collection[:-1] + '_types', [])
            }
            definitions_by_kind[collection] = definitions
            type_mapping = {}
            for old_id, definition in definitions.items():
                found = self.type_by_name(kind, definition)
                if found:
                    type_mapping[old_id] = str(found['id'])
                else:
                    payload = {
                        key: value
                        for key, value in definition.items()
                        if key != 'id'
                    }
                    result = self.call('Put' + kind + 'Type',
                                       **{kind.lower() + '_type': payload})
                    type_mapping[old_id] = str(result['type_id'])
            for node in graph.get(collection, []):
                definition = definitions[str(node['type_id'])]
                name = self.imported_name(collection, node, definition, source)
                current = self.find_node(collection, name, definition)
                payload = copy.deepcopy(node)
                for field in ('id', 'type', 'external_id', 'system_metadata',
                              'create_time_since_epoch',
                              'last_update_time_since_epoch'):
                    payload.pop(field, None)
                payload['type_id'], payload['name'] = type_mapping[str(
                    node['type_id'])], name
                props = payload.setdefault('custom_properties', {})
                props[PROVENANCE + 'source'] = {'string_value': source}
                props[PROVENANCE + 'digest'] = {
                    'string_value': node_digest(node)
                }
                props[PROVENANCE + 'original'] = {
                    'string_value': canonical(node)
                }
                # Do not expose a stale source integer as a destination relationship.
                for field in EXECUTION_REFERENCES:
                    props.pop(field, None)
                if 'cache_fingerprint' in props:
                    props['cache_fingerprint'] = {
                        'string_value':
                            'history:' + source + ':' + str(node['id'])
                    }
                if current:
                    new_id = str(current['id'])
                else:
                    result = self.call('Put' + kind + 's',
                                       **{collection: [payload]})
                    new_id = str(result[collection[:-1] + '_ids'][0])
                payload['id'] = new_id
                mapping[collection][str(node['id'])] = new_id
                imported_nodes[(collection, str(node['id']))] = payload
        # Nodes are all reserved before rewriting parent/cache execution pointers.
        for node in graph.get('executions', []):
            payload = imported_nodes[('executions', str(node['id']))]
            props = payload['custom_properties']
            for key, field in EXECUTION_REFERENCES.items():
                value = node.get('custom_properties', {}).get(key,
                                                              {}).get(field)
                if value and str(value) != '0':
                    props[key] = {field: mapping['executions'][str(value)]}
            self.call('PutExecutions', executions=[payload])
        # Events are insert-only; compare existing edges to make retries idempotent.
        for execution in graph.get('executions', []):
            new_execution = mapping['executions'][str(execution['id'])]
            existing = self.call(
                'GetEventsByExecutionIDs',
                execution_ids=[new_execution]).get('events', [])

            def event_key(event):
                return canonical({
                    key: value
                    for key, value in event.items()
                    if key != 'milliseconds_since_epoch'
                })

            existing_keys = {event_key(row) for row in existing}
            for event in graph.get('events', []):
                if str(event['execution_id']) != str(execution['id']):
                    continue
                row = copy.deepcopy(event)
                row['execution_id'] = new_execution
                row['artifact_id'] = mapping['artifacts'][str(
                    row['artifact_id'])]
                if event_key(row) not in existing_keys:
                    self.call('PutEvents', events=[row])
                    existing_keys.add(event_key(row))
        attributions = [{
            'artifact_id': mapping['artifacts'][str(row['artifact_id'])],
            'context_id': mapping['contexts'][str(row['context_id'])]
        } for row in graph.get('attributions', [])]
        associations = [{
            'execution_id': mapping['executions'][str(row['execution_id'])],
            'context_id': mapping['contexts'][str(row['context_id'])]
        } for row in graph.get('associations', [])]
        if attributions or associations:
            self.call(
                'PutAttributionsAndAssociations',
                attributions=attributions,
                associations=associations)
        parents = [{
            'child_id': mapping['contexts'][str(row['child_id'])],
            'parent_id': mapping['contexts'][str(row['parent_id'])]
        } for row in graph.get('parents', [])]
        if parents:
            # Some MLMD versions reject a duplicate parent edge. Avoid relying
            # on PutParentContexts having the association RPC's no-op semantics.
            for row in parents:
                current = self.call(
                    'GetParentContextsByContext',
                    context_id=row['child_id']).get('contexts', [])
                if row['parent_id'] not in {
                        str(node['id']) for node in current
                }:
                    self.call('PutParentContexts', parent_contexts=[row])
        return mapping

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
"""Bounded, explicitly scoped KFP workload evidence held only in memory.

Returned records include potentially sensitive API fields. Callers must
report only reviewed summaries; this module deliberately has no
serialization or CLI.
"""

from datetime import datetime

from kfp_http import CollectionError
from kfp_inventory import field
from kfp_inventory import identifier

MAX_RECORDS = 10000
MAX_PAGES = 100
API = '/apis/v2beta1/'
RESOURCES = {
    'experiments': ('experiment_id', 'experimentId', 'experiments'),
    'pipelines': ('pipeline_id', 'pipelineId', 'pipelines'),
    'pipeline_versions':
        ('pipeline_version_id', 'pipelineVersionId', 'pipelineVersions'),
    'runs': ('run_id', 'runId', 'runs'),
    'recurring_runs': ('recurring_run_id', 'recurringRunId', 'recurringRuns'),
}


def _record(raw):
    if not isinstance(raw, dict):
        raise CollectionError('invalid_record')
    if raw.get('error'):
        raise CollectionError('record_error')
    # Collection annotations cannot come from the remote endpoint.
    return {
        key: value
        for key, value in raw.items()
        if not key.startswith('_readiness_')
    }


def _id(raw, resource):
    value = field(raw, *RESOURCES[resource][:2])
    identifier(value)
    return value


def _spec(raw):
    return field(raw, 'pipeline_spec', 'pipelineSpec')


def _created_at(raw):
    value = field(raw, 'created_at', 'createdAt')
    try:
        if not isinstance(value, str):
            raise ValueError()
        parsed = datetime.fromisoformat(value.replace('Z', '+00:00'))
        if parsed.utcoffset() is None:
            raise ValueError()
        return parsed
    except ValueError:
        raise CollectionError('version_creation_time_unresolved') from None


class _Collector:

    def __init__(self, client, namespaces, include_shared,
                 source_single_user_namespace):
        self.client = client
        self.namespaces = list(dict.fromkeys(namespaces))
        for namespace in self.namespaces:
            identifier(namespace)
            if namespace == '-':
                raise CollectionError('invalid_namespace')
        if (source_single_user_namespace is not None and
                self.namespaces != [source_single_user_namespace]):
            raise CollectionError('invalid_single_user_namespace')
        self.single_user_namespace = source_single_user_namespace
        self.include_shared = include_shared
        self.inventory = {resource: {} for resource in RESOURCES}
        self.failures = []
        self.resources = {}
        self.version_lists = {}
        self.hydrated = set()

    def annotate_failure(self, raw, reason):
        errors = raw.setdefault('_readiness_collection_errors', [])
        if reason not in errors:
            errors.append(reason)

    def fail(self, namespace, resource, reason, traversal=None):
        self.failures.append(
            dict(namespace=namespace, resource=resource, reason=reason))
        if traversal is not None:
            traversal['failed_checks'] += 1

    def scope(self, raw, namespace, inherited=False):
        actual = raw.get('namespace')
        if inherited and actual in (None, ''):
            evidence = 'parent_resource'
        elif self.single_user_namespace == namespace and actual in (None, '',
                                                                    '-'):
            evidence = 'operator_asserted_single_user'
        elif namespace in ('', '-') and actual in (None, '', '-'):
            evidence = 'shared_pipeline'
        elif actual == namespace:
            evidence = 'api_namespace'
        else:
            raise CollectionError('namespace_mismatch')
        raw['namespace'] = namespace
        raw['_readiness_namespace_evidence'] = evidence
        return raw

    def store(self, resource, raw):
        key = _id(raw, resource)
        previous = self.inventory[resource].get(key)
        if previous is not None:
            if previous['namespace'] != raw['namespace']:
                raise CollectionError('record_scope_changed')
            if (resource == 'pipeline_versions' and
                    field(previous, 'pipeline_id', 'pipelineId') != field(
                        raw, 'pipeline_id', 'pipelineId')):
                raise CollectionError('pipeline_version_mismatch')
            return previous
        if sum(map(len, self.inventory.values())) >= MAX_RECORDS:
            raise CollectionError('record_limit')
        self.inventory[resource][key] = raw
        return raw

    def pages(self, resource, namespace, path, params, accept, parent_id=None):
        key = (resource, namespace, parent_id)
        if key in self.resources:
            return self.resources[key]
        traversal = dict(
            namespace=namespace,
            resource=resource,
            list_complete=False,
            pages=0,
            listed_records=0,
            unique_listed_records=0,
            records=0,
            failed_checks=0)
        if parent_id is not None:
            traversal['parent_id'] = parent_id
        self.resources[key] = traversal
        token, tokens, ids = '', set(), set()
        total_size = None
        try:
            for _ in range(MAX_PAGES):
                response = self.client.get(
                    path, dict(params, page_size=100, page_token=token))
                response = _record(response)
                traversal['pages'] += 1
                records = field(response, resource, RESOURCES[resource][2])
                if records is None:
                    records = []
                if not isinstance(records, list):
                    raise CollectionError('invalid_resource_list')
                size = field(response, 'total_size', 'totalSize')
                if size is not None:
                    if isinstance(
                            size,
                            bool) or not isinstance(size, int) or size < 0:
                        raise CollectionError('invalid_total_size')
                    if total_size is not None and size != total_size:
                        self.fail(namespace, resource, 'list_size_changed',
                                  traversal)
                    total_size = size
                page_ids = set()
                for raw in records:
                    traversal['listed_records'] += 1
                    try:
                        raw = _record(raw)
                        record_id = _id(raw, resource)
                        already_seen = record_id in ids or record_id in page_ids
                        page_ids.add(record_id)
                        if not already_seen:
                            accept(raw)
                        else:
                            # Validate overlapping pages too: a changed scope or
                            # parent must not be hidden by ID deduplication.
                            accepted = self.inventory[resource].get(record_id)
                            if accepted is not None:
                                self.scope(
                                    raw,
                                    namespace,
                                    inherited=resource
                                    not in ('experiments', 'pipelines'))
                                if (resource == 'pipeline_versions' and
                                        field(raw, 'pipeline_id', 'pipelineId')
                                        != field(accepted, 'pipeline_id',
                                                 'pipelineId')):
                                    raise CollectionError(
                                        'pipeline_version_mismatch')
                                if (resource in ('runs', 'recurring_runs') and
                                        field(raw, 'experiment_id',
                                              'experimentId') != field(
                                                  accepted, 'experiment_id',
                                                  'experimentId')):
                                    raise CollectionError('experiment_mismatch')
                    except CollectionError as error:
                        if error.reason == 'record_limit':
                            raise
                        self.fail(namespace, resource, error.reason, traversal)
                new_ids = page_ids - ids
                ids.update(page_ids)
                traversal['unique_listed_records'] = len(ids)
                next_token = field(response, 'next_page_token', 'nextPageToken')
                if next_token is None:
                    next_token = ''
                if not isinstance(next_token, str) or len(next_token) > 16384:
                    raise CollectionError('invalid_page_token')
                if not next_token:
                    if total_size is not None and len(ids) != total_size:
                        raise CollectionError('list_count_mismatch')
                    traversal['list_complete'] = True
                    break
                if next_token in tokens:
                    raise CollectionError('repeated_page_token')
                if records and not new_ids:
                    raise CollectionError('repeated_page_records')
                tokens.add(next_token)
                token = next_token
            else:
                raise CollectionError('page_limit')
        except CollectionError as error:
            self.fail(namespace, resource, error.reason, traversal)
        return traversal

    def parent(self, resource, record_id, namespace):
        identifier(record_id)
        previous = self.inventory[resource].get(record_id)
        if previous is not None:
            if previous['namespace'] != namespace:
                raise CollectionError('namespace_mismatch')
            return previous
        raw = _record(
            self.client.get(API + resource + '/' + identifier(record_id)))
        if _id(raw, resource) != record_id:
            raise CollectionError('referenced_id_mismatch')
        return self.store(resource, self.scope(raw, namespace))

    def pipeline(self, pipeline_id):
        identifier(pipeline_id)
        previous = self.inventory['pipelines'].get(pipeline_id)
        if previous is not None:
            return previous
        raw = _record(
            self.client.get(API + 'pipelines/' + identifier(pipeline_id)))
        if _id(raw, 'pipelines') != pipeline_id:
            raise CollectionError('referenced_id_mismatch')
        namespace = raw.get('namespace')
        if namespace in (None, '', '-') and self.single_user_namespace:
            namespace = self.single_user_namespace
        elif namespace in (None, '', '-') and self.include_shared:
            namespace = namespace or ''
        elif namespace not in self.namespaces:
            raise CollectionError('referenced_pipeline_outside_scope')
        return self.store('pipelines', self.scope(raw, namespace))

    def hydrate(self, resource, raw, path):
        record_id = _id(raw, resource)
        key = (resource, record_id)
        if isinstance(_spec(raw), dict) and _spec(raw):
            return raw
        if key in self.hydrated:
            return raw
        self.hydrated.add(key)
        full = _record(self.client.get(path))
        if _id(full, resource) != record_id:
            raise CollectionError('referenced_id_mismatch')
        if resource == 'pipeline_versions':
            if field(full, 'pipeline_id',
                     'pipelineId') != field(raw, 'pipeline_id', 'pipelineId'):
                raise CollectionError('pipeline_version_mismatch')
        else:
            if field(full, 'experiment_id',
                     'experimentId') != field(raw, 'experiment_id',
                                              'experimentId'):
                raise CollectionError('experiment_mismatch')
        self.scope(full, raw['namespace'], inherited=True)
        # Keep one coherent response: field aliases from two responses may
        # otherwise create conflicts, and old oneof members must not survive.
        raw.clear()
        raw.update(full)
        return raw

    def versions(self, pipeline):
        pipeline_id = _id(pipeline, 'pipelines')
        if pipeline_id in self.version_lists:
            return self.version_lists[pipeline_id]
        namespace = pipeline['namespace']
        path = API + 'pipelines/' + identifier(pipeline_id) + '/versions'

        def accept(raw):
            if field(raw, 'pipeline_id', 'pipelineId') != pipeline_id:
                raise CollectionError('pipeline_version_mismatch')
            raw = self.store('pipeline_versions',
                             self.scope(raw, namespace, inherited=True))
            self.hydrate('pipeline_versions', raw,
                         path + '/' + identifier(_id(raw, 'pipeline_versions')))
            if not isinstance(_spec(raw), dict) or not _spec(raw):
                raise CollectionError('pipeline_spec_unavailable')

        traversal = self.pages(
            'pipeline_versions',
            namespace,
            path,
            dict(sort_by='created_at desc'),
            accept,
            parent_id=pipeline_id)
        self.version_lists[pipeline_id] = traversal
        return traversal

    def resolve(self, raw):
        reference = field(raw, 'pipeline_version_reference',
                          'pipelineVersionReference')
        legacy_version = field(raw, 'pipeline_version_id', 'pipelineVersionId')
        if legacy_version not in (None, ''):
            identifier(legacy_version)
        if sum((reference is not None, legacy_version
                not in (None, ''), _spec(raw) is not None)) > 1:
            raise CollectionError('conflicting_pipeline_source')
        if reference is None and not legacy_version:
            if not isinstance(_spec(raw), dict) or not _spec(raw):
                raise CollectionError('pipeline_spec_unavailable')
            return
        annotation = dict(kind='pinned', resolution='unresolved')
        raw['_readiness_version_reference'] = annotation
        if reference is None:
            identifier(legacy_version)
            annotation['kind'] = 'legacy_version_id'
            annotation['pipeline_version_id'] = legacy_version
            version = self.inventory['pipeline_versions'].get(legacy_version)
            if version is None:
                raise CollectionError('legacy_version_parent_unresolved')
            pipeline_id = field(version, 'pipeline_id', 'pipelineId')
            version_id = legacy_version
        else:
            if not isinstance(reference, dict):
                raise CollectionError('invalid_pipeline_reference')
            pipeline_id = field(reference, 'pipeline_id', 'pipelineId')
            identifier(pipeline_id)
            annotation['pipeline_id'] = pipeline_id
            version_id = field(reference, 'pipeline_version_id',
                               'pipelineVersionId')
            if version_id not in (None, ''):
                identifier(version_id)
            pipeline = self.pipeline(pipeline_id)
            traversal = self.versions(pipeline)
            if not version_id:
                annotation['kind'] = 'moving_latest'
                candidates = [
                    v for v in self.inventory['pipeline_versions'].values()
                    if field(v, 'pipeline_id', 'pipelineId') == pipeline_id
                ]
                if not traversal['list_complete'] or traversal[
                        'failed_checks'] or not candidates:
                    raise CollectionError('moving_latest_version_unresolved')
                # The source's latest query sorts by creation time only. A tie
                # has no stable winner, even if a list response has an order.
                timestamps = [_created_at(version) for version in candidates]
                newest = max(timestamps)
                if timestamps.count(newest) != 1:
                    raise CollectionError('moving_latest_version_ambiguous')
                version_id = _id(candidates[timestamps.index(newest)],
                                 'pipeline_versions')
            identifier(version_id)
            version = self.inventory['pipeline_versions'].get(version_id)
            if version is None:
                path = API + 'pipelines/' + identifier(
                    pipeline_id) + '/versions/' + identifier(version_id)
                version = _record(self.client.get(path))
                if _id(version, 'pipeline_versions') != version_id:
                    raise CollectionError('referenced_id_mismatch')
                if field(version, 'pipeline_id', 'pipelineId') != pipeline_id:
                    raise CollectionError('pipeline_version_mismatch')
                version = self.store(
                    'pipeline_versions',
                    self.scope(version, pipeline['namespace'], inherited=True))
        if field(version, 'pipeline_id', 'pipelineId') != pipeline_id:
            raise CollectionError('pipeline_version_mismatch')
        if not isinstance(_spec(version), dict) or not _spec(version):
            raise CollectionError('pipeline_spec_unavailable')
        annotation.update(
            pipeline_id=pipeline_id,
            pipeline_version_id=version_id,
            resolution='observed')
        raw['_readiness_pipeline_spec'] = _spec(version)

    def recurring_parent(self, raw):
        recurring_id = field(raw, 'recurring_run_id', 'recurringRunId')
        if not recurring_id:
            return
        identifier(recurring_id)
        parent = self.inventory['recurring_runs'].get(recurring_id)
        if parent is None:
            parent = _record(
                self.client.get(API + 'recurringruns/' +
                                identifier(recurring_id)))
            if _id(parent, 'recurring_runs') != recurring_id:
                raise CollectionError('referenced_id_mismatch')
            experiment_id = field(parent, 'experiment_id', 'experimentId')
            self.parent('experiments', experiment_id, raw['namespace'])
            parent = self.store(
                'recurring_runs',
                self.scope(parent, raw['namespace'], inherited=True))
        if (parent['namespace'] != raw['namespace'] or
                field(parent, 'experiment_id', 'experimentId') != field(
                    raw, 'experiment_id', 'experimentId')):
            raise CollectionError('recurring_run_scope_mismatch')

    def workloads(self, resource, namespace):
        endpoint = 'recurringruns' if resource == 'recurring_runs' else resource

        def accept(raw):
            experiment_id = field(raw, 'experiment_id', 'experimentId')
            self.parent('experiments', experiment_id, namespace)
            raw = self.store(resource,
                             self.scope(raw, namespace, inherited=True))
            try:
                self.hydrate(
                    resource, raw,
                    API + endpoint + '/' + identifier(_id(raw, resource)))
            except CollectionError as error:
                self.annotate_failure(raw, error.reason)
                raise

        self.pages(resource, namespace, API + endpoint,
                   dict(namespace=namespace), accept)

    def collect(self):
        for namespace in self.namespaces:
            for resource in ('experiments', 'pipelines'):
                self.pages(
                    resource,
                    namespace,
                    API + resource,
                    dict(namespace=namespace),
                    lambda raw, resource=resource, namespace=namespace: self.
                    store(resource, self.scope(raw, namespace)))
        if self.include_shared and self.single_user_namespace is None:
            for namespace in ('', '-'):
                self.pages(
                    'pipelines',
                    namespace,
                    API + 'pipelines',
                    dict(namespace=namespace),
                    lambda raw, namespace=namespace: self.store(
                        'pipelines', self.scope(raw, namespace)))
        for pipeline in list(self.inventory['pipelines'].values()):
            self.versions(pipeline)
        for namespace in self.namespaces:
            for resource in ('runs', 'recurring_runs'):
                self.workloads(resource, namespace)
        for resource in ('runs', 'recurring_runs'):
            for record_id, raw in self.inventory[resource].items():
                try:
                    if resource == 'runs':
                        self.recurring_parent(raw)
                    self.resolve(raw)
                except CollectionError as error:
                    self.annotate_failure(raw, error.reason)
                    self.fail(
                        raw['namespace'], resource + '/' + record_id,
                        error.reason,
                        self.resources[(resource, raw['namespace'], None)])
        for traversal in self.resources.values():
            records = self.inventory[traversal['resource']].values()
            traversal['records'] = sum(
                record['namespace'] == traversal['namespace'] and
                ('parent_id' not in traversal or
                 field(record, 'pipeline_id',
                       'pipelineId') == traversal['parent_id'])
                for record in records)
        inventory = {
            key: list(records.values())
            for key, records in self.inventory.items()
        }
        coverage = dict(
            requested_namespaces=self.namespaces,
            include_shared=self.include_shared,
            source_single_user_namespace=self.single_user_namespace,
            resources=list(self.resources.values()),
            record_counts={
                key: len(records) for key, records in inventory.items()
            },
            failed_checks=len(self.failures),
            snapshot_consistency='not_atomic',
            complete=not self.failures and
            all(item['list_complete'] for item in self.resources.values()))
        return inventory, self.failures, coverage


def collect(client,
            namespaces,
            include_shared=False,
            source_single_user_namespace=None):
    """Collect scoped records without following package, artifact, or log URLs.

    Shared collection is limited to pipelines and their versions. Empty
    source namespaces can be bound to one chosen namespace only with the
    explicit single-user assertion; that assertion is retained in
    coverage and records.
    """
    return _Collector(client, namespaces, include_shared,
                      source_single_user_namespace).collect()


class _OfflineClient:

    def get(self, path, params=None):
        raise CollectionError('offline_reference_missing')


def validate(inventory,
             namespaces,
             include_shared=False,
             source_single_user_namespace=None):
    """Validate offline workload evidence and recompute collection annotations.

    This validates the supplied snapshot, not its completeness or
    authenticity. Missing references remain unknown. Moving-latest
    cannot be resolved from an offline list whose sorting and traversal
    were not observed by the collector.
    """
    collector = _Collector(_OfflineClient(), namespaces, include_shared,
                           source_single_user_namespace)
    if not isinstance(inventory, dict):
        raise CollectionError('invalid_workload_inventory')
    for resource in RESOURCES:
        records = inventory.get(resource, [])
        if not isinstance(records, list):
            raise CollectionError('invalid_resource_list')
    if sum(len(inventory.get(resource, []))
           for resource in RESOURCES) > MAX_RECORDS:
        raise CollectionError('record_limit')

    def namespace(raw, shared=False):
        value = raw.get('namespace')
        if value in (None, '', '-') and source_single_user_namespace:
            return source_single_user_namespace
        if shared and include_shared and value in (None, '', '-'):
            return value or ''
        if value not in collector.namespaces:
            raise CollectionError('namespace_mismatch')
        return value

    for resource in RESOURCES:
        for original in inventory.get(resource, []):
            raw = _record(original)
            record_id = _id(raw, resource)
            if record_id in collector.inventory[resource]:
                raise CollectionError('duplicate_record')
            parent = None
            if resource == 'pipeline_versions':
                parent_id = field(raw, 'pipeline_id', 'pipelineId')
                identifier(parent_id)
                parent = collector.inventory['pipelines'].get(parent_id)
            elif resource in ('runs', 'recurring_runs'):
                parent_id = field(raw, 'experiment_id', 'experimentId')
                identifier(parent_id)
                parent = collector.inventory['experiments'].get(parent_id)
            if resource in ('experiments', 'pipelines'):
                collector.scope(raw,
                                namespace(raw, shared=resource == 'pipelines'))
            elif parent is not None:
                collector.scope(raw, parent['namespace'], inherited=True)
            else:
                collector.scope(raw, namespace(raw))
                raw['_readiness_namespace_evidence'] = 'unresolved_parent'
                collector.annotate_failure(raw, 'offline_reference_missing')
            collector.store(resource, raw)
        if resource == 'pipelines':
            for pipeline_id in collector.inventory['pipelines']:
                collector.version_lists[pipeline_id] = dict(
                    list_complete=False, failed_checks=0)

    for resource in ('runs', 'recurring_runs'):
        for raw in collector.inventory[resource].values():
            try:
                if resource == 'runs':
                    collector.recurring_parent(raw)
                collector.resolve(raw)
            except CollectionError as error:
                collector.annotate_failure(raw, error.reason)
    for raw in collector.inventory['pipeline_versions'].values():
        if not isinstance(_spec(raw), dict) or not _spec(raw):
            collector.annotate_failure(raw, 'pipeline_spec_unavailable')
    return {
        resource: list(records.values())
        for resource, records in collector.inventory.items()
    }

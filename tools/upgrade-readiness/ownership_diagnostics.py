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
"""Bounded API evidence for ownership recovery; never an authorization
verdict."""

import json

from kfp_http import CollectionError
from kfp_inventory import field
from kfp_inventory import identifier
from kfp_inventory import MAX_PAGES
from kfp_inventory import MAX_RECORDS

TERMINAL = {'SUCCEEDED', 'FAILED', 'CANCELED', 'SKIPPED', 3, 4, 5, 7}
ACTION = (
    'Preserve the original Workflow and reporting evidence; inspect API-server '
    'and persistence-agent diagnostics. See docs/operator-guides/workflow-report-recovery.md. Do not '
    'infer ownership from incoming reports or edit database ownership fields.')


def collect(client, namespaces):
    findings, completed = [], []
    count = 0

    def add(resource, reason, status='unknown'):
        findings.append(
            dict(
                rule='ownership.' + reason,
                status=status,
                resource=resource,
                evidence=reason,
                action=ACTION,
                verification='Partial, non-atomic source API evidence; '
                'not proof of recovery or authorization.'))

    def experiment(resource, exp_id, namespace):
        if not exp_id:
            add(resource, 'missing_experiment_reference', 'review_required')
            return
        try:
            exp = client.get('/apis/v2beta1/experiments/' + identifier(exp_id))
            if field(exp, 'experiment_id', 'experimentId') != exp_id:
                raise CollectionError('experiment_identity_mismatch')
            owner = exp.get('namespace')
            if owner in (None, '', '-'):
                add(resource, 'missing_experiment_namespace', 'review_required')
            elif owner != namespace:
                add(resource, 'experiment_namespace_mismatch',
                    'review_required')
            else:
                add(resource, 'experiment_namespace_present', 'observed')
        except CollectionError as error:
            add(resource, 'experiment_unavailable_' + error.reason)

    def run(namespace, resource, run_id):
        detail = client.get('/apis/v1beta1/runs/' + identifier(run_id))
        record = detail.get('run')
        if not isinstance(record, dict) or record.get(
                'id') != run_id or record.get('error'):
            raise CollectionError('invalid_run_detail')
        refs = field(record, 'resource_references', 'resourceReferences')
        if refs is None:
            refs = []
        if not isinstance(refs, list):
            raise CollectionError('invalid_resource_references')
        owners = set()
        for ref in refs:
            if not isinstance(ref, dict) or not isinstance(
                    ref.get('key'), dict):
                raise CollectionError('invalid_resource_reference')
            if ref.get('relationship') in (
                    'OWNER', 1) and ref['key'].get('type') in ('NAMESPACE', 5):
                value = ref['key'].get('id')
                if not isinstance(value, str):
                    raise CollectionError('invalid_namespace_reference')
                if value not in ('', '-'):
                    owners.add(value)
        if not owners:
            add(resource, 'missing_namespace_reference', 'review_required')
        elif owners != {namespace}:
            add(resource, 'namespace_reference_mismatch', 'review_required')
        else:
            add(resource, 'namespace_reference_present', 'observed')
        runtime = field(detail, 'pipeline_runtime', 'pipelineRuntime')
        if runtime is None:
            runtime = {}
        if not isinstance(runtime, dict):
            raise CollectionError('invalid_runtime')
        manifest = (
            field(runtime, 'workflow_manifest', 'workflowManifest') or
            field(runtime, 'pipeline_manifest', 'pipelineManifest'))
        if not manifest:
            add(resource, 'missing_stored_identity', 'review_required')
            return
        try:
            workflow = json.loads(manifest)
            metadata = workflow.get('metadata', {})
            if workflow.get('kind') != 'Workflow' or not isinstance(
                    metadata, dict):
                raise ValueError()
            if not all(
                    isinstance(metadata.get(key), str) and metadata[key]
                    for key in ('name', 'uid')):
                add(resource, 'incomplete_stored_identity', 'review_required')
            elif metadata.get('namespace') not in (None, '', namespace):
                add(resource, 'stored_identity_namespace_mismatch',
                    'review_required')
            else:
                add(resource, 'stored_identity_present', 'observed')
        except (ValueError, TypeError, AttributeError):
            add(resource, 'stored_identity_unreadable')

    for namespace in namespaces:
        for endpoint, key, camel, id_key, id_camel in (('runs', 'runs', 'runs',
                                                        'run_id', 'runId'),
                                                       ('recurringruns',
                                                        'recurring_runs',
                                                        'recurringRuns',
                                                        'recurring_run_id',
                                                        'recurringRunId')):
            token, seen = '', set()
            scope = endpoint + '/' + namespace
            try:
                for _ in range(MAX_PAGES):
                    page = client.get(
                        '/apis/v2beta1/' + endpoint,
                        dict(
                            namespace=namespace,
                            page_size=100,
                            page_token=token))
                    records = field(page, key, camel)
                    if records is None:
                        records = []
                    if not isinstance(records, list):
                        raise CollectionError('invalid_list')
                    for raw in records:
                        count += 1
                        if count > MAX_RECORDS:
                            raise CollectionError('record_limit')
                        if not isinstance(raw, dict) or raw.get('error'):
                            raise CollectionError('invalid_record')
                        record_id = field(raw, id_key, id_camel)
                        identifier(record_id)
                        # Unknown states remain in scope. Disabled schedules still
                        # need ownership evidence before an operator enables them.
                        if endpoint == 'runs' and isinstance(
                                raw.get('state'),
                            (str, int)) and raw.get('state') in TERMINAL:
                            continue
                        resource = scope + '/' + record_id
                        try:
                            experiment(
                                resource,
                                field(raw, 'experiment_id', 'experimentId'),
                                namespace)
                            if endpoint == 'runs':
                                run(namespace, resource, record_id)
                            else:
                                owner = raw.get('namespace')
                                reason = ('missing_schedule_namespace'
                                          if owner in (None, '', '-') else
                                          'schedule_namespace_present'
                                          if owner == namespace else
                                          'schedule_namespace_mismatch')
                                add(
                                    resource, reason, 'observed' if owner
                                    == namespace else 'review_required')
                                add(resource,
                                    'schedule_stored_identity_not_exposed')
                        except CollectionError as error:
                            add(resource, error.reason)
                    token = field(page, 'next_page_token',
                                  'nextPageToken') or ''
                    if not isinstance(token, str) or len(token) > 16384:
                        raise CollectionError('invalid_page_token')
                    if not token:
                        completed.append(scope)
                        break
                    if token in seen:
                        raise CollectionError('repeated_page_token')
                    seen.add(token)
                else:
                    raise CollectionError('page_limit')
            except CollectionError as error:
                add(scope, 'collection_' + error.reason)
    return findings, dict(
        completed_scopes=completed,
        records_examined=min(count, MAX_RECORDS),
        snapshot_consistency='not_atomic',
        coverage='Only API-visible records in selected namespaces; '
        'orphaned or unauthorized records may be invisible.')

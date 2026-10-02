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
"""Bounded source evidence collection; never substitutes current target
settings."""

from urllib.parse import quote

from kfp_http import CollectionError

MAX_RECORDS = 10000
MAX_PAGES = 100


def field(record, snake, camel):
    if snake in record and camel in record and record[snake] != record[camel]:
        raise CollectionError('conflicting_fields')
    return record.get(snake, record.get(camel))


def identifier(value):
    if not isinstance(value, str) or not value or len(value) > 253:
        raise CollectionError('invalid_identifier')
    return quote(value, safe='')


def v2_template(spec):
    # API pipeline_spec is the stored V2 IR, not an Argo workflow manifest.
    # This identifies its shape only; compilation and additional identities
    # remain unassessed.
    if not isinstance(spec, dict):
        return False
    info = field(spec, 'pipeline_info', 'pipelineInfo')
    return (isinstance(info, dict) and isinstance(info.get('name'), str) and
            bool(info['name']) and isinstance(spec.get('root'), dict) and
            'apiVersion' not in spec and 'kind' not in spec)


def collect(client, namespaces, record_budget=MAX_RECORDS):
    record_budget = min(MAX_RECORDS, record_budget)
    jobs, experiments, failures = [], {}, []
    versions = {}
    completed = []

    def fail(namespace, resource, reason):
        failures.append(
            dict(namespace=namespace, resource=resource, reason=reason))

    for namespace in namespaces:
        token, seen = '', set()
        try:
            for page in range(MAX_PAGES):
                response = client.get(
                    '/apis/v2beta1/recurringruns',
                    dict(namespace=namespace, page_size=100, page_token=token))
                records = field(response, 'recurring_runs', 'recurringRuns')
                if records is None:
                    records = []
                if not isinstance(records, list):
                    raise CollectionError('invalid_recurring_run_list')
                for raw in records:
                    if len(jobs) + len(experiments) >= record_budget:
                        raise CollectionError('record_limit')
                    if not isinstance(raw, dict):
                        raise CollectionError('invalid_recurring_run')
                    if raw.get('error'):
                        raise CollectionError('recurring_run_error')
                    job = {
                        snake: field(raw, snake, camel)
                        for snake, camel in (('recurring_run_id',
                                              'recurringRunId'),
                                             ('experiment_id', 'experimentId'),
                                             ('service_account',
                                              'serviceAccount'))
                    }
                    identifier(job['recurring_run_id'])
                    job['namespace'] = raw.get('namespace')
                    if job['namespace'] not in (None, '', namespace):
                        raise CollectionError('namespace_mismatch')
                    jobs.append(job)
                    try:
                        exp_id = job['experiment_id']
                        exp_path = identifier(exp_id)
                        if exp_id not in experiments:
                            exp = client.get('/apis/v2beta1/experiments/' +
                                             exp_path)
                            if field(exp, 'experiment_id',
                                     'experimentId') != exp_id or exp.get(
                                         'namespace') != namespace:
                                raise CollectionError(
                                    'experiment_scope_mismatch')
                            if len(jobs) + len(experiments) >= record_budget:
                                raise CollectionError('record_limit')
                            experiments[exp_id] = dict(
                                experiment_id=exp_id, namespace=namespace)
                        elif experiments[exp_id]['namespace'] != namespace:
                            raise CollectionError('experiment_scope_mismatch')
                        if job['service_account'] in (None, ''):
                            spec = field(raw, 'pipeline_spec', 'pipelineSpec')
                            ref = field(raw, 'pipeline_version_reference',
                                        'pipelineVersionReference')
                            if ref is not None:
                                if not isinstance(ref, dict):
                                    raise CollectionError(
                                        'invalid_pipeline_reference')
                                pipeline_id = field(ref, 'pipeline_id',
                                                    'pipelineId')
                                version_id = field(ref, 'pipeline_version_id',
                                                   'pipelineVersionId')
                                if not version_id:
                                    raise CollectionError(
                                        'moving_latest_version_unresolved')
                                key = (identifier(pipeline_id),
                                       identifier(version_id))
                                if key not in versions:
                                    version = client.get(
                                        '/apis/v2beta1/pipelines/' + key[0] +
                                        '/versions/' + key[1])
                                    if version.get('error'):
                                        raise CollectionError(
                                            'pipeline_version_error')
                                    if field(
                                            version, 'pipeline_id', 'pipelineId'
                                    ) != pipeline_id or field(
                                            version, 'pipeline_version_id',
                                            'pipelineVersionId') != version_id:
                                        raise CollectionError(
                                            'pipeline_version_mismatch')
                                    versions[key] = v2_template(
                                        field(version, 'pipeline_spec',
                                              'pipelineSpec'))
                                is_v2 = versions[key]
                            else:
                                is_v2 = v2_template(spec)
                            if is_v2:
                                job['_readiness_v2_default'] = True
                            else:
                                raise CollectionError(
                                    'template_identity_unresolved')
                    except CollectionError as error:
                        fail(namespace,
                             'RecurringRun/' + job['recurring_run_id'],
                             error.reason)
                token = field(response, 'next_page_token',
                              'nextPageToken') or ''
                if not isinstance(token, str) or len(token) > 16384:
                    raise CollectionError('invalid_page_token')
                if not token:
                    completed.append(namespace)
                    break
                if token in seen:
                    raise CollectionError('repeated_page_token')
                seen.add(token)
            else:
                raise CollectionError('page_limit')
        except CollectionError as error:
            fail(namespace, 'KFP recurring runs', error.reason)
    return dict(
        recurring_runs=jobs,
        experiments=list(experiments.values())), failures, dict(
            requested_namespaces=namespaces,
            list_completed_namespaces=completed,
            recurring_run_records=len(jobs),
            experiment_records=len(experiments),
            failed_checks=len(failures),
            snapshot_consistency='not_atomic')

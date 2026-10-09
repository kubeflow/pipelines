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
import copy
import unittest
from unittest import mock

from kfp_http import CollectionError
import workload_inventory

API = '/apis/v2beta1/'
SPEC = {'pipelineInfo': {'name': 'test'}, 'root': {}}


class Client:
    """Strict endpoint fixture: every unexpected GET fails the collection."""

    def __init__(self):
        self.routes = {}
        self.calls = []

    def list(self,
             resource,
             records,
             namespace='team',
             token='',
             next_token='',
             total=None,
             parent=None):
        endpoint = 'recurringruns' if resource == 'recurring_runs' else resource
        params = dict(namespace=namespace, page_size=100, page_token=token)
        if parent is not None:
            endpoint = 'pipelines/' + parent + '/versions'
            params = dict(
                sort_by='created_at desc', page_size=100, page_token=token)
        response = {resource: records, 'next_page_token': next_token}
        if total is not None:
            response['total_size'] = total
        self.routes[(API + endpoint, tuple(sorted(params.items())))] = response

    def detail(self, path, record):
        self.routes[(API + path, ())] = record

    def get(self, path, params=None):
        self.calls.append((path, params))
        key = (path, tuple(sorted((params or {}).items())))
        if key in self.routes:
            response = self.routes[key]
            if isinstance(response, Exception):
                raise response
            return copy.deepcopy(response)
        if params is not None and path in [
                API + p
                for p in ('experiments', 'pipelines', 'runs', 'recurringruns')
        ]:
            return {}
        raise CollectionError('unexpected_request')


def experiment(identifier='exp', namespace='team'):
    return dict(experiment_id=identifier, namespace=namespace)


def pipeline(identifier='pipe', namespace='team'):
    return dict(pipeline_id=identifier, namespace=namespace)


def version(identifier='version',
            parent='pipe',
            spec=SPEC,
            created_at='2026-01-01T00:00:00Z'):
    return dict(
        pipeline_id=parent,
        pipeline_version_id=identifier,
        pipeline_spec=spec,
        created_at=created_at)


def run(identifier='run', spec=SPEC, parent='exp', **extra):
    return dict(
        run_id=identifier, experiment_id=parent, pipeline_spec=spec, **extra)


def recurring(identifier='job', spec=SPEC, parent='exp', **extra):
    return dict(
        recurring_run_id=identifier,
        experiment_id=parent,
        pipeline_spec=spec,
        **extra)


class WorkloadInventoryTest(unittest.TestCase):

    def test_complete_empty_scopes_are_distinct_from_failure(self):
        client = Client()
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team', 'other'])
        self.assertTrue(coverage['complete'])
        self.assertEqual(failures, [])
        self.assertEqual(len(coverage['resources']), 8)
        self.assertTrue(
            all(item['list_complete'] for item in coverage['resources']))
        self.assertTrue(all(not records for records in inventory.values()))
        self.assertEqual({params['namespace'] for _, params in client.calls},
                         {'team', 'other'})

    def test_complete_inventory_includes_empty_experiments_and_all_versions(
            self):
        client = Client()
        client.list('experiments', [experiment(), experiment('empty')])
        client.list('pipelines', [pipeline()])
        client.list(
            'pipeline_versions', [version('new')],
            parent='pipe',
            next_token='older',
            total=2)
        client.list(
            'pipeline_versions', [version('old', spec=None)],
            parent='pipe',
            token='older',
            total=2)
        client.detail('pipelines/pipe/versions/old', version('old'))
        client.list('runs', [run()])
        client.list('recurring_runs', [recurring(status='DISABLED')])
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(failures, [])
        self.assertTrue(coverage['complete'])
        self.assertEqual(
            coverage['record_counts'],
            dict(
                experiments=2,
                pipelines=1,
                pipeline_versions=2,
                runs=1,
                recurring_runs=1))
        self.assertEqual(inventory['runs'][0]['namespace'], 'team')
        self.assertEqual(inventory['recurring_runs'][0]['status'], 'DISABLED')
        self.assertEqual(inventory['pipeline_versions'][1]['pipeline_spec'],
                         SPEC)

    def test_source_camelcase_records_and_hydration(self):
        client = Client()
        client.list('experiments', [{
            'experimentId': 'exp',
            'namespace': 'team'
        }])
        client.list('runs', [{'runId': 'run', 'experimentId': 'exp'}])
        client.detail('runs/run', {
            'runId': 'run',
            'experimentId': 'exp',
            'pipelineSpec': SPEC
        })
        client.list('recurring_runs', [{
            'recurringRunId': 'job',
            'experimentId': 'exp',
            'pipelineSpec': SPEC
        }])
        key = (API + 'recurringruns',
               tuple(
                   sorted(
                       dict(namespace='team', page_size=100,
                            page_token='').items())))
        client.routes[key]['recurringRuns'] = client.routes[key].pop(
            'recurring_runs')
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(failures, [])
        self.assertTrue(coverage['complete'])
        self.assertEqual(inventory['runs'][0]['pipelineSpec'], SPEC)
        self.assertEqual(len(inventory['recurring_runs']), 1)

    def test_list_pagination_deduplicates_overlapping_records(self):
        client = Client()
        client.list('experiments', [experiment()], next_token='next', total=2)
        client.list(
            'experiments', [experiment(), experiment('empty')],
            token='next',
            total=2)
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team', 'team'])
        self.assertEqual(failures, [])
        self.assertEqual(len(inventory['experiments']), 2)
        traversal = coverage['resources'][0]
        self.assertEqual(traversal['pages'], 2)
        self.assertEqual(traversal['listed_records'], 3)
        self.assertEqual(traversal['unique_listed_records'], 2)
        self.assertEqual(coverage['requested_namespaces'], ['team'])

    def test_partial_page_failure_keeps_evidence_and_other_resource_results(
            self):
        client = Client()
        client.list('experiments', [experiment()], next_token='next')
        key = (API + 'experiments',
               tuple(
                   sorted(
                       dict(namespace='team', page_size=100,
                            page_token='next').items())))
        client.routes[key] = CollectionError('request_budget_exceeded')
        client.list('runs', [run()])
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(len(inventory['experiments']), 1)
        self.assertEqual(len(inventory['runs']), 1)
        self.assertFalse(coverage['complete'])
        self.assertFalse(coverage['resources'][0]['list_complete'])
        self.assertEqual(failures[0]['reason'], 'request_budget_exceeded')

    def test_combined_budget_counts_every_resource_and_retains_partial_records(
            self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('pipelines', [pipeline()])
        client.list('pipeline_versions', [version()], parent='pipe')
        client.list('runs', [run()])
        with mock.patch.object(workload_inventory, 'MAX_RECORDS', 3):
            inventory, failures, coverage = workload_inventory.collect(
                client, ['team'])
        self.assertEqual(sum(map(len, inventory.values())), 3)
        self.assertEqual(inventory['runs'], [])
        self.assertFalse(coverage['complete'])
        self.assertIn('record_limit', [item['reason'] for item in failures])

    def test_repeated_tokens_and_repeated_records_fail_closed(self):
        for same_records in (False, True):
            with self.subTest(same_records=same_records):
                client = Client()
                client.list('experiments', [experiment()], next_token='next')
                client.list(
                    'experiments', [experiment()] if same_records else [],
                    token='next',
                    next_token='last' if same_records else 'next')
                _, failures, coverage = workload_inventory.collect(
                    client, ['team'])
                self.assertFalse(coverage['complete'])
                self.assertEqual(
                    failures[0]['reason'], 'repeated_page_records'
                    if same_records else 'repeated_page_token')

    def test_page_limit_and_total_size_mismatch_stay_incomplete(self):
        client = Client()
        client.list('experiments', [experiment()], next_token='next')
        with mock.patch.object(workload_inventory, 'MAX_PAGES', 1):
            _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures[0]['reason'], 'page_limit')
        self.assertFalse(coverage['complete'])
        client.list('experiments', [], total=3)
        _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures[0]['reason'], 'list_count_mismatch')
        self.assertFalse(coverage['resources'][0]['list_complete'])

    def test_wrong_namespaces_are_rejected_before_retaining_raw_records(self):
        client = Client()
        client.list('experiments', [experiment(namespace='outside')])
        client.list('pipelines', [pipeline(namespace='outside')])
        client.list('runs', [run()])
        client.detail('experiments/exp', experiment(namespace='outside'))
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertTrue(all(not records for records in inventory.values()))
        self.assertEqual({item['reason'] for item in failures},
                         {'namespace_mismatch'})
        self.assertFalse(coverage['complete'])

    def test_unlisted_referenced_experiment_is_verified_and_collected(self):
        client = Client()
        client.list('runs', [run()])
        client.detail('experiments/exp', experiment())
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(failures, [])
        self.assertEqual(len(inventory['experiments']), 1)
        self.assertEqual(coverage['resources'][0]['records'], 1)
        self.assertEqual(coverage['resources'][0]['unique_listed_records'], 0)

    def test_wrong_detail_ids_or_parent_are_never_accepted(self):
        for replacement, reason in ((run('other'), 'referenced_id_mismatch'),
                                    (run(parent='different'),
                                     'experiment_mismatch')):
            with self.subTest(reason=reason):
                client = Client()
                client.list('experiments', [experiment()])
                client.list('runs', [run(spec=None)])
                client.detail('runs/run', replacement)
                inventory, failures, coverage = workload_inventory.collect(
                    client, ['team'])
                self.assertFalse(coverage['complete'])
                self.assertIn(reason, [item['reason'] for item in failures])
                self.assertIsNone(inventory['runs'][0]['pipeline_spec'])

    def test_inaccessible_referenced_record_is_failure_not_empty_success(self):
        client = Client()
        client.list('runs', [run()])
        client.detail('experiments/exp', CollectionError('access_denied'))
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(inventory['runs'], [])
        self.assertEqual(failures[0]['reason'], 'access_denied')
        self.assertFalse(coverage['complete'])
        traversal = next(item for item in coverage['resources']
                         if item['resource'] == 'runs')
        self.assertTrue(traversal['list_complete'])
        self.assertEqual(traversal['failed_checks'], 1)

    def test_pinned_reference_observes_spec_without_overwriting_run_source(
            self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('pipelines', [pipeline()])
        client.list('pipeline_versions', [version()], parent='pipe')
        record = run(
            spec=None,
            pipeline_version_reference=dict(
                pipeline_id='pipe', pipeline_version_id='version'))
        client.list('runs', [record])
        client.detail('runs/run', record)
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(failures, [])
        self.assertTrue(coverage['complete'])
        collected = inventory['runs'][0]
        self.assertIsNone(collected['pipeline_spec'])
        self.assertEqual(collected['_readiness_pipeline_spec'], SPEC)
        self.assertEqual(
            collected['_readiness_version_reference'],
            dict(
                kind='pinned',
                resolution='observed',
                pipeline_id='pipe',
                pipeline_version_id='version'))

    def test_latest_is_observation_and_uses_newest_first_api_order(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('pipelines', [pipeline()])
        client.list(
            'pipeline_versions', [
                version('newest', created_at='2026-01-02T00:00:00Z'),
                version('old')
            ],
            parent='pipe')
        record = recurring(
            spec=None, pipeline_version_reference=dict(pipeline_id='pipe'))
        client.list('recurring_runs', [record])
        client.detail('recurringruns/job', record)
        inventory, failures, _ = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures, [])
        self.assertEqual(
            inventory['recurring_runs'][0]['_readiness_version_reference'],
            dict(
                kind='moving_latest',
                resolution='observed',
                pipeline_id='pipe',
                pipeline_version_id='newest'))
        self.assertTrue(
            any(
                params and params.get('sort_by') == 'created_at desc'
                for _, params in client.calls))

    def test_latest_from_incomplete_version_list_is_unresolved(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('pipelines', [pipeline()])
        client.list(
            'pipeline_versions', [version('newest')],
            parent='pipe',
            next_token='missing')
        record = recurring(
            spec=None, pipeline_version_reference=dict(pipeline_id='pipe'))
        client.list('recurring_runs', [record])
        client.detail('recurringruns/job', record)
        inventory, failures, _ = workload_inventory.collect(client, ['team'])
        self.assertIn('moving_latest_version_unresolved',
                      [item['reason'] for item in failures])
        self.assertNotIn('_readiness_pipeline_spec',
                         inventory['recurring_runs'][0])

    def test_legacy_version_is_resolved_from_scoped_inventory(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('pipelines', [pipeline()])
        client.list('pipeline_versions', [version()], parent='pipe')
        record = run(spec=None, pipeline_version_id='version')
        client.list('runs', [record])
        client.detail('runs/run', record)
        inventory, failures, _ = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures, [])
        self.assertEqual(
            inventory['runs'][0]['_readiness_version_reference']['kind'],
            'legacy_version_id')

    def test_legacy_version_without_scoped_parent_is_unresolved(self):
        client = Client()
        client.list('experiments', [experiment()])
        record = run(spec=None, pipeline_version_id='missing')
        client.list('runs', [record])
        client.detail('runs/run', record)
        _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures[0]['reason'],
                         'legacy_version_parent_unresolved')
        self.assertFalse(coverage['complete'])

    def test_pipeline_reference_outside_selected_scope_does_not_fetch_versions(
            self):
        client = Client()
        client.list('experiments', [experiment()])
        record = run(
            spec=None,
            pipeline_version_reference=dict(
                pipeline_id='outside', pipeline_version_id='version'))
        client.list('runs', [record])
        client.detail('runs/run', record)
        client.detail('pipelines/outside', pipeline('outside', 'other'))
        inventory, failures, _ = workload_inventory.collect(client, ['team'])
        self.assertEqual(inventory['pipelines'], [])
        self.assertEqual(failures[0]['reason'],
                         'referenced_pipeline_outside_scope')
        self.assertFalse(any('/versions' in path for path, _ in client.calls))

    def test_shared_inventory_requires_opt_in_and_never_unscopes_workloads(
            self):
        for include_shared in (False, True):
            with self.subTest(include_shared=include_shared):
                client = Client()
                client.list('pipelines', [pipeline('shared', '')], namespace='')
                client.list(
                    'pipeline_versions', [version(parent='shared')],
                    parent='shared')
                inventory, failures, coverage = workload_inventory.collect(
                    client, ['team'], include_shared=include_shared)
                self.assertEqual(failures, [])
                self.assertTrue(coverage['complete'])
                self.assertEqual(
                    len(inventory['pipelines']), int(include_shared))
                unscoped = [
                    (path, params['namespace'])
                    for path, params in client.calls
                    if params is not None and params.get('namespace') in ('',
                                                                          '-')
                ]
                self.assertEqual(unscoped, [(API + 'pipelines', ''),
                                            (API + 'pipelines',
                                             '-')] if include_shared else [])

    def test_single_user_binding_is_explicit_and_limited_to_one_namespace(self):
        client = Client()
        client.list('experiments', [experiment(namespace='')])
        client.list('pipelines', [pipeline(namespace='')])
        client.list('pipeline_versions', [version()], parent='pipe')
        client.list('runs', [run()])
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'], source_single_user_namespace='team')
        self.assertEqual(failures, [])
        self.assertEqual(coverage['source_single_user_namespace'], 'team')
        self.assertEqual(
            inventory['experiments'][0]['_readiness_namespace_evidence'],
            'operator_asserted_single_user')
        self.assertEqual(inventory['pipelines'][0]['namespace'], 'team')
        self.assertEqual(inventory['runs'][0]['namespace'], 'team')
        with self.assertRaisesRegex(CollectionError,
                                    'invalid_single_user_namespace'):
            workload_inventory.collect(
                client, ['team', 'other'], source_single_user_namespace='team')
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertFalse(coverage['complete'])
        self.assertEqual(inventory['experiments'], [])

    def test_embedded_urls_and_annotations_are_never_followed_or_trusted(self):
        client = Client()
        client.list('experiments', [experiment()])
        record = run(
            plugins_input={'test': {
                'token': 'private'
            }},
            runtime_config={
                'pipeline_root': 'https://not-kfp.example/artifacts'
            },
            _readiness_version_reference={'resolution': 'observed'})
        client.list('runs', [record])
        inventory, failures, _ = workload_inventory.collect(client, ['team'])
        self.assertEqual(failures, [])
        self.assertEqual(inventory['runs'][0]['plugins_input']['test']['token'],
                         'private')
        self.assertNotIn('_readiness_version_reference', inventory['runs'][0])
        self.assertTrue(all(path.startswith(API) for path, _ in client.calls))

    def test_spec_missing_from_version_get_does_not_fetch_package_url(self):
        client = Client()
        client.list('pipelines', [pipeline()])
        raw = version(spec=None)
        raw['package_url'] = {'pipeline_url': 'https://outside.example/secret'}
        client.list('pipeline_versions', [raw], parent='pipe')
        client.detail('pipelines/pipe/versions/version', raw)
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(len(inventory['pipeline_versions']), 1)
        self.assertFalse(coverage['complete'])
        self.assertEqual(failures[0]['reason'], 'pipeline_spec_unavailable')
        self.assertTrue(all(path.startswith(API) for path, _ in client.calls))

    def test_wrong_pipeline_version_parent_is_rejected(self):
        client = Client()
        client.list('pipelines', [pipeline()])
        client.list(
            'pipeline_versions', [version(parent='outside')], parent='pipe')
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(inventory['pipeline_versions'], [])
        self.assertEqual(failures[0]['reason'], 'pipeline_version_mismatch')
        self.assertFalse(coverage['complete'])

    def test_conflicting_field_aliases_and_error_rows_remain_failures(self):
        client = Client()
        client.list('experiments', [
            dict(experiment_id='exp', experimentId='other', namespace='team'),
            dict(experiment_id='broken', error={'message': 'do not expose'})
        ])
        _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertEqual([item['reason'] for item in failures],
                         ['conflicting_fields', 'record_error'])
        self.assertNotIn('do not expose', str(failures))
        self.assertFalse(coverage['complete'])

    def test_unlisted_recurring_parent_is_verified(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('runs', [run(recurring_run_id='job')])
        client.detail('recurringruns/job', recurring())
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertEqual(failures, [])
        self.assertTrue(coverage['complete'])
        self.assertEqual(len(inventory['recurring_runs']), 1)

    def test_recurring_parent_must_belong_to_same_experiment(self):
        client = Client()
        client.list('experiments', [experiment(), experiment('different')])
        client.list('runs', [run(recurring_run_id='job')])
        client.list('recurring_runs', [recurring(parent='different')])
        _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertFalse(coverage['complete'])
        self.assertEqual(failures[0]['reason'], 'recurring_run_scope_mismatch')

    def test_inaccessible_recurring_parent_is_reported(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('runs', [run(recurring_run_id='missing')])
        client.detail('recurringruns/missing', CollectionError('access_denied'))
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertFalse(coverage['complete'])
        self.assertEqual(failures[0]['reason'], 'access_denied')
        self.assertEqual(inventory['runs'][0]['_readiness_collection_errors'],
                         ['access_denied'])

    def test_duplicate_run_cannot_change_experiment(self):
        client = Client()
        client.list('experiments', [experiment(), experiment('other')])
        client.list('runs', [run(), run(parent='other')])
        _, failures, coverage = workload_inventory.collect(client, ['team'])
        self.assertFalse(coverage['complete'])
        self.assertEqual(failures[0]['reason'], 'experiment_mismatch')

    def test_non_string_empty_page_token_is_not_end_of_list(self):
        for token in (False, 0, []):
            with self.subTest(token=token):
                client = Client()
                client.list('experiments', [], next_token=token)
                _, failures, coverage = workload_inventory.collect(
                    client, ['team'])
                self.assertFalse(coverage['complete'])
                self.assertEqual(failures[0]['reason'], 'invalid_page_token')

    def test_latest_tied_or_missing_creation_times_remain_unknown(self):
        for second_time, expected in (('2026-01-01T00:00:00Z',
                                       'moving_latest_version_ambiguous'),
                                      (None,
                                       'version_creation_time_unresolved')):
            with self.subTest(second_time=second_time):
                client = Client()
                client.list('experiments', [experiment()])
                client.list('pipelines', [pipeline()])
                client.list(
                    'pipeline_versions',
                    [version('a'),
                     version('b', created_at=second_time)],
                    parent='pipe')
                record = recurring(
                    spec=None,
                    pipeline_version_reference=dict(pipeline_id='pipe'))
                client.list('recurring_runs', [record])
                client.detail('recurringruns/job', record)
                inventory, failures, coverage = workload_inventory.collect(
                    client, ['team'])
                self.assertFalse(coverage['complete'])
                self.assertEqual(failures[0]['reason'], expected)
                self.assertNotIn('_readiness_pipeline_spec',
                                 inventory['recurring_runs'][0])

    def test_non_string_version_ids_are_never_latest_references(self):
        for value in (False, 0, [], {}):
            with self.subTest(value=value):
                client = Client()
                client.list('experiments', [experiment()])
                record = recurring(
                    spec=None,
                    pipeline_version_reference=dict(
                        pipeline_id='pipe', pipeline_version_id=value))
                client.list('recurring_runs', [record])
                client.detail('recurringruns/job', record)
                _, failures, coverage = workload_inventory.collect(
                    client, ['team'])
                self.assertFalse(coverage['complete'])
                self.assertEqual(failures[0]['reason'], 'invalid_identifier')

    def test_conflicting_oneof_pipeline_sources_are_unknown(self):
        client = Client()
        client.list('experiments', [experiment()])
        client.list('runs', [
            run(
                pipeline_version_reference=dict(
                    pipeline_id='pipe', pipeline_version_id='version'))
        ])
        inventory, failures, coverage = workload_inventory.collect(
            client, ['team'])
        self.assertFalse(coverage['complete'])
        self.assertEqual(failures[0]['reason'], 'conflicting_pipeline_source')
        self.assertEqual(inventory['runs'][0]['_readiness_collection_errors'],
                         ['conflicting_pipeline_source'])


class OfflineInventoryTest(unittest.TestCase):

    def test_offline_records_are_scoped_and_pinned_references_resolve(self):
        raw = dict(
            experiments=[experiment()],
            pipelines=[pipeline()],
            pipeline_versions=[version()],
            runs=[
                run(spec=None,
                    pipeline_version_reference=dict(
                        pipeline_id='pipe', pipeline_version_id='version'))
            ])
        inventory = workload_inventory.validate(raw, ['team'])
        self.assertEqual(inventory['runs'][0]['namespace'], 'team')
        self.assertEqual(inventory['runs'][0]['_readiness_pipeline_spec'], SPEC)
        self.assertNotIn('namespace', raw['runs'][0])
        self.assertEqual(inventory['recurring_runs'], [])

    def test_offline_annotations_cannot_assert_scope_or_resolved_spec(self):
        raw = dict(
            experiments=[experiment()],
            runs=[
                run(spec=None,
                    _readiness_pipeline_spec=SPEC,
                    _readiness_version_reference={'resolution': 'observed'},
                    _readiness_collection_errors=[])
            ])
        inventory = workload_inventory.validate(raw, ['team'])
        collected = inventory['runs'][0]
        self.assertNotIn('_readiness_pipeline_spec', collected)
        self.assertNotIn('_readiness_version_reference', collected)
        self.assertEqual(collected['_readiness_collection_errors'],
                         ['pipeline_spec_unavailable'])

    def test_missing_references_with_explicit_scope_are_unknown(self):
        raw = dict(runs=[
            run(namespace='team',
                spec=None,
                pipeline_version_reference=dict(
                    pipeline_id='missing', pipeline_version_id='v'))
        ])
        inventory = workload_inventory.validate(raw, ['team'])
        collected = inventory['runs'][0]
        self.assertEqual(collected['_readiness_namespace_evidence'],
                         'unresolved_parent')
        self.assertEqual(collected['_readiness_collection_errors'],
                         ['offline_reference_missing'])
        self.assertEqual(
            collected['_readiness_version_reference']['resolution'],
            'unresolved')

    def test_missing_parent_without_namespace_cannot_invent_scope(self):
        with self.assertRaisesRegex(CollectionError, 'namespace_mismatch'):
            workload_inventory.validate(dict(runs=[run()]), ['team'])

    def test_missing_latest_never_claims_a_version_from_offline_array_order(
            self):
        raw = dict(
            experiments=[experiment()],
            pipelines=[pipeline()],
            pipeline_versions=[version('old'), version('new')],
            recurring_runs=[
                recurring(
                    spec=None,
                    pipeline_version_reference=dict(pipeline_id='pipe'))
            ])
        inventory = workload_inventory.validate(raw, ['team'])
        collected = inventory['recurring_runs'][0]
        self.assertNotIn('_readiness_pipeline_spec', collected)
        self.assertEqual(collected['_readiness_collection_errors'],
                         ['moving_latest_version_unresolved'])

    def test_duplicates_wrong_types_and_combined_record_limit_rejected(self):
        cases = [([], 'invalid_workload_inventory'),
                 ({
                     'runs': {}
                 }, 'invalid_resource_list'),
                 ({
                     'experiments': [experiment(), experiment()]
                 }, 'duplicate_record'), ({
                     'runs': ['bad']
                 }, 'invalid_record')]
        for raw, reason in cases:
            with self.subTest(reason=reason):
                with self.assertRaisesRegex(CollectionError, reason):
                    workload_inventory.validate(raw, ['team'])
        with mock.patch.object(workload_inventory, 'MAX_RECORDS', 1):
            with self.assertRaisesRegex(CollectionError, 'record_limit'):
                workload_inventory.validate(
                    dict(experiments=[experiment()], runs=[run()]), ['team'])

    def test_cross_namespace_parents_are_rejected(self):
        raw = dict(experiments=[experiment()], runs=[run(namespace='outside')])
        with self.assertRaisesRegex(CollectionError, 'namespace_mismatch'):
            workload_inventory.validate(raw, ['team', 'outside'])

    def test_offline_shared_pipeline_requires_explicit_opt_in(self):
        raw = dict(
            pipelines=[pipeline(namespace='')], pipeline_versions=[version()])
        with self.assertRaisesRegex(CollectionError, 'namespace_mismatch'):
            workload_inventory.validate(raw, ['team'])
        inventory = workload_inventory.validate(
            raw, ['team'], include_shared=True)
        self.assertEqual(inventory['pipeline_versions'][0]['namespace'], '')

    def test_single_user_namespace_assertion_is_retained_offline(self):
        raw = dict(experiments=[experiment(namespace='')], runs=[run()])
        inventory = workload_inventory.validate(
            raw, ['team'], source_single_user_namespace='team')
        self.assertEqual(
            inventory['experiments'][0]['_readiness_namespace_evidence'],
            'operator_asserted_single_user')
        self.assertEqual(inventory['runs'][0]['namespace'], 'team')


if __name__ == '__main__':
    unittest.main()

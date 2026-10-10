# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
import base64
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import live_pagination as fixture


def response(values, token=''):
    return {
        'experiments': [{
            'experiment_id': value
        } for value in values],
        'next_page_token': token
    }


class PaginationTest(unittest.TestCase):

    def test_transport_requires_exact_structured_restart_detail(self):
        detail = {
            '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
            'reason': 'PAGINATION_RESTART_REQUIRED',
            'domain': 'kubeflow.org'
        }
        valid = {'code': 9, 'details': [detail], 'message': 'private-message'}
        cases = [(400, valid, True), (503, valid, False),
                 (400, dict(valid, code=3), False),
                 (400, dict(valid, details=[]), False),
                 (400, dict(valid, details=[dict(detail,
                                                 domain='other')]), False),
                 (400, dict(valid, details=[dict(detail,
                                                 reason='OTHER')]), False),
                 (400, [], False)]
        for status, body, expected in cases:
            with self.subTest(status=status, body=body), \
                 mock.patch.object(fixture.http.client, 'HTTPConnection') as connection:
                response = connection.return_value.getresponse.return_value
                response.status = status
                response.read.return_value = json.dumps(body).encode()
                with self.assertRaises(fixture.ApiError) as caught:
                    fixture.request(8888, 'GET')
                self.assertEqual(caught.exception.restart_required, expected)
                self.assertNotIn('private-message', str(caught.exception))

    def test_alternating_readers_and_exact_sequence(self):
        with mock.patch.object(
                fixture,
                'page',
                side_effect=[
                    response(['a'], 'one'),
                    response(['b'], 'two'),
                    response(['c'])
                ]) as page:
            result = fixture.walk([8889, 8888], 'criteria', ['a', 'b', 'c'])
        self.assertEqual(result['ports'], [8889, 8888, 8889])
        self.assertEqual(page.call_args_list, [
            mock.call(8889, 'criteria', '', True),
            mock.call(8888, 'criteria', 'one', True),
            mock.call(8889, 'criteria', 'two', True)
        ])

    def test_saved_page_is_not_refetched(self):
        with mock.patch.object(
                fixture, 'page', return_value=response(['b'])) as page:
            fixture.walk([8888],
                         'criteria', ['a', 'b'],
                         response(['a'], 'old-token'),
                         repeat=False)
        page.assert_called_once_with(8888, 'criteria', 'old-token', False)

    def test_duplicate_missing_extra_and_reordered_rows_fail(self):
        for values in (['a', 'a'], ['a'], ['a', 'b', 'c'], ['b', 'a']):
            with self.subTest(values=values), mock.patch.object(
                    fixture, 'page', return_value=response(values)):
                with self.assertRaises(ValueError):
                    fixture.walk([8888], 'criteria', ['a', 'b'])

    def test_saved_source_trace_is_explicit(self):
        result = fixture.walk([8888], 'criteria', ['a'], response(['a']))
        self.assertEqual(result['ports'], ['saved_source_page'])

    def test_filter_operation_matches_api_version(self):
        for version, key in (('v1beta1', 'op'), ('v2beta1', 'operation')):
            predicate = json.loads(
                fixture.criteria('fixture', version=version))['predicates'][0]
            self.assertEqual(predicate[key], 'EQUALS')
            self.assertNotIn('operation' if key == 'op' else 'op', predicate)

    def test_source_serializes_original_and_extended_cases(self):
        self.capture_source_with_transport()

    def test_source_records_old_metric_sort_failure_after_independent_inventory(
            self):
        self.capture_source_with_transport(metric_failure=True)

    def capture_source_with_transport(self, metric_failure=False):
        rows = {'experiments': [], 'runs': []}
        seen_endpoints = set()

        def transport(port,
                      method,
                      body=None,
                      endpoint='/apis/v2beta1/experiments',
                      **params):
            seen_endpoints.add(endpoint)
            if endpoint.endswith(':reportMetrics'):
                return {'results': [{'status': 'OK'}]}
            collection = endpoint.rsplit('/', 1)[1]
            v1 = '/v1beta1/' in endpoint
            identity = 'id' if v1 else collection[:-1] + '_id'
            if method == 'POST':
                value = str(len(rows[collection]))
                rows[collection].append({'id': value, **body})
                return ({
                    'run': {
                        'id': value
                    }
                } if collection == 'runs' else {
                    identity: value
                })
            if metric_failure and params.get('sort_by',
                                             '').startswith('metric:'):
                raise fixture.ApiError(500, endpoint)
            selected = rows[collection]
            token = params.get('page_token')
            offset = 0
            if token:
                saved = json.loads(token)
                selected = [
                    row for row in selected if row['id'] in saved['ids']
                ]
                offset = saved['offset']
            elif 'filter' in params:
                predicate = json.loads(params['filter'])['predicates'][0]
                self.assertIn('op' if v1 else 'operation', predicate)
                marker = predicate.get(
                    'string_value') or predicate['string_values']['values'][0]
                operation = predicate['op' if v1 else 'operation']
                selected = [
                    row for row in selected
                    if (marker in row['description'] if operation ==
                        'IS_SUBSTRING' else row['description'] == marker)
                ]
            size = params['page_size']
            result = {
                collection: [{
                    identity: row['id']
                } for row in selected[offset:offset + size]]
            }
            if offset + size < len(selected):
                result['next_page_token'] = json.dumps({
                    'ids': [row['id'] for row in selected],
                    'offset': offset + size
                })
            return result

        with tempfile.TemporaryDirectory() as directory, mock.patch.object(
                fixture, 'request', side_effect=transport):
            path = Path(directory) / 'state.json'
            fixture.source(path, 'example/api:2.17.2')
            state = json.loads(path.read_text())
            # Later fixture creation must not change any original filter's set.
            for case in state['cases']:
                fixture.walk([8888], case['filter'], state['expected'])
        self.assertEqual([case['operation'] for case in state['cases']],
                         ['EQUALS', 'IS_SUBSTRING', 'IN'])
        self.assertEqual(len(state['extended']), 32)
        for case in state['cases'] + state['extended']:
            if metric_failure and case.get('sort', '').startswith('metric:'):
                self.assertEqual(case['source_baseline']['phase'],
                                 'sorted_inventory')
                self.assertIsNone(case['source_order'])
                self.assertIsNone(case['first'])
            else:
                self.assertEqual(case['source_baseline']['outcome'], 'passed')
                self.assertTrue(case['first']['next_page_token'])
        self.assertIn('/apis/v1beta1/experiments', seen_endpoints)
        self.assertIn('/apis/v1beta1/runs', seen_endpoints)
        self.assertIn('/apis/v2beta1/runs', seen_endpoints)

    def test_source_image_pin_is_required(self):
        with self.assertRaisesRegex(ValueError, 'pinned'):
            fixture.source(None, 'example/api:latest')

    def test_token_cycle_fails(self):
        with mock.patch.object(
                fixture, 'page', return_value=response(['a'], 'same')):
            with self.assertRaisesRegex(ValueError, 'repeated page token'):
                fixture.walk([8888], 'criteria', ['a'])

    def test_page_budget(self):
        with mock.patch.object(
                fixture,
                'page',
                side_effect=[response([str(i)], str(i)) for i in range(20)]):
            with self.assertRaisesRegex(ValueError, 'page budget'):
                fixture.walk([8888], 'criteria', [])


class ExtendedPaginationTest(unittest.TestCase):

    def setUp(self):
        self.case = {
            'endpoint': '/apis/v2beta1/runs',
            'collection': 'runs',
            'identity': 'run_id',
            'params': {
                'experiment_id': 'experiment'
            },
            'sort': 'display_name asc',
            'created_ids': ['a', 'b', 'c'],
            'source_order': ['a', 'b', 'c'],
            'first': {
                'runs': [{
                    'run_id': 'a'
                }],
                'next_page_token': 'old'
            },
            'source_baseline': {
                'outcome': 'passed'
            }
        }

    def test_affected_legacy_sort_requires_structured_restart(self):
        self.case['sort'] = 'scheduled_at asc'
        restart = {
            'outcome': 'restart_required',
            'reason': 'PAGINATION_RESTART_REQUIRED'
        }
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value=restart), \
             mock.patch.object(fixture, 'old_reader_boundary', return_value=restart), \
             mock.patch.object(fixture, 'reader_version_boundary', return_value={'outcome': 'old_reader_rejected'}):
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['disposition'],
                         'explicit_restart_for_legacy_nullable_order')
        for invalid in ({'outcome': 'passed'}, {'outcome': 'failed'}):
            with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
                 mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
                 mock.patch.object(fixture, 'observe_walk', return_value=invalid):
                with self.assertRaisesRegex(ValueError,
                                            'did not require restart'):
                    fixture.validate_extended([self.case])

    def test_restart_check_rejects_generic_errors_and_silent_success(self):
        for error in (fixture.ApiError(400), fixture.ApiError(503)):
            with mock.patch.object(fixture, 'case_page', side_effect=error):
                with self.assertRaises(fixture.ApiError):
                    fixture.require_restart(8888, self.case, 'old')
        with mock.patch.object(fixture, 'case_page', return_value={}):
            with self.assertRaisesRegex(ValueError,
                                        'accepted without explicit restart'):
                fixture.require_restart(8888, self.case, 'old')
        with mock.patch.object(
                fixture,
                'case_page',
                side_effect=fixture.ApiError(400, restart_required=True)):
            self.assertEqual(
                fixture.require_restart(8888, self.case, 'old')['outcome'],
                'restart_required')

    def test_new_envelope_must_be_rejected_by_old_reader(self):
        token = 'kfp1:' + base64.b64encode(
            json.dumps({
                'OrderingVersion': 1
            }).encode()).decode()
        first = {'next_page_token': token}
        with mock.patch.object(
                fixture,
                'case_page',
                side_effect=[first, fixture.ApiError(400,
                                                     grpc_code=3)]) as page:
            result = fixture.reader_version_boundary(self.case)
        self.assertEqual(result['outcome'], 'old_reader_rejected')
        page.assert_called_with(8889, self.case, token)
        for response in ({}, {'next_page_token': 'stripped'}):
            with mock.patch.object(
                    fixture, 'case_page', side_effect=[first, response]):
                with self.assertRaisesRegex(ValueError,
                                            'accepted incompatible'):
                    fixture.reader_version_boundary(self.case)
        with mock.patch.object(
                fixture, 'case_page',
                side_effect=[first, fixture.ApiError(500)]):
            with self.assertRaises(fixture.ApiError):
                fixture.reader_version_boundary(self.case)

    def test_null_metric_desc_envelope_cannot_return_old_reader_page(self):
        self.case['sort'] = 'metric:pagination_score desc'
        token = 'kfp1:' + base64.b64encode(
            json.dumps({
                'OrderingVersion': 1,
                'SortByFieldIsNull': True,
                'SortByFieldValue': None,
                'IsDesc': True
            }).encode()).decode()
        first = {'runs': [{'run_id': 'a'}], 'next_page_token': token}
        with mock.patch.object(
                fixture,
                'case_page',
                side_effect=[first, fixture.ApiError(400, grpc_code=3)]):
            self.assertEqual(
                fixture.reader_version_boundary(self.case)['outcome'],
                'old_reader_rejected')
        # The old reader ignoring the NULL marker and silently returning page
        # one must fail, even if it presents an otherwise plausible response.
        with mock.patch.object(
                fixture, 'case_page', side_effect=[first, first]):
            with self.assertRaisesRegex(ValueError, 'accepted incompatible'):
                fixture.reader_version_boundary(self.case)

    def test_source_null_metric_failure_does_not_invent_desc_cursor(self):
        self.case.update(
            sort='metric:pagination_score desc',
            first=None,
            source_order=None,
            source_baseline={'outcome': 'failed'})
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'old_reader_boundary', return_value={'outcome': 'not_available'}), \
             mock.patch.object(fixture, 'reader_version_boundary', return_value={'outcome': 'old_reader_rejected'}):
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['saved_source_continuation']['outcome'],
                         'not_available')
        self.assertEqual(result['disposition'],
                         'explicit_restart_for_legacy_nullable_order')

    def test_selected_nullable_cases_require_restart_in_both_directions(self):
        for sort, affected in (('metric:x asc', True), ('metric:x desc', True),
                               ('scheduled_at asc',
                                True), ('finished_at asc',
                                        True), ('recurring_run_id asc', True),
                               ('display_name asc', False), ('finished_at desc',
                                                             True)):
            self.assertEqual(
                fixture.changed_nullable_order(dict(self.case, sort=sort)),
                affected)

    def test_source_matrix_has_nullable_metrics_and_both_api_versions(self):
        created = []

        def create(port, method, body, endpoint='/apis/v2beta1/experiments'):
            created.append((endpoint, body))
            if endpoint.endswith(':reportMetrics'):
                return {'results': [{'status': 'OK'}]}
            value = str(len(created))
            if endpoint.endswith('/runs'):
                return {'run': {'id': value}}
            return {'experiment_id': value}

        with mock.patch.object(fixture, 'request', side_effect=create), \
             mock.patch.object(fixture, 'full_inventory', side_effect=lambda port, case: case['created_ids']), \
             mock.patch.object(fixture, 'case_page', return_value={'next_page_token': 'actual-source'}), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}):
            cases = fixture.prepare_extended('marker')
        self.assertEqual(len(cases), 32)
        self.assertEqual({case['endpoint'] for case in cases}, {
            '/apis/v1beta1/experiments', '/apis/v2beta1/experiments',
            '/apis/v1beta1/runs', '/apis/v2beta1/runs'
        })
        self.assertEqual(sum('metric:' in case['sort'] for case in cases), 4)
        self.assertEqual(
            sum(path.endswith(':reportMetrics') for path, _ in created), 3)
        runs = [body for path, body in created if path.endswith('/runs')]
        self.assertEqual(len(runs), 7)
        self.assertEqual(runs[0]['name'].lower(), runs[1]['name'].lower())
        self.assertNotEqual(runs[0]['name'], runs[1]['name'])

    def test_source_membership_failure_is_fatal(self):
        for error in (fixture.ApiError(500), ValueError('missing fixture row')):
            with self.subTest(error=error), mock.patch.object(
                    fixture, 'full_inventory', side_effect=error):
                with self.assertRaises(type(error)):
                    fixture.capture_source_case(self.case)

    def test_source_request_errors_are_fatal(self):
        for status in (400, 401, 403, 404):
            with self.subTest(status=status), mock.patch.object(
                    fixture,
                    'full_inventory',
                    side_effect=[['a', 'b', 'c'],
                                 fixture.ApiError(status)]):
                with self.assertRaises(fixture.ApiError):
                    fixture.capture_source_case(self.case)

    def test_source_sorted_membership_mismatch_is_fatal(self):
        with mock.patch.object(
                fixture,
                'full_inventory',
                side_effect=[['a', 'b', 'c'],
                             ValueError('lost fixture row')]):
            with self.assertRaisesRegex(ValueError, 'lost fixture row'):
                fixture.capture_source_case(self.case)

    def test_unknown_source_order_is_not_a_comparison_change(self):
        self.case.update(
            source_order=None,
            first=None,
            source_baseline={
                'outcome': 'failed',
                'phase': 'sorted_inventory'
            })
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value={'outcome': 'failed'}):
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['disposition'],
                         'preexisting_source_pagination_failure')
        self.assertEqual(result['saved_source_continuation']['outcome'],
                         'not_available')

    def test_source_first_page_auth_and_availability_errors_are_fatal(self):
        for status in (401, 403, 503):
            with self.subTest(status=status), mock.patch.object(
                    fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
                 mock.patch.object(fixture, 'case_page', side_effect=fixture.ApiError(status)):
                with self.assertRaises(fixture.ApiError):
                    fixture.capture_source_case(self.case)

    def test_source_later_page_failure_preserves_real_saved_cursor(self):
        first = self.case['first']
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_page', return_value=first), \
             mock.patch.object(fixture, 'case_walk', side_effect=fixture.ApiError(400)):
            result = fixture.capture_source_case(self.case)
        self.assertEqual(result['first'], first)
        self.assertEqual(result['source_baseline']['phase'], 'traversal')

    def test_source_later_page_auth_error_is_fatal(self):
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_page', return_value=self.case['first']), \
             mock.patch.object(fixture, 'case_walk', side_effect=fixture.ApiError(403)):
            with self.assertRaises(fixture.ApiError):
                fixture.capture_source_case(self.case)

    def test_source_first_page_failure_does_not_invent_a_saved_cursor(self):
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_page', side_effect=fixture.ApiError(400)):
            result = fixture.capture_source_case(self.case)
        self.assertIsNone(result['first'])
        self.assertEqual(result['source_baseline']['phase'], 'first_page')
        self.assertEqual(result['source_order'], ['a', 'b', 'c'])

    def test_unavailable_source_cursor_is_not_reported_as_fresh_continuation(
            self):
        self.case['first'] = None
        self.case['source_baseline'] = {
            'outcome': 'failed',
            'phase': 'first_page'
        }
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value={'outcome': 'failed'}) as observe:
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['saved_source_continuation']['outcome'],
                         'not_available')
        self.assertEqual(observe.call_count, 2)

    def test_run_endpoint_and_sort_are_preserved(self):
        self.case['sort'] = 'metric:pagination_score asc'
        with mock.patch.object(fixture, 'request') as request:
            fixture.case_page(8888, self.case, 'old')
        request.assert_called_once_with(
            8888,
            'GET',
            endpoint='/apis/v2beta1/runs',
            experiment_id='experiment',
            page_size=2,
            sort_by='metric:pagination_score asc',
            page_token='old')

    def test_full_inventory_checks_membership_not_just_count(self):
        value = {'runs': [{'run_id': value} for value in ('a', 'b', 'foreign')]}
        with mock.patch.object(fixture, 'case_page', return_value=value):
            with self.assertRaisesRegex(ValueError, 'lost or added'):
                fixture.full_inventory(8888, self.case)

    def test_walk_retains_null_metric_rows_by_identity(self):
        last = {'runs': [{'run_id': 'b'}, {'run_id': 'c'}]}
        with mock.patch.object(fixture, 'case_page', return_value=last) as page:
            fixture.case_walk([8888], self.case, ['a', 'b', 'c'],
                              self.case['first'])
        page.assert_called_once_with(8888, self.case, 'old')

    def test_compatible_source_failure_is_new_regression(self):
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value={'outcome': 'failed'}):
            with self.assertRaisesRegex(ValueError,
                                        'new pagination regression'):
                fixture.validate_extended([self.case])

    def test_changed_order_requires_explicit_restart(self):
        with mock.patch.object(fixture, 'full_inventory', return_value=['c', 'b', 'a']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value={'outcome': 'failed'}):
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['disposition'],
                         'restart_after_comparison_order_change')
        self.assertEqual(result['saved_source_continuation']['outcome'],
                         'failed')

    def test_historical_failure_is_identified(self):
        self.case['source_baseline'] = {'outcome': 'failed'}
        with mock.patch.object(fixture, 'full_inventory', return_value=['a', 'b', 'c']), \
             mock.patch.object(fixture, 'case_walk', return_value={'outcome': 'passed'}), \
             mock.patch.object(fixture, 'observe_walk', return_value={'outcome': 'failed'}):
            result = fixture.validate_extended([self.case])[0]
        self.assertEqual(result['disposition'],
                         'preexisting_source_pagination_failure')

    def test_fresh_candidate_failure_cannot_be_excused_by_changed_order(self):
        with mock.patch.object(fixture, 'full_inventory', return_value=['c', 'b', 'a']), \
             mock.patch.object(fixture, 'case_walk', side_effect=ValueError('missing')):
            with self.assertRaisesRegex(ValueError, 'missing'):
                fixture.validate_extended([self.case])


if __name__ == '__main__':
    unittest.main()

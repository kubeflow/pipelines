#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
"""Real v2 experiment pagination acceptance in the disposable upgrade cluster.

The old API replica and candidate use the same database, with explicit
endpoints so every page's serving version is known. No rows mutate
during traversal.
"""
import argparse
import http.client
import itertools
import json
import os
from pathlib import Path
import urllib.parse
import uuid


class ApiError(RuntimeError):

    def __init__(self, status):
        super().__init__('experiment API HTTP ' + str(status))
        self.status = status


def request(port,
            method,
            body=None,
            endpoint='/apis/v2beta1/experiments',
            **params):
    connection = http.client.HTTPConnection('127.0.0.1', port, timeout=20)
    try:
        path = endpoint
        if params:
            path += '?' + urllib.parse.urlencode(params)
        connection.request(
            method,
            path,
            body=json.dumps(body) if body else None,
            headers={'Content-Type': 'application/json'})
        response = connection.getresponse()
        raw = response.read(1024 * 1024 + 1)
        if len(raw) > 1024 * 1024:
            raise ValueError('response limit exceeded')
        if response.status != 200:
            raise ApiError(response.status)
        return json.loads(raw)
    finally:
        connection.close()


def criteria(marker, operation='EQUALS'):
    value = {
        'string_values': {
            'values': [marker]
        }
    } if operation == 'IN' else {
        'string_value': marker
    }
    return json.dumps({
        'predicates': [{
            'key': 'description',
            'operation': operation,
            **value
        }]
    })


def page(port, spec, token='', repeat=True):
    params = {'namespace': 'kubeflow', 'page_size': 2}
    if not token or repeat:
        params.update(filter=spec, sort_by='display_name')
    if token:
        params['page_token'] = token
    return request(port, 'GET', **params)


def ids(value):
    rows = value.get('experiments', [])
    values = [row['experiment_id'] for row in rows]
    if any(not isinstance(value, str) or not value for value in values):
        raise ValueError('missing experiment identity')
    return values


def walk(ports, spec, expected, first=None, repeat=True):
    """Bound traversals; fail on missing/extra/duplicate IDs or token
    cycles."""
    result, tokens, trace = [], set(), []
    token = ''
    for index, port in enumerate(itertools.cycle(ports)):
        if index >= 20:
            raise ValueError('page budget exceeded')
        value = first if index == 0 and first is not None else page(
            port, spec, token, repeat)
        result.extend(ids(value))
        trace.append(
            'saved_source_page' if index == 0 and first is not None else port)
        token = value.get('next_page_token', '')
        if not token:
            break
        if token in tokens:
            raise ValueError('repeated page token')
        tokens.add(token)
    if len(set(result)) != len(result):
        raise ValueError('duplicate experiment across pages')
    if result != expected:
        raise ValueError('missing, extra, or out-of-order experiments')
    return {
        'outcome': 'passed',
        'rows': len(result),
        'pages': len(trace),
        'ports': trace
    }


def source(state_path, source_image):
    if not source_image.endswith(':2.17.2'):
        raise ValueError('source API image is not pinned to 2.17.2')
    marker = 'pagination-' + uuid.uuid4().hex
    expected = []
    for index in range(7):
        row = request(
            8888, 'POST', {
                'display_name': marker + '-' + str(index),
                'description': marker,
                'namespace': 'kubeflow'
            })
        expected.append(row['experiment_id'])
    cases = []
    for operation in ('EQUALS', 'IS_SUBSTRING', 'IN'):
        spec = criteria(marker, operation)
        first = page(8888, spec)
        if not first.get('next_page_token'):
            raise ValueError('source did not generate a pagination token')
        # 2.17.2 cannot compare a repeated IN criterion after token JSON decode.
        baseline = walk([8888], spec, expected, first, repeat=operation != 'IN')
        cases.append({
            'operation': operation,
            'filter': spec,
            'first': first,
            'source_baseline': baseline
        })
    state_path.write_text(
        json.dumps({
            'source_version': '2.17.2',
            'source_image': source_image,
            'expected': expected,
            'cases': cases,
            'extended': prepare_extended(marker)
        }))


def target(state_path, report_path):
    state = json.loads(state_path.read_text())
    results = []
    for case in state['cases']:
        spec, expected = case['filter'], state['expected']
        # Continue the exact page saved before deployment, including repeat-IN
        # on the fixed candidate and token-only requests on both readers.
        for repeat in (False, True):
            outcome = walk([8888], spec, expected, case['first'], repeat=repeat)
            results.append({
                'case': case['operation'],
                'mode': 'upgrade',
                'repeat_filter': repeat,
                **outcome
            })
        for ports in ([8889, 8888], [8888, 8889]):
            repeats = (False,) if case['operation'] == 'IN' else (False, True)
            for repeat in repeats:
                outcome = walk(ports, spec, expected, repeat=repeat)
                results.append({
                    'case': case['operation'],
                    'mode': 'mixed',
                    'repeat_filter': repeat,
                    **outcome
                })
        # Repeated changed criteria must never be silently accepted.
        for port in (8888, 8889):
            try:
                page(port, criteria('unrelated'),
                     case['first']['next_page_token'])
            except ApiError as error:
                if error.status != 400:
                    raise
            else:
                raise ValueError('changed filter unexpectedly accepted')
        if case['operation'] == 'IN':
            for token_port in (8888, 8889):
                token = page(token_port, spec)['next_page_token']
                try:
                    page(8889, spec, token)
                except ApiError as error:
                    if error.status != 400:
                        raise
                else:
                    raise ValueError(
                        'historical repeated IN limitation changed')
    report_path.write_text(
        json.dumps(
            {
                'outcome':
                    'passed',
                'scope':
                    'v1_v2_experiments_runs_mysql_stable_data',
                'source_version':
                    '2.17.2',
                'source_image':
                    state['source_image'],
                'candidate_revision':
                    os.environ['GITHUB_SHA'],
                'candidate_port':
                    8888,
                'source_port':
                    8889,
                'results':
                    results,
                'extended':
                    validate_extended(state['extended']),
                'limitations': [
                    'old reader repeated IN rejects with HTTP 400 (old and new tokens)',
                    'pipeline, pipeline-version, task and recurring-run list endpoints are not covered; MySQL only'
                ]
            },
            indent=2) + '\n')


def case_page(port, case, token='', size=2):
    params = {**case['params'], 'page_size': size, 'sort_by': case['sort']}
    if token:
        params['page_token'] = token
    return request(port, 'GET', endpoint=case['endpoint'], **params)


def case_ids(value, case):
    return [row[case['identity']] for row in value.get(case['collection'], [])]


def case_walk(ports, case, expected, first=None):
    result, tokens = [], set()
    token = ''
    for index, port in enumerate(itertools.cycle(ports)):
        if index == 32:
            raise ValueError('page budget exceeded')
        value = first if index == 0 and first is not None else case_page(
            port, case, token)
        result.extend(case_ids(value, case))
        token = value.get('next_page_token', '')
        if not token:
            break
        if token in tokens:
            raise ValueError('repeated page token')
        tokens.add(token)
    if result != expected or len(set(result)) != len(result):
        raise ValueError('missing, duplicate, extra, or out-of-order rows')
    return {'outcome': 'passed', 'rows': len(result)}


def observe_walk(ports, case, expected, first=None):
    # Store classifications only: tokens/API bodies stay in private fixture state.
    try:
        return case_walk(ports, case, expected, first)
    except (ApiError, ValueError) as error:
        return {'outcome': 'failed', 'reason': str(error)}


def full_inventory(port, case):
    value = case_page(port, case, size=64)
    values = case_ids(value, case)
    if value.get('next_page_token') or len(values) != len(set(values)):
        raise ValueError('independent inventory is incomplete or duplicated')
    if set(values) != set(case['created_ids']):
        raise ValueError('independent inventory lost or added fixture rows')
    return values


def prepare_extended(marker):
    """Capture actual old cursors, with one-page inventory as an independent
    oracle."""
    experiment_ids, run_ids = [], []
    suffixes = ('alpha', 'Bravo', 'CHARLIE', 'delta', 'Echo', 'foxtrot', 'GOLF')
    for suffix in suffixes:
        row = request(
            8888, 'POST', {
                'display_name': marker + '-mixed-' + suffix,
                'description': marker + '-mixed',
                'namespace': 'kubeflow'
            })
        experiment_ids.append(row['experiment_id'])
    # A suspended V1 workflow keeps mutable run state out of the sort oracle and
    # needs no task image. The disposable cluster is the fixture's cleanup unit.
    workflow = {
        'apiVersion': 'argoproj.io/v1alpha1',
        'kind': 'Workflow',
        'metadata': {
            'generateName': 'pagination-',
            'labels': {
                'pipelines.kubeflow.org/pagination-fixture': 'true'
            }
        },
        'spec': {
            'entrypoint': 'hold',
            'templates': [{
                'name': 'hold',
                'suspend': {}
            }]
        }
    }
    for index, suffix in enumerate(suffixes):
        row = request(
            8888,
            'POST', {
                'name':
                    marker + '-' + ('ALPHA' if index == 1 else suffix),
                'description':
                    marker,
                'pipeline_spec': {
                    'workflow_manifest': json.dumps(workflow)
                },
                'resource_references': [{
                    'key': {
                        'type': 'EXPERIMENT',
                        'id': experiment_ids[0]
                    },
                    'relationship': 'OWNER'
                }]
            },
            endpoint='/apis/v1beta1/runs')
        run_id = row['run']['id']
        run_ids.append(run_id)
        if index < 3:
            reported = request(
                8888,
                'POST', {
                    'metrics': [{
                        'name': 'pagination_score',
                        'node_id': 'fixture',
                        'number_value': index,
                        'format': 'RAW'
                    }]
                },
                endpoint='/apis/v1beta1/runs/' + run_id + ':reportMetrics')
            if any(
                    result.get('status') != 'OK' for result in reported.get(
                        'results', [])) or not reported.get('results'):
                raise ValueError('source metric preparation failed')
    cases = []
    for version in ('v1beta1', 'v2beta1'):
        v1 = version == 'v1beta1'
        for collection, created_ids, params, sorts in (
            ('experiments', experiment_ids, {
                **({
                    'resource_reference_key.type': 'NAMESPACE',
                    'resource_reference_key.id': 'kubeflow'
                } if v1 else {
                       'namespace': 'kubeflow'
                   }), 'filter':
                    criteria(marker + '-mixed')
            }, ['name' if v1 else 'display_name', 'description',
                'created_at']), ('runs', run_ids, {
                    'resource_reference_key.type': 'EXPERIMENT',
                    'resource_reference_key.id': experiment_ids[0]
                } if v1 else {
                    'namespace': 'kubeflow',
                    'experiment_id': experiment_ids[0]
                }, [
                    'name' if v1 else 'display_name', 'metric:pagination_score',
                    'recurring_run_id', 'scheduled_at', 'finished_at'
                ])):
            for sort in sorts:
                for direction in ('asc', 'desc'):
                    case = {
                        'endpoint': '/apis/' + version + '/' + collection,
                        'collection': collection,
                        'identity': 'id' if v1 else collection[:-1] + '_id',
                        'params': params,
                        'sort': sort + ' ' + direction,
                        'created_ids': created_ids
                    }
                    expected = full_inventory(8888, case)
                    first = case_page(8888, case)
                    if not first.get('next_page_token'):
                        raise ValueError(
                            'source did not generate an extended cursor')
                    case.update(
                        source_order=expected,
                        first=first,
                        source_baseline=observe_walk([8888], case, expected))
                    cases.append(case)
    return cases


def validate_extended(cases):
    results = []
    for case in cases:
        expected = full_inventory(8888, case)
        # A changed old cursor can require restarting; a broken fresh candidate
        # traversal is never accepted as a historical compatibility exception.
        fresh = case_walk([8888], case, expected)
        continuation = observe_walk([8888], case, expected, case['first'])
        source_good = case['source_baseline']['outcome'] == 'passed'
        changed_order = case['source_order'] != expected
        mixed = [
            observe_walk(ports, case, expected)
            for ports in ([8889, 8888], [8888, 8889])
        ]
        if source_good and not changed_order:
            if continuation['outcome'] != 'passed' or any(
                    outcome['outcome'] != 'passed' for outcome in mixed):
                raise ValueError('new pagination regression: ' +
                                 case['endpoint'] + ' ' + case['sort'])
        results.append({
            'endpoint':
                case['endpoint'],
            'sort':
                case['sort'],
            'source_baseline':
                case['source_baseline'],
            'fresh_candidate':
                fresh,
            'saved_source_continuation':
                continuation,
            'mixed_readers':
                mixed,
            'disposition':
                ('restart_after_comparison_order_change'
                 if changed_order else 'preexisting_source_pagination_failure'
                 if not source_good else 'compatible')
        })
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('phase', choices=('source', 'target'))
    parser.add_argument('--state', required=True, type=Path)
    parser.add_argument('--report', type=Path)
    parser.add_argument('--source-image')
    args = parser.parse_args()
    if args.phase == 'source':
        if not args.source_image:
            parser.error('--source-image is required for source')
        source(args.state, args.source_image)
    else:
        if not args.report:
            parser.error('--report is required for target')
        args.report.write_text(
            json.dumps({
                'outcome': 'inconclusive',
                'candidate_revision': os.environ.get('GITHUB_SHA'),
                'reason': 'validation_not_completed'
            }) + '\n')
        target(args.state, args.report)


if __name__ == '__main__':
    main()

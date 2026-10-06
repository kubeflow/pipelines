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


def request(port, method, body=None, **params):
    connection = http.client.HTTPConnection('127.0.0.1', port, timeout=20)
    try:
        path = '/apis/v2beta1/experiments'
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
            'cases': cases
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
                    'v2_experiments_mysql_stable_lowercase_data',
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
                'limitations': [
                    'old reader repeated IN rejects with HTTP 400 (old and new tokens)',
                    'numeric repeated filters, changed case/collation semantics, nullable sort cursors, and other resource endpoints are not covered'
                ]
            },
            indent=2) + '\n')


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

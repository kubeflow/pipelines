# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Compare serving KFP APIs with the pinned 2.18 migration contract."""

import argparse
import json
from pathlib import Path
from urllib.parse import urlencode
from urllib.request import Request, urlopen


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('endpoint')
    parser.add_argument('fixtures', type=Path)
    parser.add_argument('--legacy', action='store_true')
    args = parser.parse_args()
    prefix = '/apis/v2beta1' if args.legacy else '/apis/v2'

    def request(path, params=(), body=None):
        query = '?' + urlencode(params, doseq=True) if params else ''
        data = json.dumps(body).encode() if body is not None else None
        request = Request(
            f'{args.endpoint}{prefix}{path}{query}',
            data=data,
            headers={'Content-Type': 'application/json'},
        )
        with urlopen(request, timeout=30) as response:
            return json.load(response)

    captured = json.loads((args.fixtures / 'api_runs.json').read_text())
    expected = json.loads(
        (args.fixtures / 'migration_fixture.json').read_text()
    )['expected']
    captured_by_id = {run['run_id']: run for run in captured.values()}
    listed = request('/runs', {'page_size': 100})['runs']
    assert {run['run_id'] for run in listed} == set(captured_by_id)
    if args.legacy:
        for legacy in captured.values():
            run = request(f'/runs/{legacy["run_id"]}')
            assert run['state'] == legacy['state'], run['run_id']
            actual = run['run_details']['task_details']
            original = legacy['run_details']['task_details']
            assert {task['task_id'] for task in actual} == {
                task['task_id'] for task in original
            }, run['run_id']
        print('Verified released run and task history')
        return

    for run in expected['api']:
        run_id = run['run_id']
        details = request(f'/runs/{run_id}', {'view': 'FULL'})
        assert details['state'] == captured_by_id[run_id]['state'], run_id
        assert {task['task_id'] for task in details.get('tasks', [])} == set(
            run['task_ids']
        ), run_id
        tasks = request(f'/runs/{run_id}/tasks', {'page_size': 100}).get(
            'tasks', []
        )
        assert {task['task_id'] for task in tasks} == set(run['task_ids']), (
            run_id
        )

    for task in expected['tasks']:
        fields = task['fields']
        run_id, task_id = fields['RunUUID'], task['target_uuid']
        result = request(f'/runs/{run_id}/tasks/{task_id}')
        assert result['task_id'] == task_id, task_id
        assert (result.get('parent_task_id') or None) == fields[
            'ParentTaskUUID'
        ], task_id
        assert (result.get('cached_task_id') or None) == fields.get(
            'CachedTaskUUID'
        ), task_id

    artifacts = request(
        '/artifacts', {'namespace': 'kubeflow', 'page_size': 100}
    )['artifacts']
    assert {artifact['artifact_id'] for artifact in artifacts} == {
        artifact['target_uuid'] for artifact in expected['artifacts']
    }
    for artifact in expected['artifacts']:
        artifact_id = artifact['target_uuid']
        result = request(f'/artifacts/{artifact_id}')
        assert result['artifact_id'] == artifact_id, artifact_id
        assert result['name'] == artifact['fields']['Name'], artifact_id

    links = request(
        '/artifact_tasks',
        {
            'run_ids': [run['run_id'] for run in expected['api']],
            'page_size': 100,
        },
    )['artifact_tasks']
    assert {
        (link['id'], link['artifact_id'], link['task_id'], link['key'])
        for link in links
    } == {
        (
            link['target_uuid'],
            link['fields']['ArtifactID'],
            link['fields']['TaskID'],
            link['fields']['ArtifactKey'],
        )
        for link in expected['artifact_tasks']
    }
    fingerprint = (
        (args.fixtures / 'produce_native_fingerprint.txt').read_text().strip()
    )
    response = request(
        '/tasks:findCached',
        body={'namespace': 'kubeflow', 'cache_fingerprint': fingerprint},
    )
    producer = next(
        task['target_uuid']
        for task in expected['tasks']
        if task['source'] == 'mlmd.executions[id=2]'
    )
    assert response['task']['task_id'] == producer, response
    print('Verified native run, task, artifact, lineage, and cache reads')


if __name__ == '__main__':
    main()

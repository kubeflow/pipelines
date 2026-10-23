#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Bounded HTTP qualification against disposable, independently deployed
stores."""

import argparse
import hashlib
import json
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime
from pathlib import Path

API = 'http://127.0.0.1:8888/apis/v2beta1/'
# Single-user APIs store experiments/catalog in the empty logical namespace;
# the independent Kubernetes runtime namespaces are both kubeflow.
NAMESPACE = ''
TEXT = 'namespace transfer preserves parameters'


def request(path, body=None, *, raw=False, **query):
    url = API + path + '?' + urllib.parse.urlencode(query)
    data = (
        body
        if isinstance(body, bytes)
        else (None if body is None else json.dumps(body).encode())
    )
    req = urllib.request.Request(
        url, data=data, headers={'Content-Type': 'application/json'}
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as response:
            return response.read() if raw else json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read(16384).decode('utf-8', errors='replace')
        raise RuntimeError(
            f'{req.get_method()} {path}: HTTP {error.code}: {detail}'
        ) from error


def records(collection):
    result, token = [], ''
    for _ in range(100):
        page = request(
            collection, namespace=NAMESPACE, page_size=100, page_token=token
        )
        result.extend(
            page.get(
                'recurringRuns'
                if collection == 'recurringruns'
                else collection,
                [],
            )
        )
        token = page.get('next_page_token', '')
        if not token:
            return result
    raise AssertionError(f'Unbounded pagination for {collection}')


def wait_run(run_id):
    deadline = time.monotonic() + 600
    run = None
    while time.monotonic() < deadline:
        run = request(f'runs/{run_id}', view='FULL')
        state = run.get('state')
        if run.get('error'):
            raise AssertionError(
                f'Fixture run {run_id} API error: {json.dumps(run)}'
            )
        if state == 'SUCCEEDED':
            return run
        if state in ('FAILED', 'CANCELED'):
            raise AssertionError(
                f'Fixture run {run_id} ended {state}: {json.dumps(run)}'
            )
        time.sleep(5)
    raise AssertionError(
        f'Fixture run {run_id} did not complete in 600s: {json.dumps(run)}'
    )


def artifact_uris(value):
    if isinstance(value, dict):
        return (
            {value['uri']} if isinstance(value.get('uri'), str) else set()
        ).union(*(artifact_uris(item) for item in value.values()))
    if isinstance(value, list):
        return set().union(*(artifact_uris(item) for item in value))
    return set()


def wait_export(run_id):
    deadline = time.monotonic() + 180
    while time.monotonic() < deadline:
        wire = request('transfer/export', {}, raw=True, namespace=NAMESPACE)
        archive = json.loads(wire)
        assert archive['format'] == 'kfp-namespace-transfer-mlmd-2.18/v2'
        graph = archive['metadata']
        if (
            any(item['run']['UUID'] == run_id for item in archive['runs'])
            and len(graph.get('executions', [])) >= 2
            and graph.get('artifacts')
            and graph.get('events')
        ):
            return wire, archive
        time.sleep(5)
    raise AssertionError(
        'Export did not include completed fixture history and MLMD graph in 180s'
    )


def source(directory):
    experiment = request(
        'experiments',
        {'display_name': 'transfer-source', 'namespace': NAMESPACE},
    )
    fixture = 'test_data/sdk_compiled_pipelines/valid/critical/two_step_pipeline_containerized.yaml'
    pipeline = json.loads(
        subprocess.check_output(
            [
                'curl',
                '--fail-with-body',
                '--silent',
                '--show-error',
                '--max-time',
                '120',
                '-F',
                f'uploadfile=@{fixture}',
                API
                + 'pipelines/upload?name=transfer-source&namespace='
                + NAMESPACE,
            ],
            text=True,
        )
    )
    pipeline_id = pipeline['pipeline_id']
    versions = request(f'pipelines/{pipeline_id}/versions', page_size=100)[
        'pipeline_versions'
    ]
    assert len(versions) == 1
    version = versions[0]
    reference = {
        'pipeline_id': pipeline_id,
        'pipeline_version_id': version['pipeline_version_id'],
    }
    base = {
        'experiment_id': experiment['experiment_id'],
        'namespace': NAMESPACE,
        'pipeline_version_reference': reference,
        'runtime_config': {'parameters': {'text': TEXT}},
    }
    run = request('runs', dict(base, display_name='transfer-completed'))
    complete = wait_run(run['run_id'])
    request(
        'recurringruns',
        dict(
            base,
            display_name='transfer-disabled',
            mode='DISABLE',
            max_concurrency='1',
            no_catchup=True,
            trigger={'periodic_schedule': {'interval_second': '3600'}},
        ),
    )
    wire, archive = wait_export(run['run_id'])
    assert (
        archive['runs']
        and archive['schedules']
        and archive['pipeline_versions']
    )
    archived_artifacts = archive['metadata']['artifacts']
    # Legacy GetRun reads SQL and does not hydrate the MLMD task graph.
    # Verify persisted execution/artifact metadata through the exporter.
    assert len(archive['metadata']['executions']) >= 2
    uris = artifact_uris(archived_artifacts)
    assert uris, 'Export must preserve produced and consumed artifact URIs'
    (directory / 'archive.json').write_bytes(wire)
    followup = request('runs', dict(base, display_name='source-after-export'))
    wait_run(followup['run_id'])
    evidence = {
        'archive_sha256': hashlib.sha256(wire).hexdigest(),
        'run': complete,
        'source_runs': records('runs'),
        'pipeline_id': pipeline_id,
        'version_id': version['pipeline_version_id'],
        'artifact_uris': sorted(uris),
        'source_after_export_run_id': followup['run_id'],
    }
    (directory / 'source-evidence.json').write_text(
        json.dumps(evidence, indent=2)
    )


def verify_root_completion(run):
    roots = [task for task in run['tasks'] if task['type'] == 'ROOT']
    assert len(roots) == 1 and roots[0]['end_time'] == run['finished_at']
    root_end = datetime.fromisoformat(
        roots[0]['end_time'].replace('Z', '+00:00')
    )
    assert all(
        datetime.fromisoformat(task['end_time'].replace('Z', '+00:00'))
        <= root_end
        for task in run['tasks']
        if task.get('end_time')
    )


def destination(directory):
    wire = (directory / 'archive.json').read_bytes()
    evidence = json.loads((directory / 'source-evidence.json').read_text())
    assert hashlib.sha256(wire).hexdigest() == evidence['archive_sha256']
    retained = request(
        'experiments',
        {'display_name': 'destination-existing', 'namespace': NAMESPACE},
    )
    collections = ('experiments', 'pipelines', 'runs', 'recurringruns')
    before = {kind: records(kind) for kind in collections}
    preview = request(
        'transfer/import',
        wire,
        namespace=NAMESPACE,
        name_prefix='legacy-',
        dry_run='true',
    )
    assert preview['dry_run'] and preview['counts']['runs'] >= 1
    assert before == {kind: records(kind) for kind in collections}, (
        'Preview published resources'
    )
    applied = request(
        'transfer/import',
        wire,
        namespace=NAMESPACE,
        name_prefix='legacy-',
        dry_run='false',
    )
    assert not applied['dry_run'] and applied['imported'] > 0
    after = {kind: records(kind) for kind in collections}
    imported = [
        r for r in after['runs'] if r['display_name'] == 'transfer-completed'
    ]
    assert len(imported) == 1
    run = request(f'runs/{imported[0]["run_id"]}', view='FULL')
    assert run['state'] == 'SUCCEEDED'
    assert run['runtime_config']['parameters']['text'] == TEXT
    assert len(run['tasks']) >= 2
    verify_root_completion(run)
    assert set(evidence['artifact_uris']) <= artifact_uris(run), (
        'Artifact references changed during import'
    )
    pipelines = [
        p for p in after['pipelines'] if p['name'] == 'legacy-transfer-source'
    ]
    assert len(pipelines) == 1
    versions = request(
        f'pipelines/{pipelines[0]["pipeline_id"]}/versions', page_size=100
    )['pipeline_versions']
    assert len(versions) == 1
    definition = request(
        f'pipelines/{pipelines[0]["pipeline_id"]}/versions/{versions[0]["pipeline_version_id"]}'
    )
    assert definition['pipeline_spec']
    schedules = [
        s
        for s in after['recurringruns']
        if s['display_name'] == 'legacy-transfer-disabled'
    ]
    assert (
        len(schedules) == 1
        and schedules[0]['status'] == 'DISABLED'
        and schedules[0]['no_catchup']
    )
    assert schedules[0]['runtime_config']['parameters']['text'] == TEXT
    assert request(f'experiments/{retained["experiment_id"]}') == retained
    repeated = request(
        'transfer/import',
        wire,
        namespace=NAMESPACE,
        name_prefix='legacy-',
        dry_run='false',
    )
    assert repeated['imported'] == 0 and repeated['skipped'] > 0
    assert after == {kind: records(kind) for kind in collections}, (
        'Repeat import changed existing resources'
    )
    (directory / 'destination-evidence.json').write_text(
        json.dumps(
            {
                'preview': preview,
                'applied': applied,
                'repeated': repeated,
                'imported_run': run,
                'imported_schedule': schedules[0],
            },
            indent=2,
        )
    )


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('source', 'destination'))
    parser.add_argument(
        '--evidence', type=Path, default=Path('transfer-evidence')
    )
    args = parser.parse_args()
    args.evidence.mkdir(parents=True, exist_ok=True)
    {'source': source, 'destination': destination}[args.mode](args.evidence)

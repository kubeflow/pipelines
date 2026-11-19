# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Verify two completed demo runs through Argo and the disposable kind
cluster."""

import argparse
import json
import math
import subprocess
from urllib.parse import quote


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('first_run')
    parser.add_argument('second_run')
    args = parser.parse_args()

    def argo(path):
        # The demo port-forward uses Argo's self-signed quick-start certificate.
        return json.loads(
            subprocess.check_output(
                [
                    'curl',
                    '--fail',
                    '--silent',
                    '--show-error',
                    '--insecure',
                    'https://localhost:2746/' + path,
                ],
                text=True,
            )
        )

    runs = [
        argo('api/v1/workflows/argo/' + quote(name, safe=''))
        for name in (args.first_run, args.second_run)
    ]
    summaries = []
    for index, run in enumerate(runs):
        assert run['status']['phase'] == 'Succeeded', run['status']
        nodes = {
            node['displayName']: node
            for node in run['status']['nodes'].values()
            if node['type'] == 'Pod'
        }
        assert set(nodes) == {'prepare', 'train', 'evaluate'}
        assert nodes['prepare']['memoizationStatus']['hit'] is bool(index)
        metadata = {}
        for name, node in nodes.items():
            metadata[name] = argo(
                'artifacts/argo/'
                + run['metadata']['name']
                + '/'
                + node['id']
                + '/kfp-metadata'
            )
            for artifact in metadata[name]['artifacts']:
                stored = next(
                    a
                    for a in node['outputs']['artifacts']
                    if a['name'] == artifact['port']
                )
                repository = run['status']['artifactRepositoryRef'][
                    'artifactRepository'
                ]['s3']
                bucket = stored['s3'].get('bucket', repository['bucket'])
                assert (
                    artifact['uri']
                    == 's3://' + bucket + '/' + stored['s3']['key']
                )
        expected = (2 / 3, 1.0)[index]
        assert math.isclose(metadata['evaluate']['metrics']['score'], expected)
        assert math.isclose(
            float(nodes['evaluate']['outputs']['parameters'][0]['value']),
            expected,
        )
        assert (
            metadata['train']['inputs'][0]['producer']
            == metadata['prepare']['producer']
        )
        pods = json.loads(
            subprocess.check_output(
                [
                    'kubectl',
                    '--context',
                    'kind-kfp-argo',
                    '-n',
                    'argo',
                    'get',
                    'pods',
                    '-l',
                    'workflows.argoproj.io/workflow=' + run['metadata']['name'],
                    '-o',
                    'json',
                ],
                text=True,
            )
        )
        assert len(pods['items']) == (3, 2)[index], (
            'Unexpected task or driver pod'
        )
        assert all(
            [c['name'] for c in p['spec']['containers']] == ['wait', 'main']
            for p in pods['items']
        )
        summaries.append(metadata)
        print(
            f'{run["metadata"]["name"]}: score={expected}, pods={len(pods["items"])}, prepare cached={bool(index)}'
        )
    assert summaries[0]['prepare'] == summaries[1]['prepare'], (
        'Cache changed producer identity'
    )
    deployments = json.loads(
        subprocess.check_output(
            [
                'kubectl',
                '--context',
                'kind-kfp-argo',
                '-n',
                'argo',
                'get',
                'deployments',
                '-o',
                'json',
            ],
            text=True,
        )
    )
    assert {d['metadata']['name'] for d in deployments['items']} == {
        'argo-server',
        'workflow-controller',
        'minio',
        'httpbin',
    }
    print(
        'PASS: native Argo execution, storage identities, metrics, cache lineage; no KFP backend.'
    )


if __name__ == '__main__':
    main()

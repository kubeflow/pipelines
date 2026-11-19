# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Compile, lint, execute and assert the native compatibility corpus on
Kubernetes."""

import argparse
import importlib.util
import json
import subprocess
import tempfile
from pathlib import Path

from kfp import compiler


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--context', required=True)
    parser.add_argument('--namespace', default='argo')
    parser.add_argument(
        '--image-digest',
        help='Verified Docker identity; create its content-named tag with compile_demo.py first.',
    )
    args = parser.parse_args()
    spec = importlib.util.spec_from_file_location(
        'corpus', Path(__file__).with_name('compatibility.py')
    )
    corpus = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(corpus)
    argo = ['argo', '--context', args.context, '-n', args.namespace]
    cases = [
        (corpus.control_flow, {'value': 2}, 'Succeeded', '6'),
        (corpus.control_flow, {'value': -1}, 'Succeeded', '6'),
        (corpus.control_flow, {'fail': True}, 'Failed', None),
        (corpus.exit_flow, {'fail': False}, 'Succeeded', 'SUCCEEDED'),
        (corpus.exit_flow, {'fail': True}, 'Failed', 'FAILED'),
        (corpus.exit_flow, {'fail_cleanup': True}, 'Failed', None),
        (corpus.container_artifacts, {}, 'Succeeded', '6'),
        (corpus.large_metadata, {}, 'Succeeded', '100000'),
        (corpus.retry_failure, {}, 'Failed', None),
    ]
    with tempfile.TemporaryDirectory(prefix='kfp-argo-corpus-') as directory:
        for index, (pipeline, parameters, phase, expected) in enumerate(cases):
            path = str(Path(directory) / f'{index}.yaml')
            compiler.Compiler(
                image_digests={'kfp-argo-demo:dev': args.image_digest}
                if args.image_digest
                else None
            ).compile(
                pipeline,
                path,
                pipeline_parameters=parameters,
                resource_kind='Workflow',
            )
            subprocess.run(['argo', 'lint', '--offline', path], check=True)
            name = subprocess.check_output(
                argo + ['submit', path, '-o', 'name'], text=True
            ).strip()
            # Deliberate failures are assertions, not tests to retry into success.
            result = subprocess.run(argo + ['wait', name], timeout=300)
            assert result.returncode == (0 if phase == 'Succeeded' else 1), name
            run = json.loads(
                subprocess.check_output(
                    argo + ['get', name, '-o', 'json'], text=True
                )
            )
            assert run['status']['phase'] == phase, run['status']
            nodes = list(run['status']['nodes'].values())
            if pipeline is corpus.exit_flow:
                cleanup = next(
                    node for node in nodes if node['displayName'] == 'exit'
                )
                assert cleanup['phase'] == (
                    'Succeeded' if expected else 'Failed'
                ), cleanup
                if expected:
                    assert (
                        cleanup['outputs']['parameters'][0]['value'] == expected
                    ), cleanup
            elif expected is not None:
                root = run['status']['nodes'][name]
                assert root['outputs']['parameters'][0]['value'] == expected, (
                    root
                )
            if pipeline is corpus.retry_failure:
                attempts = [node for node in nodes if node['type'] == 'Pod']
                assert len(attempts) == 2 and all(
                    node['phase'] == 'Failed' for node in attempts
                ), attempts
                pods = json.loads(
                    subprocess.check_output(
                        [
                            'kubectl',
                            '--context',
                            args.context,
                            '-n',
                            args.namespace,
                            'get',
                            'pods',
                            '-l',
                            'workflows.argoproj.io/workflow=' + name,
                            '-o',
                            'json',
                        ],
                        text=True,
                    )
                )
                assert len(pods['items']) == 2
                for pod in pods['items']:
                    main = next(
                        container
                        for container in pod['spec']['containers']
                        if container['name'] == 'main'
                    )
                    assert main['resources']['requests']['cpu'] == '50m'
                    assert main['resources']['limits']['memory'] == '64Mi'
            if pipeline is corpus.large_metadata:
                identities = [
                    parameter
                    for node in nodes
                    for parameter in node.get('outputs', {}).get(
                        'parameters', []
                    )
                    if parameter['name'].startswith('kfp-meta-')
                ]
                assert identities and all(
                    len(parameter['value']) < 1000 for parameter in identities
                )
            if pipeline is corpus.control_flow:
                skipped = [node for node in nodes if node['phase'] == 'Skipped']
                assert len(skipped) == 1, nodes
                total = next(
                    node for node in nodes if node['displayName'] == 'total'
                )
                assert total['phase'] == (
                    'Succeeded' if expected else 'Omitted'
                ), total
                if parameters.get('value', 2) > 0:
                    assert skipped[0]['displayName'] == 'condition-3', skipped
                else:
                    assert skipped[0]['displayName'] == 'condition-2', skipped
            print(
                f'PASS {name}: {phase}, expected output={expected}', flush=True
            )


if __name__ == '__main__':
    main()

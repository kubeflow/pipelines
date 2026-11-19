# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Compile the demo against a content-named local component image snapshot."""

import argparse
import importlib.util
import subprocess
from pathlib import Path

from kfp import compiler
from kfp.compiler.argo_compiler import local_image_reference


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True)
    parser.add_argument(
        '--kind',
        choices=['WorkflowTemplate', 'Workflow', 'CronWorkflow'],
        default='WorkflowTemplate',
    )
    parser.add_argument('--schedule')
    parser.add_argument('--namespace', default='kfp-argo')
    args = parser.parse_args()
    image = 'kfp-argo-demo:dev'
    digest = subprocess.check_output(
        ['docker', 'image', 'inspect', image, '--format', '{{.Id}}'], text=True
    ).strip()
    snapshot = local_image_reference(image, digest)
    existing = subprocess.run(
        ['docker', 'image', 'inspect', snapshot, '--format', '{{.Id}}'],
        capture_output=True,
        text=True,
    )
    if existing.returncode == 0 and existing.stdout.strip() != digest:
        raise RuntimeError(
            f'Refusing to overwrite image snapshot {snapshot}; its content does not match its name.'
        )
    subprocess.run(['docker', 'tag', image, snapshot], check=True)
    source = Path(__file__).with_name('components.py')
    spec = importlib.util.spec_from_file_location('native_demo', source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    compiler.Compiler(image_digests={image: digest}).compile(
        module.pipeline,
        args.output,
        resource_kind=args.kind,
        schedule=args.schedule,
        namespace=args.namespace,
    )
    print(f'Compiled {args.output} using {snapshot}')


if __name__ == '__main__':
    main()

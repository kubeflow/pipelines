#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Regenerate authoring/config schemas, without KFP RPC or Go generators."""

import importlib.util
import subprocess
import sys
import tempfile
from pathlib import Path

import grpc_tools

ROOT = Path(__file__).resolve().parents[1]


def main():
    includes = [
        ROOT / 'api/v2alpha1',
        ROOT / 'kubernetes_platform/proto',
        Path(grpc_tools.__file__).parent / '_proto',
    ]
    status = importlib.util.find_spec('google.rpc.status_pb2')
    includes.append(Path(status.origin).parents[2])
    with tempfile.TemporaryDirectory(prefix='kfp-schema-') as directory:
        subprocess.run(
            [
                sys.executable,
                '-m',
                'grpc_tools.protoc',
                *['-I' + str(path) for path in includes],
                '--python_out=' + directory,
                str(ROOT / 'api/v2alpha1/pipeline_spec.proto'),
                str(
                    ROOT
                    / 'kubernetes_platform/proto/kubernetes_executor_config.proto'
                ),
            ],
            check=True,
        )
        for name, target in [
            ('pipeline_spec_pb2.py', 'pipeline_spec'),
            ('kubernetes_executor_config_pb2.py', 'kubernetes'),
        ]:
            source = (Path(directory) / name).read_text()
            source = source.replace(
                'import pipeline_spec_pb2 as ',
                'from kfp.pipeline_spec import pipeline_spec_pb2 as ',
            )
            (ROOT / 'sdk/python/kfp' / target / name).write_text(source)


if __name__ == '__main__':
    main()

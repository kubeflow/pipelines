# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify installed SDK compilation and CLI imports."""

import importlib.metadata
from pathlib import Path
import subprocess
import sys
import tempfile

from kfp import compiler
from kfp import dsl
from kfp import kubernetes
from packaging.version import Version


@dsl.container_component
def echo():
    return dsl.ContainerSpec(
        image='busybox:1.36', command=['echo', 'sdk-smoke'])


@dsl.pipeline(name='sdk-python-compatibility')
def pipeline():
    kubernetes.add_pod_label(echo(), 'sdk-smoke', 'true')


def verify_versions(expected_version):
    expected = Version(expected_version)
    for package in ('kfp', 'kfp-kubernetes', 'kfp-server-api',
                    'kfp-pipeline-spec'):
        actual = importlib.metadata.version(package)
        if Version(actual) != expected:
            raise RuntimeError(
                f'{package}: expected {expected_version}, got {actual}')


def main():
    expected_version = sys.argv[1]
    verify_versions(expected_version)
    with tempfile.TemporaryDirectory() as directory:
        output = Path(directory) / 'pipeline.yaml'
        compiler.Compiler().compile(pipeline, package_path=str(output))
        compiled = output.read_text()
        if 'platforms:' not in compiled or 'sdk-smoke' not in compiled:
            raise RuntimeError(
                'Compiled pipeline lost Kubernetes configuration')
        subprocess.run([sys.executable, '-m', 'kfp.cli', '--help'],
                       cwd=directory,
                       check=True,
                       stdout=subprocess.DEVNULL)
    print(f'Installed SDK {expected_version} smoke passed on {sys.version}')


if __name__ == '__main__':
    main()

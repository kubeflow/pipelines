#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Remove SDK installation only from the two controlled cache fixtures."""

import argparse
import copy
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = {
    'hello-world-with-returning-component.yaml': {
        'exec-comp': ('python:3.12', '2.13.0')
    },
    'pvc-mount.yaml': {
        'exec-consumer': ('python:3.11', '2.14.2'),
        'exec-producer': ('python:3.11', '2.14.2'),
    },
}


def installer(version):
    """Match the complete generated installer, rejecting extra dependencies."""
    return (
        '\nif ! [ -x "$(command -v pip)" ]; then\n'
        '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\n'
        'fi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location '
        f"'kfp=={version}' '--no-deps' "
        '\'typing-extensions>=3.7.4,<5; python_version<"3.9"\' '
        '&& "$0" "$@"\n'
    )


def transform(name, documents, image):
    """Change only the image and SDK installer prefix."""
    if name not in FIXTURES or not documents:
        raise ValueError(f'Unsupported cache fixture: {name}')
    result = copy.deepcopy(documents)
    executors = result[0]['deploymentSpec']['executors']
    if set(executors) != set(FIXTURES[name]):
        raise ValueError(f'Unexpected executors in {name}')
    for name, (base, version) in FIXTURES[name].items():
        container = executors[name]['container']
        command = container['command']
        if (
            container['image'] != base
            or len(command) != 7
            or command[:3] != ['sh', '-c', installer(version)]
            or command[3:5] != ['sh', '-ec']
            or '_KFP_RUNTIME=true python3 -m kfp.dsl.executor_main'
            not in command[5]
        ):
            raise ValueError(
                f'Unexpected SDK installer or executor command in {name}'
            )
        container['image'] = image
        container['command'] = command[3:]
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    parser.add_argument('--image', default='kfp-cache-fixture:ci')
    args = parser.parse_args()
    prepared = {}
    for name in FIXTURES:
        source = ROOT / 'backend/test/v2/resources' / name
        documents = list(yaml.safe_load_all(source.read_text()))
        prepared[name] = transform(name, documents, args.image)
    args.output.mkdir(parents=True, exist_ok=True)
    for name, documents in prepared.items():
        (args.output / name).write_text(
            yaml.safe_dump_all(documents, sort_keys=False)
        )


if __name__ == '__main__':
    main()

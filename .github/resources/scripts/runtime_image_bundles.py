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
"""Select runtime preload bundles without changing the fixture inventory."""

import argparse
from pathlib import Path

BUNDLES = ('fixtures', 'mysql', 'argo-v4.0.13', 'argo-v4.1.5', 'modelcar')
ROOT = Path(__file__).resolve().parents[3]


def bundle_images(bundle, inventory=None):
    if bundle not in BUNDLES:
        raise ValueError(f'Unknown runtime image bundle: {bundle}')
    if inventory is None:
        inventory = (ROOT / '.github/resources/runtime-base-images.txt'
                    ).read_text().splitlines()
    selected = []
    for line in inventory:
        image = line.strip()
        if not image or image.startswith('#'):
            continue
        if image.startswith('quay.io/argoproj/'):
            group = 'argo-' + image.rsplit(':', 1)[1]
        elif image.startswith('docker.io/library/mysql:'):
            group = 'mysql'
        else:
            group = 'fixtures'
        if group not in BUNDLES:
            raise ValueError(f'Runtime image has no supported bundle: {image}')
        if group == bundle:
            selected.append(image)
    return selected


def selected_bundles(argo_version='', db_type='', modelcar=False):
    version = argo_version or (ROOT /
                               'third_party/argo/VERSION').read_text().strip()
    argo = 'argo-' + version
    if argo not in BUNDLES:
        # Preserve arbitrary deploy overrides: preload the legacy full inventory;
        # deployment can still pull the requested version as it did previously.
        return [
            bundle for bundle in BUNDLES if bundle != 'modelcar' or modelcar
        ]
    if db_type not in ('', 'mysql', 'pgx'):
        raise ValueError(f'Unsupported runtime database: {db_type}')
    result = ['fixtures', argo]
    if db_type != 'pgx':
        result.append('mysql')
    if modelcar:
        result.append('modelcar')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('bundle', choices=BUNDLES)
    parser.add_argument('--inventory', type=Path)
    args = parser.parse_args()
    inventory = args.inventory.read_text().splitlines(
    ) if args.inventory else None
    print('\n'.join(bundle_images(args.bundle, inventory)))


if __name__ == '__main__':
    main()

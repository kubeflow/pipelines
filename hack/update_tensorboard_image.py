#!/usr/bin/env python3
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
"""Stamp the native TensorBoard runtime alongside the other release images."""

import argparse
from pathlib import Path
import re

IMAGE = 'ghcr.io/kubeflow/kfp-tensorboard:'
TAG = r'[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}'
IMAGE_REFERENCE = re.compile(re.escape(IMAGE) + TAG + r'(?![A-Za-z0-9_.@-])')
REQUIRED_PATHS = (
    'backend/src/crd/controller/viewer/reconciler/reconciler.go',
    'frontend/src/components/viewers/Tensorboard.tsx',
    'frontend/src/components/viewers/Tensorboard.test.tsx',
    'frontend/src/components/viewers/__snapshots__/Tensorboard.test.tsx.snap',
)


def update_images(root: Path, tag: str) -> None:
    """Validate every required source before updating source and bundled
    assets."""
    if re.fullmatch(TAG, tag) is None:
        raise ValueError(
            'Invalid image tag; use a Docker tag of at most 128 characters.')
    changes = {}
    for relative in REQUIRED_PATHS:
        path = root / relative
        if not path.is_file():
            raise ValueError(
                f'Missing TensorBoard source {relative}; restore it before stamping the release.'
            )
        content = path.read_text()
        updated, count = IMAGE_REFERENCE.subn(IMAGE + tag, content)
        if count == 0:
            raise ValueError(
                f'No native TensorBoard image in {relative}; update the release image paths before stamping.'
            )
        changes[path] = updated
    assets = root / 'sdk/python/kfp/cli/viewer_assets'
    if assets.is_dir():
        for path in assets.rglob('*'):
            if path.is_file() and path.suffix in {
                    '.js', '.html', '.css', '.json', '.map'
            }:
                content = path.read_text()
                if IMAGE in content:
                    changes[path] = IMAGE_REFERENCE.sub(IMAGE + tag, content)
    for path, updated in changes.items():
        if path.read_text() != updated:
            path.write_text(updated)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('tag')
    args = parser.parse_args()
    try:
        update_images(Path(__file__).resolve().parents[1], args.tag)
    except ValueError as error:
        parser.error(str(error))


if __name__ == '__main__':
    main()

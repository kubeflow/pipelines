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

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

CONTAINER_LIST_PATTERN = re.compile(r'^(?:initContainers|containers):\s*$')
IMAGE_PATTERN = re.compile(
    r'''^(?:-\s*)?image:\s*["']?([^\s"'#]+)["']?\s*(?:#.*)?$''')


def extract_images(rendered_manifest: str) -> set[str]:
    """Returns the unique container images in a rendered Kubernetes manifest."""
    images = set()
    container_list_indent = None
    for line in rendered_manifest.splitlines():
        stripped_line = line.lstrip()
        line_indent = len(line) - len(stripped_line)

        if CONTAINER_LIST_PATTERN.match(stripped_line):
            container_list_indent = line_indent
            continue

        if container_list_indent is None:
            continue

        if (line_indent < container_list_indent or
            (line_indent == container_list_indent and
             not stripped_line.startswith('-'))):
            container_list_indent = None
            continue

        image_match = IMAGE_PATTERN.match(stripped_line)
        if image_match:
            images.add(image_match.group(1))

    return images


def render_overlay(kustomize: Path, overlay: Path) -> str:
    """Renders one Kustomize overlay and returns its Kubernetes resources."""
    result = subprocess.run(
        [
            str(kustomize),
            'build',
            '--load-restrictor',
            'LoadRestrictionsNone',
            str(overlay),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout


def create_matrix(images: set[str]) -> dict[str, list[dict[str, str]]]:
    """Creates a stable GitHub Actions matrix for the discovered images."""
    return {
        'include': [{
            'image': image,
            'category': hashlib.sha256(image.encode()).hexdigest()[:16],
        } for image in sorted(images)]
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Render KFP Kustomize overlays and emit an image matrix.')
    parser.add_argument('--kustomize', required=True, type=Path)
    parser.add_argument('--overlay', action='append', required=True, type=Path)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    images: set[str] = set()
    for overlay in args.overlay:
        images.update(extract_images(render_overlay(args.kustomize, overlay)))

    if not images:
        raise RuntimeError(
            'No container images found in rendered KFP manifests')

    print(json.dumps(create_matrix(images), separators=(',', ':')))


if __name__ == '__main__':
    main()

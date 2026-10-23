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
import subprocess
from pathlib import Path

import yaml


def extract_images(rendered_manifest: str) -> set[str]:
    """Returns unique container images from rendered Kubernetes resources."""
    images = set()

    def visit(node):
        if isinstance(node, dict):
            for key, value in node.items():
                if key in ('initContainers', 'containers') and isinstance(
                    value, list
                ):
                    for container in value:
                        if isinstance(container, dict):
                            image = container.get('image')
                            if isinstance(image, str) and image:
                                images.add(image)
                visit(value)
        elif isinstance(node, list):
            for item in node:
                visit(item)

    # ConfigMaps can embed source manifests as strings for hashing. Those are
    # not deployed containers, even when their text contains a containers list.
    for document in yaml.safe_load_all(rendered_manifest):
        visit(document)

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


def create_image_map(images: set[str]) -> dict[str, str]:
    """Maps stable repository categories to the exact images to scan."""
    image_map = {}
    for image in sorted(images):
        repository = image.split('@', 1)[0]
        # Only the final path component can contain a tag; preserve registry ports.
        if ':' in repository.rsplit('/', 1)[-1]:
            repository = repository.rsplit(':', 1)[0]
        category = hashlib.sha256(repository.encode()).hexdigest()[:16]
        if category in image_map:
            raise ValueError(
                f'Multiple image references share scan identity {repository}: '
                f'{image_map[category]} and {image}. Align their versions or '
                'define separate stable scan targets before scanning both.'
            )
        image_map[category] = image
    return image_map


def create_matrix(images: set[str]) -> dict[str, list[dict[str, str]]]:
    """Keeps versioned references out of GitHub's analysis environment."""
    return {
        'include': [
            {'category': category} for category in create_image_map(images)
        ]
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Render KFP Kustomize overlays and emit an image matrix.'
    )
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
            'No container images found in rendered KFP manifests'
        )

    inventory = {
        'matrix': create_matrix(images),
        'images': create_image_map(images),
    }
    print(json.dumps(inventory, separators=(',', ':')))


if __name__ == '__main__':
    main()

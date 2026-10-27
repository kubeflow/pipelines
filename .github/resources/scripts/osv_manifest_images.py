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
from arm64_smoke import CI_IMAGES, IMAGES
from osv_master_images import merge_published_images, repository_name

# Components removed from master remain first-party on supported release
# branches. Names match the release-2.18 image-builds-release.yml inventory;
# legacy aliases are the previously published gcr.io/ml-pipeline names.
RELEASE_IMAGES = {
    'kfp-visualization-server': 'visualization-server',
    'kfp-tensorboard': 'tensorboard',
    'kfp-cache-deployer': 'cache-deployer',
    'kfp-cache-server': 'cache-server',
    'kfp-metadata-envoy': 'metadata-envoy',
    'kfp-metadata-writer': 'metadata-writer',
    'kfp-inverse-proxy-agent': 'inverse-proxy-agent',
}


def is_first_party(image: str, owner: str = 'kubeflow') -> bool:
    """Classify built KFP components, not every image in an owned registry."""
    repository = repository_name(image)
    prefix = f'ghcr.io/{owner.lower()}/'
    if repository.startswith(prefix):
        name = repository.removeprefix(prefix)
        if name in IMAGES or name in RELEASE_IMAGES:
            return True
        if name.startswith('kfp-'):
            raise ValueError(f'Unclassified KFP image repository: {repository}')
    legacy = (
        set(CI_IMAGES.values())
        | set(RELEASE_IMAGES.values())
        | {name.removeprefix('kfp-') for name in IMAGES}
    )
    return repository in {f'gcr.io/ml-pipeline/{name}' for name in legacy}


def select_images(images: set[str], scope: str, owner: str) -> set[str]:
    """Partition coverage without changing image or category identities."""
    if scope == 'all':
        return images
    if scope not in ('first-party', 'third-party'):
        raise ValueError(f'Invalid image scope: {scope}')
    selected = {
        image
        for image in images
        if is_first_party(image, owner) == (scope == 'first-party')
    }
    if not selected:
        raise ValueError(f'No {scope} images found in rendered manifests')
    return selected


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
        repository = repository_name(image)
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
    parser.add_argument('--published-records', type=Path)
    parser.add_argument('--source-sha')
    parser.add_argument('--image-owner')
    parser.add_argument(
        '--image-scope',
        choices=('all', 'first-party', 'third-party'),
        default='all',
    )
    args = parser.parse_args()
    if any(
        (args.published_records, args.source_sha, args.image_owner)
    ) and not all((args.published_records, args.source_sha, args.image_owner)):
        parser.error(
            '--published-records, --source-sha and --image-owner must be supplied together'
        )
    return args


def main() -> None:
    args = parse_args()
    images: set[str] = set()
    for overlay in args.overlay:
        images.update(extract_images(render_overlay(args.kustomize, overlay)))

    if not images:
        raise RuntimeError(
            'No container images found in rendered KFP manifests'
        )

    if args.published_records:
        images = merge_published_images(
            images, args.published_records, args.source_sha, args.image_owner
        )

    images = select_images(
        images, args.image_scope, args.image_owner or 'kubeflow'
    )

    inventory = {
        'matrix': create_matrix(images),
        'images': create_image_map(images),
    }
    print(json.dumps(inventory, separators=(',', ':')))


if __name__ == '__main__':
    main()

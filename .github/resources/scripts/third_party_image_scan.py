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
"""Collect branch-pinned third-party inventories and scan immutable images."""

import argparse
import hashlib
import json
import os
import subprocess
from pathlib import Path

from kfpr.deployed_images import SOURCE_PATHS, capture
from osv_manifest_images import (
    create_image_map,
    is_first_party,
)
from osv_manifest_source import download_manifests
from osv_master_images import github_api, repository_name

RELEASE_BRANCH = 'release-2.18'


def create_inventory(sources, branch_images, default_branch='master'):
    if default_branch not in ('master', 'main'):
        raise ValueError('Expected master or main as the default branch')
    branches = (default_branch, RELEASE_BRANCH)
    if set(sources) != set(branches) or set(branch_images) != set(branches):
        raise ValueError(
            'Inventory sources must match the default and release branches'
        )
    rows = {}
    uploads = []
    for branch in branches:
        images = branch_images[branch]
        if not images:
            raise ValueError(f'No third-party images found for {branch}')
        if isinstance(images, set):
            targets = [
                {
                    'target': repository_name(image),
                    'image': image,
                    'scope': 'standard',
                    'profiles': [],
                }
                for image in create_image_map(images).values()
            ]
        else:
            targets = images
        categories = set()
        for target in targets:
            image = target['image']
            category = hashlib.sha256(target['target'].encode()).hexdigest()[
                :16
            ]
            if category in categories:
                raise ValueError('Duplicate stable scan target in ' + branch)
            categories.add(category)
            key = hashlib.sha256(image.encode()).hexdigest()[:16]
            row = rows.setdefault(
                key, {'key': key, 'image': image, 'branches': [], 'targets': []}
            )
            if row['image'] != image:
                raise ValueError('Image key collision')
            if branch not in row['branches']:
                row['branches'].append(branch)
            row['targets'].append({'branch': branch, **target})
            uploads.append(
                {
                    'key': key,
                    'image': image,
                    'branch': branch,
                    'sha': sources[branch],
                    'category': (
                        f'kubeflow-pipelines-osv-image-{category}'
                        if branch == default_branch
                        else f'{branch}-third-party-image-{category}-amd64'
                    ),
                }
            )
    development = {
        target['target']: r
        for r in rows.values()
        for target in r['targets']
        if target['branch'] == default_branch
    }
    for row in rows.values():
        candidates = {
            development[target['target']]['key']
            for target in row['targets']
            if target['branch'] == 'release-2.18'
            and target['target'] in development
        }
        candidate = (
            rows[next(iter(candidates))] if len(candidates) == 1 else None
        )
        if candidate and candidate['key'] != row['key']:
            row['candidate_key'] = candidate['key']
    return {
        'schema': 1,
        'default_branch': default_branch,
        'sources': sources,
        'images': list(rows.values()),
        'uploads': uploads,
        'scans': [
            {'key': r['key'], 'image': r['image']} for r in rows.values()
        ],
    }


def discover(repository, kustomize, output):
    default_branch = github_api(f'repos/{repository}')['default_branch']
    if default_branch not in ('master', 'main'):
        raise ValueError('Expected master or main as the default branch')
    branches = (default_branch, RELEASE_BRANCH)
    sources, images = {}, {}
    for branch in branches:
        sha = github_api(f'repos/{repository}/commits/{branch}')['sha']
        sources[branch] = sha
        destination = output / branch
        download_manifests(
            repository, sha, destination, extra_paths=SOURCE_PATHS
        )
        captured = capture(
            destination, kustomize, output / f'{branch}-deployed'
        )
        images[branch] = [
            target
            for target in captured['targets']
            if not is_first_party(target['image'])
        ]
    inventory = create_inventory(sources, images, default_branch)
    inventory['workflow_sha'] = os.environ['GITHUB_WORKFLOW_SHA']
    inventory['run_id'] = os.environ['GITHUB_RUN_ID']
    inventory['run_attempt'] = os.environ['GITHUB_RUN_ATTEMPT']
    (output / 'inventory.json').write_text(
        json.dumps(inventory, indent=2) + '\n'
    )
    with open(os.environ['GITHUB_OUTPUT'], 'a') as stream:
        stream.write(
            'matrix=' + json.dumps({'include': inventory['scans']}) + '\n'
        )
        stream.write(
            'uploads=' + json.dumps({'include': inventory['uploads']}) + '\n'
        )


def scan(image, output):
    from kfpr.osv_image_scan import scan_image

    output.mkdir(parents=True, exist_ok=False)
    subprocess.run(
        ['bash', '.github/resources/scripts/docker-pull-with-retry.sh', image],
        check=True,
    )
    metadata = json.loads(
        subprocess.check_output(
            ['docker', 'image', 'inspect', image], text=True
        )
    )
    if len(metadata) != 1:
        raise ValueError('Expected one pulled image')
    item = metadata[0]
    platform = item['Os'] + '/' + item['Architecture']
    if platform != 'linux/amd64':
        raise ValueError('Third-party inventory currently requires linux/amd64')
    subject = {
        'image': image,
        'image_id': item['Id'],
        'platform': platform,
        'repo_digests': item['RepoDigests'],
    }
    (output / 'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
    try:
        scan_image(item['Id'], platform, output, report_format='json')
        github_output = output / 'github'
        github_output.mkdir()
        scan_image(item['Id'], platform, github_output, report_format='sarif')
    except (ValueError, subprocess.CalledProcessError) as error:
        (output / 'coverage.json').write_text(
            json.dumps(
                {
                    'status': 'unknown',
                    'image_id': item['Id'],
                    'reason': str(error),
                },
                indent=2,
            )
            + '\n'
        )
        raise
    (output / 'coverage.json').write_text(
        json.dumps(
            {
                'status': 'complete',
                'image_id': item['Id'],
            },
            indent=2,
        )
        + '\n'
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    discovery = sub.add_parser('discover')
    discovery.add_argument('--repository', required=True)
    discovery.add_argument('--kustomize', required=True, type=Path)
    discovery.add_argument('--output', required=True, type=Path)
    scanner = sub.add_parser('scan')
    scanner.add_argument('--image', required=True)
    scanner.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    if args.command == 'discover':
        args.output.mkdir(parents=True, exist_ok=False)
        discover(args.repository, args.kustomize, args.output)
    else:
        scan(args.image, args.output)


if __name__ == '__main__':
    main()

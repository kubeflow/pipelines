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

from kfpr.third_party import extract_images
from osv_manifest_images import (
    create_image_map,
    is_first_party,
    render_overlay,
)
from osv_manifest_source import download_manifests
from osv_master_images import github_api, repository_name

BRANCHES = ('master', 'release-2.18')


def create_inventory(sources, branch_images):
    rows = {}
    uploads = []
    for branch in BRANCHES:
        images = branch_images[branch]
        if not images:
            raise ValueError(f'No third-party images found for {branch}')
        image_map = create_image_map(images)
        for category, image in image_map.items():
            key = hashlib.sha256(image.encode()).hexdigest()[:16]
            row = rows.setdefault(
                key, {'key': key, 'image': image, 'branches': []}
            )
            if row['image'] != image:
                raise ValueError('Image key collision')
            row['branches'].append(branch)
            uploads.append(
                {
                    'key': key,
                    'image': image,
                    'branch': branch,
                    'sha': sources[branch],
                    'category': (
                        f'kubeflow-pipelines-osv-image-{category}'
                        if branch == 'master'
                        else f'{branch}-third-party-image-{category}-amd64'
                    ),
                }
            )
    master = {
        repository_name(r['image']): r
        for r in rows.values()
        if 'master' in r['branches']
    }
    for row in rows.values():
        candidate = master.get(repository_name(row['image']))
        if candidate and candidate['key'] != row['key']:
            row['candidate_key'] = candidate['key']
    return {
        'schema': 1,
        'sources': sources,
        'images': list(rows.values()),
        'uploads': uploads,
        'scans': [
            {'key': r['key'], 'image': r['image']} for r in rows.values()
        ],
    }


def discover(repository, kustomize, output):
    sources, images = {}, {}
    for branch in BRANCHES:
        sha = github_api(f'repos/{repository}/commits/{branch}')['sha']
        sources[branch] = sha
        destination = output / branch
        download_manifests(repository, sha, destination)
        deployed = set()
        for profile in ('platform-agnostic', 'platform-agnostic-multi-user'):
            rendered = render_overlay(
                kustomize, destination / 'manifests/kustomize/env' / profile
            )
            (output / f'{branch}-{profile}.yaml').write_text(rendered)
            deployed.update(extract_images(rendered))
        images[branch] = {
            image for image in deployed if not is_first_party(image)
        }
    inventory = create_inventory(sources, images)
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
    scan_image(item['Id'], platform, output, report_format='json')
    github_output = output / 'github'
    github_output.mkdir()
    scan_image(item['Id'], platform, github_output, report_format='sarif')


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

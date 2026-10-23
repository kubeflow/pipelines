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

"""Resolve one successful master publication and validate its image records."""

import argparse
import json
import re
import subprocess
from pathlib import Path

from arm64_smoke import IMAGES, image_refs

WORKFLOW = '.github/workflows/image-builds-master.yml'


def validate_producer(run, repository):
    expected = {
        'path': WORKFLOW,
        'event': 'push',
        'head_branch': 'master',
        'status': 'completed',
        'conclusion': 'success',
    }
    if any(run.get(key) != value for key, value in expected.items()):
        raise ValueError(
            'Expected a completed successful master image-build push'
        )
    if any(
        run.get(key, {}).get('full_name') != repository
        for key in ('repository', 'head_repository')
    ):
        raise ValueError(
            'Image producer must belong to the scanning repository'
        )
    if not re.fullmatch(r'[0-9a-f]{40}', run.get('head_sha', '')):
        raise ValueError('Image producer must identify a full source SHA')
    if any(
        type(run.get(key)) is not int or run[key] <= 0
        for key in ('id', 'run_attempt', 'run_number')
    ):
        raise ValueError(
            'Image producer must identify its run, attempt and number'
        )


def select_producer(runs, repository, trigger=None):
    """Choose source order, not completion order, for every trusted wakeup."""
    if not runs:
        raise ValueError(
            'No successful master image build; publish images before scanning'
        )
    for run in runs:
        validate_producer(run, repository)
    if trigger is not None:
        validate_producer(trigger, repository)
    # A late completion can replace a newer pending workflow in GitHub's
    # concurrency queue. Treat it as a wakeup to scan the latest publication.
    return max(runs, key=lambda run: run['run_number'])


def repository_name(image):
    repository = image.split('@', 1)[0]
    if ':' in repository.rsplit('/', 1)[-1]:
        repository = repository.rsplit(':', 1)[0]
    return repository


def merge_published_images(images, records, source_sha, owner):
    """Replace only known KFP repositories and include runtime driver/launcher."""
    refs = image_refs(records, source_sha, 'published')
    repositories = {f'ghcr.io/{owner.lower()}/{name}' for name in IMAGES}
    if {repository_name(ref) for ref in refs.values()} != repositories:
        raise ValueError(
            'Published image records must use the expected GHCR owner'
        )
    return {
        image for image in images if repository_name(image) not in repositories
    } | set(refs.values())


def github_api(endpoint):
    result = subprocess.run(
        ['gh', 'api', endpoint],
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(result.stdout)


def resolve_producer(repository, trigger=None):
    # The workflow runs API lists newest-created runs first. The first 100
    # successful master pushes contain the newest run_number even if an older
    # run completed or was rerun later.
    endpoint = (
        f'repos/{repository}/actions/workflows/image-builds-master.yml/runs'
    )
    runs = github_api(
        endpoint + '?branch=master&event=push&status=success&per_page=100'
    )['workflow_runs']
    selected = select_producer(runs, repository, trigger)
    current = github_api(f'repos/{repository}/actions/runs/{selected["id"]}')
    validate_producer(current, repository)
    if any(
        current[key] != selected[key]
        for key in ('id', 'run_attempt', 'head_sha')
    ):
        raise ValueError(
            'Producer changed during selection; retry after its latest attempt completes'
        )
    return current


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repository', required=True)
    parser.add_argument('--event-name', required=True)
    parser.add_argument('--event-path', required=True, type=Path)
    parser.add_argument('--github-output', required=True, type=Path)
    args = parser.parse_args()
    trigger = None
    if args.event_name == 'workflow_run':
        trigger = json.loads(args.event_path.read_text())['workflow_run']
    elif args.event_name not in ('schedule', 'workflow_dispatch'):
        raise ValueError(
            'Image scans require a producer completion, schedule or manual dispatch'
        )
    selected = resolve_producer(args.repository, trigger)
    with args.github_output.open('a') as output:
        output.write(f'run-id={selected["id"]}\n')
        output.write(f'run-attempt={selected["run_attempt"]}\n')
        output.write(f'sha={selected["head_sha"]}\n')


if __name__ == '__main__':
    main()

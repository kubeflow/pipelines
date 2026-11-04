"""Promote existing published indexes through trusted CI without rebuilding."""

import argparse
import base64
import importlib.util
import json
import os
import re
import subprocess
import uuid
from pathlib import Path

import yaml

from .core import (
    REPO,
    CommandRunner,
    ReleaseContext,
    ReleaseMetadata,
    ReleaseState,
    default_branch,
)
from .qualification import _matrix_records
from .steps import _should_set_backend_latest
from .workflow_runs import dispatch_and_watch, validate_run

WORKFLOW = 'release-promote-latest.yml'


def api(path):
    return json.loads(
        subprocess.check_output(
            ['gh', 'api', f'repos/{REPO}/{path}'], text=True
        )
    )


def discover(version):
    if not re.fullmatch(
        r'(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)', version
    ):
        raise ValueError('Promotion requires a stable X.Y.Z release tag')
    release = api(f'releases/tags/{version}')
    if (
        release.get('tag_name') != version
        or release.get('draft') is not False
        or release.get('prerelease') is not False
        or not release.get('published_at')
    ):
        raise ValueError(
            'Promotion requires an existing published stable backend release'
        )
    context = ReleaseContext(
        Path.cwd(),
        ReleaseState(Path('unused-promotion-state.json')),
        CommandRunner(),
        ReleaseMetadata.from_version('patch', version),
        'origin',
        True,
        False,
    )
    if not _should_set_backend_latest(context):
        raise ValueError(
            'A newer stable backend release exists; refusing latest downgrade'
        )
    sha = api(f'commits/{version}').get('sha', '')
    if not re.fullmatch(r'[0-9a-f]{40}', sha):
        raise ValueError('Invalid released source commit')
    source = api(
        f'contents/.github/workflows/image-builds-release.yml?ref={sha}'
    )
    records = _matrix_records(
        yaml.safe_load(base64.b64decode(source['content'], validate=False))
    )
    images = {}
    for record in records:
        name, platform = record['name'], record['platform']
        if not re.fullmatch(
            r'[a-z0-9]+(?:-[a-z0-9]+)*', name
        ) or platform not in ('linux/amd64', 'linux/arm64'):
            raise ValueError('Invalid released image inventory')
        platforms = images.setdefault(name, [])
        if platform in platforms:
            raise ValueError('Duplicate released image platform')
        platforms.append(platform)
    if not images:
        raise ValueError('Empty released image inventory')
    return {
        'version': version,
        'source_sha': sha,
        'images': {
            name: sorted(platforms)
            for name, platforms in sorted(images.items())
        },
    }


def index_tools():
    path = (
        Path(__file__).resolve().parents[2]
        / '.github/resources/scripts/publish_image_index.py'
    )
    spec = importlib.util.spec_from_file_location(
        'kfpr_publish_image_index', path
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def promote(inventory, output):
    """Preflight every index, copy only latest, retain evidence even on
    failure."""
    report = {
        'schema': 1,
        'release': inventory,
        'workflow_sha': os.environ.get('GITHUB_SHA'),
        'run_id': os.environ.get('GITHUB_RUN_ID'),
        'run_attempt': os.environ.get('GITHUB_RUN_ATTEMPT'),
        'images': [],
        'passed': False,
    }
    output.parent.mkdir(parents=True, exist_ok=True)
    try:
        if discover(inventory['version']) != inventory:
            raise ValueError(
                'Release inventory changed after read-only discovery'
            )
        tools = index_tools()
        for name, platforms in inventory['images'].items():
            image = f'ghcr.io/kubeflow/{name}'
            source = tools.inspect(f'{image}:{inventory["version"]}')
            source_digest = tools.digest(source.get('digest'))
            tools.validate_index(source, set(platforms))
            previous = tools.inspect(f'{image}:latest')
            previous_digest = tools.digest(previous.get('digest'))
            # Validate immutable retrieval as well as the versioned tag snapshot.
            immutable = tools.inspect(f'{image}@{source_digest}')
            tools.validate_index(immutable, set(platforms))
            if immutable != source:
                raise ValueError(
                    'Immutable registry index differs from published version'
                )
            report['images'].append(
                {
                    'image': image,
                    'source_digest': source_digest,
                    'previous_digest': previous_digest,
                    'platforms': platforms,
                    'source_index': source,
                    'previous_index': previous,
                    'promoted': False,
                }
            )
        if discover(inventory['version']) != inventory:
            raise ValueError(
                'Release eligibility or source changed before promotion'
            )
        for record in report['images']:
            if discover(inventory['version']) != inventory:
                raise ValueError('Release eligibility changed during promotion')
            image, digest = record['image'], record['source_digest']
            if (
                tools.inspect(f'{image}:{inventory["version"]}').get('digest')
                != digest
            ):
                raise ValueError('Versioned index changed before promotion')
            tools.imagetools(
                'create',
                '--prefer-index=false',
                '--tag',
                f'{image}:latest',
                f'{image}@{digest}',
            )
            latest = tools.inspect(f'{image}:latest')
            if latest != record['source_index']:
                raise ValueError(
                    'Promoted latest index is not identical to released index'
                )
            record.update(promoted=True, new_digest=latest['digest'])
        for record in report['images']:
            for tag in (inventory['version'], 'latest'):
                if (
                    tools.inspect(f'{record["image"]}:{tag}')
                    != record['source_index']
                ):
                    raise ValueError('Final index verification failed')
        if discover(inventory['version']) != inventory:
            raise ValueError(
                'Release eligibility changed during final verification'
            )
        report['passed'] = True
    except Exception as error:
        report['error'] = str(error)
        raise
    finally:
        output.write_text(json.dumps(report, indent=2) + '\n')
    return report


def dispatch(version, dry_run=False):
    """Dispatch upstream CI; local credentials need no package-write access."""
    if not re.fullmatch(
        r'(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)', version
    ):
        raise ValueError('Promotion requires a stable X.Y.Z release tag')
    if dry_run:
        print(
            f'[dry-run] would dispatch upstream {WORKFLOW} for published {version}; no local image operations'
        )
        return
    discover(version)
    workflow_ref = default_branch(api)
    workflow_sha = api(f'commits/{workflow_ref}')['sha']
    if not re.fullmatch(r'[0-9a-f]{40}', workflow_sha):
        raise ValueError('Invalid upstream workflow SHA')
    request = uuid.uuid4().hex
    title = f'KFP latest promotion {version} {request}'
    identity = {
        'workflow': f'.github/workflows/{WORKFLOW}',
        'workflow_sha': workflow_sha,
        'workflow_ref': workflow_ref,
        'title': title,
    }
    final = dispatch_and_watch(
        api,
        subprocess.run,
        identity,
        {'version': version, 'request_id': request},
        attempts=30,
        interval=2,
    )
    validate_run(final, identity)
    print(
        f'Latest promotion evidence: https://github.com/{REPO}/actions/runs/{identity["run_id"]}'
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['discover', 'promote'])
    args = parser.parse_args()
    if (
        os.environ.get('GITHUB_REPOSITORY') != REPO
        or os.environ.get('GITHUB_EVENT_NAME') != 'workflow_dispatch'
        or os.environ.get('GITHUB_REF') != f'refs/heads/{default_branch(api)}'
    ):
        raise ValueError(
            'Promotion must run in the upstream default-branch workflow'
        )
    if args.command == 'discover':
        inventory = discover(os.environ['VERSION'])
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            output.write(
                'inventory='
                + json.dumps(inventory, separators=(',', ':'))
                + '\n'
            )
    else:
        promote(
            json.loads(os.environ['PROMOTION_INVENTORY']),
            Path('evidence/promotion.json'),
        )


if __name__ == '__main__':
    main()

#!/usr/bin/env python3
"""Fingerprint runtime-image generation and select a trusted producer run."""

import argparse
from datetime import datetime
from datetime import timezone
import hashlib
import json
from pathlib import Path
import sys
from typing import Any, Iterable, Optional

from runtime_image_bundles import BUNDLES
from runtime_image_bundles import selected_bundles


def fingerprint_files(paths: Iterable[Path]) -> str:
    """Return a stable fingerprint of each path name and file contents."""
    digest = hashlib.sha256()
    for path in paths:
        path_bytes = str(path).encode()
        contents = path.read_bytes()
        digest.update(len(path_bytes).to_bytes(8, 'big'))
        digest.update(path_bytes)
        digest.update(len(contents).to_bytes(8, 'big'))
        digest.update(contents)
    return digest.hexdigest()


def select_producer_run_id(payload: Any, source_sha: str) -> Optional[int]:
    """Return the newest current-source or trusted-master producer run ID."""
    if not isinstance(payload, dict):
        raise ValueError('Expected an artifact API response object')
    artifacts = payload.get('artifacts')
    if not isinstance(artifacts, list):
        raise ValueError('Artifact API response must contain an artifacts list')

    candidates = []
    for artifact in artifacts:
        if not isinstance(artifact,
                          dict) or artifact.get('expired') is not False:
            continue
        workflow_run = artifact.get('workflow_run')
        if not isinstance(workflow_run, dict):
            continue

        is_current_source = workflow_run.get('head_sha') == source_sha
        head_repository_id = workflow_run.get('head_repository_id')
        is_trusted_master = (
            workflow_run.get('head_branch') == 'master' and
            isinstance(head_repository_id, int) and
            head_repository_id == workflow_run.get('repository_id'))
        if is_current_source or is_trusted_master:
            candidates.append(artifact)

    if not candidates:
        return None

    newest = max(
        candidates, key=lambda artifact: artifact.get('created_at', ''))
    run_id = newest['workflow_run'].get('id')
    if not isinstance(run_id, int):
        raise ValueError(
            'Selected artifact is missing an integer workflow run ID')
    return run_id


GENERATION_INPUTS = (
    '.github/resources/scripts/runtime_image_bundles.py',
    '.github/resources/runtime-base-images.txt',
    '.github/resources/scripts/helper-functions.sh',
    '.github/resources/scripts/build-runtime-base-images.sh',
    '.github/workflows/runtime-base-images.yml',
    'test_data/sdk_compiled_pipelines/valid/critical/modelcar/Dockerfile',
)


def expected_artifact_name() -> str:
    return 'runtime-base-images-' + fingerprint_files(
        Path(path) for path in GENERATION_INPUTS)


def eligible_artifact(artifact, name, source_sha, repository_id):
    """Validate API metadata, never trusting identity from a reference file."""
    if (type(repository_id) is not int or repository_id <= 0 or
            not isinstance(source_sha, str) or not source_sha or
            not isinstance(artifact, dict)):
        return False
    run = artifact.get('workflow_run', {})
    if not isinstance(run, dict):
        return False
    if (artifact.get('name') != name or artifact.get('expired') is not False or
            type(artifact.get('id')) is not int or artifact['id'] <= 0 or
            type(run.get('id')) is not int or run['id'] <= 0 or
            type(run.get('repository_id')) is not int or
            type(run.get('head_repository_id')) is not int or
            run['head_repository_id'] <= 0 or
            run.get('repository_id') != repository_id):
        return False
    try:
        expiry = datetime.fromisoformat(artifact['expires_at'].replace(
            'Z', '+00:00'))
        created = datetime.fromisoformat(artifact['created_at'].replace(
            'Z', '+00:00'))
        if expiry <= datetime.now(timezone.utc) or created.tzinfo is None:
            return False
    except (KeyError, TypeError, ValueError, AttributeError):
        return False
    return (run.get('head_sha') == source_sha or
            (run.get('head_branch') == 'master' and
             run.get('head_repository_id') == repository_id))


def select_artifact(pages, name, source_sha, repository_id):
    if not isinstance(pages, list):
        raise ValueError('Expected paginated artifact API responses')
    candidates = [
        artifact for page in pages for artifact in page['artifacts']
        if eligible_artifact(artifact, name, source_sha, repository_id)
    ]
    return max(
        candidates,
        key=lambda item: (datetime.fromisoformat(item['created_at'].replace(
            'Z', '+00:00')), item['id']),
        default=None)


def bundle_reference(path):
    reference = json.loads(path.read_text())
    if (not isinstance(reference, dict) or set(reference) != {'bundles'} or
            not isinstance(reference['bundles'], dict) or
            set(reference['bundles']) != set(BUNDLES) or any(
                type(value) is not int or value <= 0
                for value in reference['bundles'].values()) or
            len(set(reference['bundles'].values())) != len(BUNDLES)):
        raise ValueError(
            'Expected a complete runtime bundle reference with unique positive IDs'
        )
    return reference['bundles']


def select_bundles(pages, source_sha, repository_id):
    # A refresh may be partially uploaded. Never combine mutable image snapshots
    # from different producer runs just because their generation code matches.
    runs = {}
    prefix = expected_artifact_name()
    for page in pages:
        for artifact in page['artifacts']:
            for bundle in BUNDLES:
                name = prefix + '-' + bundle
                if eligible_artifact(artifact, name, source_sha, repository_id):
                    run = runs.setdefault(artifact['workflow_run']['id'], {})
                    previous = run.get(bundle)
                    if previous is None or (artifact['created_at'],
                                            artifact['id']) > (
                                                previous['created_at'],
                                                previous['id']):
                        run[bundle] = artifact
                    break
    complete = [run for run in runs.values() if set(run) == set(BUNDLES)]
    if not complete:
        return None
    newest = max(
        complete,
        key=lambda run: max(
            (item['created_at'], item['id']) for item in run.values()))
    return {'bundles': {bundle: newest[bundle]['id'] for bundle in BUNDLES}}


def reference_id(path):
    reference = json.loads(path.read_text())
    if (not isinstance(reference, dict) or set(reference) != {'artifact_id'} or
            type(reference['artifact_id']) is not int or
            reference['artifact_id'] <= 0):
        raise ValueError(
            'Expected a runtime reference with a positive artifact_id')
    return reference['artifact_id']


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)

    fingerprint_parser = subparsers.add_parser('fingerprint')
    fingerprint_parser.add_argument('paths', nargs='+', type=Path)

    select_parser = subparsers.add_parser('select-producer')
    select_parser.add_argument('--source-sha', required=True)

    shared_parser = subparsers.add_parser('select-artifact')
    bundles_parser = subparsers.add_parser('select-bundles')
    bundle_parser = subparsers.add_parser('validate-bundle')
    bundle_parser.add_argument('--bundle', choices=BUNDLES, required=True)
    bundle_parser.add_argument('--reference', type=Path, required=True)
    selection_parser = subparsers.add_parser('reference-bundles')
    selection_parser.add_argument('path', type=Path)
    selection_parser.add_argument('--argo-version', default='')
    selection_parser.add_argument('--db-type', default='')
    selection_parser.add_argument(
        '--modelcar', choices=('true', 'false'), default='false')
    validate_parser = subparsers.add_parser('validate-reference')
    for command_parser in (shared_parser, validate_parser, bundles_parser,
                           bundle_parser):
        command_parser.add_argument('--source-sha', required=True)
        command_parser.add_argument('--repository-id', type=int, required=True)
    validate_parser.add_argument('--reference', type=Path, required=True)
    reference_parser = subparsers.add_parser('reference-id')
    reference_parser.add_argument('path', type=Path)

    args = parser.parse_args()
    try:
        if args.command == 'fingerprint':
            print(fingerprint_files(args.paths))
        elif args.command == 'reference-bundles':
            reference = bundle_reference(args.path)
            for bundle in selected_bundles(args.argo_version, args.db_type,
                                           args.modelcar == 'true'):
                print(f'{bundle} {reference[bundle]}')
        elif args.command == 'select-bundles':
            selected = select_bundles(
                json.load(sys.stdin), args.source_sha, args.repository_id)
            if selected:
                print(json.dumps(selected))
        elif args.command == 'validate-bundle':
            artifact = json.load(sys.stdin)
            reference = bundle_reference(args.reference)
            if (not eligible_artifact(
                    artifact,
                    expected_artifact_name() + '-' + args.bundle,
                    args.source_sha, args.repository_id) or
                    artifact['id'] != reference[args.bundle]):
                raise ValueError(
                    'Shared runtime bundle metadata failed validation')
            print(artifact['id'])
        elif args.command == 'reference-id':
            print(reference_id(args.path))
        elif args.command == 'select-artifact':
            selected = select_artifact(
                json.load(sys.stdin), expected_artifact_name(), args.source_sha,
                args.repository_id)
            if selected:
                print(json.dumps({'artifact_id': selected['id']}))
        elif args.command == 'validate-reference':
            artifact = json.load(sys.stdin)
            if (not eligible_artifact(artifact, expected_artifact_name(),
                                      args.source_sha, args.repository_id) or
                    artifact['id'] != reference_id(args.reference)):
                raise ValueError(
                    'Shared runtime artifact metadata failed validation')
            print(artifact['id'])
        else:
            run_id = select_producer_run_id(
                json.load(sys.stdin), args.source_sha)
            if run_id is not None:
                print(run_id)
    except (KeyError, OSError, TypeError, ValueError,
            json.JSONDecodeError) as error:
        print(
            f'Cannot process runtime base-image artifacts: {error}',
            file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

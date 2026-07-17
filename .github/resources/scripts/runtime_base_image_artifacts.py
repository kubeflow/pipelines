#!/usr/bin/env python3
"""Fingerprint runtime-image generation and select a trusted producer run."""

import argparse
import hashlib
import json
import sys
from pathlib import Path
from typing import Any, Iterable, Optional


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
        if not isinstance(artifact, dict) or artifact.get('expired') is not False:
            continue
        workflow_run = artifact.get('workflow_run')
        if not isinstance(workflow_run, dict):
            continue

        is_current_source = workflow_run.get('head_sha') == source_sha
        head_repository_id = workflow_run.get('head_repository_id')
        is_trusted_master = (
            workflow_run.get('head_branch') == 'master'
            and isinstance(head_repository_id, int)
            and head_repository_id == workflow_run.get('repository_id')
        )
        if is_current_source or is_trusted_master:
            candidates.append(artifact)

    if not candidates:
        return None

    newest = max(candidates, key=lambda artifact: artifact.get('created_at', ''))
    run_id = newest['workflow_run'].get('id')
    if not isinstance(run_id, int):
        raise ValueError('Selected artifact is missing an integer workflow run ID')
    return run_id


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)

    fingerprint_parser = subparsers.add_parser('fingerprint')
    fingerprint_parser.add_argument('paths', nargs='+', type=Path)

    select_parser = subparsers.add_parser('select-producer')
    select_parser.add_argument('--source-sha', required=True)

    args = parser.parse_args()
    try:
        if args.command == 'fingerprint':
            print(fingerprint_files(args.paths))
        else:
            run_id = select_producer_run_id(json.load(sys.stdin), args.source_sha)
            if run_id is not None:
                print(run_id)
    except (KeyError, OSError, TypeError, ValueError, json.JSONDecodeError) as error:
        print(f'Cannot process runtime base-image artifacts: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

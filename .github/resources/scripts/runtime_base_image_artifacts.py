#!/usr/bin/env python3
"""Fingerprint runtime-image generation and select a trusted producer run."""

import argparse
import hashlib
import json
import sys
from pathlib import Path
from typing import Any, Iterable, Optional

QUEUE_BRANCH_PREFIX = 'gh-readonly-queue/release-2.18/'
QUEUE_PRODUCER_WORKFLOW = '.github/workflows/runtime-base-images-merge-group.yml'


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


def select_producer_run_id(payload: Any,
                           source_sha: str,
                           require_source_sha: bool = False,
                           queue_runs: Any = None) -> Optional[int]:
    """Return a matching producer, requiring queue provenance for merge groups."""
    if not isinstance(payload, dict):
        raise ValueError('Expected an artifact API response object')
    artifacts = payload.get('artifacts')
    if not isinstance(artifacts, list):
        raise ValueError('Artifact API response must contain an artifacts list')
    if require_source_sha:
        if (not isinstance(queue_runs, dict) or
                not isinstance(queue_runs.get('workflow_runs'), list)):
            raise ValueError(
                'Queue producer API response must contain workflow runs')
        runs_by_id = {
            run['id']: run
            for run in queue_runs['workflow_runs']
            if isinstance(run, dict) and isinstance(run.get('id'), int)
        }

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
        is_upstream_source = (
            isinstance(head_repository_id, int) and
            head_repository_id == workflow_run.get('repository_id'))
        is_trusted_master = (
            workflow_run.get('head_branch') == 'master' and is_upstream_source)
        head_branch = workflow_run.get('head_branch')
        is_trusted_queue = (
            is_current_source and is_upstream_source and
            isinstance(head_branch, str) and
            head_branch.startswith(QUEUE_BRANCH_PREFIX))
        if is_trusted_queue:
            run = runs_by_id.get(
                workflow_run.get('id')) if require_source_sha else None
            repository = run.get('repository') if isinstance(run,
                                                             dict) else None
            head_repository = run.get('head_repository') if isinstance(
                run, dict) else None
            run_path = run.get('path') if isinstance(run, dict) else None
            is_trusted_queue = (
                isinstance(repository, dict) and
                isinstance(head_repository, dict) and
                isinstance(run_path, str) and
                run_path.split('@', 1)[0] == QUEUE_PRODUCER_WORKFLOW and
                run.get('event') == 'merge_group' and
                run.get('head_sha') == source_sha and
                run.get('head_branch') == head_branch and
                run.get('status') == 'completed' and
                run.get('conclusion') == 'success' and
                isinstance(repository.get('id'), int) and
                repository['id'] == workflow_run.get('repository_id') and
                head_repository.get('id') == repository['id'])
        if (is_trusted_queue if require_source_sha else
            (is_current_source or is_trusted_master)):
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


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)

    fingerprint_parser = subparsers.add_parser('fingerprint')
    fingerprint_parser.add_argument('paths', nargs='+', type=Path)

    select_parser = subparsers.add_parser('select-producer')
    select_parser.add_argument('--source-sha', required=True)
    select_parser.add_argument('--require-source-sha', action='store_true')
    select_parser.add_argument('--queue-runs-file', type=Path)

    args = parser.parse_args()
    try:
        if args.command == 'fingerprint':
            print(fingerprint_files(args.paths))
        else:
            queue_runs = (
                json.loads(args.queue_runs_file.read_text())
                if args.queue_runs_file else None)
            run_id = select_producer_run_id(
                json.load(sys.stdin), args.source_sha, args.require_source_sha,
                queue_runs)
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

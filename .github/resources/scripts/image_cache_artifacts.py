#!/usr/bin/env python3
"""Select an exact image-cache artifact from eligible repository provenance."""

import argparse
from datetime import datetime
from datetime import timezone
import json
import sys
from typing import Any, Optional


def positive_integer(value: Any) -> bool:
    return type(value) is int and value > 0


def select_artifact(payload: Any, name: str, head_repository_id: int,
                    head_branch: str) -> Optional[dict[str, int]]:
    """Select from the current repository's paginated artifacts API
    response."""
    if not name or not positive_integer(head_repository_id) or not head_branch:
        raise ValueError(
            'Provide a cache name, positive head repository ID and head branch')
    if not isinstance(payload, list):
        raise ValueError(
            'Expected paginated artifact responses from gh api --paginate --slurp'
        )

    candidates = []
    for page in payload:
        if not isinstance(page, dict) or not isinstance(
                page.get('artifacts'), list):
            raise ValueError(
                'Each artifact response must contain an artifacts list')
        for artifact in page['artifacts']:
            if not isinstance(artifact, dict):
                continue
            if artifact.get('name') != name or artifact.get(
                    'expired') is not False:
                continue
            if not positive_integer(artifact.get('id')):
                continue
            run = artifact.get('workflow_run')
            if not isinstance(run, dict) or not all(
                    positive_integer(run.get(field))
                    for field in ('id', 'repository_id', 'head_repository_id')):
                continue
            same_head_branch = (
                run['head_repository_id'] == head_repository_id and
                run.get('head_branch') == head_branch)
            trusted_master = (
                run['head_repository_id'] == run['repository_id'] and
                run.get('head_branch') == 'master')
            if not (same_head_branch or trusted_master):
                continue
            created_at = artifact.get('created_at')
            if not isinstance(created_at, str):
                continue
            try:
                created = datetime.strptime(
                    created_at,
                    '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=timezone.utc)
            except ValueError:
                continue
            candidates.append((created, artifact['id'], run['id']))
    if not candidates:
        return None
    _, artifact_id, run_id = max(candidates)
    return {'artifact_id': artifact_id, 'run_id': run_id}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--name', required=True)
    parser.add_argument('--head-repository-id', type=int, required=True)
    parser.add_argument('--head-branch', required=True)
    args = parser.parse_args()
    try:
        selection = select_artifact(
            json.load(sys.stdin), args.name, args.head_repository_id,
            args.head_branch)
        print(json.dumps(selection or {}))
        return 0
    except (OSError, TypeError, ValueError) as error:
        print(f'Cannot select image-cache artifact: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())

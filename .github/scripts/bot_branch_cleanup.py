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
"""Delete the branch of a bot pull request that closed without merging.

Repository auto-delete covers merged pull requests only, so Dependabot
and Copybara branches whose pull requests are closed unmerged stay
behind. The branch is deleted only if no open pull request uses it as
its head or base, and the deletion is conditional on the branch still
pointing at the closed pull request's head, so a newer push survives. A
deleted branch can be restored with the "Restore branch" button on the
pull request.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import urllib.parse

# Each bot is matched by login and by its own branch prefix, so only branches
# that bot created are ever deleted.
BOT_BRANCH_PREFIXES = {
    'dependabot[bot]': 'dependabot/',
    'copybara-service[bot]': 'test_',
}
# Branch names are interpolated into API paths and printed to the workflow
# log, which parses `::` commands, so only plain ref characters are accepted.
BRANCH_PATTERN = re.compile(r'[A-Za-z0-9][A-Za-z0-9._/+-]*')
SHA_PATTERN = re.compile(r'[0-9a-f]{40}')
ZERO_SHA = '0' * 40
# updateRefs applies the update only if the ref still points at beforeOid;
# an afterOid of all zeros deletes the ref.
DELETE_REF_IF_UNCHANGED = """
mutation($repositoryId: ID!, $name: GitRefname!, $beforeOid: GitObjectID!,
         $afterOid: GitObjectID!) {
  updateRefs(input: {
    repositoryId: $repositoryId,
    refUpdates: [{name: $name, beforeOid: $beforeOid, afterOid: $afterOid}]
  }) { clientMutationId }
}
"""


class NotFoundError(Exception):
    """The requested GitHub resource does not exist."""


def is_cleanup_candidate(
    author: str, head_ref: str, head_repo: str, repo: str, merged: bool
) -> bool:
    """Return whether the closed pull request left a bot branch behind."""
    prefix = BOT_BRANCH_PREFIXES.get(author)
    return (
        prefix is not None
        and not merged
        and head_repo == repo
        and head_ref.startswith(prefix)
    )


def is_safe_branch_name(name: str) -> bool:
    """Return whether name is a plain branch name safe to use in a path."""
    return (
        BRANCH_PATTERN.fullmatch(name) is not None
        and '..' not in name
        and '//' not in name
        and not name.endswith(('/', '.', '.lock'))
    )


def gh_api(*args: str):
    """Call the GitHub API through gh and return the decoded JSON body.

    Raises NotFoundError for HTTP 404 so callers can treat a missing
    branch or pull request as a distinct outcome.
    """
    result = subprocess.run(
        ['gh', 'api', *args],
        capture_output=True,
        text=True,
        timeout=30,
    )
    if result.returncode != 0:
        if 'HTTP 404' in result.stderr:
            raise NotFoundError(args[-1])
        raise RuntimeError(
            f'gh api {" ".join(args)} failed: {result.stderr.strip()}'
        )
    return json.loads(result.stdout) if result.stdout.strip() else None


def branch_path(repo: str, branch: str) -> str:
    # `git/ref` matches one branch exactly; `git/refs` would also return
    # branches that share the prefix.
    return f'repos/{repo}/git/ref/heads/{urllib.parse.quote(branch)}'


def branch_tip(repo: str, branch: str) -> str | None:
    """Return the branch's current commit SHA, or None if it is gone."""
    try:
        return gh_api(branch_path(repo, branch))['object']['sha']
    except NotFoundError:
        return None


def has_open_pull_request(repo: str, **query: str) -> bool:
    """Return whether any open pull request matches the head/base query."""
    args = ['--method', 'GET', f'repos/{repo}/pulls']
    for key, value in {'state': 'open', 'per_page': '1', **query}.items():
        args += ['-f', f'{key}={value}']
    return bool(gh_api(*args))


def delete_branch_if_unchanged(
    repo: str, repo_id: str, branch: str, head_sha: str
) -> bool:
    """Delete the branch only if it still points at head_sha.

    Returns True if the branch was deleted and False if it is already
    gone or moved, so a push racing the cleanup always survives.
    """
    try:
        gh_api(
            'graphql',
            '-f',
            f'query={DELETE_REF_IF_UNCHANGED}',
            '-f',
            f'repositoryId={repo_id}',
            '-f',
            f'name=refs/heads/{branch}',
            '-f',
            f'beforeOid={head_sha}',
            '-f',
            f'afterOid={ZERO_SHA}',
        )
    except RuntimeError:
        # A rejected update does not say why, so read the branch again:
        # gone or moved is an expected outcome, anything else is an error.
        if branch_tip(repo, branch) != head_sha:
            return False
        raise
    return True


def cleanup(
    repo: str,
    pr_number: str,
    author: str,
    branch: str,
    head_sha: str,
    head_repo: str,
    merged: bool,
) -> str:
    """Delete the closed pull request's bot branch if it is safe to do so.

    Returns a one-line description of the outcome for the workflow log.
    """
    if not is_cleanup_candidate(author, branch, head_repo, repo, merged):
        return 'Not a closed, unmerged bot pull request; nothing to do.'
    if not is_safe_branch_name(branch):
        raise ValueError('Branch name contains unexpected characters.')
    if not (pr_number.isdigit() and SHA_PATTERN.fullmatch(head_sha)):
        raise ValueError('Pull request number and head SHA are malformed.')

    # The event may be stale: the pull request could have been reopened.
    pull = gh_api(f'repos/{repo}/pulls/{pr_number}')
    if pull['state'] != 'closed' or pull['merged']:
        return f'#{pr_number} is no longer closed and unmerged; kept {branch}.'

    tip = branch_tip(repo, branch)
    if tip is None:
        return f'{branch} is already deleted.'
    if tip != head_sha:
        return f'{branch} moved after #{pr_number} closed; kept it.'

    owner = repo.split('/')[0]
    if has_open_pull_request(repo, head=f'{owner}:{branch}'):
        return f'{branch} is the head of an open pull request; kept it.'
    if has_open_pull_request(repo, base=branch):
        return f'{branch} is the base of an open pull request; kept it.'

    repo_id = pull['base']['repo']['node_id']
    if not delete_branch_if_unchanged(repo, repo_id, branch, head_sha):
        return f'{branch} moved or was deleted during cleanup; left it.'
    return (
        f'Deleted {branch} at {head_sha} (#{pr_number} closed without '
        'merging). Restore it from the pull request page if needed.'
    )


def main() -> int:
    """Handle one pull_request_target closed event from the environment."""
    print(
        cleanup(
            repo=os.environ['GH_REPO'],
            pr_number=os.environ['PR_NUMBER'],
            author=os.environ['PR_AUTHOR'],
            branch=os.environ['HEAD_REF'],
            head_sha=os.environ['HEAD_SHA'],
            head_repo=os.environ['HEAD_REPO'],
            # Anything but an explicit `false` keeps the branch.
            merged=os.environ['PR_MERGED'] != 'false',
        )
    )
    return 0


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as error:
        print(f'Bot branch cleanup failed: {error}', file=sys.stderr)
        raise SystemExit(1)

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
"""Apply the `ready` label when an OWNERS approver comments `/ready`."""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

import yaml
from kubeflow_membership import LOGIN_PATTERN, _UniqueKeySafeLoader

READY_LABEL = 'ready'
# The command must be alone on its line so prose such as "is this /ready?"
# does not admit an issue.
READY_COMMAND = re.compile(r'^[ \t]*/ready[ \t\r]*$', re.MULTILINE)


def has_ready_command(comment_body: str) -> bool:
    """Return whether the comment contains `/ready` on its own line."""
    return READY_COMMAND.search(comment_body) is not None


def load_owners_approvers(root: Path) -> set[str]:
    """Return lowercase `approvers` from every OWNERS file under root.

    Raises RuntimeError on malformed files so a parsing problem fails
    the workflow instead of being reported to the commenter as a denial.
    """
    approvers: set[str] = set()
    for owners_path in sorted(root.rglob('OWNERS')):
        try:
            owners = yaml.load(
                owners_path.read_text(encoding='utf-8'),
                Loader=_UniqueKeySafeLoader,
            )
        except yaml.YAMLError as error:
            raise RuntimeError(f'Invalid YAML in {owners_path}') from error
        # Reviewer-only OWNERS files are valid and contribute no approvers.
        logins = (owners or {}).get('approvers', [])
        if not isinstance(logins, list) or not all(
            isinstance(login, str) and re.fullmatch(LOGIN_PATTERN, login)
            for login in logins
        ):
            raise RuntimeError(
                f'Expected a list of GitHub usernames for approvers in '
                f'{owners_path}'
            )
        approvers.update(login.lower() for login in logins)
    if not approvers:
        raise RuntimeError(f'No OWNERS approvers found under {root}')
    return approvers


def gh_api(*args: str) -> None:
    """Call the GitHub API through the authenticated gh CLI."""
    subprocess.run(
        ['gh', 'api', *args],
        check=True,
        stdout=subprocess.DEVNULL,
        timeout=30,
    )


def main() -> int:
    """Handle one issue comment event described by environment variables."""
    if not has_ready_command(os.environ['COMMENT_BODY']):
        print('No /ready command found.')
        return 0

    repo = os.environ['GH_REPO']
    issue_number = os.environ['ISSUE_NUMBER']
    comment_id = os.environ['COMMENT_ID']
    commenter = os.environ['COMMENTER']
    # Event IDs are interpolated into API paths, so accept only digits.
    if not (issue_number.isdigit() and comment_id.isdigit()):
        raise ValueError('Issue number and comment ID must be numeric.')
    approvers = load_owners_approvers(Path(os.environ.get('OWNERS_ROOT', '.')))
    reactions_path = f'repos/{repo}/issues/comments/{comment_id}/reactions'

    # Deny with a reaction rather than a reply so unauthorized commenters
    # cannot make the bot post comments.
    if commenter.lower() not in approvers:
        gh_api('--method', 'POST', reactions_path, '-f', 'content=-1')
        print(f'Denied /ready for {commenter}.')
        return 0

    gh_api(
        '--method',
        'POST',
        f'repos/{repo}/issues/{issue_number}/labels',
        '-f',
        f'labels[]={READY_LABEL}',
    )
    gh_api('--method', 'POST', reactions_path, '-f', 'content=+1')
    print(f'Applied {READY_LABEL} label for {commenter}.')
    return 0


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as error:
        print(f'/ready command failed: {error}', file=sys.stderr)
        raise SystemExit(1)

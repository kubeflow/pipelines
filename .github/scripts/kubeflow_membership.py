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

from __future__ import annotations

import os
import re
import subprocess
import sys

ACL_PATH = 'repos/kubeflow/internal-acls/contents/github-orgs/kubeflow/org.yaml?ref=master'
LOGIN_PATTERN = r'[A-Za-z0-9][A-Za-z0-9-]{0,38}'


def _parse_members(yaml_text: str) -> set[str]:
    # This is the ACL's expected layout, not a general YAML parser. Reject layout
    # changes instead of silently treating organization members as external.
    members: set[str] = set()
    in_orgs = False
    in_kubeflow_org = False
    current_list: str | None = None
    seen_lists: set[str] = set()
    complete = False

    for raw_line in yaml_text.splitlines():
        line = raw_line.replace('\t', '    ').split('#', 1)[0].rstrip()
        if not line.strip():
            continue
        if line == 'orgs:':
            in_orgs = True
            continue
        if not in_kubeflow_org:
            if in_orgs and line == '    kubeflow:':
                in_kubeflow_org = True
            continue

        if line == '        teams:':
            complete = True
            break
        if line in ('        admins:', '        members:'):
            current_list = line.strip().rstrip(':')
            seen_lists.add(current_list)
            continue
        if re.match(r'^        [a-z_]+:', line):
            current_list = None
            continue
        if current_list:
            login = line[len('        - '):] if line.startswith(
                '        - ') else ''
            if not re.fullmatch(LOGIN_PATTERN, login):
                raise RuntimeError(
                    'Unexpected Kubeflow ACL member entry. Verify the org.yaml layout before retrying.'
                )
            members.add(login.lower())
        elif not line.startswith('        '):
            break

    if not complete or seen_lists != {'admins', 'members'} or not members:
        raise RuntimeError(
            'Incomplete Kubeflow ACL membership data. Verify org.yaml contains admins and members before retrying.'
        )
    return members


def is_kubeflow_member(username: str) -> bool:
    """Check the authoritative ACL; raise on lookup errors, never infer non-membership.

    Requires the gh CLI, authenticated through GH_TOKEN or GITHUB_TOKEN in CI.
    """
    if not re.fullmatch(LOGIN_PATTERN, username):
        raise ValueError(
            'Provide a valid GitHub username for membership lookup.')
    try:
        result = subprocess.run(
            [
                'gh', 'api', ACL_PATH, '--header',
                'Accept: application/vnd.github.raw+json'
            ],
            check=True,
            capture_output=True,
            text=True,
            timeout=30,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise RuntimeError(
            'Could not fetch Kubeflow internal-acls membership. Check GitHub access and rerun the workflow.'
        ) from error
    return username.lower() in _parse_members(result.stdout)


def main() -> int:
    is_member = is_kubeflow_member(os.environ['PR_AUTHOR'])
    # Do not emit a negative result when fetching or parsing the ACL fails.
    with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
        output.write(f'is_member={str(is_member).lower()}\n')
    print(f'Kubeflow internal-acls membership: {is_member}')
    return 0


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as error:
        print(f'Membership lookup failed: {error}', file=sys.stderr)
        raise SystemExit(1)

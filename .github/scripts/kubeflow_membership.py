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

import yaml

ACL_PATH = 'repos/kubeflow/internal-acls/contents/github-orgs/kubeflow/org.yaml?ref=master'
LOGIN_PATTERN = r'[A-Za-z0-9][A-Za-z0-9-]{0,38}'


class _UniqueKeySafeLoader(yaml.SafeLoader):

    def construct_mapping(self, node, deep=False):
        mapping = super().construct_mapping(node, deep=deep)
        # SafeLoader flattens YAML merges before construction; reject overrides too.
        if len(mapping) != len(node.value):
            raise yaml.constructor.ConstructorError(
                None, None, 'Duplicate YAML mapping keys are not allowed',
                node.start_mark)
        return mapping


def _parse_members(yaml_text: str) -> set[str]:
    try:
        org = yaml.load(
            yaml_text, Loader=_UniqueKeySafeLoader)['orgs']['kubeflow']
        groups = [org['admins'], org['members']]
    except (yaml.YAMLError, KeyError, TypeError) as error:
        raise RuntimeError(
            'Invalid Kubeflow ACL membership data. Verify org.yaml contains Kubeflow admins and members before retrying.'
        ) from error

    if any(not isinstance(group, list) for group in groups) or any(
            not isinstance(login, str) or
            not re.fullmatch(LOGIN_PATTERN, login)
            for group in groups
            for login in group):
        raise RuntimeError(
            'Invalid Kubeflow ACL member lists. Expected lists of GitHub usernames; verify org.yaml before retrying.'
        )
    members = {login.lower() for group in groups for login in group}
    if not members:
        raise RuntimeError(
            'Empty Kubeflow ACL membership data. Verify org.yaml before retrying.'
        )
    return members


def is_kubeflow_member(username: str) -> bool:
    """Check the authoritative ACL, raising on lookup errors.

    Requires gh authenticated through GH_TOKEN or GITHUB_TOKEN.
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

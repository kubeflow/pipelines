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
"""Find a same-repository issue that can admit an external pull request."""

import json
import os
import re
import sys

CLOSING_REFERENCE = re.compile(
    r'^\s*(?:[-*]\s+)?(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+'
    r'(?:(?P<repository>[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+))?'
    r'#(?P<number>[1-9][0-9]*)\b',
    re.IGNORECASE,
)


def admission_issue(pr, repository, default_branch):
    """Return a linked issue, or an explicit release-branch closing
    reference."""
    for issue in pr['closingIssuesReferences']:
        issue_repository = issue['repository']
        full_name = (f"{issue_repository['owner']['login']}/"
                     f"{issue_repository['name']}")
        if full_name.casefold() == repository.casefold():
            return issue['number']

    # GitHub does not populate closingIssuesReferences from closing keywords
    # when a pull request targets a branch other than the default branch.
    if pr['baseRefName'] != default_branch:
        for line in (pr['body'] or '').splitlines():
            match = CLOSING_REFERENCE.match(line)
            if match and (not match['repository'] or
                          match['repository'].casefold()
                          == repository.casefold()):
                return int(match['number'])

    return None


if __name__ == '__main__':
    pull_request = json.load(sys.stdin)
    number = admission_issue(pull_request, os.environ['GH_REPO'],
                             os.environ['DEFAULT_BRANCH'])
    if number is not None:
        print(number)

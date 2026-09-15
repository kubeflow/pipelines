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
"""Fail when regeneration produced changes that are not already committed.

protoc-gen-go and protoc-gen-go-grpc stamp their own version into a comment at
the top of every file they emit, and those generator versions are resolved from
the Go module manifests: backend/api/Dockerfile derives protobuf_go_version and
protoc_gen_go_grpc_version from them with module_version. A Go module bump
therefore rewrites that comment in every generated file while leaving the
generated code identical, which reports the tree as out of date even though
nothing needs regenerating.

Changes confined to those version comments are reported and allowed. Anything
else -- a changed declaration, a new or deleted generated file -- still fails.
"""

import argparse
import re
import subprocess
import sys

# Emitted as "// \tprotoc-gen-go v1.36.12" by protoc-gen-go and as
# "// - protoc-gen-go-grpc v1.6.2" by protoc-gen-go-grpc.
GENERATOR_STAMP = re.compile(r'^//\s*(?:-\s*)?protoc-gen-go(?:-grpc)?\s+v\S+\s*$')

# Diff bookkeeping lines that are not themselves content changes.
DIFF_FILE_HEADER = re.compile(r'^(\+\+\+|---)\s')


def run(arguments: list[str]) -> str:
    completed = subprocess.run(
        arguments,
        check=True,
        capture_output=True,
        text=True,
    )
    return completed.stdout


def worktree_entries(status_output: str) -> list[tuple[str, str]]:
    """Return (status, path) for each entry of `git status --porcelain`."""
    entries = []
    for line in status_output.splitlines():
        if not line.strip():
            continue
        status, _, path = line[:2], line[2:3], line[3:]
        # Renames are reported as "old -> new"; the new path is what to inspect.
        if '->' in path:
            path = path.split('->', 1)[1].strip()
        entries.append((status.strip(), path.strip().strip('"')))
    return entries


def changed_lines(diff_output: str) -> list[str]:
    """Return added and removed content lines from a unified diff."""
    lines = []
    for line in diff_output.splitlines():
        if DIFF_FILE_HEADER.match(line):
            continue
        if line.startswith('+') or line.startswith('-'):
            lines.append(line[1:])
    return lines


def is_generator_stamp_only(diff_output: str) -> bool:
    lines = changed_lines(diff_output)
    if not lines:
        return False
    return all(GENERATOR_STAMP.match(line.strip()) for line in lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.parse_args()

    status_output = run(['git', 'status', '--porcelain'])
    entries = worktree_entries(status_output)
    if not entries:
        return 0

    stamp_only = []
    substantive = []
    for status, path in entries:
        # An untracked or deleted generated file is a real change regardless of
        # what its contents look like.
        if status in ('??', 'D', 'AD') or status.startswith('D'):
            substantive.append(path)
            continue
        # HEAD covers staged and unstaged changes; a plain `git diff` misses
        # anything the generation step already staged.
        diff_output = run(['git', 'diff', 'HEAD', '-U0', '--', path])
        if is_generator_stamp_only(diff_output):
            stamp_only.append(path)
        else:
            substantive.append(path)

    if substantive:
        print('ERROR: Generated files are out of date')
        print('Please regenerate using make clean all for api and '
              'kubernetes_platform')
        print('Changes found in the following files:')
        for path in substantive:
            print(f'  {path}')
        print('Diff of changes:')
        # HEAD so staged changes are shown; the previous check used a plain
        # `git diff` and printed nothing once generation had staged its output.
        sys.stdout.write(run(['git', 'diff', 'HEAD']))
        return 1

    print(f'Ignoring {len(stamp_only)} generated file(s) whose only change is '
          'the recorded generator version:')
    for path in stamp_only:
        print(f'  {path}')
    print('The generated code itself is unchanged, so no regeneration commit '
          'is required.')
    return 0


if __name__ == '__main__':
    sys.exit(main())

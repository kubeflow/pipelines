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
"""Synchronize Argo versions used by CI and its supporting documentation."""

import argparse
from pathlib import Path
import re
from typing import Dict, List, Tuple

SEMVER_PATTERN = re.compile(r'v\d+\.\d+\.\d+')
ARGO_VERSION_LINE = re.compile(r'^\s*argo_version:\s*(.*)$', re.MULTILINE)

WORKFLOW_PATHS = (
    Path('.github/workflows/e2e-test.yml'),
    Path('.github/workflows/api-server-tests.yml'),
)
CI_REFERENCE_PATHS = WORKFLOW_PATHS + (
    Path('.github/resources/runtime-base-images.txt'),
    Path('AGENTS.md'),
)


def _version_key(version: str) -> Tuple[int, int, int]:
    return tuple(int(part) for part in version[1:].split('.'))


def _read_version(path: Path) -> str:
    version = path.read_text(encoding='utf-8').strip()
    if SEMVER_PATTERN.fullmatch(version) is None:
        raise ValueError(
            f'{path} must contain one semantic version, found {version!r}')
    return version


def _workflow_versions(repo_root: Path) -> List[str]:
    versions = set()
    for relative_path in WORKFLOW_PATHS:
        contents = (repo_root / relative_path).read_text(encoding='utf-8')
        for value in ARGO_VERSION_LINE.findall(contents):
            versions.update(SEMVER_PATTERN.findall(value))
    return sorted(versions, key=_version_key)


def synchronized_contents(repo_root: Path) -> Dict[Path, str]:
    current_version = _read_version(repo_root / 'third_party/argo/VERSION')
    compatibility_version = _read_version(
        repo_root / 'third_party/argo/COMPATIBILITY_VERSION')
    if _version_key(compatibility_version) >= _version_key(current_version):
        raise ValueError('COMPATIBILITY_VERSION must be older than VERSION: '
                         f'{compatibility_version} >= {current_version}')
    existing_versions = _workflow_versions(repo_root)
    if len(existing_versions) != 2:
        raise ValueError(
            'expected exactly two Argo versions in CI matrices, found '
            f'{existing_versions}')

    replacements = {
        existing_versions[0]: compatibility_version,
        existing_versions[1]: current_version,
    }
    replacement_pattern = re.compile('|'.join(
        re.escape(version) for version in replacements))
    synchronized = {}
    for relative_path in CI_REFERENCE_PATHS:
        path = repo_root / relative_path
        contents = path.read_text(encoding='utf-8')
        contents = replacement_pattern.sub(
            lambda match: replacements[match.group(0)], contents)
        synchronized[path] = contents
    return synchronized


def sync(repo_root: Path, check: bool = False) -> List[Path]:
    changed_paths = []
    for path, expected_contents in synchronized_contents(repo_root).items():
        if path.read_text(encoding='utf-8') == expected_contents:
            continue
        changed_paths.append(path)
        if not check:
            path.write_text(expected_contents, encoding='utf-8')
    return changed_paths


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    changed_paths = sync(repo_root, check=args.check)
    if args.check and changed_paths:
        for path in changed_paths:
            print(
                f'Argo version reference is out of date: {path.relative_to(repo_root)}'
            )
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

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
"""Update the repository-wide Go compiler and pinned builder images."""

import argparse
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tempfile
from typing import Callable, Dict, Iterable, List, Optional, Set, Tuple

VERSION_PATTERN = re.compile(r'^1\.(\d+)(?:\.(\d+))?$')
EXACT_VERSION_PATTERN = re.compile(r'^1\.(\d+)\.(\d+)$')
DIGEST_PATTERN = re.compile(r'^sha256:[0-9a-f]{64}$')
GO_DIRECTIVE_PATTERN = re.compile(
    r'^go (?P<version>\d+\.\d+(?:\.\d+)?)$', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    r'^toolchain go(?P<version>\d+\.\d+\.\d+)$', re.MULTILINE)
TOOLCHAIN_DIRECTIVE_PATTERN = re.compile(r'^toolchain\s+(.+)$', re.MULTILINE)
TOOLCHAIN_LINE_PATTERN = re.compile(
    r'^toolchain go\d+\.\d+\.\d+\n?(?:\n)?', re.MULTILINE)
GO_IMAGE_PATTERN = re.compile(
    r'^(?P<prefix>FROM\s+)golang:'
    r'(?P<version>\d+\.\d+\.\d+)'
    r'(?P<flavor>-[^@\s]+)?@'
    r'(?P<digest>sha256:[0-9a-f]{64})'
    r'(?P<suffix>\s+AS\s+\w+)',
    re.IGNORECASE | re.MULTILINE,
)
GO_RUNTIME_REFERENCE_PATTERN = re.compile(r'golang:|dl\.google\.com/go/go')

SCANNED_RUNTIME_SUFFIXES = {'.sh', '.yaml', '.yml'}


def _parse_version(version: str) -> Tuple[int, int, int]:
    match = VERSION_PATTERN.fullmatch(version)
    if match is None:
        raise ValueError(f'invalid Go version {version!r}')
    return 1, int(match.group(1)), int(match.group(2) or 0)


def _parse_target_version(version: str) -> Tuple[int, int, int]:
    if EXACT_VERSION_PATTERN.fullmatch(version) is None:
        raise ValueError(
            f'Go version must use the exact stable form 1.X.Y, found '
            f'{version!r}')
    return _parse_version(version)


def _repository_paths(repo_root: Path) -> Set[Path]:
    try:
        output = subprocess.run(
            ('git', 'ls-files', '-z', '--cached', '--others',
             '--exclude-standard'),
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
        ).stdout
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        raise RuntimeError(
            f'could not discover repository files with git: {error}') from error
    return {Path(path) for path in output.split('\0') if path}


def _module_versions(
    contents: str,
    relative_path: Path,
) -> Tuple[Tuple[int, int, int], Optional[Tuple[int, int, int]]]:
    go_versions = GO_DIRECTIVE_PATTERN.findall(contents)
    if len(go_versions) != 1:
        raise ValueError(
            f'{relative_path} must contain exactly one go directive, found '
            f'{go_versions}')
    toolchain_directives = TOOLCHAIN_DIRECTIVE_PATTERN.findall(contents)
    toolchains = TOOLCHAIN_PATTERN.findall(contents)
    if len(toolchain_directives) > 1:
        raise ValueError(
            f'{relative_path} must contain at most one toolchain directive, '
            f'found {toolchain_directives}')
    if len(toolchains) != len(toolchain_directives):
        raise ValueError(
            f'{relative_path} contains an invalid toolchain directive: '
            f'{toolchain_directives}')
    go_version = _parse_version(go_versions[0])
    toolchain_version = _parse_version(toolchains[0]) if toolchains else None
    return go_version, toolchain_version


def _updated_module_contents(contents: str, relative_path: Path,
                             target: Tuple[int, int, int]) -> str:
    go_version, _ = _module_versions(contents, relative_path)
    language_floor = (go_version if go_version[:2] == target[:2] else
                      (target[0], target[1], 0))
    if language_floor > target:
        floor = '.'.join(str(part) for part in language_floor)
        compiler = '.'.join(str(part) for part in target)
        raise ValueError(
            f'{relative_path} requires Go {floor}, which exceeds the target '
            f'compiler {compiler}')
    language_version = '.'.join(str(part) for part in language_floor)
    updated = GO_DIRECTIVE_PATTERN.sub(
        f'go {language_version}', contents, count=1)
    updated = TOOLCHAIN_LINE_PATTERN.sub('', updated)
    if target[2] != 0:
        target_version = '.'.join(str(part) for part in target)
        updated = GO_DIRECTIVE_PATTERN.sub(
            lambda match: f'{match.group(0)}\n\ntoolchain go{target_version}',
            updated,
            count=1,
        )
    return updated.rstrip('\n') + '\n'


def _managed_dockerfiles(repo_root: Path,
                         repository_paths: Iterable[Path]) -> Dict[Path, str]:
    managed = {}
    unmanaged = []
    for relative_path in repository_paths:
        if not (relative_path.name.startswith('Dockerfile') or
                relative_path.suffix in SCANNED_RUNTIME_SUFFIXES):
            continue
        path = repo_root / relative_path
        if not path.exists():
            continue
        contents = path.read_text(encoding='utf-8', errors='ignore')
        if GO_RUNTIME_REFERENCE_PATTERN.search(contents) is None:
            continue
        matches = list(GO_IMAGE_PATTERN.finditer(contents))
        if not relative_path.name.startswith('Dockerfile') or len(matches) != 1:
            unmanaged.append(relative_path)
            continue
        managed[relative_path] = contents
    if unmanaged:
        paths = ', '.join(str(path) for path in sorted(unmanaged))
        raise ValueError(
            'unsupported Go runtime pins found; use exactly one digest-pinned '
            f'Golang builder image per Dockerfile: {paths}')
    if not managed:
        raise ValueError('no digest-pinned Golang builder images were found')
    return managed


def _inspect_manifest_digest(image: str) -> str:
    result = subprocess.run(
        (
            'docker',
            'buildx',
            'imagetools',
            'inspect',
            image,
            '--format',
            '{{json .Manifest}}',
        ),
        check=True,
        capture_output=True,
        text=True,
    )
    try:
        digest = json.loads(result.stdout)['digest']
    except (json.JSONDecodeError, KeyError, TypeError) as error:
        raise RuntimeError(
            f'docker buildx returned no manifest digest for {image}') from error
    if not isinstance(digest, str) or DIGEST_PATTERN.fullmatch(digest) is None:
        raise RuntimeError(f'docker buildx returned an invalid digest for '
                           f'{image}: {digest!r}')
    return digest


def resolve_docker_hub_digest(tag: str) -> str:
    images = (f'golang:{tag}', f'mirror.gcr.io/library/golang:{tag}')
    failures = []
    for image in images:
        try:
            return _inspect_manifest_digest(image)
        except FileNotFoundError as error:
            raise RuntimeError(
                'docker buildx is required to resolve Go builder image digests'
            ) from error
        except subprocess.CalledProcessError as error:
            detail = error.stderr.strip() or error.stdout.strip() or str(error)
            failures.append(f'{image}: {detail}')
    raise RuntimeError('could not resolve the Go builder image from Docker Hub '
                       f'or its configured mirror: {"; ".join(failures)}')


def synchronized_contents(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> Dict[Path, str]:
    target = _parse_target_version(target_version)
    paths = set(repository_paths or _repository_paths(repo_root))
    module_paths = sorted(
        path for path in paths
        if path.name == 'go.mod' and (repo_root / path).exists())
    if Path('go.mod') not in module_paths:
        raise ValueError('the repository root go.mod was not found')

    root_contents = (repo_root / 'go.mod').read_text(encoding='utf-8')
    root_go_version, root_toolchain_version = _module_versions(
        root_contents, Path('go.mod'))
    current = root_toolchain_version or root_go_version
    if target < current:
        current_version = '.'.join(str(part) for part in current)
        raise ValueError(
            f'refusing to downgrade Go from {current_version} to '
            f'{target_version}')

    expected_contents = {}
    for relative_path in module_paths:
        contents = (repo_root / relative_path).read_text(encoding='utf-8')
        expected_contents[relative_path] = _updated_module_contents(
            contents, relative_path, target)

    dockerfiles = _managed_dockerfiles(repo_root, paths)
    flavors = {
        match.group('flavor') or ''
        for contents in dockerfiles.values()
        for match in GO_IMAGE_PATTERN.finditer(contents)
    }
    digests = {}
    for flavor in sorted(flavors):
        tag = f'{target_version}{flavor}'
        digest = digest_resolver(tag)
        if DIGEST_PATTERN.fullmatch(digest) is None:
            raise ValueError(f'invalid digest resolved for golang:{tag}: '
                             f'{digest!r}')
        digests[flavor] = digest

    for relative_path, contents in dockerfiles.items():

        def replace_image(match: re.Match) -> str:
            flavor = match.group('flavor') or ''
            return (f"{match.group('prefix')}golang:{target_version}{flavor}@"
                    f"{digests[flavor]}{match.group('suffix')}")

        expected_contents[relative_path] = GO_IMAGE_PATTERN.sub(
            replace_image, contents)
    return expected_contents


def sync(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> List[Path]:
    expected_contents = synchronized_contents(
        repo_root,
        target_version,
        digest_resolver=digest_resolver,
        repository_paths=repository_paths,
    )
    changed_paths = []
    for relative_path, expected in expected_contents.items():
        path = repo_root / relative_path
        if path.read_text(encoding='utf-8') == expected:
            continue
        changed_paths.append(relative_path)

    temporary_paths = {}
    try:
        for relative_path in changed_paths:
            path = repo_root / relative_path
            descriptor, temporary_name = tempfile.mkstemp(
                dir=path.parent,
                prefix=f'.{path.name}.',
            )
            temporary_path = Path(temporary_name)
            temporary_paths[relative_path] = temporary_path
            os.fchmod(descriptor, stat.S_IMODE(path.stat().st_mode))
            with os.fdopen(descriptor, 'w', encoding='utf-8') as temporary_file:
                temporary_file.write(expected_contents[relative_path])
                temporary_file.flush()
                os.fsync(temporary_file.fileno())
        for relative_path in sorted(changed_paths):
            os.replace(temporary_paths[relative_path], repo_root / relative_path)
    finally:
        for temporary_path in temporary_paths.values():
            temporary_path.unlink(missing_ok=True)
    return sorted(changed_paths)


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Update every Go module and pinned builder image.')
    parser.add_argument('--version', required=True, help='Exact Go version X.Y.Z')
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    try:
        changed_paths = sync(repo_root, args.version)
    except (RuntimeError, ValueError) as error:
        print(f'error: {error}', file=sys.stderr)
        return 2

    if changed_paths:
        print('Updated Go version references:', flush=True)
        for path in changed_paths:
            print(f'  {path}', flush=True)
    else:
        print(
            f'Go version references are already current at {args.version}.',
            flush=True,
        )

    checker = repo_root / '.github/resources/scripts/go_version_consistency_test.py'
    return subprocess.run((sys.executable, str(checker)), cwd=repo_root).returncode


if __name__ == '__main__':
    raise SystemExit(main())

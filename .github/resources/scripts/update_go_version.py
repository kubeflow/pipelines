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
import time
from typing import Callable, Dict, Iterable, List, Optional, Set, Tuple

from go_version_metadata import (has_go_runtime_reference,
                                 top_level_module_matches)

DECIMAL_PATTERN = r'(?:0|[1-9][0-9]*)'
VERSION_PATTERN = re.compile(
    rf'^1\.({DECIMAL_PATTERN})(?:\.({DECIMAL_PATTERN}))?$')
EXACT_VERSION_PATTERN = re.compile(
    rf'^1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN}$')
DIGEST_PATTERN = re.compile(r'^sha256:[0-9a-f]{64}$')
GO_DIRECTIVE_PATTERN = re.compile(
    rf'^[ \t]*go[ \t]+(?P<version>1\.{DECIMAL_PATTERN}'
    rf'(?:\.{DECIMAL_PATTERN})?)'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
GO_DIRECTIVE_LINE_PATTERN = re.compile(
    r'^[ \t]*go(?:[ \t]+(.*?))?[ \t]*$', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    rf'^[ \t]*toolchain[ \t]+go(?P<version>1\.{DECIMAL_PATTERN}'
    rf'\.{DECIMAL_PATTERN})'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
TOOLCHAIN_DIRECTIVE_PATTERN = re.compile(
    r'^[ \t]*toolchain(?:[ \t]+(.*?))?[ \t]*$', re.MULTILINE)
TOOLCHAIN_LINE_PATTERN = re.compile(
    r'^[ \t]*toolchain[ \t]+go\d+\.\d+\.\d+'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*\n?(?:\n)?',
    re.MULTILINE)
GO_IMAGE_PATTERN = re.compile(
    r'^(?P<prefix>FROM\s+)golang:'
    r'(?P<version>\d+\.\d+\.\d+)'
    r'(?P<flavor>-[^@\s]+)?@'
    r'(?P<digest>sha256:[0-9a-f]{64})'
    r'(?P<suffix>\s+AS\s+\w+)',
    re.IGNORECASE | re.MULTILINE,
)
SCANNED_RUNTIME_SUFFIXES = {'.sh', '.yaml', '.yml'}
DIGEST_LOOKUP_ATTEMPTS = 3
DIGEST_LOOKUP_BACKOFF_SECONDS = (1, 2)
DIGEST_LOOKUP_SOURCE_COUNT = 2
DIGEST_LOOKUP_TIMEOUT_SECONDS = 20
DIGEST_VERIFICATION_BUDGET_SECONDS = 540


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
    go_directive_matches = top_level_module_matches(
        contents, GO_DIRECTIVE_LINE_PATTERN)
    go_directive_lines = [match.group(1) for match in go_directive_matches]
    go_versions = [
        match.group('version')
        for match in top_level_module_matches(contents, GO_DIRECTIVE_PATTERN)
    ]
    if len(go_directive_lines) != 1:
        raise ValueError(
            f'{relative_path} must contain exactly one go directive, found '
            f'{go_directive_lines}')
    if len(go_versions) != len(go_directive_lines):
        raise ValueError(
            f'{relative_path} contains an invalid go directive: '
            f'{go_directive_lines}')
    toolchain_directive_matches = top_level_module_matches(
        contents, TOOLCHAIN_DIRECTIVE_PATTERN)
    toolchain_directives = [
        match.group(1) for match in toolchain_directive_matches
    ]
    toolchains = [
        match.group('version')
        for match in top_level_module_matches(contents, TOOLCHAIN_PATTERN)
    ]
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
    go_match = top_level_module_matches(contents, GO_DIRECTIVE_PATTERN)[0]
    go_comment = go_match.group('comment') or ''
    toolchain_matches = top_level_module_matches(contents, TOOLCHAIN_PATTERN)
    toolchain_match = toolchain_matches[0] if toolchain_matches else None
    toolchain_comment = (
        toolchain_match.group('comment') if toolchain_match else '') or ''
    if go_version > target:
        floor = '.'.join(str(part) for part in go_version)
        compiler = '.'.join(str(part) for part in target)
        raise ValueError(
            f'{relative_path} requires Go {floor}, which exceeds the target '
            f'compiler {compiler}')
    language_floor = (go_version if go_version[:2] == target[:2] else
                      (target[0], target[1], 0))
    language_version = '.'.join(str(part) for part in language_floor)
    updated = GO_DIRECTIVE_PATTERN.sub(
        lambda _match: f'go {language_version}{go_comment}',
        contents,
        count=1,
    )
    updated = TOOLCHAIN_LINE_PATTERN.sub('', updated)
    if target[2] != 0:
        target_version = '.'.join(str(part) for part in target)
        updated = GO_DIRECTIVE_PATTERN.sub(
            lambda match: (f'{match.group(0)}\n\n'
                           f'toolchain go{target_version}{toolchain_comment}'),
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
        if not has_go_runtime_reference(relative_path, contents):
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
        timeout=DIGEST_LOOKUP_TIMEOUT_SECONDS,
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
    failures = {}
    for attempt in range(DIGEST_LOOKUP_ATTEMPTS):
        for image in images:
            try:
                return _inspect_manifest_digest(image)
            except FileNotFoundError as error:
                raise RuntimeError(
                    'docker buildx is required to resolve Go builder image '
                    'digests') from error
            except subprocess.CalledProcessError as error:
                detail = (error.stderr.strip() or error.stdout.strip() or
                          str(error))
                failures[image] = detail
            except subprocess.TimeoutExpired as error:
                failures[image] = f'timed out after {error.timeout} seconds'
        if attempt < DIGEST_LOOKUP_ATTEMPTS - 1:
            time.sleep(DIGEST_LOOKUP_BACKOFF_SECONDS[attempt])
    raise RuntimeError('could not resolve the Go builder image from Docker Hub '
                       'or its configured mirror after '
                       f'{DIGEST_LOOKUP_ATTEMPTS} attempts: ' + '; '.join(
                           f'{image}: {detail}'
                           for image, detail in failures.items()))


def verify_image_digests(
    repo_root: Path,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> None:
    paths = set(repository_paths or _repository_paths(repo_root))
    dockerfiles = _managed_dockerfiles(repo_root, paths)
    pins_by_tag: Dict[str, Dict[str, List[Path]]] = {}
    for relative_path, contents in dockerfiles.items():
        match = GO_IMAGE_PATTERN.search(contents)
        tag = match.group('version') + (match.group('flavor') or '')
        digest = match.group('digest')
        pins_by_tag.setdefault(tag, {}).setdefault(digest,
                                                   []).append(relative_path)

    worst_case_seconds = _digest_verification_worst_case_seconds(
        len(pins_by_tag))
    if worst_case_seconds > DIGEST_VERIFICATION_BUDGET_SECONDS:
        raise RuntimeError(
            'Go builder digest verification is configured for a worst-case '
            f'{worst_case_seconds} seconds, exceeding its '
            f'{DIGEST_VERIFICATION_BUDGET_SECONDS}-second budget')

    errors = []
    for tag, pins_by_digest in sorted(pins_by_tag.items()):
        if len(pins_by_digest) != 1:
            errors.append(
                f'golang:{tag} has inconsistent pinned digests: '
                f'{sorted(pins_by_digest)}')
            continue
        pinned_digest = next(iter(pins_by_digest))
        resolved_digest = digest_resolver(tag)
        if DIGEST_PATTERN.fullmatch(resolved_digest) is None:
            raise ValueError(f'invalid digest resolved for golang:{tag}: '
                             f'{resolved_digest!r}')
        if pinned_digest != resolved_digest:
            relative_paths = ', '.join(
                str(path) for path in sorted(pins_by_digest[pinned_digest]))
            errors.append(
                f'golang:{tag} resolves to {resolved_digest}, but '
                f'{relative_paths} pin {pinned_digest}')
    if errors:
        raise ValueError('Go builder image digest verification failed:\n  ' +
                         '\n  '.join(errors))


def _digest_verification_worst_case_seconds(tag_count: int) -> int:
    return tag_count * (
        DIGEST_LOOKUP_ATTEMPTS * DIGEST_LOOKUP_SOURCE_COUNT *
        DIGEST_LOOKUP_TIMEOUT_SECONDS +
        sum(DIGEST_LOOKUP_BACKOFF_SECONDS))


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
    original_contents = {}
    for relative_path, expected in expected_contents.items():
        path = repo_root / relative_path
        original = path.read_text(encoding='utf-8')
        if original == expected:
            continue
        changed_paths.append(relative_path)
        original_contents[relative_path] = original

    update_paths = {}
    rollback_paths = {}
    replaced_paths = []
    try:
        for relative_path in changed_paths:
            path = repo_root / relative_path
            update_paths[relative_path] = _temporary_replacement(
                path, expected_contents[relative_path], 'update')
            rollback_paths[relative_path] = _temporary_replacement(
                path, original_contents[relative_path], 'rollback')
        for relative_path in sorted(changed_paths):
            os.replace(update_paths[relative_path], repo_root / relative_path)
            replaced_paths.append(relative_path)
    except Exception as update_error:
        rollback_errors = []
        for relative_path in reversed(replaced_paths):
            try:
                os.replace(rollback_paths[relative_path],
                           repo_root / relative_path)
            except Exception as rollback_error:
                rollback_errors.append(f'{relative_path}: {rollback_error}')
        if rollback_errors:
            raise RuntimeError(
                f'failed to apply Go version update: {update_error}; '
                'also failed to restore original files: ' +
                '; '.join(rollback_errors)) from update_error
        raise RuntimeError(
            'failed to apply Go version update; restored original files: '
            f'{update_error}') from update_error
    finally:
        for temporary_path in (*update_paths.values(),
                               *rollback_paths.values()):
            temporary_path.unlink(missing_ok=True)
    return sorted(changed_paths)


def _temporary_replacement(path: Path, contents: str, purpose: str) -> Path:
    descriptor, temporary_name = tempfile.mkstemp(
        dir=path.parent,
        prefix=f'.{path.name}.{purpose}.',
    )
    temporary_path = Path(temporary_name)
    try:
        os.fchmod(descriptor, stat.S_IMODE(path.stat().st_mode))
        with os.fdopen(descriptor, 'w', encoding='utf-8') as temporary_file:
            descriptor = -1
            temporary_file.write(contents)
            temporary_file.flush()
            os.fsync(temporary_file.fileno())
    except Exception:
        if descriptor >= 0:
            os.close(descriptor)
        temporary_path.unlink(missing_ok=True)
        raise
    return temporary_path


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Update or verify every pinned Go builder image.')
    operation = parser.add_mutually_exclusive_group(required=True)
    operation.add_argument('--version', help='Exact Go version X.Y.Z')
    operation.add_argument(
        '--check-image-digests',
        action='store_true',
        help='Verify that every pinned digest resolves from its declared tag',
    )
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    if args.check_image_digests:
        try:
            verify_image_digests(repo_root)
        except (RuntimeError, ValueError) as error:
            print(f'error: {error}', file=sys.stderr)
            return 2
        print('Go builder image digests match their declared tags.')
        return 0

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

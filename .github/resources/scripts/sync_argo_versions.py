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
"""Synchronize the declared Argo runtime, module, CI, and documentation
pins."""

import argparse
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from typing import Dict, List, Optional, Tuple

SEMVER_PATTERN = re.compile(r'v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)')
ARGO_VERSION_LINE = re.compile(r'^\s*argo_version:\s*(.*)$', re.MULTILINE)
MODULE_PATTERN = re.compile(
    r'(?m)^[ \t]*(?:require[ \t]+)?(github\.com/argoproj/argo-workflows/v(\d+))[ \t]+'
    r'(v\d+\.\d+\.\d+)(?:[ \t]+//[^\n]*)?[ \t]*$')
VERSION_PATH = Path('third_party/argo/VERSION')
WORKFLOW_PATHS = (
    Path('.github/workflows/e2e-test.yml'),
    Path('.github/workflows/api-server-tests.yml'),
)
CI_REFERENCE_PATHS = WORKFLOW_PATHS + (
    Path('.github/resources/runtime-base-images.txt'),
    Path('AGENTS.md'),
)
MANIFEST_ROOT = Path('manifests/kustomize/third-party/argo')
MANIFEST_REFS = (
    (MANIFEST_ROOT / 'base/kustomization.yaml', 1),
    (MANIFEST_ROOT / 'installs/namespace/kustomization.yaml', 1),
    (MANIFEST_ROOT / 'installs/namespace/cluster-scoped/kustomization.yaml', 1),
    (MANIFEST_ROOT / 'installs/cluster/kustomization.yaml', 2),
)


def _version_key(version: str) -> Tuple[int, int, int]:
    return tuple(int(part) for part in version[1:].split('.'))


def _validate_version(version: str) -> str:
    if SEMVER_PATTERN.fullmatch(version) is None:
        raise ValueError(
            f'expected an exact stable Argo version vX.Y.Z, found {version!r}')
    return version


def _read_version(path: Path) -> str:
    return _validate_version(path.read_text(encoding='utf-8').strip())


def _workflow_versions(repo_root: Path) -> List[str]:
    versions = set()
    for relative_path in WORKFLOW_PATHS:
        contents = (repo_root / relative_path).read_text(encoding='utf-8')
        for value in ARGO_VERSION_LINE.findall(contents):
            tokens = re.findall(r'''(?<![A-Za-z0-9_.])v[0-9][^\s,\]\"']*''',
                                value.split('#', 1)[0])
            versions.update(_validate_version(token) for token in tokens)
    return sorted(versions, key=_version_key)


def synchronized_contents(repo_root: Path,
                          version: Optional[str] = None) -> Dict[Path, str]:
    current_version = _validate_version(version) if version else _read_version(
        repo_root / VERSION_PATH)
    compatibility_version = _read_version(
        repo_root / 'third_party/argo/COMPATIBILITY_VERSION')
    if _version_key(compatibility_version) >= _version_key(current_version):
        raise ValueError('COMPATIBILITY_VERSION must be older than VERSION: '
                         f'{compatibility_version} >= {current_version}')
    existing_versions = _workflow_versions(repo_root)
    if len(existing_versions) != 2:
        raise ValueError('expected exactly two Argo versions in CI matrices, '
                         f'found {existing_versions}')
    replacements = {
        existing_versions[0]: compatibility_version,
        existing_versions[1]: current_version
    }
    replacement_pattern = re.compile(r'(?<![A-Za-z0-9_.+@-])(?:' + '|'.join(
        re.escape(v) for v in replacements) + r')(?![A-Za-z0-9_.+@-])')
    synchronized = {}
    for relative_path in CI_REFERENCE_PATHS:
        path = repo_root / relative_path
        contents = path.read_text(encoding='utf-8')
        synchronized[path] = replacement_pattern.sub(
            lambda match: replacements[match.group(0)], contents)
    preload_path = repo_root / '.github/resources/runtime-base-images.txt'
    images = ('workflow-controller', 'argoexec')
    observed = [
        line.strip()
        for line in synchronized[preload_path].splitlines()
        if re.match(
            r'quay\.io/argoproj/(?:workflow-controller|argoexec)(?:[:@]|$)',
            line.strip())
    ]
    expected = [
        f'quay.io/argoproj/{image}:{release}' for image in images
        for release in (compatibility_version, current_version)
    ]
    if sorted(observed) != sorted(expected):
        raise ValueError(
            'runtime-base-images.txt must contain exactly the current and '
            'compatibility controller and executor images with stable tags')
    return synchronized


def _replace_pin(contents: str,
                 prefix: str,
                 version: str,
                 path: Path,
                 expected_count: int = 1) -> str:
    pattern = re.compile(f'({prefix})v\\d+\\.\\d+\\.\\d+' +
                         r'''(?=$|[\s\"'/&#])''')
    updated, count = pattern.subn(lambda match: match.group(1) + version,
                                  contents)
    if count != expected_count:
        raise ValueError(
            f'{path}: expected {expected_count} Argo version references, found {count}'
        )
    return updated


def _module_metadata(contents: str) -> Tuple[str, str]:
    matches = list(MODULE_PATTERN.finditer(contents))
    if len(matches) != 1:
        raise ValueError(
            'go.mod must contain exactly one stable Argo module requirement')
    match = matches[0]
    version = _validate_version(match.group(3))
    if int(match.group(2)) != _version_key(version)[0]:
        raise ValueError(
            'Argo module path and version must have the same major version')
    return match.group(1), version


def _module_contents(contents: str, version: str) -> str:
    module, current = _module_metadata(contents)
    if _version_key(current)[0] != _version_key(version)[0]:
        raise ValueError(
            f'Argo major upgrades require a code migration: {current} -> {version}'
        )
    match = MODULE_PATTERN.search(contents)
    return contents[:match.start(3)] + version + contents[match.end(3):]


def _has_module_sums(contents: str, module: str, version: str) -> bool:
    return all(
        re.search(
            rf'(?m)^{re.escape(module)} {re.escape(version + suffix)} h1:\S+$',
            contents) for suffix in ('', '/go.mod'))


def _tidied_module(repo_root: Path, expected_mod: str,
                   version: str) -> Dict[Path, str]:
    # An alternate manifest keeps failed Go resolution from changing tracked
    # files, while running at the repository root preserves local replacements.
    with tempfile.NamedTemporaryFile(
            mode='w',
            suffix='.mod',
            prefix='.argo-update-',
            dir=repo_root,
            encoding='utf-8',
            delete=False) as handle:
        handle.write(expected_mod)
        mod_path = Path(handle.name)
    sum_path = mod_path.with_suffix('.sum')
    try:
        sum_path.write_text(
            (repo_root / 'go.sum').read_text(encoding='utf-8'),
            encoding='utf-8')
        environment = dict(os.environ, GOWORK='off', GOTOOLCHAIN='local')
        subprocess.run(['go', 'mod', 'tidy', f'-modfile={mod_path}'],
                       cwd=repo_root,
                       env=environment,
                       check=True,
                       capture_output=True,
                       text=True)
        updated_mod = mod_path.read_text(encoding='utf-8')
        module, resolved_version = _module_metadata(updated_mod)
        if resolved_version != version:
            raise ValueError(
                f'Go resolved Argo {resolved_version}, expected {version}')
        compiler = re.compile(r'(?m)^(?:go|toolchain)\s+[^\n]+$')
        if compiler.findall(updated_mod) != compiler.findall(expected_mod):
            raise ValueError(
                'Argo update changes the Go compiler; update Go separately first'
            )
        updated_sum = sum_path.read_text(encoding='utf-8')
        if not _has_module_sums(updated_sum, module, version):
            raise ValueError(
                f'Go resolution did not produce checksums for {module}@{version}'
            )
        return {
            repo_root / 'go.mod': updated_mod,
            repo_root / 'go.sum': updated_sum
        }
    finally:
        mod_path.unlink(missing_ok=True)
        sum_path.unlink(missing_ok=True)


def planned_contents(repo_root: Path,
                     scope: str,
                     version: Optional[str] = None) -> Dict[Path, str]:
    if scope not in ('all', 'ci', 'manifests', 'backend', 'docs'):
        raise ValueError(f'unknown update scope: {scope}')
    if version is not None and scope != 'all':
        raise ValueError('--version requires --scope all')
    current = _read_version(repo_root / VERSION_PATH)
    target = _validate_version(version) if version is not None else current
    planned = synchronized_contents(repo_root,
                                    target) if scope in ('all', 'ci') else {}
    if scope == 'ci':
        return planned
    # Validate the supported module major even for a manifests-only update.
    module_path = repo_root / 'go.mod'
    updated_mod = _module_contents(
        module_path.read_text(encoding='utf-8'), target)
    if scope == 'all':
        planned[repo_root / VERSION_PATH] = target + '\n'
    if scope in ('all', 'backend'):
        planned[module_path] = updated_mod
    if scope in ('all', 'manifests'):
        pins = [
            (p, r'https://github\.com/argoproj/argo-workflows/[^\s?]+\?ref=',
             count) for p, count in MANIFEST_REFS
        ]
        deployment = MANIFEST_ROOT / 'base/workflow-controller-deployment-patch.yaml'
        pins += [
            (deployment, r'quay\.io/argoproj/workflow-controller:', 1),
            (deployment, r'quay\.io/argoproj/argoexec:', 1),
            (MANIFEST_ROOT / 'base/workflow-controller-configmap-patch.yaml',
             r'https://github\.com/argoproj/argo-workflows/blob/', 3)
        ]
        for relative_path, prefix, count in pins:
            path = repo_root / relative_path
            contents = planned[path] if path in planned else path.read_text(
                encoding='utf-8')
            planned[path] = _replace_pin(contents, prefix, target,
                                         relative_path, count)
    if scope in ('all', 'docs'):
        path = repo_root / 'third_party/argo/UPGRADE.md'
        planned[path] = _replace_pin(
            path.read_text(encoding='utf-8'), 'ARGO_TAG=', target, path)
    return planned


def sync(repo_root: Path,
         check: bool = False,
         scope: str = 'ci',
         version: Optional[str] = None) -> List[Path]:
    planned = planned_contents(repo_root, scope, version)
    missing_sums = []
    if scope in ('all', 'backend'):
        module_path = repo_root / 'go.mod'
        module, target = _module_metadata(planned[module_path])
        if check:
            sum_path = repo_root / 'go.sum'
            if not _has_module_sums(
                    sum_path.read_text(encoding='utf-8'), module, target):
                missing_sums.append(sum_path)
        else:
            planned.update(
                _tidied_module(repo_root, planned[module_path], target))
    changed_paths = [
        path for path, contents in planned.items()
        if path.read_text(encoding='utf-8') != contents
    ]
    if not check:
        for path in changed_paths:
            path.write_text(planned[path], encoding='utf-8')
    return changed_paths + missing_sums


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    parser.add_argument(
        '--scope',
        choices=('ci', 'all', 'manifests', 'backend', 'docs'),
        default='ci')
    parser.add_argument('--version')
    args = parser.parse_args()
    repo_root = Path(__file__).resolve().parents[3]
    try:
        changed_paths = sync(
            repo_root, check=args.check, scope=args.scope, version=args.version)
    except subprocess.CalledProcessError as error:
        print(
            f'Argo update failed: {error.stderr.strip() or error}',
            file=sys.stderr)
        return 1
    except (OSError, ValueError) as error:
        print(f'Argo update failed: {error}', file=sys.stderr)
        return 1
    if args.check and changed_paths:
        for path in changed_paths:
            print(
                f'Argo version reference is out of date: {path.relative_to(repo_root)}'
            )
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

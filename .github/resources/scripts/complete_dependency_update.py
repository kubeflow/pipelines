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
"""Complete a validated dependency proposal using trusted repository
generators.

Run this script from the trusted checkout, passing the candidate checkout as
--repo. Classification, bot identity, exact-head checks, and publication belong
in the caller. Never import a generator or execute a Makefile from --repo.
"""

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile

import sync_argo_versions as argo
import update_go_version as go

TRUSTED_ROOT = Path(__file__).resolve().parents[3]
GO_MODULE_PATHS = (
    Path('go.mod'),
    Path('api/go.mod'),
    Path('backend/api/tools/go.mod'),
    Path('kubernetes_platform/go.mod'),
    Path('test/tools/project-cleaner/go.mod'),
)
GO_OUTPUT_PATHS = GO_MODULE_PATHS + tuple(
    pin.path for pin in go.MANAGED_DOCKERFILES)
ARGO_OUTPUT_PATHS = (
    argo.CI_REFERENCE_PATHS +
    (argo.VERSION_PATH, Path('go.mod'), Path('go.sum')) +
    tuple(path for path, _ in argo.MANIFEST_REFS) +
    (argo.MANIFEST_ROOT / 'base/workflow-controller-deployment-patch.yaml',
     argo.MANIFEST_ROOT / 'base/workflow-controller-configmap-patch.yaml',
     Path('third_party/argo/UPGRADE.md')))
GH_AW_SOURCE = Path('.github/workflows/ai-analyzer.md')
GH_AW_SETUP = Path('.github/actions/setup-gh-aw/action.yml')
GH_AW_LOCK = Path('.github/workflows/ai-analyzer.lock.yml')
GH_AW_ACTIONS_LOCK = Path('.github/aw/actions-lock.json')
GH_AW_OUTPUT_PATHS = (GH_AW_SETUP, GH_AW_LOCK, GH_AW_ACTIONS_LOCK)
OUTPUT_PATHS = {
    'go': GO_OUTPUT_PATHS,
    'argo': ARGO_OUTPUT_PATHS,
    'gh-aw': GH_AW_OUTPUT_PATHS,
}
STABLE_VERSION = re.compile(r'v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)')
SETUP_PIN = re.compile(
    r'(?m)^\s*uses: github/gh-aw/actions/setup-cli@([0-9a-f]{40}) # (v\d+\.\d+\.\d+)$'
)
SETUP_VERSION = re.compile(r'(?m)^(\s*default: )(v\d+\.\d+\.\d+)$')
RUNTIME_PIN = re.compile(
    r'(?m)^\s*uses: github/gh-aw-actions/setup@([0-9a-f]{40}) # (v\d+\.\d+\.\d+)$'
)


def _read_regular(root, relative):
    path = root
    for part in relative.parts:
        path = path / part
        if path.is_symlink():
            raise ValueError(f'{relative}: symlinks are not supported')
    if not path.is_file():
        raise ValueError(f'{relative}: expected a regular file')
    return path.read_text(encoding='utf-8')


def _validate_paths(root, paths):
    for path in paths:
        _read_regular(root, path)


def _version_key(version):
    if not STABLE_VERSION.fullmatch(version):
        raise ValueError(
            f'expected an exact stable version vX.Y.Z: {version!r}')
    return tuple(map(int, version[1:].split('.')))


def complete_go(repo, version, images=()):
    """Preserve proposed immutable pins while completing the compiler
    inventory."""
    go._parse_exact_version(version)
    modules = go._module_paths(go._tracked_paths(repo))
    if set(modules) != set(GO_MODULE_PATHS):
        raise ValueError(
            'Go module inventory changed; update the trusted completion policy')
    _validate_paths(repo, GO_OUTPUT_PATHS + go.MANAGED_SETUP_GO_ACTIONS)
    metadata = [
        go._docker_metadata(_read_regular(repo, pin.path), pin)
        for pin in go.MANAGED_DOCKERFILES
    ]
    proposed = {}
    valid_tags = {version + pin.flavor for pin in go.MANAGED_DOCKERFILES}
    for image in images:
        tag, separator, digest = image.partition('@')
        if (not separator or tag not in valid_tags or
                not go.DIGEST_PATTERN.fullmatch(digest)):
            raise ValueError(f'unsupported proposed Go image {image!r}')
        if tag in proposed and proposed[tag] != digest:
            raise ValueError(f'conflicting proposed Go digests for {tag}')
        if not any(pin.version + pin.flavor == tag and pin.digest == digest
                   for pin in metadata):
            raise ValueError(
                f'proposed Go image is absent from the candidate: {image}')
        proposed[tag] = digest

    verified = set()

    def digest_resolver(tag):
        digest = proposed.get(tag)
        if digest is None:
            existing = {
                pin.digest
                for pin in metadata
                if pin.version + pin.flavor == tag
            }
            if len(existing) > 1:
                raise ValueError(
                    f'conflicting {tag} pins require an explicit proposed digest'
                )
            digest = next(
                iter(existing)) if existing else go.resolve_image_digest(tag)
        if digest not in verified:
            versions = go.resolve_image_versions(digest)
            for platform in go.REQUIRED_IMAGE_PLATFORMS:
                if versions.get(platform) != version:
                    raise ValueError(
                        f'golang@{digest} has Go {versions.get(platform)!r} '
                        f'on {platform}; expected {version}')
            verified.add(digest)
        return digest

    changed = go.update_repository(
        repo, version, digest_resolver=digest_resolver)
    go.check_repository(repo)
    return changed


def complete_argo(repo, version):
    target = _version_key(version)
    _validate_paths(
        repo,
        ARGO_OUTPUT_PATHS + (Path('third_party/argo/COMPATIBILITY_VERSION'),))
    current = _version_key(_read_regular(repo, argo.VERSION_PATH).strip())
    if target[0] != current[0] or target < current:
        raise ValueError('Argo completion supports same-major upgrades only')
    changed = argo.sync(repo, scope='all', version=version)
    if argo.sync(repo, scope='all', check=True):
        raise ValueError('Argo completion did not synchronize all references')
    return [path.relative_to(repo) for path in changed]


def _run(arguments, cwd):
    return subprocess.run(
        arguments,
        cwd=cwd,
        check=True,
        capture_output=True,
        text=True,
        timeout=600)


def _lock_metadata(contents):
    prefix = '# gh-aw-metadata: '
    first = contents.partition('\n')[0]
    if not first.startswith(prefix):
        raise ValueError('compiled workflow has no gh-aw metadata')
    return json.loads(first[len(prefix):])


def complete_gh_aw(repo, version, trusted_root=TRUSTED_ROOT):
    """Compile only trusted source; candidate code never enters the compiler
    cwd."""
    target = _version_key(version)
    _validate_paths(repo, GH_AW_OUTPUT_PATHS + (GH_AW_SOURCE,))
    source = _read_regular(trusted_root, GH_AW_SOURCE)
    if _read_regular(repo, GH_AW_SOURCE) != source:
        raise ValueError(
            'gh-aw completion requires unchanged trusted Markdown source')
    setup = _read_regular(repo, GH_AW_SETUP)
    pins = SETUP_PIN.findall(setup)
    versions = list(SETUP_VERSION.finditer(setup))
    if len(pins) != 1 or pins[0][1] != version or len(versions) != 1:
        raise ValueError(
            'expected one canonical gh-aw setup-cli pin matching the target')
    if _version_key(versions[0].group(2)) > target:
        raise ValueError('refusing to downgrade the gh-aw compiler')
    old_lock = _read_regular(trusted_root, GH_AW_LOCK)
    if _version_key(_lock_metadata(old_lock)['compiler_version']) > target:
        raise ValueError('refusing to downgrade the generated gh-aw workflow')
    expected_setup = SETUP_VERSION.sub(lambda match: match.group(1) + version,
                                       setup)
    with tempfile.TemporaryDirectory(prefix='kfp-gh-aw-') as directory:
        scratch = Path(directory)
        for relative, contents in ((GH_AW_SOURCE, source), (GH_AW_LOCK,
                                                            old_lock),
                                   (GH_AW_ACTIONS_LOCK,
                                    _read_regular(trusted_root,
                                                  GH_AW_ACTIONS_LOCK))):
            path = scratch / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents, encoding='utf-8')
        installed = _run(['gh', 'aw', '--version'], scratch)
        if (installed.stdout +
                installed.stderr).strip() != f'gh aw version {version}':
            raise ValueError(f'gh-aw compiler must be exactly {version}')
        _run(['git', 'init', '-q'], scratch)
        _run([
            'git', 'remote', 'add', 'origin',
            'https://github.com/kubeflow/pipelines.git'
        ], scratch)
        _run([
            'gh', 'aw', 'compile',
            str(GH_AW_SOURCE), '--action-mode', 'action', '--action-tag',
            version, '--no-check-update'
        ], scratch)
        if _read_regular(scratch, GH_AW_SOURCE) != source:
            raise ValueError(
                'gh-aw compiler unexpectedly changed its Markdown source')
        lock = _read_regular(scratch, GH_AW_LOCK)
        if _lock_metadata(lock)['compiler_version'] != version:
            raise ValueError(
                'generated workflow compiler version does not match target')
        action_lock = _read_regular(scratch, GH_AW_ACTIONS_LOCK)
        entry = json.loads(
            action_lock)['entries'][f'github/gh-aw-actions/setup@{version}']
        runtime_pins = RUNTIME_PIN.findall(lock)
        if (not runtime_pins or
                any(pin != (entry['sha'], version) for pin in runtime_pins)):
            raise ValueError(
                'generated gh-aw runtime pins do not match the compiler version'
            )
        expected = {
            GH_AW_SETUP: expected_setup,
            GH_AW_LOCK: lock,
            GH_AW_ACTIONS_LOCK: action_lock
        }
    changed = [
        path for path, contents in expected.items()
        if _read_regular(repo, path) != contents
    ]
    for path in changed:
        (repo / path).write_text(expected[path], encoding='utf-8')
    return changed


def complete(repo, kind, version, go_images=()):
    repo = Path(repo).resolve(strict=True)
    if kind != 'go' and go_images:
        raise ValueError('--go-image is valid only for Go completion')
    if kind == 'go':
        return complete_go(repo, version, go_images)
    if kind == 'argo':
        return complete_argo(repo, version)
    if kind == 'gh-aw':
        return complete_gh_aw(repo, version)
    raise ValueError(f'unsupported dependency completion kind {kind!r}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--kind', choices=tuple(OUTPUT_PATHS), required=True)
    parser.add_argument('--version', required=True)
    parser.add_argument(
        '--go-image', action='append', default=[], metavar='TAG@DIGEST')
    args = parser.parse_args()
    try:
        changed = complete(args.repo, args.kind, args.version, args.go_image)
    except (OSError, ValueError, KeyError, go.PolicyError,
            subprocess.SubprocessError) as error:
        print(f'Dependency completion failed: {error}', file=sys.stderr)
        return 1
    print(json.dumps({'changed_paths': [str(path) for path in changed]}))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

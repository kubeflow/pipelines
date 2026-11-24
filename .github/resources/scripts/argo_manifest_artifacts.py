#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Prepare pinned Argo sources once and install verified local CI resources."""

import argparse
import io
import json
import os
import re
import shutil
import signal
import subprocess
import tarfile
import tempfile
import time
from collections.abc import Callable, Collection, Sequence
from pathlib import Path, PurePosixPath

import shared_ci_images as shared
import sync_argo_versions as pins

ROOT = Path(__file__).resolve().parents[3]
UPSTREAM = 'https://github.com/argoproj/argo-workflows'
ARCHIVE = 'argo-manifests.tar'
METADATA = 'argo-manifests.json'
MAX_BYTES = 64 * 1024 * 1024
UPSTREAM_PATHS = (
    'manifests/base/workflow-controller',
    'manifests/base/crds/minimal',
    'manifests/namespace-install/workflow-controller-rbac',
    'manifests/cluster-install-no-crds/workflow-controller-rbac',
)


def versions(root: Path = ROOT) -> list[str]:
    return sorted(
        {
            pins._read_version(root / pins.VERSION_PATH),
            pins._read_version(root / 'third_party/argo/COMPATIBILITY_VERSION'),
        }
    )


def fetch_with_retry(
    command: Sequence[str],
    cwd: Path,
    attempts: int = 3,
    timeout: float = 180,
    delay: float = 10,
    before_attempt: Callable[[], None] | None = None,
) -> None:
    for attempt in range(1, attempts + 1):
        try:
            if before_attempt is not None:
                before_attempt()
            # Git starts transport/pack helpers. Terminate the whole fetch on
            # timeout before another attempt can reuse its repository files.
            with subprocess.Popen(
                command,
                cwd=cwd,
                stdin=subprocess.DEVNULL,
                env={**os.environ, 'GIT_TERMINAL_PROMPT': '0'},
                start_new_session=True,
            ) as process:
                try:
                    returncode = process.wait(timeout=timeout)
                except subprocess.TimeoutExpired:
                    try:
                        os.killpg(process.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    process.wait()
                    raise
                if returncode:
                    raise subprocess.CalledProcessError(returncode, command)
                return
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
            print(
                f'Argo fetch attempt {attempt}/{attempts} failed.', flush=True
            )
            if attempt == attempts:
                raise
            time.sleep(delay)


def archive_members(
    archive: tarfile.TarFile, allowed_versions: Collection[str] | None = None
) -> list[tarfile.TarInfo]:
    members = archive.getmembers()
    if (
        len(members) > 10000
        or sum(member.size for member in members) > MAX_BYTES
    ):
        raise ValueError('Argo manifest archive exceeds size limits')
    names = set()
    for member in members:
        path = PurePosixPath(member.name)
        if (
            path.is_absolute()
            or '..' in path.parts
            or not path.parts
            or member.name in names
            or not (member.isfile() or member.isdir())
        ):
            raise ValueError('Unsafe Argo manifest archive member')
        names.add(member.name)
        if allowed_versions is not None:
            if path.parts[0] not in allowed_versions or (
                len(path.parts) > 1 and path.parts[1] != 'manifests'
            ):
                raise ValueError('Unexpected Argo manifest archive path')
        elif path.parts[0] != 'manifests':
            raise ValueError('Unexpected upstream Argo archive path')
    if not any(member.isfile() for member in members):
        raise ValueError('Empty Argo manifest archive')
    return members


def extract(
    archive: tarfile.TarFile,
    destination: Path,
    allowed_versions: Collection[str] | None = None,
) -> None:
    # Validate the entire collection before writing, and never accept links.
    members = archive_members(archive, allowed_versions)
    for member in members:
        target = destination / member.name
        if member.isdir():
            target.mkdir(parents=True, exist_ok=True)
        else:
            target.parent.mkdir(parents=True, exist_ok=True)
            with (
                archive.extractfile(member) as source,
                target.open('wb') as output,
            ):
                shutil.copyfileobj(source, output)


def fetch_version(version: str, destination: Path) -> str:
    pins._validate_version(version)
    with tempfile.TemporaryDirectory() as temp:
        checkout = Path(temp)

        def reset_checkout() -> None:
            # A killed fetch can leave lockfiles. This repository is private to
            # this fetch, so each retry starts from an entirely fresh Git state.
            if (checkout / '.git').exists():
                shutil.rmtree(checkout / '.git')
            subprocess.run(['git', 'init', '-q'], cwd=checkout, check=True)

        fetch_with_retry(
            [
                'git',
                'fetch',
                '--depth=1',
                '--no-tags',
                UPSTREAM,
                f'refs/tags/{version}',
            ],
            checkout,
            before_attempt=reset_checkout,
        )
        commit = subprocess.check_output(
            ['git', 'rev-parse', 'FETCH_HEAD^{commit}'], cwd=checkout, text=True
        ).strip()
        if not re.fullmatch('[0-9a-f]{40}', commit):
            raise ValueError('Argo tag did not resolve to a commit')
        data = subprocess.check_output(
            ['git', 'archive', commit, *UPSTREAM_PATHS], cwd=checkout
        )
        with tarfile.open(fileobj=io.BytesIO(data)) as archive:
            extract(archive, destination)
        return commit


def prepare(
    output: Path, root: Path = ROOT, identity: dict | None = None
) -> None:
    identity = identity or shared.environment_identity()
    shared.validate_identity(identity)
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as temp:
        staging = Path(temp)
        commits = {}
        for version in versions(root):
            print(f'Preparing Argo {version}', flush=True)
            commits[version] = fetch_version(version, staging / version)
        with tarfile.open(output / ARCHIVE, 'w') as archive:
            for version in commits:
                archive.add(staging / version, arcname=version)
        shared.write_json(
            output / METADATA,
            {
                'version': 1,
                'identity': identity,
                'versions': commits,
                'sha256': shared.checksum(output / ARCHIVE),
            },
        )


def verify(
    directory: Path,
    state: dict,
    root: Path = ROOT,
    api: shared.GitHub | None = None,
    consumer: dict | None = None,
) -> dict:
    consumer = consumer or shared.environment_identity()
    shared.validate_state(state, consumer, state['architecture'])
    if directory.is_symlink() or not directory.is_dir():
        raise ValueError('Unsafe Argo artifact directory')
    metadata_path = directory / METADATA
    if metadata_path.is_symlink() or metadata_path.stat().st_size > 1024 * 1024:
        raise ValueError('Unsafe Argo artifact metadata')
    metadata = json.loads(metadata_path.read_text())
    if (
        metadata.get('version') != 1
        or metadata.get('identity') != state['identity']
        or set(metadata.get('versions', {})) != set(versions(root))
        or not all(
            re.fullmatch('[0-9a-f]{40}', commit)
            for commit in metadata['versions'].values()
        )
        or metadata.get('sha256') != shared.checksum(directory / ARCHIVE)
    ):
        raise ValueError(
            'Argo artifact source, version, attempt, or checksum mismatch'
        )
    with tarfile.open(directory / ARCHIVE) as archive:
        archive_members(archive, metadata['versions'])
    api = api or shared.GitHub(consumer['repository'])
    shared.validate_run(api.run(state['identity']['run_id']), state['identity'])
    return metadata


def install(directory: Path, version: str, root: Path = ROOT) -> None:
    """Localize every Argo resource; repeat deployment attempts are
    idempotent."""
    pins._validate_version(version)
    metadata = json.loads((directory / METADATA).read_text())
    if version not in metadata['versions'] or metadata[
        'sha256'
    ] != shared.checksum(directory / ARCHIVE):
        raise ValueError(
            'Selected Argo version is missing or its archive changed'
        )
    argo = root / pins.MANIFEST_ROOT
    vendor = argo / '.ci-argo'
    # Extraction always uses a fresh directory, avoiding preexisting symlink paths.
    with tempfile.TemporaryDirectory(dir=argo) as temp:
        staging = Path(temp)
        with tarfile.open(directory / ARCHIVE) as archive:
            extract(archive, staging, metadata['versions'])
        if vendor.is_symlink():
            raise ValueError('Unsafe local Argo source directory')
        if vendor.exists():
            shutil.rmtree(vendor)
        staging.rename(vendor)
    pattern = re.compile(
        r'(?m)^(\s*-\s+)(?:https://github\.com/argoproj/argo-workflows/|'
        r'(?:\.\./)*\.ci-argo/v\d+\.\d+\.\d+/)'
        r'(manifests/[^\s?]+)(?:\?ref=v\d+\.\d+\.\d+(?:&timeout=\d+s)?)?$'
    )
    planned = {}
    for relative, expected in pins.MANIFEST_REFS:
        path = root / relative

        def replacement(match: re.Match[str]) -> str:
            target = vendor / version / match[2]
            if not target.is_dir():
                raise ValueError(f'Missing pinned Argo resource: {match[2]}')
            return match[1] + os.path.relpath(target, path.parent)

        content, count = pattern.subn(replacement, path.read_text())
        if count != expected:
            raise ValueError(
                f'{relative}: expected {expected} Argo resource references, found {count}'
            )
        planned[path] = content
    deployment = argo / 'base/workflow-controller-deployment-patch.yaml'
    content = deployment.read_text()
    for image in ('workflow-controller', 'argoexec'):
        content = pins._replace_pin(
            content, rf'quay\.io/argoproj/{image}:', version, deployment
        )
    planned[deployment] = content
    for path, content in planned.items():
        path.write_text(content)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    producer = commands.add_parser('prepare')
    producer.add_argument('--output', type=Path, required=True)
    verifier = commands.add_parser('verify')
    verifier.add_argument('--directory', type=Path, required=True)
    verifier.add_argument('--state', type=Path, required=True)
    installer = commands.add_parser('install')
    installer.add_argument('--directory', type=Path, required=True)
    installer.add_argument('--version', required=True)
    args = parser.parse_args()
    if args.command == 'prepare':
        prepare(args.output)
    elif args.command == 'verify':
        verify(args.directory, json.loads(args.state.read_text()))
    else:
        install(args.directory, args.version)


if __name__ == '__main__':
    main()

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
"""Download immutable producer manifests as data, without checking out code."""

import argparse
import re
import shutil
import subprocess
import tarfile
import tempfile
from pathlib import Path, PurePosixPath


def extract_manifests(archive, destination, extra_paths=()):
    destination.mkdir(parents=True, exist_ok=False)
    count = 0
    with tarfile.open(archive) as source:
        for member in source:
            path = PurePosixPath(member.name)
            if path.is_absolute() or '..' in path.parts:
                raise ValueError('Unsafe producer archive path')
            # GitHub archives have one repository-prefix directory.
            if len(path.parts) < 2 or not (
                path.parts[1] == 'manifests'
                or str(PurePosixPath(*path.parts[1:])) in extra_paths
            ):
                continue
            if member.isdir():
                continue
            if not member.isfile():
                raise ValueError(
                    'Producer manifests must be regular files, not links'
                )
            target = destination.joinpath(*path.parts[1:])
            target.parent.mkdir(parents=True, exist_ok=True)
            with (
                source.extractfile(member) as content,
                target.open('xb') as output,
            ):
                shutil.copyfileobj(content, output)
            count += 1
    if not count:
        raise ValueError('Producer archive contains no manifests')


def download_manifests(repository, sha, destination, extra_paths=()):
    if not re.fullmatch(
        r'[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*', repository
    ):
        raise ValueError('Expected owner/repository')
    if not re.fullmatch(r'[0-9a-f]{40}', sha):
        raise ValueError('Expected full producer SHA')
    with tempfile.TemporaryDirectory() as directory:
        archive = Path(directory) / 'source.tar.gz'
        with archive.open('wb') as output:
            subprocess.run(
                ['gh', 'api', f'repos/{repository}/tarball/{sha}'],
                stdout=output,
                check=True,
            )
        extract_manifests(archive, destination, extra_paths=extra_paths)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repository', required=True)
    parser.add_argument('--sha', required=True)
    parser.add_argument('--destination', required=True, type=Path)
    args = parser.parse_args()
    download_manifests(args.repository, args.sha, args.destination)

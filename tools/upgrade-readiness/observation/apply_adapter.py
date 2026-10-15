# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Check or apply the optional observer to an exact, clean KFP 2.17.2
checkout."""

import argparse
from pathlib import Path
import shutil
import subprocess

SOURCE_REVISION = '2511cdbd74cd531c6633f2e094d235082a32917d'
PACKAGE = Path('backend/src/apiserver/readinessobservation')
SOURCES = ('recorder.go', 'recorder_test.go')


def git(source, *arguments):
    return subprocess.check_output(
        ['git', '-c', 'core.fsmonitor=false', *arguments],
        cwd=source,
        text=True,
        stderr=subprocess.PIPE,
        timeout=30).strip()


def apply(source, *, check_only=True):
    source = source.resolve()
    if Path(git(source, 'rev-parse', '--show-toplevel')).resolve() != source:
        raise ValueError('Supply the backend checkout root.')
    if git(source, 'rev-parse', 'HEAD') != SOURCE_REVISION:
        raise ValueError(
            'Observer adapter requires the exact pinned 2.17.2 source revision.'
        )
    if git(source, 'status', '--porcelain', '--untracked-files=all'):
        raise ValueError('Observer adapter requires a clean source checkout.')
    destination = source / PACKAGE
    if destination.exists() or destination.is_symlink():
        raise ValueError('Observer package destination must not already exist.')
    assets = Path(__file__).resolve().parent
    patch = assets / 'source-2.17.2.patch'
    git(source, 'apply', '--check', str(patch))
    if check_only:
        return
    destination.mkdir()
    try:
        for name in SOURCES:
            shutil.copyfile(assets / name, destination / name)
        git(source, 'apply', str(patch))
    except BaseException:
        for name in SOURCES:
            (destination / name).unlink(missing_ok=True)
        destination.rmdir()
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument(
        '--apply',
        action='store_true',
        help='Apply after validation; otherwise only check.')
    args = parser.parse_args()
    try:
        apply(args.source, check_only=not args.apply)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        reason = str(error) if isinstance(
            error,
            ValueError) else 'source validation or patch application failed'
        parser.exit(1, 'Observer adapter failed: ' + reason + '\n')
    print(('Applied' if args.apply else 'Validated') +
          ' source observation adapter for ' + SOURCE_REVISION +
          '; observation remains disabled until explicitly configured.')


if __name__ == '__main__':
    main()

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

import argparse
import ast
import os
from pathlib import Path
import subprocess
from typing import Iterable
import warnings

PYTHON_FEATURE_VERSION = 9


def repository_python_files(repository_root: Path) -> list[Path]:
    result = subprocess.run(
        [
            'git', '-C', str(repository_root), 'ls-files', '-z', '--cached',
            '--', '*.py'
        ],
        check=True,
        stdout=subprocess.PIPE,
    )
    paths = []
    for encoded_path in result.stdout.split(b'\0'):
        relative_path = os.fsdecode(encoded_path)
        path = repository_root / relative_path
        if relative_path and path.is_file():
            paths.append(path)
    return paths


def syntax_errors(paths: Iterable[Path]) -> list[tuple[Path, str, int, int]]:
    errors = []
    for path in paths:
        try:
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', SyntaxWarning)
                ast.parse(
                    path.read_bytes(),
                    filename=str(path),
                    feature_version=PYTHON_FEATURE_VERSION,
                )
        except (SyntaxError, ValueError) as error:
            message = error.msg if isinstance(error, SyntaxError) else str(error)
            errors.append((
                path,
                message,
                getattr(error, 'lineno', None) or 1,
                getattr(error, 'offset', None) or 1,
            ))
    return errors


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Check repository Python files for valid syntax.')
    parser.add_argument(
        '--repository-root',
        type=Path,
        default=Path(__file__).resolve().parents[3],
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    repository_root = args.repository_root.resolve()
    python_files = repository_python_files(repository_root)
    errors = syntax_errors(python_files)
    for path, message, line, column in errors:
        relative_path = path.relative_to(repository_root)
        print(f'{relative_path}:{line}:{column}: {message}')
    if errors:
        print(f'Found syntax errors in {len(errors)} Python file(s).')
        return 1
    print(f'Checked {len(python_files)} Python file(s).')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

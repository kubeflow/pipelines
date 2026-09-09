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
import json
import os
from pathlib import Path
import tempfile
from typing import Any


def _result_key(result: dict[str, Any]) -> str:
    """Returns a stable key without discarding SARIF result metadata."""
    return json.dumps(result, sort_keys=True, separators=(',', ':'))


def deduplicate_sarif(sarif: dict[str, Any]) -> int:
    """Removes structurally identical results within each SARIF run in place."""
    runs = sarif.get('runs')
    if not isinstance(runs, list):
        raise ValueError('SARIF document must contain a runs array')

    removed_results = 0
    for run_index, run in enumerate(runs):
        if not isinstance(run, dict):
            raise ValueError(f'SARIF run {run_index} must be an object')

        results = run.get('results')
        if results is None:
            continue
        if not isinstance(results, list):
            raise ValueError(
                f'SARIF run {run_index} results must be an array')

        unique_results = []
        seen_results = set()
        for result_index, result in enumerate(results):
            if not isinstance(result, dict):
                raise ValueError(
                    f'SARIF run {run_index} result {result_index} must be an '
                    'object'
                )
            result_key = _result_key(result)
            if result_key in seen_results:
                removed_results += 1
                continue
            seen_results.add(result_key)
            unique_results.append(result)
        run['results'] = unique_results

    return removed_results


def write_sarif(path: Path, sarif: dict[str, Any]) -> None:
    """Atomically writes a SARIF document to path."""
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary_file = None
    try:
        with tempfile.NamedTemporaryFile(
                mode='w',
                encoding='utf-8',
                dir=path.parent,
                prefix=f'.{path.name}.',
                suffix='.tmp',
                delete=False,
        ) as output_file:
            temporary_file = Path(output_file.name)
            json.dump(sarif, output_file, separators=(',', ':'))
            output_file.write('\n')
        os.replace(temporary_file, path)
    finally:
        if temporary_file is not None and temporary_file.exists():
            temporary_file.unlink()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Remove exact duplicate results from an OSV SARIF report.')
    parser.add_argument('--input', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    try:
        sarif = json.loads(args.input.read_text(encoding='utf-8'))
    except (OSError, json.JSONDecodeError) as error:
        raise RuntimeError(
            f'Unable to read SARIF input {args.input}: {error}') from error
    if not isinstance(sarif, dict):
        raise ValueError('SARIF document must be a JSON object')

    removed_results = deduplicate_sarif(sarif)
    write_sarif(args.output, sarif)
    print(f'Removed {removed_results} exact duplicate SARIF result(s)')


if __name__ == '__main__':
    main()

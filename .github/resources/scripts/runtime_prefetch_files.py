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
"""Bind reusable runtime downloads to an exact CI invocation and artifact."""

import json
import os
from pathlib import Path
import sys


def identity(artifact_id, required, allow_reference):
    keys = ('GITHUB_REPOSITORY', 'GITHUB_RUN_ID', 'GITHUB_RUN_ATTEMPT',
            'SOURCE_SHA', 'RUNTIME_ARGO_VERSION', 'RUNTIME_DB_TYPE',
            'LOAD_MODELCAR_FIXTURE')
    context = {key: os.environ.get(key, '') for key in keys}
    if not all(context[key] for key in keys[:4]):
        raise ValueError(
            'Prefetch requires repository, run, attempt and source SHA')
    return dict(
        context=context,
        artifact_id=artifact_id,
        required=required.splitlines(),
        allow_reference=allow_reference)


def main():
    command, directory, artifact_id, required, allow_reference = sys.argv[1:]
    root = Path(directory)
    marker = root / '.runtime-prefetch.json'
    try:
        expected = identity(artifact_id, required, allow_reference)
        if command == 'verify':
            record = json.loads(marker.read_text())
            if record['identity'] != expected:
                return 1
            files = record['files']
            names = expected['required']
            if allow_reference == 'true' and set(files) == {
                    'runtime-reference.json'
            }:
                names = ['runtime-reference.json']
            if set(files) != set(names):
                return 1
            for name, size in files.items():
                path = root / name
                if path.is_symlink() or not path.is_file(
                ) or size <= 0 or path.stat().st_size != size:
                    return 1
            print('\n'.join(names))
        elif command == 'save':
            names = expected['required']
            if allow_reference == 'true' and (
                    root / 'runtime-reference.json').is_file():
                names = ['runtime-reference.json']
            sizes = {name: (root / name).stat().st_size for name in names}
            temporary = marker.with_suffix('.tmp')
            temporary.write_text(
                json.dumps(dict(identity=expected, files=sizes)))
            temporary.replace(marker)
        else:
            raise ValueError('Unknown prefetch marker command')
    except (OSError, ValueError, KeyError, TypeError):
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

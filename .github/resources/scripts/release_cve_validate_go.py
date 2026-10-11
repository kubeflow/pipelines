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
"""Compile every tracked Go module changed by a compiler remediation."""

import argparse
from pathlib import Path

from release_cve_prepare import regular_source_path
from release_cve_prepare import run


def validate(source):
    tracked = run(['git', 'ls-files', '-z'], source).stdout.split('\0')
    modules = sorted(path for path in tracked if Path(path).name == 'go.mod')
    if not modules:
        raise ValueError('No tracked Go modules to validate')
    for path in modules:
        directory = regular_source_path(source, path).parent
        packages = run(['go', 'list', '-mod=readonly', './...'], directory)
        if packages.stdout.strip():
            # Compile all packages and test binaries, without invoking cluster
            # integration tests that require a separately provisioned runtime.
            run(['go', 'test', '-mod=readonly', '-run=^$', './...'], directory)
        else:
            # Tool-only modules are built by the native generator image lane.
            run(['go', 'mod', 'download'], directory)
            run(['go', 'mod', 'verify'], directory)
    run(['go', 'test', '-mod=readonly', './backend/src/...'], source)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', required=True, type=Path)
    validate(parser.parse_args().source.resolve())

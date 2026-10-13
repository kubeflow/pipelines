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
"""Prefetch Go dependencies without retrying policy conformance tests."""

import argparse
from pathlib import Path
import subprocess
import time

ATTEMPTS = 3
TIMEOUT_SECONDS = 120


def download(source):
    source = source.resolve()
    for attempt in range(1, ATTEMPTS + 1):
        try:
            subprocess.run(['go', 'mod', 'download'],
                           cwd=source,
                           check=True,
                           timeout=TIMEOUT_SECONDS)
            return
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
            if attempt == ATTEMPTS:
                raise
            delay = 5 * attempt
            print(
                f'Policy dependency download attempt {attempt} failed; '
                f'retrying in {delay}s.',
                flush=True)
            time.sleep(delay)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--backend-source', type=Path, required=True)
    args = parser.parse_args()
    try:
        download(args.backend_source)
    except (OSError, subprocess.CalledProcessError,
            subprocess.TimeoutExpired) as error:
        parser.exit(
            1, f'Policy dependency download failed: {error}\n'
            'Check Go module proxy connectivity and retry the job.\n')


if __name__ == '__main__':
    main()

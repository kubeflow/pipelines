#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Record the native ARM build identity consumed by the installation smoke."""

import argparse
import json
from pathlib import Path
import re

from arm64_smoke import CI_IMAGES


def image_record(ci_image, source_sha, registry):
    images = {value: key for key, value in CI_IMAGES.items()}
    if ci_image not in images:
        raise ValueError(f'Unknown CI image: {ci_image}')
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha):
        raise ValueError('source_sha must be a full commit SHA')
    if registry != 'kind-registry:5000':
        raise ValueError(
            'ARM presubmit images must use the CI registry namespace')
    return {
        'image': images[ci_image],
        'source_sha': source_sha,
        'reference': f'{registry}/{ci_image}:ci',
        'platforms': ['linux/arm64'],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('ci_image')
    parser.add_argument('source_sha')
    parser.add_argument('registry')
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    record = image_record(args.ci_image, args.source_sha, args.registry)
    args.output.write_text(json.dumps(record, indent=2) + '\n')


if __name__ == '__main__':
    main()

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
"""Migrate trusted, rendered Istio manifests off retired image registries.

The injector stores its hub in embedded JSON/YAML strings, so a
Kubernetes image transformer alone cannot update all consumers. Preserve
the rendered manifest verbatim except for exact registry prefixes and
standalone hub values.
"""

import re
import sys

RETIRED = re.compile(
    r'(?<![\w./-])(?:registry\.istio\.io/release|gcr\.io/istio-release)'
    r'(?=[/\s\"\']|$)')


def migrate(manifest):
    if not manifest.strip():
        raise ValueError(
            'Empty Istio manifest; check the upstream rendering step.')
    result = RETIRED.sub('docker.io/istio', manifest)
    if any(
            host in result
            for host in ('registry.istio.io', 'gcr.io/istio-release',
                         'istio-release.storage.googleapis.com')):
        raise ValueError(
            'Unsupported retired Istio endpoint; inspect the pinned '
            'manifest and update its migration before deployment.')
    return result


def main():
    try:
        result = migrate(sys.stdin.read())
    except ValueError as error:
        sys.exit(str(error))
    sys.stdout.write(result)


if __name__ == '__main__':
    main()

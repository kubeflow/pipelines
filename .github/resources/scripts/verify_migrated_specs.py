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
"""Require each migrated behavior to execute successfully in a Ginkgo
report."""

import argparse
import json
from pathlib import Path


def verify(reports, required):
    """Return missing, ambiguous, or unsuccessful migration labels."""
    if not required or len(required) != len(set(required)):
        raise ValueError('Expected a nonempty list of unique migration labels')
    states = {label: [] for label in required}
    for report in reports:
        for spec in report.get('SpecReports', []):
            labels = set((spec.get('LeafNodeLabels') or []))
            for container in spec.get('ContainerHierarchyLabels') or []:
                labels.update(container or [])
            for label in labels & states.keys():
                states[label].append(spec.get('State'))
    return {
        label: found for label, found in states.items() if found != ['passed']
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('report', type=Path)
    parser.add_argument('manifest', type=Path)
    args = parser.parse_args()
    failures = verify(
        json.loads(args.report.read_text()),
        json.loads(args.manifest.read_text()),
    )
    if failures:
        raise SystemExit(
            f'Migrated specs did not each pass exactly once: {failures}'
        )
    print('All required migrated specs executed and passed.')


if __name__ == '__main__':
    main()

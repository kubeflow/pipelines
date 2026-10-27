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
"""Compare full OSV evidence for manifest image upgrades on Linux amd64."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import urllib.request

import yaml

TARGETS = {
    'profile-controller': (
        'manifests/kustomize/base/installs/multi-user/'
        'pipelines-profile-controller/deployment.yaml',
        'profile-controller',
    ),
    'seaweedfs': (
        'manifests/kustomize/third-party/seaweedfs/base/seaweedfs/'
        'seaweedfs-deployment.yaml',
        'seaweedfs',
    ),
    'argo-controller': (
        'manifests/kustomize/third-party/argo/base/'
        'workflow-controller-deployment-patch.yaml',
        'workflow-controller',
    ),
    'argo-executor': (
        'manifests/kustomize/third-party/argo/base/'
        'workflow-controller-deployment-patch.yaml',
        'workflow-controller',
    ),
}
CERT_MANAGER_TARGETS = {
    'cert-manager-controller': 'cert-manager-controller',
    'cert-manager-cainjector': 'cert-manager-cainjector',
    'cert-manager-webhook': 'cert-manager-webhook',
    'cert-manager-acmesolver': 'cert-manager-controller',
}
TARGETS.update({
    target: ('backend/Makefile', container)
    for target, container in CERT_MANAGER_TARGETS.items()
})
PLATFORM = 'linux/amd64'


def cert_manager_reference(checkout: Path, target: str,
                           output: Path) -> tuple[str, dict]:
    """Resolve a component from the install manifest and retain that evidence."""
    makefile = (checkout / 'backend/Makefile').read_text()
    versions = re.findall(
        r'^CERT_MANAGER_VERSION\s*\?=\s*(v[0-9]+\.[0-9]+\.[0-9]+)\s*$',
        makefile, re.MULTILINE)
    if len(versions) != 1:
        raise ValueError('Expected exactly one pinned cert-manager version')
    url = ('https://github.com/cert-manager/cert-manager/releases/download/' +
           versions[0] + '/cert-manager.yaml')
    with urllib.request.urlopen(url, timeout=60) as response:
        raw = response.read()
    path = output / 'cert-manager.yaml'
    path.write_bytes(raw)
    containers = []
    for document in yaml.safe_load_all(raw):
        if (not isinstance(document, dict) or
                document.get('kind') != 'Deployment' or document.get(
                    'metadata', {}).get('namespace') != 'cert-manager'):
            continue
        containers.extend(
            container
            for container in document['spec']['template']['spec']['containers']
            if container.get('name') == CERT_MANAGER_TARGETS[target])
    if len(containers) != 1:
        raise ValueError('Expected exactly one cert-manager target container')
    container = containers[0]
    if target == 'cert-manager-acmesolver':
        flag = '--acme-http01-solver-image'
        args = container.get('args', [])
        values = [
            value.split('=', 1)[1]
            for value in args
            if isinstance(value, str) and value.startswith(flag + '=')
        ]
        values += [
            args[index + 1]
            for index, value in enumerate(args[:-1])
            if value == flag
        ]
        if len(values) != 1:
            raise ValueError('Expected exactly one ACME solver image')
        reference = values[0]
    else:
        reference = container.get('image')
    if (not isinstance(reference, str) or not reference or
            reference.startswith('-') or any(c.isspace() for c in reference)):
        raise ValueError('Invalid cert-manager image reference')
    return reference, {
        'upstream_manifest': {
            'url': url,
            'file': path.name,
            'sha256': hashlib.sha256(raw).hexdigest(),
        }
    }


def image_reference(checkout: Path, target: str) -> str:
    """Read one named manifest container or Argo executor image reference."""
    relative, name = TARGETS[target]
    manifest = yaml.safe_load((checkout / relative).read_text())
    containers = manifest['spec']['template']['spec']['containers']
    matches = [item for item in containers if item['name'] == name]
    if len(matches) != 1:
        raise ValueError('Expected exactly one target container')
    container = matches[0]
    if target == 'argo-executor':
        args = container.get('args', [])
        values = [
            args[i + 1]
            for i, value in enumerate(args[:-1])
            if value == '--executor-image'
        ]
        values += [
            item['value']
            for item in container.get('env', [])
            if item.get('name') == 'ARGO_EXECUTOR_IMAGE'
        ]
        if len(values) != 1:
            raise ValueError('Expected exactly one executor image reference')
        reference = values[0]
    else:
        reference = container['image']
    if (not isinstance(reference, str) or not reference or
            reference.startswith('-')):
        raise ValueError('Invalid image reference')
    return reference


def compare_reports(baseline: dict, candidate: dict) -> dict:
    """Group transitive advisory aliases across both reports, not packages.

    Absence from the candidate scan is an observation, not an
    applicability determination or a release-gate exemption. Raw package
    occurrences remain.
    """
    from kfpr.osv_reports import findings

    rows = [findings(baseline), findings(candidate)]
    parents = {}

    # Join aliases across both reports before comparing their connected groups.
    # An alias bridge introduced by either report must also join the other side.
    def root(value: str) -> str:
        parents.setdefault(value, value)
        while parents[value] != value:
            value = parents[value]
        return value

    for side in rows:
        for finding in side:
            ids = [finding['id'], *finding['aliases']]
            for identifier in ids[1:]:
                parents[root(identifier)] = root(ids[0])
            root(ids[0])
    groups = {}
    for identifier in parents:
        groups.setdefault(root(identifier), []).append(identifier)
    present = [{root(item['id']) for item in side} for side in rows]

    def aliases(identifiers):
        return sorted(sorted(groups[key]) for key in identifiers)

    return {
        'baseline_occurrences': len(rows[0]),
        'candidate_occurrences': len(rows[1]),
        'baseline_advisory_groups': len(present[0]),
        'candidate_advisory_groups': len(present[1]),
        'absent_from_candidate': aliases(present[0] - present[1]),
        'new_in_candidate': aliases(present[1] - present[0]),
        'present_in_both': aliases(present[0] & present[1]),
    }


def capture(checkout: Path, target: str, output: Path) -> tuple[dict, dict]:
    """Scan the same immutable Linux AMD64 image in JSON and SARIF formats."""
    from kfpr.osv_image_scan import scan_image
    from kfpr.osv_image_scan import validate_report

    output.mkdir(parents=True, exist_ok=True)
    if target in CERT_MANAGER_TARGETS:
        reference, provenance = cert_manager_reference(checkout, target, output)
    else:
        reference, provenance = image_reference(checkout, target), {}
    sha = subprocess.check_output(
        ['git', '-C', str(checkout), 'rev-parse', 'HEAD'], text=True).strip()
    subprocess.run(['docker', 'pull', '--platform', PLATFORM, reference],
                   check=True)
    inspected = json.loads(
        subprocess.check_output(['docker', 'image', 'inspect', reference],
                                text=True))
    if len(inspected) != 1:
        raise ValueError('Expected exactly one inspected image')
    metadata = inspected[0]
    image_id = metadata.get('Id', '')
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
        raise ValueError('Invalid immutable image ID')
    if (metadata.get('Os'), metadata.get('Architecture')) != ('linux', 'amd64'):
        raise ValueError('Image platform does not match Linux amd64')
    digests = metadata.get('RepoDigests')
    if not isinstance(digests, list) or not digests:
        raise ValueError('Pulled image has no repository digest')
    subject = {
        'source_sha': sha,
        'manifest': TARGETS[target][0],
        'image': reference,
        'image_id': image_id,
        'repo_digests': digests,
        'platform': PLATFORM,
        **provenance,
    }
    (output / 'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
    report = None
    for report_format in ('json', 'sarif'):
        directory = output / report_format
        directory.mkdir()
        scan_image(image_id, PLATFORM, directory, report_format=report_format)
        envelope = json.loads((directory / 'scan.json').read_text())
        validate_report(envelope, image_id, report_format)
        if report_format == 'json':
            report = envelope['report']
    return subject, report


def main() -> None:
    """Capture baseline/candidate evidence and write their alias comparison."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--target', choices=TARGETS, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    before, old = capture(args.baseline, args.target, args.output / 'baseline')
    after, new = capture(args.candidate, args.target, args.output / 'candidate')
    comparison = {
        'schema': 1,
        'target': args.target,
        'platform': PLATFORM,
        'baseline': before,
        'candidate': after,
        'comparison': compare_reports(old, new),
    }
    (args.output /
     'comparison.json').write_text(json.dumps(comparison, indent=2) + '\n')
    print(json.dumps(comparison, indent=2))


if __name__ == '__main__':
    main()

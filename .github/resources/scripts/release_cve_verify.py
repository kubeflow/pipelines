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
"""Rebuild and rescan a remediation patch without publishing images."""

import argparse
import json
from pathlib import Path
import platform
import re
import subprocess
import sys
import tempfile
import uuid

from check_fixable_cves import find_blocking_cves
from release_cve_prepare import load_plan
from release_cve_prepare import patch_hash
from release_cve_prepare import regular_source_path
from release_cve_prepare import require_source
from release_cve_prepare import run

KEY_FIELDS = ('target', 'cve', 'ecosystem', 'package')


def finding_key(finding):
    if not isinstance(finding, dict) or any(
            not isinstance(finding.get(field), str) or not finding[field]
            for field in KEY_FIELDS):
        raise ValueError('Invalid finding identity in remediation plan')
    return tuple(finding[field] for field in KEY_FIELDS)


def inventory(plan, architecture):
    images = plan.get('verify_images')
    reports = plan.get('reports')
    targets = plan.get('targets')
    if not isinstance(images, list) or not images or not isinstance(
            reports, list) or not isinstance(targets, list) or not targets:
        raise ValueError(
            'Remediation requires images, baseline reports, and explicit targets'
        )
    if any(not isinstance(record, dict) for record in reports + targets):
        raise ValueError('Reports and targets must contain objects')
    if any(
            report.get('outcome') not in ('pass', 'blocked', 'overridden')
            for report in reports):
        raise ValueError(
            'Baseline reports must have completed policy evaluation')
    names = set()
    selected = []
    for image in images:
        if not isinstance(image, dict) or not isinstance(
                image.get('image'), str) or not re.fullmatch(
                    r'[a-z0-9]+(?:[._-][a-z0-9]+)*', image['image']):
            raise ValueError('Invalid image name')
        name = image['image']
        if name in names or image.get('context') != '.':
            raise ValueError('Duplicate image or unsupported build context')
        names.add(name)
        for arch in ('amd64', 'arm64'):
            matches = [
                report for report in reports if report.get('image') == name and
                report.get('platform') == f'linux/{arch}'
            ]
            if len(matches) != 1 or matches[0].get(
                    'source_sha') != plan['source_sha']:
                raise ValueError(
                    f'Missing exact-source baseline for {name} linux/{arch}')
            if not isinstance(matches[0].get('findings'), list):
                raise ValueError('Invalid baseline findings')
            baseline = {
                finding_key(finding) for finding in matches[0]['findings']
            }
            if arch == architecture:
                selected.append((image, baseline))
    if any(target.get('image') not in names for target in targets):
        raise ValueError('Every remediation target must be rebuilt')
    for target in targets:
        finding_key(target)
    return selected


def check_findings(report, baseline, targets):
    current = {tuple(finding[:4]) for finding in find_blocking_cves(report)}
    remaining = current & targets
    introduced = current - baseline
    if remaining:
        raise ValueError(
            f'{len(remaining)} targeted CVE finding(s) remain after rebuild: ' +
            json.dumps(sorted(remaining)[:20]))
    if introduced:
        raise ValueError(
            f'Rebuild introduced {len(introduced)} new blocking CVE finding(s): '
            + json.dumps(sorted(introduced)[:20]))
    return len(current)


def apply_patch(source, bundle, plan):
    require_source(source, plan)
    patch = bundle / 'remediation.patch'
    if not patch.is_file() or not patch.stat().st_size:
        raise ValueError('Verification requires a nonempty remediation patch')
    digest = patch_hash(bundle)
    expected = (bundle / 'patch.sha256').read_text(encoding='utf-8').strip()
    if expected != digest:
        raise ValueError('Remediation patch checksum mismatch')
    run(['git', 'apply', '--check', str(patch)], source)
    run(['git', 'apply', str(patch)], source)
    return digest


def verify(source, bundle, architecture, output):
    # A failed re-verification must never leave an earlier success certificate.
    if output.exists():
        output.unlink()
    plan = load_plan(bundle)
    host = {
        'x86_64': 'amd64',
        'aarch64': 'arm64',
        'arm64': 'arm64'
    }.get(platform.machine())
    if host != architecture:
        raise ValueError(
            'Image verification must run on a native architecture runner')
    selected = inventory(plan, architecture)
    digest = apply_patch(source, bundle, plan)
    # Build metadata uses the blocked release commit. It does not publish a tag.
    build_args = [
        '--build-arg', f'COMMIT_SHA={plan["source_sha"]}', '--build-arg',
        f'COMMIT_HASH={plan["source_sha"]}', '--build-arg',
        'TAG_NAME=cve-remediation'
    ]
    if (source / 'frontend/.nvmrc').exists():
        node = regular_source_path(
            source, 'frontend/.nvmrc').read_text(encoding='utf-8').strip()
        if not re.fullmatch(r'v?\d+\.\d+\.\d+', node):
            raise ValueError('Frontend Node version must be an exact version')
        build_args.extend(
            ['--build-arg', f'NODE_VERSION={node.removeprefix("v")}'])
    verified = []
    remaining = {}
    image_platform = f'linux/{architecture}'
    for image, baseline in selected:
        name = image['image']
        dockerfile = regular_source_path(source, image.get('dockerfile', ''))
        local_tag = f'cve-remediation-{uuid.uuid4().hex}:local'
        targets = {
            finding_key(target)
            for target in plan['targets']
            if target['image'] == name
        }
        try:
            run([
                'docker', 'buildx', 'build', '--pull', '--load', '--platform',
                image_platform, '--file',
                str(dockerfile), '--tag', local_tag, *build_args, '.'
            ], source)
            with tempfile.TemporaryDirectory(
                    prefix='cve-remediation-') as directory:
                archive = Path(directory) / 'image.tar'
                report_path = Path(directory) / 'osv-results.json'
                run([
                    'docker', 'image', 'save', '--platform', image_platform,
                    '--output',
                    str(archive), local_tag
                ], source)
                scan = run([
                    'osv-scanner', 'scan', 'image', '--archive',
                    str(archive), '--all-vulns', '--format', 'json',
                    '--output-file',
                    str(report_path)
                ],
                           source,
                           check=False)
                if scan.returncode not in (0, 1):
                    raise ValueError(
                        f'OSV-Scanner failed with exit code {scan.returncode}')
                report = json.loads(report_path.read_text(encoding='utf-8'))
                remaining[name] = check_findings(report, baseline, targets)
                verified.append(name)
        finally:
            run(['docker', 'image', 'rm', '--force', local_tag],
                source,
                check=False)
    certificate = {
        'source_sha': plan['source_sha'],
        'patch_sha256': digest,
        'architecture': architecture,
        'images': verified,
        'verified': True,
        'remaining_baseline_findings': remaining
    }
    output.write_text(
        json.dumps(certificate, indent=2) + '\n', encoding='utf-8')
    return certificate


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--bundle', required=True, type=Path)
    parser.add_argument(
        '--architecture', required=True, choices=('amd64', 'arm64'))
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args(argv)
    try:
        verify(args.source.resolve(), args.bundle.resolve(), args.architecture,
               args.output.resolve())
        return 0
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'Cannot verify remediation: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())

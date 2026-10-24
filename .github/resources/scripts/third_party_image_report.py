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
"""Produce image-level review evidence without filtering release findings."""

import argparse
import hashlib
import json
import re
from pathlib import Path

from kfpr.osv_image_scan import validate_report
from kfpr.osv_reports import findings
from kfpr.osv_reports import inventory as package_inventory


def repository(image):
    """Strip a tag or digest without stripping a registry port."""
    without_digest = image.split('@', 1)[0]
    prefix, separator, name = without_digest.rpartition('/')
    return prefix + separator + name.split(':', 1)[0]


def read_json(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError(f'Missing regular evidence file: {path}')
    return json.loads(path.read_text())


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def validate_inventory(document):
    if not isinstance(document, dict) or document.get('schema') != 1:
        raise ValueError('Invalid inventory schema')
    sources = document.get('sources')
    if not isinstance(sources, dict) or set(sources) != {
        'master',
        'release-2.18',
    }:
        raise ValueError(
            'Inventory must identify master and release-2.18 sources'
        )
    if any(
        not isinstance(sha, str) or not re.fullmatch(r'[0-9a-f]{40}', sha)
        for sha in sources.values()
    ):
        raise ValueError('Inventory source must be an immutable Git commit')
    scans = document.get('scans')
    images = document.get('images')
    if (
        not isinstance(scans, list)
        or not scans
        or not isinstance(images, list)
        or not images
    ):
        raise ValueError('Inventory requires deployed images and scans')
    by_key = {}
    refs = set()
    for scan in scans:
        if not isinstance(scan, dict):
            raise ValueError('Invalid scan inventory entry')
        key, image = scan.get('key'), scan.get('image')
        if not isinstance(key, str) or not re.fullmatch(
            r'[0-9a-f]{16,64}', key
        ):
            raise ValueError('Invalid scan key')
        if (
            not isinstance(image, str)
            or not image
            or any(c.isspace() for c in image)
        ):
            raise ValueError('Invalid image reference')
        if key in by_key or image in refs:
            raise ValueError('Duplicate scan key or image reference')
        by_key[key] = scan
        refs.add(image)
    deployed = set()
    used = set()
    for image in images:
        if not isinstance(image, dict) or image.get('key') not in by_key:
            raise ValueError('Deployed image is missing its scan')
        key = image['key']
        if key in deployed or image.get('image') != by_key[key]['image']:
            raise ValueError('Duplicate or mismatched deployed image')
        branches = image.get('branches')
        if (
            not isinstance(branches, list)
            or not branches
            or any(b not in sources for b in branches)
            or len(set(branches)) != len(branches)
        ):
            raise ValueError('Invalid deployed image branches')
        deployed.add(key)
        used.add(key)
        candidate = image.get('candidate_key')
        if candidate is not None:
            if (
                candidate not in by_key
                or candidate == key
                or repository(by_key[candidate]['image'])
                != repository(image['image'])
            ):
                raise ValueError(
                    'Candidate must identify a different image in the same repository'
                )
            used.add(candidate)
    master_keys = {
        image['key'] for image in images if 'master' in image['branches']
    }
    for image in images:
        candidate = image.get('candidate_key')
        if candidate and (
            'release-2.18' not in image['branches']
            or candidate not in master_keys
        ):
            raise ValueError(
                'Candidate must be a deployed master image for a release image'
            )
    if used != set(by_key):
        raise ValueError('Inventory contains unused scans')
    return by_key


def alias_groups(scans):
    """Union aliases across both sides so advisory ID changes are not fixes."""
    parents = {}

    def root(identifier):
        parents.setdefault(identifier, identifier)
        while parents[identifier] != identifier:
            parents[identifier] = parents[parents[identifier]]
            identifier = parents[identifier]
        return identifier

    for scan in scans.values():
        for finding in scan['findings']:
            identifiers = [finding['id'], *finding['aliases']]
            for identifier in identifiers:
                parents[root(identifier)] = root(identifiers[0])
    grouped = {}
    for identifier in parents:
        grouped.setdefault(root(identifier), []).append(identifier)
    canonical = {}
    groups = {}
    for identifiers in grouped.values():
        identifiers.sort()
        cves = [
            identifier
            for identifier in identifiers
            if identifier.startswith('CVE-')
        ]
        label = cves[0] if cves else identifiers[0]
        groups[label] = identifiers
        canonical.update({identifier: label for identifier in identifiers})
    return canonical, groups


def aggregate(document, evidence):
    expected = validate_inventory(document)
    if evidence.is_symlink() or not evidence.is_dir():
        raise ValueError('Evidence directory is missing or is a symlink')
    if {path.name for path in evidence.iterdir()} != set(expected):
        raise ValueError('Evidence lanes are missing or unexpected')
    scans = {}
    for key, entry in expected.items():
        lane = evidence / key
        if lane.is_symlink() or not lane.is_dir():
            raise ValueError('Evidence lane must be a regular directory')
        subject = read_json(lane / 'subject.json')
        if (
            not isinstance(subject, dict)
            or subject.get('image') != entry['image']
            or subject.get('platform') != 'linux/amd64'
        ):
            raise ValueError(
                'Scan subject does not match inventory image/platform'
            )
        image_id = subject.get('image_id')
        if not isinstance(image_id, str) or not re.fullmatch(
            r'sha256:[0-9a-f]{64}', image_id
        ):
            raise ValueError('Scan subject requires an immutable image ID')
        envelope = read_json(lane / 'scan.json')
        raw = read_json(lane / 'osv-report.json')
        validate_report(envelope, image_id, 'json')
        if raw != envelope['report']:
            raise ValueError('Raw OSV report does not match scan envelope')
        scans[key] = {
            'image': entry['image'],
            'image_id': image_id,
            'archive_sha256': envelope['archive_sha256'],
            'evidence_sha256': {
                name: sha256(lane / name)
                for name in ('subject.json', 'scan.json', 'osv-report.json')
            },
            'packages': [
                dict(
                    zip(
                        (
                            'target',
                            'ecosystem',
                            'package',
                            'installed_version',
                        ),
                        package,
                    )
                )
                for package in package_inventory(raw)
            ],
            'findings': findings(raw),
        }
    canonical, groups = alias_groups(scans)
    for scan in scans.values():
        scan['advisories'] = sorted(
            {canonical[f['id']] for f in scan['findings']}
        )
        scan['raw_finding_count'] = len(scan['findings'])
        scan['support_channel_review_count'] = sum(
            bool(f.get('support_channel_evidence')) for f in scan['findings']
        )
        scan['unique_advisory_count'] = len(scan['advisories'])
        scan['package_fix_count'] = len(
            {
                (f['target'], f['package'], f['installed_version'])
                for f in scan['findings']
                if f['fixable']
            }
        )
    rows = []
    for image in document['images']:
        current = scans[image['key']]
        row = {
            **image,
            **current,
            'action': 'Review applicability or seek an upstream image update; no candidate image verified.',
        }
        if image.get('candidate_key'):
            candidate = scans[image['candidate_key']]
            current_ids, candidate_ids = (
                set(current['advisories']),
                set(candidate['advisories']),
            )
            row['candidate_comparison'] = {
                'image': candidate['image'],
                'image_id': candidate['image_id'],
                'resolved': sorted(current_ids - candidate_ids),
                'added': sorted(candidate_ids - current_ids),
                'remaining': sorted(current_ids & candidate_ids),
                'compatibility': 'Review required; vulnerability comparison does not establish compatibility or applicability.',
            }
            row['action'] = (
                'Review candidate image upgrade and any added advisories; compatibility is unverified.'
            )
        elif not current['findings']:
            row['action'] = 'No findings reported; retain scan evidence.'
        rows.append(row)
    return {
        'schema': 1,
        'sources': document['sources'],
        'platform': 'linux/amd64',
        'images': rows,
        'scans': scans,
        'advisory_aliases': groups,
    }


def markdown(summary):
    lines = [
        '# Third-party image review',
        '',
        'Raw OSV evidence is retained. Package fix versions do not establish an available image upgrade. Candidate comparisons require compatibility and applicability review. No findings are suppressed. Release qualification evaluates its candidate-bound evidence separately; this report does not authorize release.',
        '',
        '| Image | Branches | Raw findings | Unique advisories | Packages with fix versions | Image action |',
        '| --- | --- | ---: | ---: | ---: | --- |',
    ]
    for row in summary['images']:
        action = row['action']
        if 'candidate_comparison' in row:
            candidate = row['candidate_comparison']
            action = f'Review `{candidate["image"]}`: {len(candidate["resolved"])} advisories absent, {len(candidate["added"])} added; compatibility unverified'
        if row['support_channel_review_count']:
            action += (
                f'; {row["support_channel_review_count"]} findings need support-channel review'
                ' (Ubuntu Pro evidence is not a standard-channel fix)'
            )
        values = [
            row['image'],
            ', '.join(row['branches']),
            row['raw_finding_count'],
            row['unique_advisory_count'],
            row['package_fix_count'],
            action,
        ]
        lines.append(
            '| '
            + ' | '.join(
                str(value).replace('|', '\\|').replace('\n', ' ')
                for value in values
            )
            + ' |'
        )
    return '\n'.join(lines) + '\n'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inventory', required=True, type=Path)
    parser.add_argument('--evidence', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    summary = aggregate(read_json(args.inventory), args.evidence)
    args.output.mkdir(parents=True, exist_ok=True)
    for name in ('summary.json', 'summary.md'):
        destination = args.output / name
        if destination.exists():
            raise ValueError(
                f'Refusing to overwrite existing output: {destination}'
            )
    (args.output / 'summary.json').write_text(
        json.dumps(summary, indent=2) + '\n'
    )
    (args.output / 'summary.md').write_text(markdown(summary))


if __name__ == '__main__':
    main()

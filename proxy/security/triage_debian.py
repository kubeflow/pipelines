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
"""Report Debian source-package findings without changing scan acceptance."""

import argparse
import hashlib
import json
from pathlib import Path


def _text(value, label):
    if not isinstance(value, str) or not value:
        raise ValueError('Missing ' + label)
    return value


def _object(value, label):
    if not isinstance(value, dict):
        raise ValueError(label + ' must be an object')
    return value


def _list(value, label, nonempty=False):
    if not isinstance(value, list) or nonempty and not value:
        raise ValueError(label + ' must be ' +
                         ('a nonempty list' if nonempty else 'a list'))
    return value


def _validate(envelope, tracker):
    _object(tracker, 'tracker')
    report = _object(_object(envelope, 'envelope').get('report'), 'report')
    for result in _list(report.get('results'), 'results', nonempty=True):
        _object(result, 'result')
        source = _object(result.get('source'), 'source')
        _text(source.get('path'), 'source path')
        _text(source.get('type'), 'source type')
        for item in _list(result.get('packages'), 'packages', nonempty=True):
            package = _object(
                _object(item, 'package entry').get('package'), 'package')
            for key in ('name', 'version', 'ecosystem'):
                _text(package.get(key), 'package ' + key)
            if 'os_package_name' in package:
                _text(package['os_package_name'], 'binary package')
            for vuln in _list(
                    item.get('vulnerabilities', []), 'vulnerabilities'):
                _text(_object(vuln, 'vulnerability').get('id'), 'advisory ID')
                for alias in _list(vuln.get('aliases', []), 'aliases'):
                    _text(alias, 'advisory alias')
                for affected in _list(vuln.get('affected', []), 'affected'):
                    subject = _object(
                        _object(affected, 'affected entry').get('package'),
                        'affected package')
                    _text(subject.get('name'), 'affected name')
                    _text(subject.get('ecosystem'), 'affected ecosystem')
                    for interval in _list(affected.get('ranges', []), 'ranges'):
                        for event in _list(
                                _object(interval, 'range').get('events'),
                                'events'):
                            _object(event, 'event')
                            if 'fixed' in event:
                                _text(event['fixed'], 'fixed version')


def triage(envelope, tracker):
    """Join raw OSV occurrences to an explicitly supplied tracker snapshot."""
    _validate(envelope, tracker)
    rows = {}
    non_debian = 0
    for result in envelope['report']['results']:
        for item in result['packages']:
            package = item['package']
            vulnerabilities = item.get('vulnerabilities', [])
            ecosystem = _text(package.get('ecosystem'), 'ecosystem')
            if not ecosystem.startswith('Debian:'):
                non_debian += len(vulnerabilities)
                continue
            name = _text(package.get('name'), 'source package')
            version = _text(package.get('version'), 'reported version')
            for vulnerability in vulnerabilities:
                identifier = _text(vulnerability.get('id'), 'advisory ID')
                key = ecosystem, name, version, identifier
                row = rows.setdefault(
                    key, {
                        'ecosystem': ecosystem,
                        'source_package': name,
                        'reported_version': version,
                        'advisory_id': identifier,
                        'raw_occurrences': [],
                        'scan_fixed_versions': [],
                        'scan_advisory_ids': [],
                    })
                row['raw_occurrences'].append({
                    'source': result['source'],
                    'binary_package': package.get('os_package_name', name),
                    'advisory_id': identifier,
                })
                row['scan_advisory_ids'].extend(
                    [identifier, *vulnerability.get('aliases', [])])
                for affected in vulnerability.get('affected', []):
                    if affected.get(
                            'package',
                        {}).get('name') != name or affected['package'].get(
                            'ecosystem') != ecosystem:
                        continue
                    for interval in affected.get('ranges', []):
                        row['scan_fixed_versions'].extend(
                            event['fixed']
                            for event in interval.get('events', [])
                            if 'fixed' in event)
    for key, row in sorted(rows.items()):
        ecosystem, name, _, identifier = key
        cve = identifier.removeprefix('DEBIAN-')
        advisory = None
        release = None
        if ecosystem == 'Debian:13':
            advisory = _object(tracker.get(name, {}),
                               'tracker package').get(cve)
            if advisory is not None:
                releases = _object(
                    _object(advisory, 'tracker advisory').get('releases', {}),
                    'tracker releases')
                release = releases.get('trixie')
                if release is not None:
                    _object(release, 'tracker release')
        status = {
            'release':
                'trixie',
            'applies_to_scanned_ecosystem':
                ecosystem == 'Debian:13',
            'lookup':
                'present' if release is not None else
                ('missing_advisory' if advisory is None else 'missing_release'),
            'status':
                None,
            'fixed_version':
                None,
            'urgency':
                None,
            'nodsa':
                None,
            'nodsa_reason':
                None,
            'interpretation':
                'unknown',
        }
        if ecosystem != 'Debian:13':
            status['lookup'] = 'unsupported_ecosystem'
        if release is not None:
            for field in ('status', 'fixed_version', 'urgency', 'nodsa',
                          'nodsa_reason'):
                status[field] = release.get(field)
            if release.get('fixed_version') == '0':
                status['interpretation'] = 'not_affected'
            elif release.get('status') == 'resolved' and release.get(
                    'fixed_version'):
                status['interpretation'] = 'fixed_in_tracker_release'
            elif release.get('status') == 'open':
                status['interpretation'] = 'open'
            elif release.get('status') == 'undetermined':
                status['interpretation'] = 'undetermined'
        row['tracker'] = status
        row['raw_occurrence_count'] = len(row['raw_occurrences'])
        row['raw_occurrences'].sort(
            key=lambda value: json.dumps(value, sort_keys=True))
        for field in ('scan_fixed_versions', 'scan_advisory_ids'):
            row[field] = sorted(set(row[field]))
    ordered = [rows[key] for key in sorted(rows)]
    return {
        'schema':
            1,
        'reporting_only':
            True,
        'notice':
            'Tracker status is release-level context, not an installed-version verdict or suppression. Raw scans and acceptance gates remain authoritative.',
        'scan_provenance': {
            key: value for key, value in envelope.items() if key != 'report'
        },
        'counts': {
            'distinct_advisory_ids':
                len({row['advisory_id'] for row in ordered}),
            'raw_debian_occurrences':
                sum(row['raw_occurrence_count'] for row in ordered),
            'deduplicated_source_findings':
                len(ordered),
            'non_debian_occurrences_not_in_view':
                non_debian
        },
        'findings':
            ordered,
    }


def write_view(scan: Path, tracker: Path, output: Path):
    """Write a deterministic sidecar while preserving both input snapshots."""
    if output.resolve() in (scan.resolve(), tracker.resolve()
                           ) or output.exists() and (output.samefile(scan) or
                                                     output.samefile(tracker)):
        raise ValueError('Output must not overwrite either input')
    scan_bytes, tracker_bytes = scan.read_bytes(), tracker.read_bytes()
    report = triage(json.loads(scan_bytes), json.loads(tracker_bytes))
    report['inputs'] = {
        'scan_sha256': hashlib.sha256(scan_bytes).hexdigest(),
        'debian_tracker_sha256': hashlib.sha256(tracker_bytes).hexdigest(),
        'tracker_release': 'trixie',
    }
    output.write_text(json.dumps(report, indent=2, sort_keys=True) + '\n')
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--scan', type=Path, required=True)
    parser.add_argument('--tracker', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    write_view(args.scan, args.tracker, args.output)


if __name__ == '__main__':
    main()

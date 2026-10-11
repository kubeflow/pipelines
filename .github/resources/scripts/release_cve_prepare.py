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
"""Prepare a bounded CVE remediation patch without write credentials."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

from check_fixable_cves import find_blocking_cves
import update_go_version

LOCKFILE = 'frontend/server/package-lock.json'
SHA_PATTERN = re.compile(r'[0-9a-f]{40}')
CVE_PATTERN = re.compile(r'CVE-[0-9]{4}-[0-9]{4,}')


def run(arguments, source, check=True):
    # Long builds and fixes report progress directly to the workflow log.
    if arguments[0] == 'docker' or arguments[:2] == ['osv-scanner', 'fix']:
        return subprocess.run(arguments, cwd=source, check=check, text=True)
    result = subprocess.run(
        arguments, cwd=source, check=False, text=True, capture_output=True)
    if result.returncode:
        for value in (result.stdout, result.stderr):
            if value:
                print(value[-8000:], file=sys.stderr)
        if check:
            result.check_returncode()
    return result


def npm_advisories(report, requested):
    """Expand CVEs to all npm advisory IDs in their OSV alias groups."""
    find_blocking_cves(report)  # Reject malformed scanner output first.
    groups = []
    advisories = set()
    for result in report['results']:
        for entry in result['packages']:
            if entry['package']['ecosystem'] != 'npm':
                continue
            for vulnerability in entry.get('vulnerabilities', []):
                advisory_id = vulnerability['id']
                advisories.add(advisory_id)
                groups.append({
                    advisory_id, *(vulnerability.get('aliases') or []),
                    *vulnerability.get('upstream', [])
                })
            for group in entry.get('groups', []):
                groups.append(
                    set(group['ids']) | set(group.get('aliases') or []))
    selected = set(requested)
    found = set()
    while True:
        previous = selected.copy()
        for group in groups:
            if group & selected:
                found.update(group & set(requested))
                selected.update(group)
        if selected == previous:
            break
    if found != set(requested) or not advisories & selected:
        raise ValueError(
            'Requested npm CVEs could not be mapped to lockfile advisories')
    # OSV 2.5.0's explicit-ID filter ignores other IDs, including their aliases.
    # Passing the complete connected group avoids ignoring the selected CVE.
    return sorted(advisories & selected)


def load_plan(bundle):
    plan = json.loads((bundle / 'plan.json').read_text(encoding='utf-8'))
    if (not isinstance(plan, dict) or plan.get('schema_version') != 1 or
            not isinstance(plan.get('source_sha'), str) or
            not SHA_PATTERN.fullmatch(plan['source_sha'])):
        raise ValueError('Expected schema version 1 and an exact source SHA')
    return plan


def require_source(source, plan):
    if run(['git', 'rev-parse', 'HEAD'],
           source).stdout.strip() != plan['source_sha']:
        raise ValueError(
            'Source checkout does not match the blocked release SHA')
    if run(['git', 'status', '--porcelain', '--untracked-files=all'],
           source).stdout:
        raise ValueError('Remediation requires a clean source checkout')


def regular_source_path(source, relative):
    path = Path(relative)
    if path.is_absolute() or not path.parts or '..' in path.parts:
        raise ValueError(f'Unsafe source path: {relative}')
    current = source
    for part in path.parts:
        current /= part
        if current.is_symlink():
            raise ValueError(f'Symlink source path: {relative}')
    if not current.is_file():
        raise ValueError(f'Missing source file: {relative}')
    return current


def modified_paths(source):
    records = run(['git', 'diff', '--name-status', '-z', 'HEAD'],
                  source).stdout.split('\0')
    changes = []
    for offset in range(0, len(records) - 1, 2):
        status, name = records[offset:offset + 2]
        if status != 'M':
            raise ValueError(
                'Remediation may only modify existing regular files')
        regular_source_path(source, name)
        changes.append(name)
    if run(['git', 'ls-files', '--others', '--exclude-standard'],
           source).stdout:
        raise ValueError('Remediation unexpectedly created untracked files')
    return set(changes)


def patch_hash(bundle):
    return hashlib.sha256(
        (bundle / 'remediation.patch').read_bytes()).hexdigest()


def prepare(source, bundle):
    plan = load_plan(bundle)
    require_source(source, plan)
    go_version = plan.get('go_version', '')
    npm_vulns = plan.get('npm_vulns', [])
    if not isinstance(go_version, str) or (
            go_version and not re.fullmatch(r'1\.\d+\.\d+', go_version)):
        raise ValueError('Go remediation requires an exact compiler version')
    if not isinstance(npm_vulns, list) or any(
            not isinstance(cve, str) or not CVE_PATTERN.fullmatch(cve)
            for cve in npm_vulns):
        raise ValueError('npm remediation requires explicit CVE IDs')
    allowed = set()
    go_changes = set()
    if go_version:
        tracked = run(['git', 'ls-files', '-z'], source).stdout.split('\0')
        allowed.update(path for path in tracked if Path(path).name == 'go.mod')
        allowed.update(
            str(pin.path) for pin in update_go_version.MANAGED_DOCKERFILES)
        for name in allowed:
            regular_source_path(source, name)
        update_go_version.update_repository(source, go_version)
        update_go_version.check_repository(source)
        go_changes = modified_paths(source)
        if not go_changes <= allowed:
            raise ValueError(
                'Go updater changed files outside its managed inventory')
    if npm_vulns:
        lockfile = regular_source_path(source, LOCKFILE)
        # in-place only updates the lockfile, and minor excludes major upgrades.
        # Repeat --vulns as documented by OSV-Scanner 2.5.0 fix --help.
        before = {name: (source / name).read_bytes() for name in go_changes}
        # OSV also writes a .resolve.deps cache next to the input. Keep that
        # cache and any package-manager side effects outside the checkout.
        with tempfile.TemporaryDirectory(
                prefix='cve-remediation-npm-') as directory:
            temporary = Path(directory)
            candidate = temporary / 'package-lock.json'
            candidate.write_bytes(lockfile.read_bytes())
            report_path = temporary / 'osv-results.json'
            scan = run([
                'osv-scanner', 'scan', 'source', '--lockfile',
                str(candidate), '--all-vulns', '--format', 'json',
                '--output-file',
                str(report_path)
            ],
                       temporary,
                       check=False)
            if scan.returncode not in (0, 1):
                raise ValueError(
                    f'OSV-Scanner lockfile scan failed with exit code {scan.returncode}'
                )
            advisories = npm_advisories(
                json.loads(report_path.read_text(encoding='utf-8')), npm_vulns)
            command = [
                'osv-scanner', 'fix', '--strategy=in-place',
                '--upgrade-config=minor', '--no-introduce', '--lockfile',
                str(candidate)
            ]
            for advisory_id in advisories:
                command.extend(['--vulns', advisory_id])
            run(command, temporary)
            lockfile.write_bytes(candidate.read_bytes())
        if any((source / name).read_bytes() != contents
               for name, contents in before.items()):
            raise ValueError('npm remediation modified the Go update')
        npm_changes = modified_paths(source)
        if not npm_changes <= go_changes | {LOCKFILE}:
            raise ValueError('npm remediation modified unexpected files')
        allowed.add(LOCKFILE)
    changes = modified_paths(source)
    if not changes <= allowed:
        raise ValueError('Remediation modified unexpected files: ' +
                         ', '.join(sorted(changes - allowed)))
    patch = run([
        'git', 'diff', '--full-index', '--no-ext-diff', '--no-renames', 'HEAD'
    ], source).stdout
    if 'GIT binary patch' in patch or '\nBinary files ' in patch or '\nold mode ' in patch:
        raise ValueError('Remediation patch must contain only text changes')
    (bundle / 'remediation.patch').write_text(patch, encoding='utf-8')
    digest = patch_hash(bundle)
    (bundle / 'patch.sha256').write_text(digest + '\n', encoding='utf-8')
    return {
        'has_patch': bool(changes),
        'go_changed': bool(go_changes),
        'npm_changed': LOCKFILE in changes
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--bundle', required=True, type=Path)
    args = parser.parse_args(argv)
    try:
        outputs = prepare(args.source.resolve(), args.bundle.resolve())
        if os.environ.get('GITHUB_OUTPUT'):
            with open(
                    os.environ['GITHUB_OUTPUT'], 'a',
                    encoding='utf-8') as output:
                for name, value in outputs.items():
                    output.write(f'{name}={str(value).lower()}\n')
        print(json.dumps(outputs, sort_keys=True))
        return 0
    except (OSError, ValueError, subprocess.CalledProcessError,
            update_go_version.PolicyError) as error:
        print(
            f'Cannot prepare a verified remediation: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())

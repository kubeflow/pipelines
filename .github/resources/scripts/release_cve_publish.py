#!/usr/bin/env python3
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
"""Publish an independently verified dependency patch without running source
code."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.parse import quote

from release_cve_plan import GO_IMAGES
from update_go_version import DOCKER_FROM_PATTERN
from update_go_version import MANAGED_DOCKERFILES

BOT_NAME = 'github-actions[bot]'
BOT_EMAIL = '41898282+github-actions[bot]@users.noreply.github.com'
ARCHITECTURES = ('amd64', 'arm64')
SHA_PATTERN = re.compile(r'[0-9a-f]{40}')


class PublishError(ValueError):
    """The patch cannot be published safely or is no longer current."""


def _run(arguments, source=None, env=None):
    result = subprocess.run(
        arguments,
        cwd=source,
        env=env,
        text=True,
        capture_output=True,
        check=False)
    if result.returncode:
        # Git's authentication is passed through environment, never arguments.
        raise PublishError(result.stderr.strip() or result.stdout.strip() or
                           f'{arguments[0]} failed')
    return result.stdout.strip()


def _git(source, *arguments, env=None):
    return _run([
        'git', '-c', 'core.hooksPath=/dev/null', '-c', 'core.fsmonitor=false',
        *arguments
    ], source, env)


def _github_git(source, repository, operation, *arguments):
    token = os.environ.get('GH_TOKEN')
    if not token:
        raise PublishError('GH_TOKEN must contain the workflow GITHUB_TOKEN')
    authorization = base64.b64encode(
        f'x-access-token:{token}'.encode()).decode()
    env = dict(
        os.environ,
        GIT_TERMINAL_PROMPT='0',
        GIT_CONFIG_COUNT='1',
        GIT_CONFIG_KEY_0='http.https://github.com/.extraheader',
        GIT_CONFIG_VALUE_0=f'AUTHORIZATION: basic {authorization}')
    return _git(
        source,
        operation,
        f'https://github.com/{repository}.git',
        *arguments,
        env=env)


def _gh(*arguments):
    return _run(['gh', *arguments])


def _json_file(path):
    if path.is_symlink() or not path.is_file():
        raise PublishError(f'Required regular file is missing: {path.name}')
    try:
        return json.loads(path.read_text(encoding='utf-8'))
    except (OSError, ValueError) as error:
        raise PublishError(f'Invalid JSON in {path.name}') from error


def _regular_file(root, relative):
    path = Path(relative)
    if path.is_absolute() or '..' in path.parts:
        raise PublishError(f'Unsafe patch path: {relative}')
    for parent in (root / path, *(root / path).parents):
        if parent == root:
            break
        if parent.is_symlink():
            raise PublishError(
                f'Patch paths must not follow symlinks: {relative}')
    if not (root / path).is_file():
        raise PublishError(f'Patch may only modify existing files: {relative}')


def _validate_bundle(source, bundle, verification):
    plan = _json_file(bundle / 'plan.json')
    if (not isinstance(plan, dict) or
            type(plan.get('schema_version')) is not int or
            plan['schema_version'] != 1):
        raise PublishError('Unsupported remediation plan schema')
    sha = plan.get('source_sha', '')
    branch = plan.get('source_branch', '')
    if not isinstance(sha, str) or not SHA_PATTERN.fullmatch(sha):
        raise PublishError('Plan must identify the scanned source SHA')
    if not isinstance(branch, str) or not branch or branch.startswith('-'):
        raise PublishError('Plan must identify a source branch')
    _git(source, 'check-ref-format', '--branch', branch)
    if _git(source, 'rev-parse', 'HEAD') != sha:
        raise PublishError('Source checkout does not match the scanned SHA')
    if _git(source, 'status', '--porcelain', '--untracked-files=no'):
        raise PublishError('Source checkout must be clean before publication')
    version = plan.get('go_version', '')
    npm_vulns = plan.get('npm_vulns', [])
    if (not isinstance(version, str) or
        (version and not re.fullmatch(r'1\.\d+\.\d+', version)) or
            not isinstance(npm_vulns, list) or
            any(not isinstance(vuln, str) or
                not re.fullmatch(r'CVE-[0-9]{4}-[0-9]{4,}', vuln)
                for vuln in npm_vulns)):
        raise PublishError('Invalid supported remediation kinds in plan')
    images = plan.get('verify_images', [])
    if not isinstance(images, list) or not images:
        raise PublishError(
            'Plan must name the images verified on both architectures')
    expected_images = [
        image.get('image') for image in images if isinstance(image, dict)
    ]
    if (len(expected_images) != len(images) or any(
            not isinstance(name, str) or not name for name in expected_images)
            or len(set(expected_images)) != len(expected_images)):
        raise PublishError('Plan contains invalid or duplicate image names')
    required_images = {
        name: dockerfile for name, (_, dockerfile) in GO_IMAGES.items()
    } if version else {}
    if npm_vulns:
        required_images['kfp-frontend'] = 'frontend/Dockerfile'
    if (set(expected_images) != set(required_images) or any(
            image.get('context') != '.' or
            image.get('dockerfile') != required_images[image['image']]
            for image in images)):
        raise PublishError(
            'Verification must cover the complete trusted inventory for each update kind'
        )
    for name in ('remediation.patch', 'remediation.md'):
        _regular_file(bundle, name)
    patch = bundle / 'remediation.patch'
    digest = hashlib.sha256(patch.read_bytes()).hexdigest()
    for architecture in ARCHITECTURES:
        certificate = _json_file(verification /
                                 f'verification-{architecture}.json')
        if (not isinstance(certificate, dict) or
                certificate.get('source_sha') != sha or
                certificate.get('patch_sha256') != digest or
                certificate.get('architecture') != architecture or
                certificate.get('verified') is not True or
                not isinstance(certificate.get('images'), list) or
                any(not isinstance(name, str) for name in certificate['images'])
                or sorted(certificate['images']) != sorted(expected_images)):
            raise PublishError(
                f'Missing matching {architecture} verification certificate')
    tracked = set(_git(source, 'ls-files', '-z').split('\0'))
    allowed = set()
    if version:
        allowed.update(path for path in tracked if Path(path).name == 'go.mod')
        allowed.update(str(pin.path) for pin in MANAGED_DOCKERFILES)
    if npm_vulns:
        allowed.add('frontend/server/package-lock.json')
    # Git parses paths and detects binary and mode changes; do not implement
    # a second patch parser or trust filenames declared in the plan.
    if _git(source, 'apply', '--summary', str(patch)):
        raise PublishError(
            'Patch must not create, delete, rename, or change file modes')
    changes = _git(source, 'apply', '--numstat', '-z', str(patch)).split('\0')
    paths = []
    for change in filter(None, changes):
        added, removed, path = change.split('\t', 2)
        if added == '-' or removed == '-' or path not in allowed or path not in tracked:
            raise PublishError(
                f'Patch contains an unsupported file change: {path}')
        _regular_file(source, path)
        mode = _git(source, 'ls-files', '--stage', '--', path).split(' ', 1)[0]
        if mode != '100644':
            raise PublishError(
                f'Patch requires a tracked regular text file: {path}')
        paths.append(path)
    if not paths:
        raise PublishError('Remediation patch is empty')
    _git(source, 'apply', '--check', '--index', str(patch))
    return plan, patch, paths


def _validate_go_changes(source, paths, version):
    for path in paths:
        if path == 'frontend/server/package-lock.json':
            continue
        before = _git(source, 'show', f'HEAD:{path}')
        after = _git(source, 'show', f':{path}')
        if Path(path).name == 'go.mod':
            directive = re.compile(
                r'^(?:go 1\.[0-9]+(?:\.[0-9]+)?|toolchain go1\.[0-9]+\.[0-9]+)$'
            )
            retained = lambda text: [
                line for line in text.splitlines()
                if line.strip() and not directive.fullmatch(line)
            ]
            toolchain = re.findall(r'(?m)^toolchain go(1\.[0-9]+\.[0-9]+)$',
                                   after)
            go = re.findall(r'(?m)^go (1\.[0-9]+(?:\.[0-9]+)?)$', after)
            if (retained(before) != retained(after) or len(go) != 1 or
                    len(toolchain) > 1 or (toolchain or go)[0] != version):
                raise PublishError(
                    f'Go patch may only update compiler directives: {path}')
        else:
            previous = list(DOCKER_FROM_PATTERN.finditer(before))
            updated = list(DOCKER_FROM_PATTERN.finditer(after))
            if (len(previous) != 1 or len(updated) != 1 or
                    updated[0]['version'] != version or
                    previous[0]['flavor'] != updated[0]['flavor'] or
                    previous[0]['stage'] != updated[0]['stage'] or
                    DOCKER_FROM_PATTERN.sub(
                        '', before) != DOCKER_FROM_PATTERN.sub('', after)):
                raise PublishError(
                    f'Go patch may only update the existing builder pin: {path}'
                )


def _remote_sha(repository, branch):
    endpoint = f'repos/{repository}/git/matching-refs/heads/{quote(branch, safe="")}'
    refs = json.loads(_gh('api', endpoint))
    if not isinstance(refs, list):
        raise PublishError('GitHub returned an invalid branch lookup')
    for ref in refs:
        if ref.get('ref') == f'refs/heads/{branch}':
            sha = ref.get('object', {}).get('sha', '')
            if not SHA_PATTERN.fullmatch(sha):
                raise PublishError('GitHub returned an invalid branch SHA')
            return sha
    return None


def _require_current_source(repository, plan):
    if _remote_sha(repository, plan['source_branch']) != plan['source_sha']:
        raise PublishError(
            'The source branch moved; rerun the release scan before opening a fix PR'
        )


def _pr_url(repository, branch, base):
    prs = json.loads(
        _gh('pr', 'list', '--repo', repository, '--head', branch, '--state',
            'all', '--json',
            'url,state,author,baseRefName,headRefName,isCrossRepository'))
    if not prs:
        return None
    if len(prs) != 1:
        raise PublishError(
            'Multiple PRs use the remediation branch; review them manually')
    pr = prs[0]
    if (pr.get('state') != 'OPEN' or pr.get('author', {}).get('login')
            not in {BOT_NAME, 'app/github-actions'} or
            pr.get('baseRefName') != base or pr.get('headRefName') != branch or
            pr.get('isCrossRepository') is not False):
        raise PublishError(
            'The remediation branch already has a closed or unexpected PR; review it manually'
        )
    return pr['url']


def _record_url(url):
    summary = os.environ.get('GITHUB_STEP_SUMMARY')
    if summary:
        with open(summary, 'a', encoding='utf-8') as output:
            output.write(f'\nDependency fix PR: {url}\n')
    return url


def publish(source, bundle, verification, repository, source_branch):
    """Publish a verified patch or return the matching existing bot PR."""
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise PublishError('Expected repository in owner/name form')
    source, bundle, verification = source.resolve(), bundle.resolve(
    ), verification.resolve()
    plan, patch, paths = _validate_bundle(source, bundle, verification)
    if plan['source_branch'] != source_branch:
        raise PublishError(
            'Plan source branch differs from the requested release branch')
    _require_current_source(repository, plan)
    key = f'{plan["source_branch"]}:{plan["source_sha"]}'
    branch = 'codex/cve-fix-' + hashlib.sha256(key.encode()).hexdigest()[:16]
    existing = _pr_url(repository, branch, plan['source_branch'])
    _git(source, 'apply', '--index', str(patch))
    _validate_go_changes(source, paths, plan.get('go_version', ''))
    tree = _git(source, 'write-tree')
    remote_head = _remote_sha(repository, branch)
    if remote_head:
        _github_git(source, repository, 'fetch', f'refs/heads/{branch}')
        actual_head = _git(source, 'rev-parse', 'FETCH_HEAD')
        metadata = _git(source, 'show', '-s',
                        '--format=%P%n%an%n%ae%n%cn%n%ce%n%B',
                        actual_head).splitlines()
        if (actual_head != remote_head or
                _git(source, 'rev-parse', f'{actual_head}^{{tree}}') != tree or
                metadata[:5] != [
                    plan['source_sha'], BOT_NAME, BOT_EMAIL, BOT_NAME, BOT_EMAIL
                ] or
                f'Signed-off-by: {BOT_NAME} <{BOT_EMAIL}>' not in metadata[5:]):
            raise PublishError(
                'Existing remediation branch differs from the verified bot patch; review it manually'
            )
        if existing:
            _require_current_source(repository, plan)
            return _record_url(existing)
    elif existing:
        raise PublishError(
            'Existing remediation PR has no matching branch; review it manually'
        )
    else:
        _git(source, 'config', '--local', 'user.name', BOT_NAME)
        _git(source, 'config', '--local', 'user.email', BOT_EMAIL)
        title = 'fix(deps): remediate release image CVEs'
        env = dict(
            os.environ,
            GIT_AUTHOR_NAME=BOT_NAME,
            GIT_AUTHOR_EMAIL=BOT_EMAIL,
            GIT_COMMITTER_NAME=BOT_NAME,
            GIT_COMMITTER_EMAIL=BOT_EMAIL)
        _git(
            source,
            '-c',
            'commit.gpgsign=false',
            'commit',
            '-s',
            '-m',
            title,
            env=env)
        _require_current_source(repository, plan)
        # No force push: concurrent publication or human changes must not be overwritten.
        _github_git(source, repository, 'push', f'HEAD:refs/heads/{branch}')

    _require_current_source(repository, plan)
    report = (bundle / 'remediation.md').read_text(encoding='utf-8')
    body = (
        report +
        '\n\nThis dependency patch was rebuilt and rescanned on Linux AMD64 '
        'and ARM64 before publication. The original release remains blocked until '
        'its remaining CVEs are fixed or explicitly overridden.\n\n'
        'Review and merge this PR normally; automatic merge is not enabled. '
        'Because GitHub Actions created this PR with GITHUB_TOKEN, a maintainer '
        'may need to select **Approve workflows to run** for the PR checks.\n')
    with tempfile.TemporaryDirectory(prefix='release-cve-pr-') as directory:
        body_path = Path(directory) / 'body.md'
        body_path.write_text(body, encoding='utf-8')
        url = _gh('pr', 'create', '--repo', repository, '--head', branch,
                  '--base', plan['source_branch'], '--title',
                  'fix(deps): remediate release image CVEs', '--body-file',
                  str(body_path))
    return _record_url(url)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--bundle', type=Path, required=True)
    parser.add_argument('--verification', type=Path, required=True)
    parser.add_argument('--repository', required=True)
    parser.add_argument('--source-branch', required=True)
    args = parser.parse_args(argv)
    try:
        print(
            publish(args.source, args.bundle, args.verification,
                    args.repository, args.source_branch))
    except (OSError, ValueError) as error:
        print(f'Remediation PR not published: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

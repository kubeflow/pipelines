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
"""Read-only, pre-rename evidence collection; never authorizes a rename."""

import argparse
import hashlib
import json
import re
import subprocess
from datetime import datetime, timezone
from urllib.request import urlopen

import yaml

REPO = 'kubeflow/pipelines'
PROW_URL = 'https://oss.gprow.dev/config'
PREPARATION = [(REPO, 15274), ('GoogleCloudPlatform/oss-test-infra', 2762)]
RULESETS = {14287837: 'dco', 14287367: 'deletion', 14287485: 'pull_request'}
OWNER_GATES = [
    'Copybara/source mirror, Reviewable and FOSSA owners verify main support.',
    'PyPI owner verifies trusted publisher repository/workflow/environment binding.',
    'Documentation owner verifies ReadTheDocs release pins remain unchanged.',
    'Name bootstrap actor; verify existing narrow authority for the exact merge.',
    'Name rollback owner; freeze all merge paths and drain publications.',
    'Review all effective rules, bypass actors and integrations; API subset is not full policy approval.',
]


class EvidenceError(Exception):
    pass


def github(path, paginated=False):
    """Use only GET, including for paginated resources; omit raw API errors."""
    command = ['gh', 'api', '--hostname', 'github.com', '--method', 'GET', path]
    if paginated:
        command += ['--paginate', '--slurp']
    try:
        result = subprocess.run(
            command, capture_output=True, text=True, timeout=60, check=False
        )
        if result.returncode:
            raise EvidenceError(
                'GitHub GET failed; verify access and connectivity'
            )
        value = json.loads(result.stdout)
        if paginated:
            if not isinstance(value, list):
                raise EvidenceError('Invalid paginated response')
            return value
        return value
    except (OSError, subprocess.TimeoutExpired, ValueError) as exc:
        raise EvidenceError('GitHub GET unavailable or invalid JSON') from exc


def deployed_config():
    try:
        with urlopen(PROW_URL, timeout=30) as response:
            data = response.read(8 * 1024 * 1024 + 1)
        if len(data) > 8 * 1024 * 1024:
            raise EvidenceError('Deployed configuration exceeds size limit')
        value = yaml.safe_load(data)
        if not isinstance(value, dict):
            raise EvidenceError('Deployed configuration is not a mapping')
        return value, hashlib.sha256(data).hexdigest()
    except (OSError, ValueError, yaml.YAMLError) as exc:
        raise EvidenceError(
            'Public Prow configuration unavailable or invalid'
        ) from exc


def protection_ok(value):
    checks = value['required_status_checks']
    pairs = {(c['context'], c['app_id']) for c in checks['checks']}
    return (
        {('DCO', 1861), ('ci-passed', 15368)} <= pairs
        and checks['strict'] is True
        and value['allow_force_pushes']['enabled'] is False
        and value['allow_deletions']['enabled'] is False
        and value['required_linear_history']['enabled'] is True
        and value['required_conversation_resolution']['enabled'] is True
    )


def ruleset_ok(value, kind):
    refs = value['conditions']['ref_name']
    if not (
        value['enforcement'] == 'active'
        and value['target'] == 'branch'
        and {'refs/heads/master', 'refs/heads/main', '~DEFAULT_BRANCH'}
        <= set(refs['include'])
        and refs['exclude'] == []
    ):
        return False
    rules = value['rules']
    if kind == 'dco':
        return any(
            rule['type'] == 'required_status_checks'
            and {'context': 'DCO', 'integration_id': 1861}
            in rule['parameters']['required_status_checks']
            for rule in rules
        )
    kinds = {rule['type'] for rule in rules}
    return (
        {'deletion', 'non_fast_forward'} if kind == 'deletion' else {kind}
    ) <= kinds


def string_list(value):
    if not isinstance(value, list) or any(
        not isinstance(item, str) for item in value
    ):
        raise EvidenceError('Expected a list of strings')
    return value


def tide_ok(value):
    tide = value['tide']
    options = tide['context_options']
    org = options['orgs']['kubeflow']
    repo = org['repos']['pipelines']
    for branch in ('master', 'main'):
        branch_options = repo['branches'].get(branch, {})
        if 'ci-passed' not in string_list(
            branch_options.get('required-contexts', [])
        ):
            return False
        for scope in (options, org, repo, branch_options):
            if scope.get('skip-unknown-contexts', False) is not False:
                return False
            if any(
                re.search(pattern, 'ci-passed')
                for pattern in string_list(scope.get('optional-contexts', []))
            ):
                return False
    normal = bot = False
    for query in tide['queries']:
        repos = string_list(query.get('repos', []))
        orgs = string_list(query.get('orgs', []))
        relevant = REPO in repos or 'kubeflow' in orgs
        if not relevant or REPO in string_list(query.get('excludedRepos', [])):
            continue
        included = string_list(query.get('includedBranches', []))
        excluded = string_list(query.get('excludedBranches', []))
        if any(
            (included and branch not in included) or branch in excluded
            for branch in ('master', 'main')
        ):
            return False
        if 'do-not-merge/hold' not in string_list(
            query.get('missingLabels', [])
        ):
            return False
        if query.get('author') == 'dependabot[bot]':
            if 'needs-ok-to-test' not in string_list(
                query.get('missingLabels', [])
            ):
                return False
            if REPO not in string_list(query.get('repos', [])):
                return False
            bot = True
        elif not query.get('author') and {'approved', 'lgtm'} <= set(
            string_list(query.get('labels', []))
        ):
            normal = True
        else:
            return False
    return normal and bot


def policies_ok(pages):
    policies = [policy for page in pages for policy in page['branch_policies']]
    expected = {
        ('branch', 'main'),
        ('branch', 'master'),
        ('branch', 'release-*'),
        ('branch', 'kfp-kubernetes-*'),
        ('branch', 'kfp-pipeline-spec-*'),
        ('branch', 'kfp-server-api-*'),
        ('tag', 'sdk-*'),
    }
    return {(p['type'], p['name']) for p in policies} == expected


def collect(expected_sha=None, get=github, fetch_config=deployed_config):
    """Collect independent checks even if another endpoint fails."""
    checks = []
    evidence = {}

    def check(name, read, validate):
        try:
            value = read()
            passed = validate(value)
            checks.append(
                {'name': name, 'status': 'PASS' if passed else 'BLOCKED'}
            )
            return value if passed else None
        except (
            re.error,
            EvidenceError,
            KeyError,
            TypeError,
            ValueError,
            AttributeError,
        ) as exc:
            # Do not publish arbitrary remote content, auth errors or hook URLs.
            checks.append(
                {
                    'name': name,
                    'status': 'UNKNOWN',
                    'reason': type(exc).__name__,
                }
            )
            return None

    prefix = f'repos/{REPO}'
    metadata = check(
        'default branch is master',
        lambda: get(prefix),
        lambda v: v['default_branch'] == 'master',
    )
    start = check(
        'master SHA recorded',
        lambda: get(f'{prefix}/git/ref/heads/master'),
        lambda v: bool(re.fullmatch('[0-9a-f]{40}', v['object']['sha'])),
    )
    start_sha = (
        start.get('object', {}).get('sha') if isinstance(start, dict) else None
    )
    if start_sha and re.fullmatch('[0-9a-f]{40}', start_sha):
        evidence['master_sha'] = start_sha
    if expected_sha:
        check(
            'master matches operator SHA',
            lambda: start_sha,
            lambda v: v == expected_sha,
        )
    check(
        'main does not already exist',
        lambda: get(f'{prefix}/branches?per_page=100', paginated=True),
        lambda pages: (
            'master' in [b['name'] for page in pages for b in page]
            and all(b['name'] != 'main' for page in pages for b in page)
        ),
    )
    for repo, number in PREPARATION:
        check(
            f'{repo}#{number} merged',
            lambda repo=repo, number=number: get(
                f'repos/{repo}/pulls/{number}'
            ),
            lambda v: (
                v['merged'] is True
                and v['base']['ref'] == 'master'
                and bool(v['merge_commit_sha'])
            ),
        )
    check(
        'classic protection baseline',
        lambda: get(f'{prefix}/branches/master/protection'),
        protection_ok,
    )
    for rule_id, kind in RULESETS.items():
        check(
            f'ruleset {rule_id} covers both branches',
            lambda rule_id=rule_id: get(f'{prefix}/rulesets/{rule_id}'),
            lambda v, kind=kind: ruleset_ok(v, kind),
        )
    check(
        'pypi custom branch policy',
        lambda: get(f'{prefix}/environments/pypi'),
        lambda v: (
            v['deployment_branch_policy']
            == {'protected_branches': False, 'custom_branch_policies': True}
            and any(p['type'] == 'branch_policy' for p in v['protection_rules'])
        ),
    )
    check(
        'pypi branch/tag allowlist preserved with main',
        lambda: get(
            f'{prefix}/environments/pypi/deployment-branch-policies?per_page=100',
            paginated=True,
        ),
        policies_ok,
    )
    check(
        'OIDC default subject template',
        lambda: get(f'{prefix}/actions/oidc/customization/sub'),
        lambda v: v['use_default'] is True,
    )

    def read_config():
        config, digest = fetch_config()
        evidence['prow_config_sha256'] = digest
        revision = config.get('config_version_sha')
        if not isinstance(revision, str) or not re.fullmatch(
            '[0-9a-f]{40}', revision
        ):
            raise EvidenceError('No deployed revision')
        evidence['prow_revision'] = revision
        return config

    check('deployed Tide accepts and gates both branches', read_config, tide_ok)
    check(
        'master SHA unchanged during collection',
        lambda: get(f'{prefix}/git/ref/heads/master'),
        lambda v: bool(start_sha) and v['object']['sha'] == start_sha,
    )
    check(
        'default branch unchanged during collection',
        lambda: get(prefix),
        lambda v: (
            metadata is not None
            and v['default_branch'] == metadata['default_branch'] == 'master'
        ),
    )
    automated_ready = all(c['status'] == 'PASS' for c in checks)
    return {
        'schema_version': 1,
        'checked_at': datetime.now(timezone.utc).isoformat(),
        'phase': 'pre-rename',
        'decision': 'OWNER_REVIEW_REQUIRED' if automated_ready else 'NO-GO',
        'automated_checks_passed': automated_ready,
        'checks': checks,
        'evidence': evidence,
        'owner_gates': OWNER_GATES,
        'limitations': 'Point-in-time subset of policy checks. Rerun in the frozen window; not rename or bootstrap authorization.',
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        '--expected-sha',
        help='Optional recorded 40-character master SHA; drift blocks readiness',
    )
    parser.add_argument(
        '--json', action='store_true', help='Print a sanitized JSON report'
    )
    args = parser.parse_args()
    if args.expected_sha and not re.fullmatch(
        '[0-9a-f]{40}', args.expected_sha
    ):
        parser.error(
            '--expected-sha must be a lowercase 40-character commit SHA'
        )
    report = collect(args.expected_sha)
    if args.json:
        print(json.dumps(report, indent=2))
    else:
        print(f'{report["decision"]} — pre-rename readiness')
        for check in report['checks']:
            print(f'{check["status"]}: {check["name"]}')
        print('Owner confirmations still required:')
        for gate in report['owner_gates']:
            print(f'- {gate}')
        print(report['limitations'])
    return 2 if report['automated_checks_passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())

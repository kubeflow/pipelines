"""Scan an immutable open PR candidate without creating release approval."""

import json
import re
from dataclasses import dataclass
from pathlib import Path

from .ci_images_client import REPO, _api
from .ci_qualification_client import collect_ci
from .metadata import ReleaseMetadata
from .qualification import release_coverage
from .runner import CommandRunner
from .security import evaluate_policy


@dataclass
class CandidateContext:
    root: Path
    runner: CommandRunner
    metadata: ReleaseMetadata
    include_backend: bool
    include_sdk: bool


def validate_pr(context, number, expected_head):
    """Require an open upstream PR with the expected release base and head."""
    data = _api(context, f'pulls/{number}')
    expected_url = f'https://github.com/{REPO}/pull/{number}'
    base = data.get('base') or {}
    head = data.get('head') or {}
    if (
        data.get('number') != number
        or data.get('html_url') != expected_url
        or (base.get('repo') or {}).get('full_name') != REPO
    ):
        raise ValueError('Candidate PR must belong to ' + REPO)
    if data.get('state') != 'open' or data.get('merged') is not False:
        raise ValueError('Candidate PR must be open and unmerged')
    if base.get('ref') != context.metadata.release_branch:
        raise ValueError(
            'Candidate PR must target ' + context.metadata.release_branch
        )
    if head.get('sha') != expected_head:
        raise ValueError(
            'Candidate PR head changed; select its new exact SHA and rescan'
        )
    repository = (head.get('repo') or {}).get('full_name')
    if not isinstance(repository, str) or not re.fullmatch(
        r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository
    ):
        raise ValueError('Candidate PR has no valid source repository')
    return {
        'number': number,
        'url': expected_url,
        'base': base['ref'],
        'head_sha': expected_head,
        'head_repository': repository,
    }


def scan_candidate(
    *,
    pr,
    expected_head,
    version,
    output_dir,
    include_backend=True,
    include_sdk=True,
    resume=False,
    runner=None,
):
    """Collect candidate-only evidence; never load or save release state."""
    if type(pr) is not int or pr < 1:
        raise ValueError('Provide a positive upstream PR number')
    if not re.fullmatch(r'[0-9a-f]{40}', expected_head):
        raise ValueError('Provide the exact 40-character lowercase PR head SHA')
    if not (include_backend or include_sdk):
        raise ValueError('Select backend images, SDK packages, or both')
    metadata = ReleaseMetadata.from_version('patch', version)
    output = Path(output_dir).resolve()
    record_path = output / 'candidate.json'
    expected = {
        'schema': 1,
        'purpose': 'premerge-candidate',
        'pr': pr,
        'source_sha': expected_head,
        'version': version,
        'include_backend': include_backend,
        'include_sdk': include_sdk,
    }
    if resume:
        report = json.loads(record_path.read_text())
        if not isinstance(report, dict) or any(
            report.get(k) != v for k, v in expected.items()
        ):
            raise ValueError(
                'Candidate record does not match the requested PR, head, version or scope'
            )
    else:
        output.mkdir(parents=True, exist_ok=True)
        if any(output.iterdir()):
            raise ValueError(
                'Use an empty candidate output directory or --resume'
            )
        report = dict(expected)
    context = CandidateContext(
        output,
        runner or CommandRunner(),
        metadata,
        include_backend,
        include_sdk,
    )
    try:
        before = validate_pr(context, pr, expected_head)
    except (Exception, KeyboardInterrupt) as error:
        if resume:
            report.update(status='stale', stale_reason=str(error), passed=False)
            record_path.write_text(json.dumps(report, indent=2) + '\n')
        raise
    if resume and report.get('pull_request') != before:
        report.update(
            status='stale',
            stale_reason='Candidate PR identity changed',
            passed=False,
        )
        record_path.write_text(json.dumps(report, indent=2) + '\n')
        raise ValueError(
            'Candidate PR identity changed; use a new scan directory'
        )
    for key in ('error', 'stale_reason'):
        report.pop(key, None)
    report.update(pull_request=before, status='collecting', passed=False)
    record_path.write_text(json.dumps(report, indent=2) + '\n')
    try:
        source = output / 'source.git'
        if not source.exists():
            context.runner.run(
                ['git', 'init', '--bare', str(source)], cwd=output
            )
        context.runner.run(
            [
                'git',
                'fetch',
                '--no-tags',
                '--depth=1',
                f'https://github.com/{REPO}.git',
                f'refs/pull/{pr}/head',
            ],
            cwd=source,
        )
        actual = context.runner.capture(
            ['git', 'rev-parse', 'FETCH_HEAD'], cwd=source
        )
        if actual != expected_head:
            raise ValueError(
                'Fetched PR head changed; select its new SHA and rescan'
            )
        context.root = source
        if include_backend:
            actual_version = context.runner.capture(
                ['git', 'show', f'{expected_head}:VERSION'], cwd=source
            )
            if actual_version != version:
                raise ValueError(
                    'Candidate VERSION does not match the requested version'
                )
        images, minors = release_coverage(context, expected_head)
        expected_images = [
            {'name': name, 'platform': platform}
            for name, platform in sorted(images)
        ]
        if validate_pr(context, pr, expected_head) != before:
            raise ValueError('Candidate PR identity changed before dispatch')
        result = collect_ci(
            context,
            expected_head,
            expected_images,
            sorted(minors),
            output / 'ci',
            resume=resume,
        )
        (output / 'evidence.json').write_text(
            json.dumps(result, indent=2) + '\n'
        )
        policy = evaluate_policy(result['scans'], release_version=version)
        (output / 'policy.json').write_text(json.dumps(policy, indent=2) + '\n')
        report.update(
            run=result['run'],
            policy='zero-fixable',
            blocking_count=policy['blocking_count'],
            residual_count=policy['residual_count'],
            status='passed' if not policy['blocking_count'] else 'findings',
            passed=not policy['blocking_count'],
        )
    except (Exception, KeyboardInterrupt) as error:
        report.update(status='incomplete', error=str(error), passed=False)
        raise
    finally:
        try:
            if validate_pr(context, pr, expected_head) != before:
                raise ValueError(
                    'Candidate PR identity changed during scanning'
                )
        except (Exception, KeyboardInterrupt) as error:
            report.update(status='stale', stale_reason=str(error), passed=False)
            raise
        finally:
            record_path.write_text(json.dumps(report, indent=2) + '\n')
    return report

"""GitHub repository lookups and pull request and workflow polling."""

import datetime
import json
import time
from pathlib import Path

from .prompts import underline_links
from .runner import CommandRunner

REPO = 'kubeflow/pipelines'


def default_branch(api) -> str:
    """Resolve the upstream development branch without a stale-name
    fallback."""
    branch = api('').get('default_branch')
    if branch not in ('master', 'main'):
        raise ValueError(f'Unsupported upstream default branch: {branch!r}')
    return branch


def upstream_default_branch(runner, root: Path) -> str:
    """Read the authenticated upstream repository's development branch."""
    metadata = runner.capture(['gh', 'api', f'repos/{REPO}'], cwd=root)
    if runner.dry_run and not metadata:
        # Dry runs do not perform API reads or persist this placeholder.
        return '<repository-default>'
    return default_branch(lambda _: json.loads(metadata))


def wait_for_pr_merge(runner: CommandRunner, pr_url: str) -> None:
    """Poll PR status until merged.

    Args:
      runner: CommandRunner instance for executing gh commands.
      pr_url: URL of the pull request to monitor.
    """
    print(underline_links(f'Waiting for PR merge: {pr_url}'))
    if runner.dry_run:
        print('[dry-run] PR merge check skipped')
        return
    while True:
        state = runner.capture(
            [
                'gh',
                'pr',
                'view',
                pr_url,
                '--json',
                'state,mergeStateStatus,url',
                '--jq',
                '.state + " " + .mergeStateStatus',
            ]
        )
        if state.startswith('MERGED '):
            return
        if state.startswith('CLOSED '):
            raise RuntimeError(f'PR was closed without merging: {pr_url}')
        if 'DIRTY' in state or 'UNKNOWN' in state:
            print(f'PR is not ready: {state}')
        time.sleep(60)


def is_pending_merge_queue_check(check: dict) -> bool:
    """Tide's pending merge eligibility is not an unfinished CI test."""
    return (
        check.get('__typename') == 'StatusContext'
        and check.get('context') == 'tide'
        and check.get('targetUrl') == 'https://oss.gprow.dev/tide'
        and check.get('state') == 'PENDING'
    )


def watch_pr_ci(runner: CommandRunner, pr_url: str) -> None:
    """Poll PR checks until one fails or all reported checks complete.

    Args:
      runner: CommandRunner instance for executing gh commands.
      pr_url: URL of the pull request to monitor.
    """
    print(underline_links(f'Watching PR CI: {pr_url}'))
    if runner.dry_run:
        print('[dry-run] PR CI watch skipped')
        return
    reported_check_urls = set()
    while True:
        checks = json.loads(
            runner.capture(
                [
                    'gh',
                    'pr',
                    'view',
                    pr_url,
                    '--json',
                    'statusCheckRollup',
                    '--jq',
                    '.statusCheckRollup',
                ]
            )
        )
        checks = [
            check for check in checks if not is_pending_merge_queue_check(check)
        ]
        if not checks:
            print('No CI gates reported yet')
            time.sleep(60)
            continue
        latest_checks = {}
        for check in checks:
            name = (
                check.get('name')
                or check.get('workflowName')
                or check.get('context')
                or 'unknown'
            )
            timestamp = check.get('startedAt') or check.get('completedAt') or ''
            previous = latest_checks.get(name)
            if previous is None or timestamp >= previous[0]:
                latest_checks[name] = (timestamp, check)
        completed = 0
        for check in [item[1] for item in latest_checks.values()]:
            name = (
                check.get('name')
                or check.get('workflowName')
                or check.get('context')
                or 'unknown'
            )
            status = check.get('status') or check.get('state')
            conclusion = check.get('conclusion')
            details_url = check.get('detailsUrl')
            if (
                conclusion
                and conclusion not in ('SUCCESS', 'NEUTRAL', 'SKIPPED')
                or status in ('FAILURE', 'ERROR')
            ):
                suffix = f': {details_url}' if details_url else ''
                raise RuntimeError(f'{name} failed{suffix}')
            if status == 'COMPLETED' or status in (
                'SUCCESS',
                'NEUTRAL',
                'SKIPPED',
            ):
                completed += 1
            elif details_url and details_url not in reported_check_urls:
                reported_check_urls.add(details_url)
                print(underline_links(f'Watching check: {name}: {details_url}'))
        if completed == len(latest_checks):
            print(
                f'All PR CI gates completed successfully ({completed}/{len(latest_checks)})'
            )
            return
        print(
            f'PR CI still running ({completed}/{len(latest_checks)} complete)'
        )
        time.sleep(60)


def watch_latest_workflow_run(
    runner: CommandRunner,
    workflow: str,
    branch: str,
    fresh_only: bool = True,
    expected_sha: str | None = None,
) -> None:
    """Watch the latest workflow run for a given workflow and branch.

    Args:
      runner: CommandRunner instance for executing gh commands.
      workflow: Workflow filename to watch.
      branch: Branch name to filter runs by.
    """
    if runner.dry_run:
        print(
            f'[dry-run] Watching latest workflow run skipped: {workflow} on {branch}'
        )
        return
    started_at = datetime.datetime.fromtimestamp(
        time.time() - 10, datetime.timezone.utc
    )
    command = [
        'gh',
        'run',
        'list',
        '--workflow',
        workflow,
        '--branch',
        branch,
        '--limit',
        '1',
        '--json',
        'databaseId,url,createdAt',
        '--jq',
        '.[0] | [.databaseId, .url, .createdAt] | @tsv',
    ]
    if expected_sha:
        index = command.index('--branch')
        command[index : index + 2] = ['--commit', expected_sha]
        command.extend(['--repo', REPO, '--event', 'workflow_dispatch'])
    run_info = ''
    for _ in range(12):
        candidate = runner.capture(command)
        if candidate:
            _, _, created_at = candidate.split('\t', 2)
            created = datetime.datetime.fromisoformat(
                created_at.replace('Z', '+00:00')
            )
            if not fresh_only or created >= started_at:
                run_info = candidate
                break
        time.sleep(5)
    if not run_info:
        raise RuntimeError(
            f'Could not find workflow run for {workflow} on {branch}.'
        )
    run_id, run_url, _ = run_info.split('\t', 2)
    print(underline_links(f'Workflow run: {run_url}'))

    def verify_source():
        if expected_sha:
            actual = runner.capture(
                [
                    'gh',
                    'run',
                    'view',
                    run_id,
                    '--repo',
                    REPO,
                    '--json',
                    'headSha',
                    '--jq',
                    '.headSha',
                ]
            )
            if actual.strip() != expected_sha:
                raise RuntimeError(
                    f'Workflow {run_id} does not use the qualified release commit.'
                )

    verify_source()
    watch = ['gh', 'run', 'watch', run_id, '--exit-status']
    if expected_sha:
        watch.extend(['--repo', REPO])
    runner.run(watch)
    verify_source()

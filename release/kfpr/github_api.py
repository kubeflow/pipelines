"""GitHub repository lookups and pull request and workflow polling."""

import datetime
import json
import re
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


def is_pending_merge_queue_check(
    check: dict, *, merged_pr_url: str | None = None
) -> bool:
    """Identify queue bookkeeping that can outlive CI and the PR itself."""
    if (
        check.get('__typename') != 'StatusContext'
        or check.get('state') != 'PENDING'
    ):
        return False
    if check.get('context') == 'tide':
        return check.get('targetUrl') == 'https://oss.gprow.dev/tide'
    return (
        check.get('context') == 'merge-queue-admission'
        and merged_pr_url is not None
        and re.fullmatch(
            r'https://github\.com/kubeflow/pipelines/pull/[1-9][0-9]*',
            merged_pr_url,
        )
        is not None
        and check.get('targetUrl') == merged_pr_url
    )


def has_successful_ci_passed(checks: list[dict]) -> bool:
    aggregate = [
        check
        for check in checks
        if check.get('__typename') == 'StatusContext'
        and check.get('context') == 'ci-passed'
    ]
    return bool(aggregate) and all(
        check.get('state') == 'SUCCESS' for check in aggregate
    )


def superseded_ci_bookkeeping(api, check, checks, head):
    """Recognize cancelled aggregate/admission jobs followed by actual
    success."""
    # The reusable jobs keep the same duties as their original inline jobs.
    job_names = {
        'queue_admission': 'queue_admission / reconcile',
        'check_ci_status (0)': 'check_ci_status (0) / publish',
    }
    name = job_names.get(check.get('name'), check.get('name'))
    if not (
        re.fullmatch('[0-9a-f]{40}', head or '')
        and check.get('__typename') == 'CheckRun'
        and check.get('workflowName') == 'CI Check'
        and name
        in (*job_names.values(), 'check_ci_status (0) / wake_admission')
        and check.get('status') == 'COMPLETED'
        and check.get('conclusion') == 'CANCELLED'
        and check.get('startedAt')
    ):
        return False
    successors = [
        other
        for other in checks
        if all(
            other.get(key) == check[key]
            for key in ('__typename', 'workflowName', 'status')
        )
        and job_names.get(other.get('name'), other.get('name')) == name
        and other.get('conclusion') == 'SUCCESS'
        and (other.get('startedAt') or '') > check['startedAt']
    ]
    if not successors:
        return False
    newest = max(successors, key=lambda other: other['startedAt'])
    return all(trusted_ci_check(api, job, head) for job in (check, newest))


def trusted_ci_check(api, job, head):
    match = re.fullmatch(
        rf'https://github[.]com/{REPO}/actions/runs/([1-9][0-9]*)/job/[1-9][0-9]*',
        job.get('detailsUrl') or '',
    )
    if not match or not re.fullmatch('[0-9a-f]{40}', head or ''):
        return False
    run = api(f'actions/runs/{match[1]}')
    return (
        run.get('path') == '.github/workflows/ci-checks.yml'
        and run.get('event') == 'pull_request_target'
        and run.get('head_sha') == head
        and (run.get('repository') or {}).get('full_name') == REPO
    )


def cancelled_queue_notification(api, check, head):
    return (
        check.get('__typename') == 'CheckRun'
        and check.get('workflowName') == 'CI Check'
        and check.get('name') == 'check_ci_status (0) / wake_admission'
        and check.get('status') == 'COMPLETED'
        and check.get('conclusion') == 'CANCELLED'
        and trusted_ci_check(api, check, head)
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

    def read_run(endpoint):
        return json.loads(
            runner.capture(['gh', 'api', f'repos/{REPO}/{endpoint}'])
        )

    reported_check_urls = set()
    while True:
        pr = json.loads(
            runner.capture(
                [
                    'gh',
                    'pr',
                    'view',
                    pr_url,
                    '--json',
                    'state,url,headRefOid,statusCheckRollup',
                ]
            )
        )
        checks = [
            check
            for check in pr['statusCheckRollup']
            if not is_pending_merge_queue_check(
                check,
                merged_pr_url=pr.get('url')
                if pr['state'] == 'MERGED'
                else None,
            )
        ]
        if not checks:
            print('No CI gates reported yet')
            time.sleep(60)
            continue
        latest_checks = {}
        for check in checks:
            head = pr.get('headRefOid')
            if (
                pr['state'] == 'MERGED'
                and cancelled_queue_notification(read_run, check, head)
                or superseded_ci_bookkeeping(read_run, check, checks, head)
            ):
                continue
            name = (
                check.get('name')
                or check.get('workflowName')
                or check.get('context')
                or 'unknown'
            )
            timestamp = check.get('startedAt') or check.get('completedAt') or ''
            # A CheckRun and StatusContext can share a display name.
            key = (check.get('__typename'), check.get('workflowName'), name)
            previous = latest_checks.get(key)
            if previous is None or timestamp >= previous[0]:
                latest_checks[key] = (timestamp, check)
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
            if pr['state'] == 'MERGED' and not has_successful_ci_passed(checks):
                print('Waiting for successful ci-passed on the merged PR')
                time.sleep(60)
                continue
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

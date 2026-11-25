"""Read and revalidate an upstream attestation for a merged release commit."""

import json
import re
import tempfile
from pathlib import Path

from .ci_images_client import _api
from .github_api import REPO, default_branch
from .workflow_runs import validate_run

WORKFLOW = '.github/workflows/release-landed-ci.yml'


def verify_landed_attestation(context, run_id, pr, branch, sha):
    """Bind a successful default-branch workflow and its evidence to the
    cut."""
    from .landed_ci import collect_evidence

    if type(run_id) is not int or run_id < 1:
        raise ValueError('Expected a positive landed CI run ID')
    endpoint = f'actions/runs/{run_id}'
    run = _api(context, endpoint)
    ref = default_branch(lambda path: _api(context, path))
    workflow = _api(context, 'actions/workflows/release-landed-ci.yml')
    workflow_sha = run.get('head_sha', '')
    attempt = run.get('run_attempt')
    if (
        ref != 'main'
        or workflow.get('path') != WORKFLOW
        or type(workflow.get('id')) is not int
        or run.get('workflow_id') != workflow['id']
        or (run.get('head_repository') or {}).get('full_name') != REPO
        or not re.fullmatch('[0-9a-f]{40}', workflow_sha)
        or type(attempt) is not int
        or attempt < 1
        or context.state.answers.get('landed_ci_attempt', attempt) != attempt
    ):
        raise ValueError('Invalid landed CI workflow or changed run attempt')
    identity = {
        'run_id': run_id,
        'workflow': WORKFLOW,
        'workflow_sha': workflow_sha,
        'workflow_ref': ref,
        'title': f'Release landed CI #{pr} at {sha}',
        'run_attempt': attempt,
    }
    validate_run(run, identity)
    comparison = _api(context, f'compare/{workflow_sha}...{ref}?per_page=1')
    if (
        comparison.get('status') not in ('ahead', 'identical')
        or (comparison.get('merge_base_commit') or {}).get('sha')
        != workflow_sha
    ):
        raise ValueError('Landed CI workflow revision is outside upstream main')

    jobs = _api(context, f'{endpoint}/attempts/{attempt}/jobs?per_page=100')
    if jobs.get('total_count') != 1 or [
        (job.get('name'), job.get('status'), job.get('conclusion'))
        for job in jobs.get('jobs', [])
    ] != [('ci-passed', 'completed', 'success')]:
        raise ValueError('The landed ci-passed job must complete successfully')
    name = f'landed-ci-{run_id}-{attempt}'
    artifacts = _api(context, f'{endpoint}/artifacts?name={name}')
    if artifacts.get('total_count') != 1 or [
        (artifact.get('name'), artifact.get('expired'))
        for artifact in artifacts.get('artifacts', [])
    ] != [(name, False)]:
        raise ValueError('Missing or expired landed CI evidence artifact')
    with tempfile.TemporaryDirectory() as directory:
        context.runner.run(
            [
                'gh',
                'run',
                'download',
                str(run_id),
                '--repo',
                REPO,
                '--name',
                name,
                '--dir',
                directory,
            ],
            cwd=context.root,
        )
        path = Path(directory) / 'landed-ci.json'
        if path.is_symlink() or not path.is_file():
            raise ValueError('Missing landed CI evidence document')
        document = json.loads(path.read_text())
    replay_pr = document.get('replay_pr')
    if type(replay_pr) is not int or replay_pr < 1:
        raise ValueError('Invalid replay PR in landed CI evidence')
    current_branch, evidence = collect_evidence(pr, sha, replay_pr)
    expected = {
        'schema_version': 1,
        'repository': REPO,
        'pr': pr,
        'replay_pr': replay_pr,
        'branch': branch,
        'sha': sha,
        'run_id': run_id,
        'run_attempt': attempt,
        'workflow_sha': workflow_sha,
        'evidence': evidence,
    }
    if current_branch != branch or not evidence or document != expected:
        raise ValueError(
            'Landed CI evidence differs from current upstream results'
        )
    validate_run(_api(context, endpoint), identity)
    context.state.answers['landed_ci_attempt'] = attempt


def superseded_ci_bookkeeping(context, check, checks, head):
    """Recognize cancelled aggregate/admission jobs followed by actual
    success."""
    # The reusable jobs keep the same duties as their original inline jobs.
    job_names = {
        'queue_admission': 'queue_admission / reconcile',
        'check_ci_status (0)': 'check_ci_status (0) / publish',
    }
    name = job_names.get(check.get('name'), check.get('name'))
    if not (
        check.get('__typename') == 'CheckRun'
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
    for job in (check, newest):
        match = re.fullmatch(
            rf'https://github[.]com/{REPO}/actions/runs/([1-9][0-9]*)/job/[1-9][0-9]*',
            job.get('detailsUrl') or '',
        )
        if not match:
            return False
        run = _api(context, f'actions/runs/{match[1]}')
        if (
            run.get('path') != '.github/workflows/ci-checks.yml'
            or run.get('event') != 'pull_request_target'
            or run.get('head_sha') != head
            or (run.get('repository') or {}).get('full_name') != REPO
        ):
            return False
    return True

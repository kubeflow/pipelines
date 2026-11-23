"""Verify landed 2.18 CI using an exact-tree PR replay and branch-only
tests."""

import argparse
import datetime
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

REPO = 'kubeflow/pipelines'
SCRIPTS = Path(__file__).resolve().parents[2] / '.github/resources/scripts'
PREFIX = '.github/workflows/'
BRANCH = 'release-2.18'
# Branch filters prevent these PR jobs from running on the replay base. Only
# these reviewed definitions provide equivalent coverage on the release ref.
BRANCH_RUNS = {
    'pre-commit.yml': ('push', '4585f3b3575d1972eeeb11ed6a8499282d43840c'),
    'build-tools-images.yml': (
        'push',
        '799845961aa00c7399d431dcbd494491a12b0540',
    ),
    'postgresql-image-upgrade.yml': (
        'workflow_dispatch',
        '57a75aa09d750dbb96444db6d1865495c28e4757',
    ),
}
UPGRADE_PATH = PREFIX + 'upgrade-test.yml'
OPTIONAL_UPGRADE_JOBS = {
    'custom-role-acceptance',
    'readiness-reporting',
    'readiness-adoption',
    'readiness-schedules',
}

# For each test job family, require its substantive validation steps. GitHub
# may report success for a workflow with skipped tests or only image builds.
TEST_STEPS = {
    'api-server-tests.yml': {
        'KFP API Server tests Standalone': ('Run Tests',),
        'KFP API Server Multi User Tests': ('Run Tests',),
        'KFP API Server tests K8s Native API': ('Run Tests',),
    },
    'e2e-test.yml': {
        'End to End ' + label + ' Tests': ('Run Tests',)
        for label in (
            'E2ECritical',
            'E2ECriticalShardA',
            'E2ECriticalShardB',
            'E2EEssential',
            'E2EParallelNested',
            'E2EFailure',
            'E2EProxy',
            'dra-check',
            'gpu-scheduling-check',
            'Critical Scenario MLflow',
            'Critical Scenario Multi User E2ECriticalShardA',
            'Critical Scenario Multi User E2ECriticalShardB',
        )
    },
    'e2e-test-frontend.yml': {
        'Frontend Integration Dependency Checks': (
            'Dependency compatibility checks',
        ),
        'Frontend Integration Tests': (
            'Server integration tests',
            'Frontend integration tests',
        ),
    },
    'integration-tests-v1.yml': {
        'Initialization & Integration tests v1': (
            'Initialization tests v1',
            'API integration tests v1',
        ),
    },
    'legacy-v2-api-integration-tests.yml': {
        'API integration tests v2': ('API integration tests v2',)
    },
    'kfp-kubernetes-native-migration-tests.yaml': {
        'KFP Kubernetes Native Migration Tests': (
            'Run DB mode migration tests',
            'Run K8s mode migration tests',
        ),
    },
    'kubeflow-pipelines-manifests.yml': {
        'kubeflow-pipelines-manifests': ('Run tests',)
    },
    'upgrade-test.yml': {'KFP upgrade tests': ('Verify Upgrade',)},
    'pre-commit.yml': {'pre-commit': ('Run pre-commit on changed files',)},
    'build-tools-images.yml': {
        'Publish tools images': (
            'Build and push api-generator',
            'Build and push release tools',
        ),
    },
    'postgresql-image-upgrade.yml': {
        f'Retained PostgreSQL {major} data (image compatibility only)': (
            'Upgrade using the rendered deployment and retained PVC',
        )
        for major in (14, 18)
    },
}


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def positive(value):
    return type(value) is int and value > 0


def api(endpoint):
    return json.loads(
        subprocess.check_output(
            ['gh', 'api', f'repos/{REPO}/{endpoint}'], timeout=90
        )
    )


def paged(endpoint, field):
    separator = '&' if '?' in endpoint else '?'
    rows = []
    expected_total = None
    while True:
        page = api(
            f'{endpoint}{separator}per_page=100&page={len(rows) // 100 + 1}'
        )
        batch, total = page[field], page['total_count']
        require(
            type(total) is int
            and 0 <= total < 1000
            and isinstance(batch, list),
            f'Incomplete {field} history',
        )
        require(
            expected_total in (None, total) and len(batch) <= 100,
            f'Changing {field} history',
        )
        expected_total = total
        rows.extend(batch)
        if len(rows) == total:
            require(
                len({row['id'] for row in rows}) == total,
                f'Duplicate {field} entries',
            )
            return rows
        require(
            len(batch) == 100 and len(rows) < total, f'Incomplete {field} page'
        )


def binding(pr, sha, replay_pr):
    require(
        positive(pr)
        and positive(replay_pr)
        and pr != replay_pr
        and re.fullmatch(r'[a-f0-9]{40}', sha),
        'Invalid release or replay identity',
    )
    original = api(f'pulls/{pr}')
    replay = api(f'pulls/{replay_pr}')
    require(
        original['number'] == pr
        and original['merged'] is True
        and original['state'] == 'closed'
        and original['base']['ref'] == BRANCH
        and original['base']['repo']['full_name'] == REPO
        and original['merge_commit_sha'] == sha,
        'Release PR does not bind this landed commit',
    )
    tip = api(f'git/ref/heads/{BRANCH}')
    require(
        tip['ref'] == f'refs/heads/{BRANCH}'
        and tip['object']['type'] == 'commit'
        and tip['object']['sha'] == sha,
        'Release branch moved',
    )
    commit = api(f'git/commits/{sha}')
    require(
        commit['sha'] == sha and len(commit['parents']) == 1,
        'Release replay requires a single landed parent',
    )
    parent = commit['parents'][0]['sha']
    require(
        re.fullmatch(r'[a-f0-9]{40}', parent)
        and replay['number'] == replay_pr
        and replay['state'] == 'open'
        and replay['merged'] is False,
        'Replay PR must be open and unmerged',
    )
    for key, ref, revision in [
        ('head', f'ci/release-candidate/{sha}', sha),
        ('base', f'ci/release-replay/{sha}', parent),
    ]:
        side = replay[key]
        require(
            side['repo']['full_name'] == REPO
            and side['ref'] == ref
            and side['sha'] == revision,
            f'Unexpected replay {key}',
        )
    merge = replay['merge_commit_sha']
    require(
        isinstance(merge, str) and re.fullmatch(r'[a-f0-9]{40}', merge),
        'GitHub has not prepared the replay test merge',
    )
    tested = api(f'git/commits/{merge}')
    require(
        tested['sha'] == merge
        and tested['tree']['sha'] == commit['tree']['sha']
        and [p['sha'] for p in tested['parents']] == [parent, sha],
        'Replay did not test the landed tree and parents',
    )
    return parent, replay


def expected_workflows(parent, replay):
    # Limit recovery to bounded, reviewable changes. Never evaluate a change to
    # Actions configuration against the rules that preceded that change.
    files = api(f'pulls/{replay["number"]}/files?per_page=100')
    require(
        0 < len(files) == replay['changed_files'] <= 100,
        'Incomplete replay change list',
    )
    paths = [
        item[key]
        for item in files
        for key in ('filename', 'previous_filename')
        if key in item
    ]
    require(
        all(
            isinstance(p, str) and p and not p.startswith('.github/')
            for p in paths
        ),
        'Release CI configuration changes require normal PR qualification',
    )
    policy = SCRIPTS / 'release_replay_policy.js'
    require(
        policy.is_file(),
        'Landed CI requires kfpr from a trusted full source checkout',
    )
    environment = dict(
        os.environ,
        PATH=str(Path(sys.executable).parent) + os.pathsep + os.environ['PATH'],
    )
    return json.loads(
        subprocess.check_output(
            ['node', str(policy)],
            input=json.dumps(dict(branch=BRANCH, parent=parent, paths=paths)),
            text=True,
            env=environment,
            timeout=120,
        )
    )


def select_run(runs, path, event, ref, sha):
    candidates = [
        r
        for r in runs
        if r['path'] == path
        and r['event'] == event
        and r.get('head_branch') == ref
        and r.get('head_sha') == sha
        and r['repository']['full_name'] == REPO
        and r['head_repository']['full_name'] == REPO
    ]
    require(candidates, f'Missing {event} CI for {path} on {ref}')

    def ordering(run):
        timestamps = [datetime.datetime.fromisoformat(run['created_at'])]
        if run.get('run_started_at'):
            timestamps.append(
                datetime.datetime.fromisoformat(run['run_started_at'])
            )
        return max(timestamps), run['id']

    run = max(candidates, key=ordering)
    require(
        positive(run['id'])
        and positive(run['run_attempt'])
        and run['status'] == 'completed'
        and run['conclusion'] == 'success',
        f'{path} run {run["id"]} is {run["conclusion"] or run["status"]}',
    )
    return run


def verify_replay_run(run, replay):
    prs = run['pull_requests']
    require(
        len(prs) == 1
        and prs[0]['number'] == replay['number']
        and all(
            prs[0][side][key] == replay[side][key]
            for side in ('base', 'head')
            for key in ('sha', 'ref')
        )
        and all(
            prs[0][side]['repo']['id'] == replay[side]['repo']['id']
            for side in ('base', 'head')
        ),
        'Workflow run belongs to a different replay',
    )
    # Actions records the immutable ref and commit used for local reusable
    # workflows. A PR run's head_sha alone cannot identify the test merge.
    merge = replay['merge_commit_sha']
    refs = run['referenced_workflows']
    require(
        refs
        and all(
            item.get('ref') == f'refs/pull/{replay["number"]}/merge'
            and item.get('sha') == merge
            and item['path'].startswith(REPO + '/' + PREFIX)
            and item['path'].endswith('@' + merge)
            for item in refs
        ),
        'Missing provenance for the replay test merge',
    )


def effective_jobs(run, replay):
    """Keep the last execution of each full matrix-job name across attempts."""
    jobs = {}
    seen_ids = set()
    for attempt in range(1, run['run_attempt'] + 1):
        endpoint = f'actions/runs/{run["id"]}/attempts/{attempt}'
        if attempt != run['run_attempt']:
            previous = api(endpoint)
            require(
                previous['run_attempt'] == attempt
                and previous['status'] == 'completed'
                and all(
                    previous[key] == run[key]
                    for key in (
                        'id',
                        'workflow_id',
                        'path',
                        'event',
                        'head_branch',
                        'head_sha',
                        'check_suite_id',
                    )
                )
                and previous['repository']['full_name']
                == previous['head_repository']['full_name']
                == REPO,
                'Earlier workflow attempt has different source provenance',
            )
            if run['event'] == 'pull_request':
                verify_replay_run(previous, replay)
        records = paged(f'{endpoint}/jobs', 'jobs')
        if attempt == run['run_attempt']:
            require(records, f'No jobs registered for {endpoint}')
        names = set()
        for job in records:
            name = job['name']
            require(
                job['run_id'] == run['id']
                and job['run_attempt'] == attempt
                and job['head_sha'] == run['head_sha']
                and job['head_branch'] == run['head_branch']
                and isinstance(name, str)
                and name
                and name not in names
                and positive(job['id'])
                and job['id'] not in seen_ids,
                f'Unexpected job identity for {endpoint}',
            )
            names.add(name)
            seen_ids.add(job['id'])
            jobs[name] = job
    return list(jobs.values())


def verify_jobs(path, run, jobs):
    filename = path.removeprefix(PREFIX)
    skipped = {'queue-dispatch-guard'}
    if filename == 'build-tools-images.yml':
        skipped.add('Build tools images')
    if path == UPGRADE_PATH:
        skipped.update(OPTIONAL_UPGRADE_JOBS)
    required = TEST_STEPS.get(filename)
    require(required, f'Review missing landed-CI test profile for {path}')
    completed = set()
    seen = set()
    for job in jobs:
        name = job['name']
        require(
            job['run_id'] == run['id']
            and positive(job['run_attempt'])
            and job['run_attempt'] <= run['run_attempt']
            and job['status'] == 'completed'
            and name not in seen,
            f'Unexpected job for {path}',
        )
        seen.add(name)
        if job['conclusion'] == 'skipped' and name in skipped:
            continue
        require(job['conclusion'] == 'success', f'{path}: {name} did not pass')
        if name.startswith('build / ') or (
            path == UPGRADE_PATH and name in OPTIONAL_UPGRADE_JOBS
        ):
            continue
        family = name.split(' - K8s', 1)[0]
        require(family in required, f'Unknown test job {name!r} for {path}')
        steps = {
            s['name'] for s in job['steps'] if s['conclusion'] == 'success'
        }
        require(
            set(required[family]) <= steps,
            f'{path}: missing test steps in {name}',
        )
        completed.add(family)
    missing = set(required) - completed
    require(
        not missing,
        f'{path}: missing executed test job families {sorted(missing)}',
    )


def latest_ci_status(sha):
    """GitHub lists statuses newest first, including superseded contexts."""
    page = 1
    newest_id = None
    while True:
        rows = api(f'commits/{sha}/statuses?per_page=100&page={page}')
        if page == 1 and len(rows) == 100:
            newest_id = rows[0]['id']
        for status in rows:
            if status['context'] == 'ci-passed':
                if page > 1 and status['state'] == 'success':
                    # A newly posted failure may have shifted the page boundaries.
                    first = api(f'commits/{sha}/statuses?per_page=1&page=1')
                    require(
                        first and first[0]['id'] == newest_id,
                        'Status history changed during pagination; retry verification',
                    )
                return status
        if len(rows) < 100:
            return {}
        page += 1


def verify_aggregate(sha, replay):
    """Bind the most recent ci-passed to the official publisher and replay
    base."""
    status = latest_ci_status(sha)
    base = replay['base']
    stamp = hashlib.sha256(
        json.dumps([base['ref'], base['sha']], separators=(',', ':')).encode()
    ).hexdigest()
    owner = re.fullmatch(
        r'https://github[.]com/kubeflow/pipelines/actions/runs/([1-9][0-9]*)',
        status.get('target_url', ''),
    )
    require(
        status.get('state') == 'success'
        and owner
        and status.get('creator', {}).get('login') == 'github-actions[bot]'
        and status.get('description')
        == f'Expected CI and all checks passed; base policy {stamp}.',
        'Missing successful ci-passed for the replay base and landed SHA',
    )
    run = api(f'actions/runs/{owner[1]}')
    workflow = api('actions/workflows/ci-checks.yml')
    require(
        run['id'] == int(owner[1])
        and run['workflow_id'] == workflow['id']
        and run['path'] == workflow['path'] == PREFIX + 'ci-checks.yml'
        and run['repository']['full_name']
        == run['head_repository']['full_name']
        == REPO
        and run['status'] == 'completed'
        and run['conclusion'] == 'success',
        'ci-passed publisher is not a completed upstream CI Check',
    )
    if run['event'] == 'pull_request_target':
        raise RuntimeError(
            'ci-passed came from pull_request_target; rerun a required automatic '
            'test workflow from the replay PR to trigger fresh reconciliation on main. '
            'See release/README.md.'
        )
    require(
        run['event'] in ('schedule', 'workflow_run')
        and run['head_branch'] == 'main',
        'ci-passed publisher did not run on upstream main',
    )
    require(
        re.fullmatch(r'[0-9a-f]{40}', run['head_sha']),
        'Invalid ci-passed publisher commit',
    )
    history = api(f'compare/{run["head_sha"]}...main?per_page=1')
    require(
        history['status'] in ('ahead', 'identical')
        and history['merge_base_commit']['sha'] == run['head_sha'],
        'ci-passed publisher revision is outside upstream main',
    )

    job_name = (
        f'check_ci_status ({replay["number"]}, {sha})'
        if run['event'] == 'schedule'
        else 'check_ci_status (0)'
    )
    jobs = paged(
        f'actions/runs/{run["id"]}/attempts/{run["run_attempt"]}/jobs', 'jobs'
    )
    recorded = datetime.datetime.fromisoformat(status['created_at'])
    require(
        any(
            job['name'] == job_name
            and job['conclusion'] == 'success'
            and any(
                step['name'] == 'Reconcile status and informational label'
                and step['conclusion'] == 'success'
                and datetime.datetime.fromisoformat(step['started_at'])
                <= recorded
                <= datetime.datetime.fromisoformat(step['completed_at'])
                for step in job['steps']
            )
            for job in jobs
        ),
        'ci-passed was not issued during the matching upstream reconciliation job',
    )


def collect_evidence(pr, sha, replay_pr):
    parent, replay = binding(pr, sha, replay_pr)
    paths = expected_workflows(parent, replay)
    runs = paged(f'actions/runs?head_sha={sha}', 'workflow_runs')
    workflows = {
        w['path']: w['id'] for w in paged('actions/workflows', 'workflows')
    }
    evidence = []
    for path in sorted(paths):
        filename = path.removeprefix(PREFIX)
        if filename in BRANCH_RUNS:
            event, blob = BRANCH_RUNS[filename]
            require(
                api(f'contents/{path}?ref={sha}')['sha'] == blob,
                f'Review changed branch-only workflow {path}',
            )
            ref = BRANCH
        else:
            event, ref = 'pull_request', replay['head']['ref']
        run = select_run(runs, path, event, ref, sha)
        require(
            run['workflow_id'] == workflows.get(path)
            and positive(run['workflow_id']),
            f'Unregistered workflow {path}',
        )
        if event == 'pull_request':
            verify_replay_run(run, replay)
        verify_jobs(path, run, effective_jobs(run, replay))
        evidence.append(
            dict(
                path=path,
                workflow_id=run['workflow_id'],
                event=event,
                run_id=run['id'],
                run_attempt=run['run_attempt'],
                tested_sha=replay['merge_commit_sha']
                if event == 'pull_request'
                else sha,
            )
        )
    subprocess.run(
        [
            'node',
            str(SCRIPTS / 'release_replay_checks.js'),
            sha,
            str(replay_pr),
        ],
        check=True,
        timeout=180,
    )
    verify_aggregate(sha, replay)
    final_parent, final_replay = binding(pr, sha, replay_pr)
    require(
        parent == final_parent
        and replay['merge_commit_sha'] == final_replay['merge_commit_sha'],
        'Replay binding changed during verification',
    )
    # Run attempts and branch pointers are mutable. Make the attempt identity check
    # the final read, after the external CI and aggregate provenance gates.
    fresh = paged(f'actions/runs?head_sha={sha}', 'workflow_runs')
    for item in evidence:
        event = item['event']
        ref = replay['head']['ref'] if event == 'pull_request' else BRANCH
        current = select_run(fresh, item['path'], event, ref, sha)
        require(
            (current['id'], current['run_attempt'])
            == (item['run_id'], item['run_attempt']),
            'Release CI attempt changed during verification',
        )
    return BRANCH, evidence


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--pr', type=int, required=True)
    parser.add_argument('--replay-pr', type=int, required=True)
    parser.add_argument('--sha', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    branch, evidence = collect_evidence(args.pr, args.sha, args.replay_pr)
    report = dict(
        schema_version=1,
        repository=REPO,
        pr=args.pr,
        replay_pr=args.replay_pr,
        branch=branch,
        sha=args.sha,
        run_id=int(os.environ['GITHUB_RUN_ID']),
        run_attempt=int(os.environ['GITHUB_RUN_ATTEMPT']),
        workflow_sha=os.environ['GITHUB_SHA'],
        evidence=evidence,
    )
    args.output.write_text(
        json.dumps(report, indent=2) + '\n', encoding='utf-8'
    )


if __name__ == '__main__':
    main()

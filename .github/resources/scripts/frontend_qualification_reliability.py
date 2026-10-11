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
"""Report first-attempt job outcomes from saved GitHub Actions API
responses."""

import argparse
from collections import defaultdict
import datetime
import json
from pathlib import Path
import re
import subprocess

FAILURES = {'failure', 'timed_out', 'startup_failure', 'stale'}


def read_pages(path, key):
    """Accept one API response or gh api --paginate --slurp pages."""
    raw = json.loads(path.read_text())
    pages = raw if isinstance(raw, list) else [raw]
    if not pages or any(
            not isinstance(page, dict) or not isinstance(page.get(key), list)
            for page in pages):
        raise ValueError(f'{path}: expected API pages containing {key}; '
                         'collect the endpoint again with --paginate --slurp')
    return [item for page in pages for item in page[key]]


def api_json(endpoint, fields=None):
    """Read one bounded API page without shell expansion or credential
    output."""
    command = ['gh', 'api', '--method', 'GET', endpoint]
    for key, value in (fields or {}).items():
        command.extend(['-f', f'{key}={value}'])
    result = subprocess.run(
        command, check=True, capture_output=True, text=True, timeout=120)
    return json.loads(result.stdout)


def collect(directory, workflows, repository, since, until):
    """Retain complete, bounded API evidence; fail closed on truncation or
    races."""
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise ValueError('Repository must be an owner/name pair')
    start = datetime.date.fromisoformat(since)
    end = datetime.date.fromisoformat(until)
    if not 0 <= (end - start).days <= 30:
        raise ValueError(
            'Collection window must span at most 31 inclusive days')
    if not workflows or len(set(workflows)) != len(workflows) or any(
            not re.fullmatch(r'[A-Za-z0-9_-]+\.ya?ml', name)
            for name in workflows):
        raise ValueError('Select unique workflow filenames')
    directory.mkdir(parents=True, exist_ok=True)
    # A new directory prevents old attempt files from contaminating a partial refresh.
    if any(directory.iterdir()):
        raise ValueError('Collection requires an empty evidence directory')
    metadata = dict(
        repository=repository,
        since=since,
        until=until,
        workflows=workflows,
        complete=False)
    manifest = directory / 'collection.json'
    manifest.write_text(json.dumps(metadata, indent=2))

    def pages(endpoint, key, fields=None, limit=200):
        retained = []
        identities = set()
        total = None
        for page in range(1, (limit // 100) + 2):
            response = api_json(endpoint,
                                dict(fields or {}, per_page=100, page=page))
            count = response.get('total_count')
            if type(count) is not int or count < 0 or count > limit:
                raise ValueError(
                    f'{endpoint}: collection limit exceeded or invalid count; split the window'
                )
            if total is not None and count != total:
                raise ValueError(
                    f'{endpoint}: result count changed during collection')
            total = count
            items = response.get(key)
            if not isinstance(items, list):
                raise ValueError(f'{endpoint}: missing {key}')
            for item in items:
                identity = item.get('id')
                if type(identity) is not int or identity in identities:
                    raise ValueError(
                        f'{endpoint}: missing or duplicate result identity')
                identities.add(identity)
            retained.append(response)
            if len(identities) == total:
                return retained
            if not items or len(identities) > total:
                raise ValueError(f'{endpoint}: incomplete paginated evidence')
        raise ValueError(f'{endpoint}: pagination did not complete')

    runs = []
    seen = set()
    for name in workflows:
        responses = pages(f'repos/{repository}/actions/workflows/{name}/runs',
                          'workflow_runs', {'created': f'{since}..{until}'})
        for response in responses:
            for run in response['workflow_runs']:
                if run.get('path'
                          ) != f'.github/workflows/{name}' or run['id'] in seen:
                    raise ValueError(
                        'Workflow run identity does not match selection')
                if run.get('status') != 'completed':
                    raise ValueError(
                        'Selected runs are still active; use a completed window'
                    )
                attempts = run.get('run_attempt')
                if type(attempts) is not int or not 1 <= attempts <= 20:
                    raise ValueError(
                        'Run attempts are invalid or exceed the 20-attempt limit'
                    )
                seen.add(run['id'])
                runs.append(run)
    (directory / 'runs.json').write_text(json.dumps({'workflow_runs': runs}))
    for run in runs:
        endpoint = f"repos/{repository}/actions/runs/{run['id']}"
        for attempt in range(1, run['run_attempt'] + 1):
            responses = pages(
                f'{endpoint}/attempts/{attempt}/jobs', 'jobs', limit=1000)
            (directory / f"{run['id']}-attempt-{attempt}.json").write_text(
                json.dumps(responses))
        current = api_json(endpoint)
        if any(
                current.get(key) != run.get(key)
                for key in ('id', 'head_sha', 'run_attempt', 'status',
                            'conclusion')):
            raise ValueError(
                'Run changed during collection; repeat into a new directory')
    # Certify only after validating source and attempt identities. Interrupted
    # collection therefore leaves an explicitly incomplete manifest.
    result = _report(directory, workflows)
    metadata['complete'] = True
    manifest.write_text(json.dumps(metadata, indent=2))
    return result


def report(directory, workflows):
    """Keep failures independent of the enclosing run's final conclusion."""
    collection = directory / 'collection.json'
    if collection.exists() and json.loads(
            collection.read_text()).get('complete') is not True:
        raise ValueError(
            'Collection is incomplete; repeat into a new directory')
    return _report(directory, workflows)


def _report(directory, workflows):
    rows = []
    unknown = []
    seen = set()
    for run in read_pages(directory / 'runs.json', 'workflow_runs'):
        workflow = Path(run['path'].split('@', 1)[0]).name
        if workflow not in workflows:
            continue
        run_id = run['id']
        if run_id in seen:
            raise ValueError(f'Duplicate run {run_id}; collect each run once')
        seen.add(run_id)
        attempts = {}
        for attempt in range(1, run['run_attempt'] + 1):
            evidence_file = directory / f'{run_id}-attempt-{attempt}.json'
            evidence = {
                'workflow': workflow,
                'run_id': run_id,
                'source_sha': run['head_sha'],
                'attempt': attempt,
                'run_url': run['html_url'],
            }
            if not evidence_file.exists():
                unknown.append({**evidence, 'reason': 'Missing attempt jobs'})
                continue
            jobs = read_pages(evidence_file, 'jobs')
            by_name = {}
            for job in jobs:
                if (job.get('run_id') != run_id or
                        job.get('run_attempt') != attempt or
                        job.get('head_sha') != run['head_sha']):
                    raise ValueError(f'{evidence_file}: job identity mismatch; '
                                     'collect the matching run attempt again')
                name = job['name']
                if name in by_name:
                    raise ValueError(f'{evidence_file}: duplicate job name '
                                     f'{name}; use unique matrix job names')
                by_name[name] = job
            if not jobs:
                unknown.append({**evidence, 'reason': 'No jobs in attempt'})
            attempts[attempt] = by_name
        first = attempts.get(1, {})
        for name in sorted(
                set().union(*(set(jobs) for jobs in attempts.values()))):
            job = first.get(name)
            conclusion = job.get('conclusion') if job else None
            outcome = ('failure' if conclusion in FAILURES else
                       'success' if conclusion == 'success' else 'unknown')
            successes = [{
                'attempt': attempt,
                'job_url': jobs[name]['html_url']
            }
                         for attempt, jobs in sorted(attempts.items())
                         if attempt > 1 and name in jobs and
                         jobs[name].get('conclusion') == 'success']
            rows.append({
                'workflow': workflow,
                'lane': name,
                'run_id': run_id,
                'source_sha': run['head_sha'],
                'run_url': run['html_url'],
                'first_attempt': 1,
                'first_job_url': job['html_url'] if job else None,
                'first_conclusion': conclusion,
                'first_outcome': outcome,
                'successful_reruns': successes,
                'same_sha_recovery': outcome == 'failure' and bool(successes),
                'cause': 'unclassified',
            })
    summaries = defaultdict(lambda: {
        'success': 0,
        'failure': 0,
        'unknown': 0,
        'same_sha_recoveries': 0
    })
    for row in rows:
        summary = summaries[(row['workflow'], row['lane'])]
        summary[row['first_outcome']] += 1
        summary['same_sha_recoveries'] += int(row['same_sha_recovery'])
    return {
        'lanes': [
            dict(workflow=workflow, lane=lane, **counts)
            for (workflow, lane), counts in sorted(summaries.items())
        ],
        'evidence':
            rows,
        'incomplete_attempts':
            unknown,
        'missing_workflows':
            sorted(set(workflows) - {row['workflow'] for row in rows}),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    parser.add_argument(
        '--workflow',
        action='append',
        required=True,
        help='Workflow filename; repeat to select multiple')
    parser.add_argument('--collect', action='store_true')
    parser.add_argument('--repository')
    parser.add_argument('--since')
    parser.add_argument('--until')
    args = parser.parse_args()
    try:
        if args.collect:
            if not all((args.repository, args.since, args.until)):
                parser.error(
                    '--collect requires --repository, --since and --until')
            result = collect(args.directory, args.workflow, args.repository,
                             args.since, args.until)
        else:
            result = report(args.directory, args.workflow)
    except (OSError, ValueError, KeyError, TypeError,
            subprocess.SubprocessError) as error:
        parser.error(str(error))
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()

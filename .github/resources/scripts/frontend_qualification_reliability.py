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
"""Report first-attempt job outcomes from saved GitHub Actions API responses."""

import argparse
from collections import defaultdict
import json
from pathlib import Path

FAILURES = {'failure', 'timed_out', 'startup_failure', 'action_required'}


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


def report(directory, workflows):
    """Keep failures independent of the enclosing run's final conclusion."""
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
    args = parser.parse_args()
    try:
        result = report(args.directory, args.workflow)
    except (OSError, ValueError, KeyError, TypeError) as error:
        parser.error(str(error))
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()

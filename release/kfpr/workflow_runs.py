"""Match dispatched upstream workflows and preserve their exact attempt."""

import re
import time

from .core import REPO


def validate_run(data, identity, completed=True):
    expected = {
        'id': int(identity['run_id']),
        'path': identity['workflow'],
        'event': 'workflow_dispatch',
        'head_sha': identity['workflow_sha'],
        'display_title': identity['title'],
        'run_attempt': identity['run_attempt'],
    }
    if 'workflow_ref' in identity:
        expected['head_branch'] = identity['workflow_ref']
    if (data.get('repository') or {}).get('full_name') != REPO or any(
        data.get(key) != value for key, value in expected.items()
    ):
        raise ValueError(
            'Workflow provenance changed from the expected attempt'
        )
    if completed and (
        data.get('status') != 'completed' or data.get('conclusion') != 'success'
    ):
        raise ValueError(
            'Workflow is not completed successfully on the expected attempt'
        )


def dispatch_and_watch(
    api, run_command, identity, inputs, *, attempts=24, interval=5
):
    """Find one request, watch it, and recheck provenance before artifacts."""
    if identity.get('workflow_ref') not in ('master', 'main'):
        raise ValueError('Invalid upstream workflow branch')
    workflow = identity['workflow'].rsplit('/', 1)[-1]
    command = [
        'gh',
        'workflow',
        'run',
        workflow,
        '--repo',
        REPO,
        '--ref',
        identity['workflow_ref'],
    ]
    for name, value in inputs.items():
        command.extend(['-f', f'{name}={value}'])
    run_command(command, check=True)
    for _ in range(attempts):
        runs = api(
            f'actions/workflows/{workflow}/runs?event=workflow_dispatch&per_page=100'
        )['workflow_runs']
        matches = [
            run for run in runs if run.get('display_title') == identity['title']
        ]
        if len(matches) > 1:
            raise ValueError(
                'Ambiguous workflow dispatch: multiple matching runs'
            )
        if matches:
            run = matches[0]
            if (
                not re.fullmatch(r'[1-9][0-9]*', str(run.get('id', '')))
                or type(run.get('run_attempt')) is not int
                or run['run_attempt'] < 1
            ):
                raise ValueError('Invalid workflow run identity')
            identity.update(
                run_id=str(run['id']), run_attempt=run['run_attempt']
            )
            validate_run(run, identity, completed=False)
            run_command(
                [
                    'gh',
                    'run',
                    'watch',
                    identity['run_id'],
                    '--repo',
                    REPO,
                    '--exit-status',
                ],
                check=False,
            )
            final = api(f'actions/runs/{identity["run_id"]}')
            validate_run(final, identity, completed=False)
            return final
        time.sleep(interval)
    raise RuntimeError(
        'Workflow dispatch was not found; inspect upstream Actions before retrying'
    )

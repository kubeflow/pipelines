import json
import os
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from kfp import Client

TOOLS_DIR = Path(__file__).resolve().parent
CASES = (
    ('success-first', 'success_pipeline.yaml', {'seed': 'fixture'}),
    ('success-cache', 'success_pipeline.yaml', {'seed': 'fixture'}),
    ('success-long', 'success_pipeline.yaml', {'seed': 'L' * 512}),
    ('success-optional', 'success_pipeline.yaml', {'seed': 'run-optional'}),
    ('structure', 'structure_pipeline.yaml', {}),
    ('intentional-failure', 'failure_pipeline.yaml', {}),
    ('success-sparse', 'success_pipeline.yaml', {'seed': 'sparse-id'}),
)


def save_runs(path: Path, records: list[dict[str, str]]) -> None:
    temporary = path.with_suffix('.json.tmp')
    temporary.write_text(json.dumps(records, indent=2) + '\n')
    temporary.replace(path)


def capture_runs(
    client: 'Client', output_dir: Path, selected: set[str]
) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    run_record = output_dir / 'runs.json'
    records = json.loads(run_record.read_text()) if run_record.exists() else []
    by_case = {entry['case']: entry for entry in records}
    for name, package, arguments in CASES:
        if selected and name not in selected:
            continue
        expected_state = (
            'FAILED' if name == 'intentional-failure' else 'SUCCEEDED'
        )
        record = by_case.get(name)
        if record and record['status'] == expected_state:
            continue

        if (
            name == 'success-cache'
            and by_case.get('success-first', {}).get('status') != 'SUCCEEDED'
        ):
            raise RuntimeError(
                'success-first must succeed before success-cache'
            )

        if record is None:
            run = client.create_run_from_pipeline_package(
                str(TOOLS_DIR / package), arguments=arguments, run_name=name
            )
            record = {'case': name, 'run_id': run.run_id, 'status': 'SUBMITTED'}
            records.append(record)
            by_case[name] = record
            save_runs(run_record, records)

        # A wait failure leaves the run ID on disk for the next invocation.
        run_id = record['run_id']
        final = client.wait_for_run_completion(run_id, timeout=900)
        record['status'] = final.state
        save_runs(run_record, records)
        print(name, run_id, final.state, flush=True)
        if final.state != expected_state:
            raise RuntimeError(
                f'{name} ({run_id}) reached {final.state}, expected {expected_state}'
            )


def main() -> None:
    from kfp import Client

    output_dir = Path(
        os.environ.get('KFP218_OUTPUT')
        or TOOLS_DIR.parent / 'testdata' / 'release_218'
    )
    client = Client(host=os.environ.get('KFP_HOST', 'http://127.0.0.1:8080'))
    selected = set(filter(None, os.environ.get('FIXTURE_CASES', '').split(',')))
    capture_runs(client, output_dir, selected)


if __name__ == '__main__':
    main()

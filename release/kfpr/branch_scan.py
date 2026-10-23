"""Build and report the current release branch without publishing images."""

import argparse
import copy
import json
import os
import re
import subprocess
import tempfile
import uuid
from collections import Counter
from pathlib import Path

from . import ci_images_runner as runner

REPOSITORY = 'kubeflow/pipelines'
BRANCH = 'release-2.18'


def validate_trigger(event_name: str, event: dict) -> None:
    if event_name in ('schedule', 'workflow_dispatch'):
        return
    if event_name != 'workflow_run':
        raise ValueError(
            'Expected pre-commit completion, schedule or manual dispatch'
        )
    run = event.get('workflow_run', {})
    expected = {
        'path': '.github/workflows/pre-commit.yml',
        'event': 'push',
        'head_branch': BRANCH,
        'status': 'completed',
    }
    if any(run.get(key) != value for key, value in expected.items()) or any(
        run.get(key, {}).get('full_name') != REPOSITORY
        for key in ('repository', 'head_repository')
    ):
        raise ValueError(
            'Expected same-repository release-2.18 pre-commit push completion'
        )
    if not re.fullmatch(r'[0-9a-f]{40}', run.get('head_sha', '')):
        raise ValueError('Trigger must identify its full source SHA')


def resolve_source(event_name: str, event: dict) -> str:
    validate_trigger(event_name, event)
    result = subprocess.run(
        ['gh', 'api', f'repos/{REPOSITORY}/git/ref/heads/{BRANCH}'],
        check=True,
        capture_output=True,
        text=True,
    )
    ref = json.loads(result.stdout)
    sha = ref.get('object', {}).get('sha', '')
    if (
        ref.get('ref') != f'refs/heads/{BRANCH}'
        or ref.get('object', {}).get('type') != 'commit'
        or not re.fullmatch(r'[0-9a-f]{40}', sha)
    ):
        raise ValueError('Release branch lookup did not return an exact commit')
    return sha


def stable_matrix(source: Path, release: dict) -> dict:
    return {
        'include': [
            {key: lane[key] for key in ('name', 'platform')}
            for lane in runner.matrix(source, release)['include']
        ]
    }


def required_lanes(matrix: dict, request_id: str, attempt: int) -> list[str]:
    return [
        f'kfpr-lane-{request_id}-{attempt}-{lane["name"]}-{lane["platform"].split("/")[1]}/{filename}'
        for lane in matrix['include']
        for filename in ('lane.json', 'scan.json')
    ]


def convert_scans(source: Path, directory: Path, release: dict) -> None:
    """Convert only complete verified evidence; preserve immutable raw
    scans."""
    manifest = json.loads((directory / 'manifest.json').read_text())
    if any(manifest.get(key) != value for key, value in release.items()):
        raise ValueError('Scan manifest does not match this workflow attempt')
    expected = {
        (item['name'], item['platform'])
        for item in runner.inventory(source, release)
    }
    records = manifest.get('images', [])
    pairs = [(item.get('name'), item.get('platform')) for item in records]
    if len(pairs) != len(expected) or set(pairs) != expected:
        raise ValueError(
            'Scan manifest must contain the complete source image inventory'
        )
    # Validate the entire set before producing any SARIF, including clean scans.
    for record in records:
        filename = (
            f'scans/{record["name"]}-{record["platform"].split("/")[1]}.json'
        )
        path = directory / filename
        if record.get('scan_file') != filename or path.is_symlink():
            raise ValueError('Unexpected raw scan path')
        if (
            runner._verify_scan(path, record['image_id'])
            != record['scan_sha256']
        ):
            raise ValueError(
                'Raw scan checksum mismatch before SARIF conversion'
            )
    output = directory / 'sarif'
    output.mkdir()
    for record in records:
        raw = directory / record['scan_file']
        report = json.loads(raw.read_text())
        findings = Counter(
            vulnerability['VulnerabilityID']
            for result in report.get('Results', [])
            for vulnerability in (result.get('Vulnerabilities') or [])
        )
        presentation = copy.deepcopy(report)
        for result in presentation.get('Results', []):
            if result.get('Class') == 'os-pkgs':
                # Trivy uses this target as the SARIF location. Image digests
                # and distro versions would otherwise create alert churn.
                result['Target'] = record['name']
        destination = (
            output
            / f'{record["name"]}-{record["platform"].split("/")[1]}.sarif'
        )
        convert_report(presentation, destination)
        sarif = json.loads(destination.read_text())
        runs = sarif.get('runs', [])
        if (
            sarif.get('version') != '2.1.0'
            or len(runs) != 1
            or Counter(
                result['ruleId'] for result in runs[0].get('results', [])
            )
            != findings
        ):
            raise ValueError(
                'SARIF conversion did not preserve every vulnerability finding'
            )


def convert_report(report: dict, destination: Path) -> None:
    """Convert a presentation copy with repository/env ignore policy
    disabled."""
    environment = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith('TRIVY_')
    }
    with tempfile.TemporaryDirectory() as temporary:
        config = Path(temporary) / 'config.json'
        config.write_text('{}\n')
        ignore = Path(temporary) / 'ignore.txt'
        ignore.write_text('')
        converted = Path(temporary) / 'report.json'
        converted.write_text(json.dumps(report))
        subprocess.run(
            [
                'trivy',
                'convert',
                '--config',
                str(config),
                '--ignorefile',
                str(ignore),
                '--severity',
                'UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL',
                '--format',
                'sarif',
                '--output',
                str(destination),
                str(converted),
            ],
            check=True,
            env=environment,
        )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'command', choices=('resolve', 'inventory', 'required-lanes', 'convert')
    )
    parser.add_argument('--source', type=Path)
    parser.add_argument('--evidence', type=Path)
    args = parser.parse_args()
    tool_sha = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'],
        cwd=Path(__file__).resolve().parents[2],
        text=True,
    ).strip()
    if tool_sha != os.environ['WORKFLOW_SHA']:
        raise ValueError('Tool checkout does not match the workflow SHA')
    if args.command == 'resolve':
        event = json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text())
        sha = resolve_source(os.environ['GITHUB_EVENT_NAME'], event)
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            print(f'source-sha={sha}', file=output)
            print(f'request-id={uuid.uuid4().hex}', file=output)
        return
    if not args.source:
        parser.error('command requires --source')
    if args.command == 'inventory':
        # VERSION comes from the resolved source, not the branch's spelling.
        os.environ['VERSION'] = (args.source / 'VERSION').read_text().strip()
    release = runner.identity()
    matrix = stable_matrix(args.source, release)
    if args.command == 'inventory':
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            print(f'version={release["version"]}', file=output)
            print(
                'matrix=' + json.dumps(matrix, separators=(',', ':')),
                file=output,
            )
    elif args.command == 'required-lanes':
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            print('required-files<<EOF', file=output)
            print(
                '\n'.join(
                    required_lanes(
                        matrix, release['request_id'], release['run_attempt']
                    )
                ),
                file=output,
            )
            print('EOF', file=output)
    else:
        if not args.evidence:
            parser.error('convert requires --evidence')
        convert_scans(args.source, args.evidence, release)


if __name__ == '__main__':
    main()

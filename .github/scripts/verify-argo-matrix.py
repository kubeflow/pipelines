#!/usr/bin/env python3
import re
import sys
from pathlib import Path
from typing import List, Set

from ruamel.yaml import YAML


def parse_minor(version: str) -> str:
    match = re.match(r'(v\d+\.\d+)', version)
    if not match:
        raise ValueError(f'Cannot parse minor from version: {version}')
    return match.group(1)


def _append_argo_version(versions: List[str], value: object) -> None:
    if isinstance(value, str) and value:
        versions.append(value)
    elif isinstance(value, list):
        versions.extend(
            version for version in value if isinstance(version, str) and version
        )


def _collect_argo_versions(value: object, versions: List[str]) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            if key == 'argo_version':
                _append_argo_version(versions, child)
                continue
            _collect_argo_versions(child, versions)
    elif isinstance(value, list):
        for child in value:
            _collect_argo_versions(child, versions)


def _extract_argo_versions_from_workflow(workflow_yaml_text: str) -> List[str]:
    yaml = YAML(typ='safe')
    data = yaml.load(workflow_yaml_text) or {}

    versions: List[str] = []
    jobs = data.get('jobs') or {}
    if not isinstance(jobs, dict):
        return versions

    for job in jobs.values():
        if not isinstance(job, dict):
            continue
        strategy = job.get('strategy') or {}
        if not isinstance(strategy, dict):
            continue
        matrix = strategy.get('matrix') or {}
        _collect_argo_versions(matrix, versions)

    return versions


def main() -> int:
    repo_root = Path('.')
    workflow_paths = [
        repo_root / '.github' / 'workflows' / 'e2e-test.yml',
        repo_root / '.github' / 'workflows' / 'api-server-tests.yml',
    ]
    version_path = repo_root / 'third_party' / 'argo' / 'VERSION'
    readme_path = repo_root / 'README.md'

    # Extract argo versions from compatibility workflows (drop patch later).
    argo_versions: List[str] = []
    for workflow_path in workflow_paths:
        try:
            workflow_content = workflow_path.read_text(encoding='utf-8')
        except Exception as exc:
            print(f'ERROR: Failed to read {workflow_path}: {exc}', file=sys.stderr)
            return 1

        workflow_versions = _extract_argo_versions_from_workflow(workflow_content)
        if not workflow_versions:
            print(f'ERROR: No argo_version found in {workflow_path}', file=sys.stderr)
            return 1

        argo_versions.extend(workflow_versions)

    # Read VERSION and derive minor
    try:
        version_content = version_path.read_text(encoding='utf-8').strip()
    except Exception as exc:
        print(f'ERROR: Failed to read {version_path}: {exc}', file=sys.stderr)
        return 1

    m = re.search(r'v\d+\.\d+', version_content)
    if not m:
        print('ERROR: Could not parse third_party/argo/VERSION', file=sys.stderr)
        return 1
    argo_versions.append(m.group(0))

    minor_set: Set[str] = {parse_minor(v) for v in argo_versions}
    argo_minors = sorted(
        minor_set,
        key=lambda s: tuple(int(x) for x in s[1:].split('.')),
    )

    expected = ", ".join(argo_minors)

    # Read README and extract Argo Workflows row
    try:
        readme = readme_path.read_text(encoding='utf-8')
    except Exception as exc:
        print(f'ERROR: Failed to read {readme_path}: {exc}', file=sys.stderr)
        return 1

    row_match = re.search(r'^\|\s*Argo Workflows\s*\|\s*([^|]+)\|', readme, re.MULTILINE)
    if not row_match:
        print('ERROR: Could not find "Argo Workflows" row in README.md', file=sys.stderr)
        return 1
    cell = row_match.group(1).strip()

    if cell != expected:
        print('ERROR: README.md "Dependencies Compatibility Matrix" for Argo Workflows is out of date.', file=sys.stderr)
        print(f'  Found:    "{cell}"', file=sys.stderr)
        print(f'  Expected: "{expected}"', file=sys.stderr)
        return 1

    print('Argo Workflows compatibility matrix in README.md is up to date.')
    return 0


if __name__ == '__main__':
    sys.exit(main())

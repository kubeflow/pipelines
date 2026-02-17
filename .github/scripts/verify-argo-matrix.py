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


def _extract_argo_versions_from_e2e(e2e_yaml_text: str) -> List[str]:
    yaml = YAML(typ='safe')
    data = yaml.load(e2e_yaml_text) or {}
    jobs = data.get('jobs') or {}
    return jobs.get('end-to-end-scenario-tests', {}).get('strategy', {}).get('matrix', {}).get('argo_version',[])


def main() -> int:
    repo_root = Path('.')
    e2e_path = repo_root / '.github' / 'workflows' / 'e2e-test.yml'
    version_path = repo_root / 'third_party' / 'argo' / 'VERSION'
    readme_path = repo_root / 'README.md'

    # Extract argo versions from e2e workflow (drop patch)
    try:
        e2e_content = e2e_path.read_text(encoding='utf-8')
    except Exception as exc:
        print(f'ERROR: Failed to read {e2e_path}: {exc}', file=sys.stderr)
        return 1

    argo_versions = _extract_argo_versions_from_e2e(e2e_content)

    if not argo_versions or not isinstance(argo_versions, list):
        print('ERROR: No argo_version found in .github/workflows/e2e-test.yml', file=sys.stderr)
        return 1

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



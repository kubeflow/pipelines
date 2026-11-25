"""Check active Argo Workflows against one KFP installation's run inventory."""

import argparse
import json
import sys
from pathlib import Path

TERMINAL_PHASES = {'Succeeded', 'Failed', 'Error'}


def active_workflows(
    runs: list[dict], workflows: dict
) -> list[tuple[str, str]]:
    """Return the namespace and name of each active Workflow in the
    inventory."""
    namespaces = {run['run_id']: run['namespace'] for run in runs}
    if len(namespaces) != len(runs):
        raise ValueError('run inventory contains duplicate run IDs')

    active = []
    for workflow in workflows['items']:
        metadata = workflow['metadata']
        run_id = (metadata.get('labels') or {}).get('pipeline/runid')
        if run_id not in namespaces:
            continue
        if not namespaces[run_id]:
            raise ValueError(f'run {run_id} has no namespace in the inventory')
        if metadata['namespace'] != namespaces[run_id]:
            continue
        if (workflow.get('status') or {}).get('phase') not in TERMINAL_PHASES:
            active.append((metadata['namespace'], metadata['name']))
    return sorted(active)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run_inventory', type=Path)
    args = parser.parse_args()
    try:
        runs = json.loads(args.run_inventory.read_text(encoding='utf-8'))
        active = active_workflows(runs, json.load(sys.stdin))
    except (OSError, ValueError, KeyError, TypeError) as error:
        parser.error(str(error))
    for namespace, name in active:
        print(f'{namespace}/{name}')
    return int(bool(active))


if __name__ == '__main__':
    sys.exit(main())

#!/usr/bin/env python3
"""Join sanitized actual Kubernetes audit caller records to APF HTTP completion."""
import argparse
import json
from pathlib import Path


def join(audit, http, schemas, priorities):
    result, unmatched, ambiguous = [], 0, 0
    prefixes = {}
    for h in http:
        if h.get('audit_prefix'):
            prefixes.setdefault(h['audit_prefix'], []).append(h)
    lengths = {len(prefix) for prefix in prefixes}
    owners = {}
    for a in audit:
        for length in lengths:
            owners.setdefault(a['audit_id'][:length], set()).add(a['audit_id'])
    for row in audit:
        matches = [h for length in lengths for h in prefixes.get(row['audit_id'][:length], [])]
        # HTTP IDs are truncated: never silently join a collision or duplicate.
        if len(matches) != 1:
            unmatched += not matches
            ambiguous += bool(matches)
            continue
        h = matches[0]
        if len(owners[h['audit_prefix']]) != 1:
            ambiguous += 1
            continue
        fs, pl = schemas.get(h['flow_name']), priorities.get(h['priority_name'])
        if not fs or not pl or fs['spec']['priorityLevelConfiguration']['name'] != h['priority_name']:
            unmatched += 1
            continue
        result.append({**row, 'flow_uid': fs['metadata']['uid'], 'priority_uid': pl['metadata']['uid'],
                       'priority_type': pl['spec']['type']})
    return {'records': result, 'unmatched': unmatched, 'ambiguous': ambiguous,
            'complete_for_captured_records': bool(audit) and not unmatched and not ambiguous,
            'claim': 'actual Kubernetes caller classification; not KFP tenant or RPC correlation'}


def read_lines(path):
    p = Path(path)
    if p.stat().st_size > 64 * 1024 * 1024:
        raise RuntimeError('projection exceeds bound')
    return [json.loads(x) for x in p.read_text().splitlines()]


def main():
    p = argparse.ArgumentParser()
    for name in ('audit', 'http', 'schemas', 'priorities', 'output'):
        p.add_argument('--' + name, required=True)
    args = p.parse_args()
    schemas = {r['metadata']['name']: r for r in json.loads(Path(args.schemas).read_text())['items']}
    priorities = {r['metadata']['name']: r for r in json.loads(Path(args.priorities).read_text())['items']}
    Path(args.output).write_text(json.dumps(join(read_lines(args.audit), read_lines(args.http), schemas, priorities), indent=2) + '\n')

if __name__ == '__main__':
    main()

#!/usr/bin/env python3
"""Bounded sanitized projection; stream stdin, never persist raw audit/HTTP logs."""
import argparse
import json
import sys

LIMIT = 64 * 1024 * 1024
MAX_LINE = 1024 * 1024


def project(record, kind, users):
    if not isinstance(record, dict):
        return None
    if kind == 'audit':
        if record.get('user', {}).get('username') not in users or record.get('stage') != 'ResponseComplete':
            return None
        ref = record.get('objectRef', {})
        return {'audit_id': record.get('auditID'), 'user': record['user']['username'],
                'verb': record.get('verb'), 'group': ref.get('apiGroup', ''),
                'resource': ref.get('resource'), 'namespace': ref.get('namespace'),
                'code': record.get('responseStatus', {}).get('code'),
                'received': record.get('requestReceivedTimestamp'), 'completed': record.get('stageTimestamp')}
    if record.get('msg') != 'HTTP' or not record.get('apf_fs') or not record.get('apf_pl'):
        return None
    return {'audit_prefix': record.get('audit-ID'), 'flow_name': record['apf_fs'],
            'priority_name': record['apf_pl'], 'code': record.get('resp'), 'ts': record.get('ts')}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--kind', choices=['audit', 'http'], required=True)
    parser.add_argument('--users', required=True, help='JSON array of exact KFP service-account names')
    args = parser.parse_args()
    users = set(json.loads(args.users))
    count = 0
    while True:
        line = sys.stdin.buffer.readline(MAX_LINE + 1)
        if not line:
            break
        count += len(line)
        if len(line) > MAX_LINE or count > LIMIT:
            raise SystemExit('capture limit exceeded; evidence incomplete')
        try:
            row = project(json.loads(line), args.kind, users)
        except (ValueError, TypeError):
            continue
        if row is not None:
            print(json.dumps(row), flush=True)

if __name__ == '__main__':
    main()

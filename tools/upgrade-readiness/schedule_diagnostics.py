# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Bounded, sanitized scheduler evidence for isolated live CI fixtures."""
import argparse
from datetime import datetime
from datetime import timezone
import json
import re
import time

from kubectl_inventory import kubectl_get
from provision_live_schedules import read_object


def identifier(value):
    if not isinstance(value, str) or not re.fullmatch(
            r'[A-Za-z0-9][A-Za-z0-9_.-]{0,252}', value):
        raise ValueError('invalid_identity')
    return value


def optional_identifier(value):
    return None if value is None else identifier(value)


def stamp(value):
    if value is None:
        return None
    result = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if result.tzinfo is None:
        raise ValueError('invalid_timestamp')
    return result.isoformat()


def items(context, namespace, resource):
    result, error = kubectl_get(context, namespace, resource)
    if error or not isinstance(result, dict) or not isinstance(
            result.get('items'), list):
        raise ValueError('collection_failed')
    if len(result['items']) > 10000 or not all(
            isinstance(item, dict) for item in result['items']):
        raise ValueError('collection_failed')
    return result['items']


def schedule_evidence(cases, schedules, namespace):
    rows = []
    for case in cases:
        name, uid = identifier(case['schedule_name']), identifier(
            case['schedule_uid'])
        matches = [
            s for s in schedules if s.get('metadata', {}).get('name') == name
        ]
        if len(matches) != 1:
            raise ValueError('schedule_identity_mismatch')
        swf = matches[0]
        meta, spec = swf['metadata'], swf.get('spec', {})
        if meta.get('namespace') != namespace or meta.get(
                'uid') != uid or swf.get('kind') != 'ScheduledWorkflow':
            raise ValueError('schedule_identity_mismatch')
        enabled = spec.get('enabled', False)
        concurrency = spec.get('maxConcurrency', 1)
        trigger = swf.get('status', {}).get('trigger', {})
        index = trigger.get('lastWorkflowIndex', 0)
        if type(enabled) is not bool or type(concurrency) is not int or type(
                index) is not int:
            raise ValueError('invalid_schedule_status')
        rows.append(
            dict(
                schedule_name=name,
                schedule_uid=uid,
                enabled=enabled,
                max_concurrency=concurrency,
                last_index=index,
                last_triggered=stamp(trigger.get('lastTriggeredTime')),
                next_triggered=stamp(trigger.get('nextTriggeredTime')),
                workflows=[],
                events=[]))
    return rows


def snapshot(context, namespace, cases):
    rows = schedule_evidence(cases,
                             items(context, namespace, 'scheduledworkflows'),
                             namespace)
    by_uid = {r['schedule_uid']: r for r in rows}
    for wf in items(context, namespace, 'workflows'):
        meta = wf.get('metadata', {})
        owners = meta.get('ownerReferences', [])
        selected = [o for o in owners if o.get('uid') in by_uid]
        if not selected:
            continue
        if len(selected) != 1 or len(owners) != 1:
            raise ValueError('workflow_identity_mismatch')
        owner = selected[0]
        row = by_uid[owner['uid']]
        if (meta.get('namespace') != namespace or
                owner.get('kind') != 'ScheduledWorkflow' or
                owner.get('name') != row['schedule_name'] or
                owner.get('controller') is not True):
            raise ValueError('workflow_identity_mismatch')
        labels = meta.get('labels', {})
        phase = wf.get('status', {}).get('phase', '')
        completed = labels.get('workflows.argoproj.io/completed')
        if phase not in ('', 'Pending', 'Running', 'Succeeded', 'Failed',
                         'Error') or completed not in (None, 'true', 'false'):
            raise ValueError('invalid_workflow_status')
        row['workflows'].append(
            dict(
                uid=identifier(meta.get('uid')),
                name=identifier(meta.get('name')),
                phase=phase or 'UNREPORTED',
                completed=completed,
                run_id=optional_identifier(labels.get('pipeline/runid')),
                owner_uid=owner['uid'],
                owner_name=owner['name'],
                schedule_label=optional_identifier(
                    labels.get(
                        'scheduledworkflows.kubeflow.org/scheduledWorkflowName')
                ),
                workflow_index=optional_identifier(
                    labels.get(
                        'scheduledworkflows.kubeflow.org/workflowIndex'))))
    for event in items(context, namespace, 'events'):
        involved = event.get('involvedObject', {})
        row = by_uid.get(involved.get('uid'))
        if row is None:
            continue
        if involved.get('namespace') != namespace or involved.get(
                'name') != row['schedule_name']:
            raise ValueError('event_identity_mismatch')
        count = event.get('count', 1)
        if type(count) is not int or count < 1:
            raise ValueError('invalid_event_count')
        reason = event.get('reason')
        # Classify raw messages in memory; never serialize their contents.
        message = str(event.get('message', '')).lower()
        classification = 'other'
        for code, patterns in (('authorization_denied',
                                ('permissiondenied', 'permission denied',
                                 'forbidden')),
                               ('api_concurrency',
                                ('reached maximum concurrency',
                                 'wait for an active execution')),
                               ('pending_tick',
                                ('previous tick is still pending',
                                 'retry the previous scheduled tick')),
                               ('disabled', ('recurring run is disabled',)),
                               ('not_due', ('no authorized tick is due',)),
                               ('status_conflict',
                                ('object has been modified',
                                 'operation cannot be fulfilled')),
                               ('sync_complete',
                                ('all done',)), ('sync_again',
                                                 ('partially done',))):
            if any(pattern in message for pattern in patterns):
                classification = code
                break
        row['events'].append(
            dict(
                reason=reason if reason in ('Synced', 'Failed') else 'OTHER',
                count=count,
                classification=classification,
                first_seen=stamp(event.get('firstTimestamp')),
                last_seen=stamp(event.get('lastTimestamp'))))
    return dict(
        outcome='collected',
        collected_at=datetime.now(timezone.utc).isoformat(),
        cases=rows)


def wait_enabled(context, namespace, cases):
    deadline = time.monotonic() + 60
    while True:
        rows = schedule_evidence(
            cases, items(context, namespace, 'scheduledworkflows'), namespace)
        if all(row['enabled'] for row in rows):
            return
        if time.monotonic() >= deadline:
            raise ValueError('enable_not_observed')
        time.sleep(2)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--context', required=True)
    parser.add_argument('--namespace', required=True)
    parser.add_argument('--cases', required=True)
    parser.add_argument('--wait-enabled', action='store_true')
    args = parser.parse_args()
    try:
        bundle = read_object(args.cases)
        if bundle.get('namespace') != args.namespace or not 1 <= len(
                bundle['cases']) <= 3:
            raise ValueError('invalid_cases')
        if args.wait_enabled:
            wait_enabled(args.context, args.namespace, bundle['cases'])
        print(
            json.dumps(snapshot(args.context, args.namespace, bundle['cases'])))
        return 0
    except (ValueError, TypeError, KeyError, AttributeError, OSError):
        print(
            json.dumps(
                dict(
                    outcome='inconclusive',
                    reason='scheduler_evidence_unavailable')))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())

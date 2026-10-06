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
"""Evaluate recreated fixtures on the target; never a pre-upgrade
prediction."""

import argparse
import copy
import json

from kfp_http import Client
from kfp_inventory import collect
from kubectl_inventory import kubectl_get
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
import schedule_policy


def assess(client, context, fixture, policy):
    if (context != CONTEXT or fixture.get('context') != CONTEXT or
            fixture.get('namespace') != NAMESPACE or
            not fixture.get('recreated') or not fixture.get('prepared') or
            fixture.get('enabled') is not False):
        raise ValueError('disabled_recreated_fixtures_required')
    records, failures, coverage = collect(client, [NAMESPACE])
    if failures or coverage['list_completed_namespaces'] != [NAMESPACE]:
        raise ValueError('complete_target_fixture_evidence_required')
    bundle = copy.deepcopy(policy)
    bundle.update(records)
    schedule_policy.validate(bundle)
    data, error = kubectl_get(context, NAMESPACE,
                              'scheduledworkflows.kubeflow.org')
    if error or not isinstance(data, dict) or not isinstance(
            data.get('items'), list):
        raise ValueError('fixture_schedule_collection_failed')
    findings = []
    for case in fixture['schedules']:
        schedules = [
            s for s in data['items']
            if s.get('metadata', {}).get('uid') == case['schedule_uid'] and
            s.get('metadata', {}).get('name') == case['schedule_name'] and
            s.get('metadata', {}).get('namespace') == NAMESPACE
        ]
        if len(schedules) != 1:
            raise ValueError('fixture_schedule_identity_mismatch')
        status, evidence, action = schedule_policy.assess(schedules[0], bundle)
        if status == 'unknown':
            raise ValueError('fixture_policy_unresolved')
        findings.append(
            dict(
                rule='schedule.targetMainAccount',
                status=status,
                resource='ScheduledWorkflow/' + NAMESPACE + '/' +
                case['schedule_name'],
                evidence=evidence,
                action=action))
    if len(findings) != 3:
        raise ValueError('three_fixture_cases_required')
    return dict(
        scope='post_recreation_target_policy_check',
        pre_upgrade_prediction_validated=False,
        target_revision=bundle['target_revision'],
        mode=bundle['mode'],
        findings=findings)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('context', 'fixture-state', 'policy', 'endpoint',
                 'token-file'):
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    try:
        result = assess(
            Client(args.endpoint, args.token_file), args.context,
            read_object(args.fixture_state), read_object(args.policy))
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        parser.exit(1,
                    'Unable to establish recreated fixture policy evidence.\n')
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()

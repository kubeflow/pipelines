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
"""Conditional main-account predictions from explicit target evidence."""

import re

from target_rbac import evaluate_use

CONTRACT = '14363-main-account-preview.1'
POLICY_SOURCE = '698819580262320715ee616c7479b33c62e0a4b7'


def unique(records, field, value):
    matches = [
        r for r in records if isinstance(r, dict) and r.get(field) == value
    ]
    return matches[0] if value and len(matches) == 1 else None


def validate(bundle):
    if not isinstance(bundle,
                      dict) or bundle.get('policy_contract') != CONTRACT:
        raise ValueError('Unsupported schedule policy contract.')
    if not re.fullmatch(r'[0-9a-f]{40}', bundle.get('target_revision', '')):
        raise ValueError('Supply a full target revision.')
    for name in ('recurring_runs', 'experiments', 'rbac'):
        if not isinstance(bundle.get(name), list):
            raise ValueError('Policy evidence requires lists.')
    if sum(
            len(bundle[name])
            for name in ('recurring_runs', 'experiments', 'rbac')) > 10000:
        raise ValueError('Target evidence exceeds 10000 records; reduce scope.')
    for name in ('multi_user', 'rbac_complete', 'rbac_only'):
        if not isinstance(bundle.get(name), bool):
            raise ValueError(
                'Policy evidence requires explicit boolean settings.')
    if bundle.get('mode') not in ('enforce', 'audit'):
        raise ValueError('Invalid service-account mode.')
    for name in ('default_service_account', 'controller_user'):
        if not isinstance(bundle.get(name), str) or not bundle[name]:
            raise ValueError('Missing target identity settings.')
    allow = bundle.get('allowed_service_accounts')
    if not isinstance(allow, list) or not all(
            isinstance(a, str) for a in allow):
        raise ValueError('Invalid account allowlist.')
    return bundle


def assess(schedule, bundle):
    """Never equate one account check with successful schedule execution."""
    unknown = (
        'unknown', 'Target main-account authorization could not be resolved.',
        'Collect the persisted recurring run and experiment, resolve its pipeline identity, and verify the target settings.'
    )
    if not bundle['multi_user']:
        return (
            'unknown',
            'Single-user policy is outside this multi-user prediction contract.',
            'Verify single-user execution separately.')
    metadata = schedule['metadata']
    job = unique(bundle['recurring_runs'], 'recurring_run_id',
                 metadata.get('uid'))
    if job is None:
        return unknown
    # The API record is authoritative; do not trust the editable CR identity.
    experiment = unique(bundle['experiments'], 'experiment_id',
                        job.get('experiment_id'))
    if experiment is None:
        return unknown
    namespace = experiment.get('namespace')
    if not isinstance(
            namespace,
            str) or not namespace or namespace != metadata['namespace']:
        return unknown
    if job.get('namespace') not in (None, '', namespace):
        return unknown
    account = job.get('service_account')
    if not isinstance(account, str) or not re.fullmatch(
            r'[a-z0-9]([-a-z0-9.]*[a-z0-9])?', account):
        # An empty request can acquire an identity from the pipeline/compiler.
        return unknown
    spec = schedule.get('spec', {})
    workflow = spec.get('workflow') if isinstance(spec, dict) else None
    if isinstance(workflow, dict) and workflow.get('spec') is not None:
        return unknown
    evidence = (
        'Persisted recurring-run account ' + namespace + '/' + account +
        '; prediction conditional on supplied target revision/settings and controller identity. '
    )
    if account == bundle['default_service_account']:
        return (
            'no_issue_detected', evidence +
            'Configured default account is exempt from this use check.',
            'Verify resolved compiled/plugin identities and other schedule checks separately.'
        )
    allowed = bundle['allowed_service_accounts']
    if account not in [name.strip() for name in allowed]:
        decision = 'denied'
        reason = 'The target account allowlist excludes this account.'
    else:
        # KFP IsAuthorized submits User without Groups for this policy contract.
        decision = evaluate_use(
            bundle['rbac'],
            bundle['controller_user'], [],
            namespace,
            account,
            complete=bundle['rbac_complete'] and bundle['rbac_only'])
        reason = 'Target RBAC ' + decision + ' for core serviceaccounts/use on the named account (user only, no groups).'
    if decision == 'denied':
        if bundle['mode'] == 'audit':
            return (
                'operational_impact',
                evidence + reason + ' Audit mode records the policy denial.',
                'Correct the allowlist or grant only the named account before switching to enforce; authorization infrastructure failures can still block.'
            )
        return (
            'policy_rejection', evidence + reason,
            'Correct the target allowlist or add a narrowly scoped named-account use grant to the controller; rerun the assessment.'
        )
    if decision == 'allowed':
        return (
            'no_issue_detected', evidence + reason,
            'This covers the main account check only; validate workflow identities, run creation, pipeline access and schedule execution separately.'
        )
    return (
        'unknown', evidence + reason,
        'Supply complete target RBAC-only evidence or verify the exact authorization request against the target authorizer; current-cluster permissions are not target evidence.'
    )

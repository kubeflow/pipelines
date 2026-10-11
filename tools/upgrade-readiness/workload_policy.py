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
"""Conditional workload authorization from explicit target and identity
evidence.

These checks model selected authorization requests, never
authentication, run-scoped token restrictions, admission, or successful
workload execution.
"""

import re

from kfp_inventory import field
from kfp_inventory import v2_template
from target_rbac import evaluate_access

CONTRACT = '2.18-workloads-preview.1'
PIPELINES_GROUP = 'pipelines.kubeflow.org'
MAX_INSPECTION_ITEMS = 10000
MAX_FINDINGS = 10000
OPERATIONS = {
    'read_logs': (PIPELINES_GROUP, 'runs', ('readLog',)),
    'viewer_logs': ('kubeflow.org', 'viewers', ('get',)),
    'manage_viewers': ('kubeflow.org', 'viewers', ('get', 'create', 'delete')),
    'upload_pipeline': (PIPELINES_GROUP, 'pipelines', ('create',)),
    'upload_version': (PIPELINES_GROUP, 'pipelines', ('create',)),
}


def _text(value):
    return isinstance(value, str) and 0 < len(value) <= 1024 and not any(
        ord(c) < 32 for c in value)


def _account(value):
    return isinstance(value, str) and len(value) <= 253 and re.fullmatch(
        r'[a-z0-9]([-a-z0-9.]*[a-z0-9])?', value)


def _label(value):
    return value if isinstance(value, str) and re.fullmatch(
        r'[a-zA-Z0-9][a-zA-Z0-9_.-]{0,252}', value) else '_unresolved'


def validate(bundle):
    if not isinstance(bundle,
                      dict) or bundle.get('policy_contract') != CONTRACT:
        raise ValueError('Unsupported workload policy contract.')
    revision = bundle.get('target_revision')
    if not isinstance(revision, str) or not re.fullmatch(
            r'[0-9a-f]{40}', revision):
        raise ValueError('Supply a full target revision.')
    for key in ('multi_user', 'shared_read', 'compiler_patch_empty',
                'rbac_complete', 'rbac_only'):
        if type(bundle.get(key)) is not bool:
            raise ValueError(
                'Workload policy requires explicit boolean settings.')
    if 'plugins_disabled' in bundle and type(
            bundle['plugins_disabled']) is not bool:
        raise ValueError('plugins_disabled must be an explicit boolean.')
    for key in ('service_account_mode', 'workflow_identity_mode'):
        if bundle.get(key) not in ('enforce', 'audit'):
            raise ValueError(
                'Workload authorization modes must be enforce or audit.')
    if not _account(bundle.get('default_service_account')):
        raise ValueError('Supply a literal default service account.')
    accounts = bundle.get('allowed_service_accounts')
    if not isinstance(accounts, list) or not all(
            isinstance(value, str) and len(value) <= 1024 and not any(
                ord(c) < 32 for c in value) for value in accounts):
        raise ValueError('Supply a bounded list of account allowlist strings.')
    for key in ('rbac', 'identities'):
        if not isinstance(bundle.get(key), list):
            raise ValueError(
                'Workload policy requires RBAC and identity lists.')
    checks = bundle.get('access_checks', [])
    if not isinstance(checks, list) or sum(
            len(values) for values in (bundle['rbac'], bundle['identities'],
                                       checks, accounts)) > 10000:
        raise ValueError('Target evidence exceeds the record budget.')
    seen = set()
    for entry in bundle['identities']:
        if (not isinstance(entry, dict) or
                entry.get('resource_kind') not in ('run', 'recurring_run') or
                not _text(entry.get('resource_id')) or
                not _text(entry.get('user'))):
            raise ValueError(
                'Supply explicit workload identifiers and initiating identities.'
            )
        key = (entry['resource_kind'], entry['resource_id'])
        if key in seen:
            raise ValueError(
                'A workload must have one explicit initiating identity.')
        seen.add(key)
    for entry in checks:
        if (not isinstance(entry, dict) or
                entry.get('operation') not in OPERATIONS or
                not _text(entry.get('namespace')) or
                not _text(entry.get('user')) or
            ('resource_name' in entry and not _text(entry['resource_name']))):
            raise ValueError(
                'Supply supported access checks with explicit namespace and identity.'
            )
    return bundle


def _finding(rule, status, resource, evidence):
    return dict(
        rule=rule,
        status=status,
        resource=resource,
        evidence=evidence +
        ' Conditional on the supplied target revision, settings and identity; this is a selected policy check.',
        action='Resolve missing evidence and correct scoped target grants or settings before migration.',
        verification='Verify the exact target authorization request and an isolated execution; authentication, token scope and admission remain separate checks.'
    )


def _access(bundle,
            user,
            namespace,
            verb,
            group,
            resource,
            name='',
            name_known=True):
    if not bundle['multi_user'] or (bundle['shared_read'] and
                                    verb in ('get', 'list')):
        return 'allowed'
    if not user or not namespace or namespace == '-':
        return 'unknown'
    result = evaluate_access(
        bundle['rbac'],
        user, [],
        namespace,
        verb,
        group,
        resource,
        name,
        complete=bundle['rbac_complete'] and bundle['rbac_only'])
    return 'unknown' if result == 'denied' and not name_known else result


def _status(decision, mode='enforce'):
    if decision == 'denied':
        return 'operational_impact' if mode == 'audit' else 'policy_rejection'
    return 'no_issue_detected' if decision == 'allowed' else 'unknown'


def _account_decision(bundle, user, namespace, account, mode):
    if not _account(account):
        return 'unknown'
    if account == bundle['default_service_account']:
        return 'allowed'
    listed = account in [
        value.strip() for value in bundle['allowed_service_accounts']
    ]
    if not listed and mode == 'enforce':
        return 'denied'
    decision = _access(bundle, user, namespace, 'use', '', 'serviceaccounts',
                       account)
    # Audit still issues the SAR after an allowlist denial; missing SAR evidence
    # cannot become a prediction that audit will allow execution.
    return 'denied' if not listed and decision == 'allowed' else decision


def _unique(records, snake, camel, value):
    matches = [item for item in records if field(item, snake, camel) == value]
    return matches[0] if value and len(matches) == 1 else None


def _source(record, inventory):
    """Return observed spec, owning pipeline, and whether a reference
    exists."""
    reference = field(record, 'pipeline_version_reference',
                      'pipelineVersionReference')
    version_id = field(record, 'pipeline_version_id', 'pipelineVersionId')
    if reference is None and not version_id:
        return field(record, 'pipeline_spec', 'pipelineSpec'), None, False
    pipeline_id = None
    if isinstance(reference, dict):
        pipeline_id = field(reference, 'pipeline_id', 'pipelineId')
        version_id = field(reference, 'pipeline_version_id',
                           'pipelineVersionId')
    annotation = record.get('_readiness_version_reference', {})
    if not version_id and isinstance(
            annotation, dict) and annotation.get('resolution') == 'observed':
        version_id = annotation.get('pipeline_version_id')
    version = _unique(
        inventory.get('pipeline_versions', []), 'pipeline_version_id',
        'pipelineVersionId', version_id)
    if version is None:
        return None, None, True
    owner = field(version, 'pipeline_id', 'pipelineId')
    if not owner or (pipeline_id and owner != pipeline_id):
        return None, None, True
    pipeline = _unique(
        inventory.get('pipelines', []), 'pipeline_id', 'pipelineId', owner)
    return field(version, 'pipeline_spec', 'pipelineSpec'), pipeline, True


def _workflow_accounts(workflow, main):
    """Inspect literal inline templates; unresolved patches/helpers stay
    unknown."""
    accounts = set()
    complete = True
    nodes = [workflow.get('spec')]
    visited = 0
    if workflow.get('status'):
        complete = False
    while nodes:
        node = nodes.pop()
        visited += 1
        if visited > MAX_INSPECTION_ITEMS:
            return accounts, False
        if not isinstance(node, dict):
            complete = False
            continue
        for key in ('podSpecPatch', 'executorPlugins', 'hooks'):
            if node.get(key):
                complete = False
        for key in ('workflowTemplateRef', 'templateRef', 'artifactGC',
                    'resource'):
            if node.get(key) is not None:
                complete = False
        for source in (node, node.get('executor', {})):
            if not isinstance(source, dict):
                complete = False
                continue
            value = source.get('serviceAccountName')
            if value == '{{workflow.serviceAccountName}}':
                value = main
            if value:
                if _account(value):
                    accounts.add(value)
                    if len(accounts) > 100:
                        return set(sorted(accounts)[:100]), False
                else:
                    complete = False
        outputs = node.get('outputs', {})
        if not isinstance(outputs, dict) or outputs.get('artifacts'):
            # Artifact GC inheritance also depends on retained outputs and defaults.
            complete = False
        if node.get('templateDefaults') is not None:
            if visited + len(nodes) >= MAX_INSPECTION_ITEMS:
                return accounts, False
            nodes.append(node['templateDefaults'])
        templates = node.get('templates', [])
        if not isinstance(templates, list):
            complete = False
        else:
            if visited + len(nodes) + len(templates) > MAX_INSPECTION_ITEMS:
                return accounts, False
            nodes.extend(templates)
        dag = node.get('dag', {})
        if not isinstance(dag, dict):
            complete = False
            continue
        tasks = dag.get('tasks', [])
        steps = node.get('steps', [])
        if not isinstance(tasks, list) or not isinstance(steps, list):
            complete = False
            continue
        # Queued nodes already reserve future inspection slots. Copy only
        # the bounded task list so flattening steps cannot mutate the input.
        if visited + len(nodes) + len(tasks) > MAX_INSPECTION_ITEMS:
            return accounts, False
        tasks = list(tasks)
        for group in steps:
            visited += 1
            if visited + len(nodes) + len(tasks) > MAX_INSPECTION_ITEMS:
                return accounts, False
            if isinstance(group, list):
                if visited + len(nodes) + len(tasks) + len(
                        group) > MAX_INSPECTION_ITEMS:
                    return accounts, False
                tasks.extend(group)
            else:
                complete = False
        visited += len(tasks)
        for task in tasks:
            if not isinstance(task, dict):
                complete = False
            elif task.get('templateRef') is not None or task.get('hooks'):
                complete = False
            elif task.get('inline') is not None:
                if visited + len(nodes) >= MAX_INSPECTION_ITEMS:
                    return accounts, False
                nodes.append(task['inline'])
    return accounts, complete


class _FindingBudgetReached(Exception):
    pass


def _check_finding_budget(findings):
    # Reserve one report entry to make unassessed work explicit.
    if len(findings) >= MAX_FINDINGS - 1:
        raise _FindingBudgetReached


def _workload(record, kind, inventory, bundle, results):
    _check_finding_budget(results)
    uid = field(record, kind + '_id',
                'runId' if kind == 'run' else 'recurringRunId')
    namespace = record.get('namespace')
    resource = kind + '/' + _label(namespace) + '/' + _label(uid)
    experiment_id = field(record, 'experiment_id', 'experimentId')
    experiment = _unique(
        inventory.get('experiments', []), 'experiment_id', 'experimentId',
        experiment_id)
    if (record.get('_readiness_collection_errors') or
            not isinstance(namespace, str) or namespace in ('', '-') or
            experiment is None or experiment.get('namespace') != namespace):
        results.append(
            _finding(
                'workload.authorization', 'unknown', resource,
                'Consistent workload, parent namespace and pipeline-source evidence could not be established.'
            ))
        return
    identity = next(
        (entry['user']
         for entry in bundle['identities']
         if entry['resource_kind'] == kind and entry['resource_id'] == uid),
        None)
    spec, pipeline, referenced = _source(record, inventory)
    reference = field(record, 'pipeline_version_reference',
                      'pipelineVersionReference')
    moving_latest = isinstance(reference, dict) and not field(
        reference, 'pipeline_version_id', 'pipelineVersionId')
    account = field(record, 'service_account', 'serviceAccount')
    invalid_account = account not in (None, '') and not isinstance(account, str)
    if not isinstance(account, str):
        account = None
    argo = isinstance(spec,
                      dict) and spec.get('kind') == 'Workflow' and isinstance(
                          spec.get('spec'), dict)
    if not account and not invalid_account and not moving_latest and bundle[
            'compiler_patch_empty']:
        if argo:
            account = spec['spec'].get('serviceAccountName')
            if account in (None, '', 'pipeline-runner'):
                account = bundle['default_service_account']
        elif v2_template(spec):
            account = bundle['default_service_account']
    if not isinstance(account, str):
        account = None
    decision = _account_decision(bundle, identity, namespace, account,
                                 bundle['service_account_mode'])
    if not bundle['compiler_patch_empty']:
        decision = 'unknown'
    results.append(
        _finding(
            'workload.mainAccount',
            _status(decision, bundle['service_account_mode']), resource,
            'Main-account allowlist/default/use decision: ' + decision +
            '. Audit relaxes policy denials only; authorizer failures can still block.'
        ))
    _check_finding_budget(results)
    # The request's explicit main account overrides the embedded workflow account.
    effective = dict(
        spec, spec=dict(spec['spec'],
                        serviceAccountName=account)) if argo else None
    accounts, complete = _workflow_accounts(effective,
                                            account) if argo else (set(), False)
    for extra in sorted(accounts - {account}):
        _check_finding_budget(results)
        decision = _account_decision(bundle, identity, namespace, extra,
                                     bundle['workflow_identity_mode'])
        if moving_latest:
            decision = 'unknown'
        results.append(
            _finding(
                'workload.embeddedAccount',
                _status(decision, bundle['workflow_identity_mode']), resource,
                'Literal embedded account ' + extra + ': ' + decision +
                ' under the independent workflow-identity mode.'))
    _check_finding_budget(results)
    if (not complete or moving_latest or not bundle['compiler_patch_empty'] or
            bundle.get('plugins_disabled') is not True or
            field(record, 'plugins_input', 'pluginsInput')):
        results.append(
            _finding(
                'workload.identityCoverage', 'unknown', resource,
                'Complete effective identities are unresolved: moving versions, compiler output, patches, external templates, retained execution state or registered plugins require target execution evidence.'
            ))
    else:
        results.append(
            _finding(
                'workload.identityCoverage', 'no_issue_detected', resource,
                'Literal inline workflow identity locations inspected under asserted empty compiler patch and disabled plugins. Individual account decisions are reported separately.'
            ))
    _check_finding_budget(results)
    decision = _access(bundle, identity, namespace, 'create', PIPELINES_GROUP,
                       'runs')
    results.append(
        _finding(
            'workload.createRun', _status(decision), resource,
            'Initiating identity runs/create request with an empty resource name: '
            + decision + '.'))
    _check_finding_budget(results)
    decision = 'unknown'
    if not referenced and isinstance(spec, dict) and spec:
        decision = 'allowed'
    elif pipeline is not None:
        owner_namespace = pipeline.get('namespace')
        if owner_namespace in (
                '', '-') or bundle['shared_read'] or not bundle['multi_user']:
            decision = 'allowed'
        elif isinstance(
                owner_namespace, str
        ) and owner_namespace and namespace and owner_namespace != namespace:
            decision = 'denied'
        elif owner_namespace == namespace:
            name = pipeline.get('name')
            decision = _access(
                bundle,
                identity,
                namespace,
                'get',
                PIPELINES_GROUP,
                'pipelines',
                name if isinstance(name, str) else '',
                name_known=bool(name))
    results.append(
        _finding(
            'workload.pipelineRead', _status(decision), resource,
            'Stored pipeline ownership/namespace/read decision (inline specs need no stored pipeline read): '
            + decision + '.'))


def assess(inventory, bundle):
    validate(bundle)
    findings = []
    try:
        for kind, key in (('run', 'runs'), ('recurring_run', 'recurring_runs')):
            for record in inventory.get(key, []):
                _workload(record, kind, inventory, bundle, findings)
        for index, check in enumerate(bundle.get('access_checks', [])):
            group, resource, verbs = OPERATIONS[check['operation']]
            named = check['operation'] == 'read_logs'
            name = check.get('resource_name', '') if named else ''
            for verb in verbs:
                _check_finding_budget(findings)
                decision = _access(
                    bundle,
                    check['user'],
                    check['namespace'],
                    verb,
                    group,
                    resource,
                    name,
                    name_known=not named or bool(name))
                findings.append(
                    _finding(
                        'access.' + check['operation'], _status(decision),
                        'access_check/' + str(index) + '/' +
                        _label(check['namespace']),
                        group + '/' + resource + '/' + verb +
                        ' (user only, no groups): ' + decision + '.'))
    except _FindingBudgetReached:
        findings.append(
            _finding(
                'workload.coverage', 'unknown', 'workload_policy',
                'Finding budget reached; remaining workload and access checks are unassessed. Earlier partial results are retained.'
            ))
    return findings

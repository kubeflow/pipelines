#!/usr/bin/env python3
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
"""Read-only, deliberately partial KFP 2.18 migration-plan assessment."""

import argparse
from collections import Counter
from datetime import datetime
from datetime import timezone
import json
import os
from pathlib import Path
import re
import selectors
import signal
import subprocess
import sys
import time

MAX_BYTES = 16 * 1024 * 1024
MAX_ITEMS = 10000
RULESET = '2.18-preview.1'
SUPPORTED_KINDS = {'Deployment', 'Role', 'RoleBinding', 'ClusterRole'}
GAPS = [
    'Effective user/group/service-account authorization and cluster-wide bindings',
    'Stored pipelines, versions, runs, schedules, workflow identities and ownership',
    'Artifact origins, profile proxies, archive credentials and HTTP base validation',
    'Upload/compressed/extracted/metrics sizes and historical request bodies',
    'Legacy and V2 cache entries, hit rates, recomputation time and cost',
    'SDK compilation, pip mirrors, embedded artifacts and external API consumers',
    'Observed traffic, infrequent schedules, pagination and live upgrade acceptance',
]


def finding(rule, status, resource, evidence, action, verification):
    return dict(
        rule=rule,
        status=status,
        resource=resource,
        evidence=evidence,
        action=action,
        verification=verification)


def resource_id(obj):
    metadata = obj.get('metadata', {})
    return '/'.join([
        obj.get('kind', 'Unknown'),
        metadata.get('namespace', '_cluster'),
        metadata.get('name', '_unnamed')
    ])


def read_json(path):
    with open(path, 'rb') as stream:
        raw = stream.read(MAX_BYTES + 1)
    if len(raw) > MAX_BYTES:
        raise ValueError(
            'Inventory exceeds the 16 MiB input limit; reduce scope.')
    return json.loads(raw)


def validate_inventory(inventory):
    if not isinstance(inventory, dict) or not isinstance(
            inventory.get('items'), list):
        raise ValueError(
            'Inventory must be an object containing an items list.')
    if len(inventory['items']) > MAX_ITEMS:
        raise ValueError('Inventory exceeds 10000 objects; reduce scope.')
    for obj in inventory['items']:
        if not isinstance(obj, dict) or obj.get('kind') not in SUPPORTED_KINDS:
            raise ValueError(
                'Only Deployment, Role, RoleBinding and ClusterRole objects are accepted.'
            )
        if not isinstance(obj.get('metadata'), dict):
            raise ValueError('Each inventory object requires metadata.')
        if not isinstance(obj['metadata'].get('name'), str):
            raise ValueError('Each inventory object requires a name.')
        if obj['kind'] != 'ClusterRole' and (
                not isinstance(obj['metadata'].get('namespace'), str) or
                not re.fullmatch(r'[a-z0-9]([-a-z0-9]*[a-z0-9])?',
                                 obj['metadata']['namespace']) or
                len(obj['metadata']['namespace']) > 63):
            raise ValueError(
                'Namespace-scoped objects require metadata.namespace.')
        if obj['kind'] in ('Role', 'ClusterRole'):
            rules = obj.get('rules', [])
            if not isinstance(rules, list):
                raise ValueError('Role rules must be a list.')
            for rule in rules:
                if not isinstance(rule, dict):
                    raise ValueError('Role rules must be objects.')
                for field in ('apiGroups', 'resources', 'verbs',
                              'resourceNames', 'nonResourceURLs'):
                    values = rule.get(field, [])
                    if not isinstance(values, list) or not all(
                            isinstance(v, str) for v in values):
                        raise ValueError(
                            'Role rule fields must be string lists.')
    return inventory


def kill_process_group(process):
    """Stop kubectl and inherited exec-plugin processes in its private
    session."""
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def kubectl_get(context, namespace, resource):
    """Bound time and buffered output; never print kubectl stderr or raw
    objects."""
    command = ['kubectl', '--context', context, '--request-timeout=20s']
    if namespace:
        command += ['--namespace', namespace]
    command += ['get', resource, '--chunk-size=200', '-o', 'json']
    # Cap bytes while streaming, not after kubectl has filled memory or disk.
    chunks = bytearray()
    try:
        with subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                start_new_session=True) as process:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                deadline = time.monotonic() + 30
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        kill_process_group(process)
                        return None, 'collection_timed_out'
                    chunk = process.stdout.read1(
                        min(65536, MAX_BYTES + 1 - len(chunks)))
                    if not chunk:
                        break
                    chunks.extend(chunk)
                    if len(chunks) > MAX_BYTES:
                        kill_process_group(process)
                        return None, 'collection_exceeded_16_mib'
                try:
                    code = process.wait(
                        timeout=max(0.01, deadline - time.monotonic()))
                except subprocess.TimeoutExpired:
                    kill_process_group(process)
                    return None, 'collection_timed_out'
                if code:
                    return None, 'collection_failed'
    except OSError:
        return None, 'collection_failed'
    try:
        return json.loads(chunks), None
    except (ValueError, UnicodeError):
        return None, 'collection_invalid_json'


def collect(context, namespaces):
    items, failures = [], []
    inventory_bytes = 0

    def retain(objects, namespace, resource):
        nonlocal inventory_bytes
        size = sum(len(json.dumps(obj).encode('utf-8')) for obj in objects)
        if inventory_bytes + size > MAX_BYTES:
            failures.append(
                dict(
                    namespace=namespace or '_cluster',
                    resource=resource,
                    reason='inventory_budget_exceeded_remaining_scope_not_collected'
                ))
            return False
        items.extend(objects)
        inventory_bytes += size
        if len(items) > MAX_ITEMS:
            raise ValueError('Inventory exceeds 10000 objects; reduce scope.')
        return True

    for namespace in namespaces:
        for resource in [
                'deployments.apps', 'roles.rbac.authorization.k8s.io',
                'rolebindings.rbac.authorization.k8s.io'
        ]:
            data, error = kubectl_get(context, namespace, resource)
            if error or not isinstance(data, dict) or not isinstance(
                    data.get('items'), list):
                failures.append(
                    dict(
                        namespace=namespace,
                        resource=resource,
                        reason=error or 'collection_invalid_list'))
                continue
            if not retain(data['items'], namespace, resource):
                return validate_inventory(dict(items=items)), failures
    refs = sorted({
        obj.get('roleRef', {}).get('name')
        for obj in items
        if obj.get('kind') == 'RoleBinding' and obj.get('roleRef', {}).get(
            'kind') == 'ClusterRole' and obj.get('roleRef', {}).get('name')
    })
    if len(refs) > 256:
        raise ValueError('More than 256 referenced ClusterRoles; reduce scope.')
    for name in refs:
        # Names are Kubernetes names, not options or shell expressions.
        if not re.fullmatch(r'[a-zA-Z0-9_.:-]+', name):
            failures.append(
                dict(resource='ClusterRole', reason='invalid_reference'))
            continue
        data, error = kubectl_get(
            context, None, 'clusterroles.rbac.authorization.k8s.io/' + name)
        if error or not isinstance(data,
                                   dict) or data.get('kind') != 'ClusterRole':
            failures.append(
                dict(
                    resource='ClusterRole/' + name,
                    reason=error or 'collection_invalid_object'))
        else:
            if not retain([data], None, 'ClusterRole/' + name):
                return validate_inventory(dict(items=items)), failures
    return validate_inventory(dict(items=items)), failures


def permits(role, group, resource, verb):
    # This inspects one role, not effective authorization. resourceNames and
    # additional bindings may change the actual caller's permission.
    return any(
        (group in r.get('apiGroups', []) or '*' in r.get('apiGroups', [])) and
        (resource in r.get('resources', []) or '*' in r.get('resources', [])
        ) and (verb in r.get('verbs', []) or '*' in r.get('verbs', []))
        for r in role.get('rules', []))


def analyze(inventory,
            system_namespace,
            namespaces,
            ui_name,
            cache_name,
            source_version,
            failures=None,
            mode='offline'):
    inventory = validate_inventory(inventory)
    items = [
        obj for obj in inventory['items'] if obj['kind'] == 'ClusterRole' or
        obj['metadata'].get('namespace') in namespaces
    ]
    findings = []
    failures = list(failures or [])
    if mode == 'offline':
        failures.append(
            dict(
                resource='offline_inventory',
                reason='offline_inventory_completeness_unverified'))
    deployments = [
        obj for obj in items if obj['kind'] == 'Deployment' and
        obj['metadata'].get('namespace') == system_namespace
    ]
    ui = next(
        (obj for obj in deployments if obj['metadata']['name'] == ui_name),
        None)
    if ui is None:
        for rule in ('tensorboard.key', 'tensorboard.rollout'):
            findings.append(
                finding(
                    rule, 'unknown', system_namespace + '/' + ui_name,
                    'Selected UI deployment was not collected.',
                    'Check scope, deployment name and read permissions.',
                    'Rerun with the correct --ui-deployment and installation namespace.'
                ))
    else:
        containers = ui.get('spec', {}).get('template',
                                            {}).get('spec',
                                                    {}).get('containers', [])
        ui_container = next(
            (c for c in containers if c.get('name') == 'ml-pipeline-ui'), None)
        if ui_container is None and len(containers) == 1:
            ui_container = containers[0]
        env = (ui_container or {}).get('env', [])
        key = next((e for e in env
                    if e.get('name') == 'TENSORBOARD_PROXY_SIGNING_SECRET'),
                   None)
        evidence = 'Dedicated signing-key configuration could not be established from explicit environment entries.'
        if key is not None:
            evidence = 'An explicit signing-key entry exists; its value, validity and replica consistency were not inspected.'
        elif (ui_container or {}).get('envFrom'):
            evidence = 'Environment imports may supply a key; imported values were not inspected.'
        findings.append(
            finding(
                'tensorboard.key', 'unknown', resource_id(ui), evidence,
                'Verify a persistent dedicated key shared by all UI replicas without exposing its value.',
                'After adoption, verify refreshed TensorBoard URLs across replicas and restarts.'
            ))
        strategy = ui.get('spec', {}).get('strategy',
                                          {}).get('type', 'RollingUpdate')
        if strategy not in ('RollingUpdate', 'Recreate'):
            strategy = 'unrecognized'
        findings.append(
            finding(
                'tensorboard.rollout', 'operational_impact', resource_id(ui),
                'Declared deployment strategy is ' + str(strategy) +
                '; target first-adoption behavior depends on #14362.',
                'Plan coordinated first shared-key adoption, a UI interruption and refreshed TensorBoard URLs. '
                'Do not infer that a currently rolling deployment is ready for mixed-key replicas.',
                'Validate the final target manifests and key-adoption procedure on a representative installation.'
            ))
    cache = next(
        (obj for obj in deployments if obj['metadata']['name'] == cache_name),
        None)
    findings.append(
        finding(
            'cache.legacy', 'unknown',
            resource_id(cache) if cache else system_namespace + '/' +
            cache_name,
            'A legacy cache deployment was collected; usage and historical ownership are unknown.'
            if cache else
            'No selected legacy cache deployment was collected; absence is not proof it is unused.',
            'Confirm V1/raw-Argo cache usage and budget recomputation when affected tasks next run, not at startup.',
            'Assess cache data and representative runs separately; deployment inventory cannot estimate cost.'
        ))
    roles = {
        (obj['kind'], obj['metadata'].get('namespace',
                                          ''), obj['metadata']['name']):
            obj for obj in items if obj['kind'] in ('Role', 'ClusterRole')
    }
    for binding in sorted(
        (obj for obj in items if obj['kind'] == 'RoleBinding'),
            key=resource_id):
        ref = binding.get('roleRef', {})
        namespace = binding['metadata'].get(
            'namespace', '') if ref.get('kind') == 'Role' else ''
        role = roles.get((ref.get('kind'), namespace, ref.get('name')))
        if role is None or role.get(
                'aggregationRule') and not role.get('rules'):
            findings.append(
                finding(
                    'rbac.coverage', 'unknown', resource_id(binding),
                    'Referenced role is missing or its aggregate rules are unavailable.',
                    'Collect the referenced role and let Kubernetes resolve any aggregation.',
                    'Check effective access with the actual caller identity.'))
            continue
        if any(
                permits(role, 'pipelines.kubeflow.org', 'runs', v)
                for v in ['get', 'list']):
            declared = permits(role, 'pipelines.kubeflow.org', 'runs',
                               'readLog')
            findings.append(
                finding(
                    'rbac.readLog',
                    'no_issue_detected' if declared else 'unknown',
                    resource_id(binding),
                    'Referenced role declares runs/readLog; effective caller access is not assessed.'
                    if declared else
                    'Referenced role grants run reads but does not declare runs/readLog. Other grants may supply it.',
                    'Verify log readers have the readLog verb on runs in pipelines.kubeflow.org in the run namespace.',
                    'Use an administrator-authorized identity check and a real log request; include group membership.'
                ))
        if permits(role, 'kubeflow.org', 'viewers', 'get'):
            manages = all(
                permits(role, 'kubeflow.org', 'viewers', v)
                for v in ['create', 'delete'])
            findings.append(
                finding(
                    'rbac.tensorboard',
                    'no_issue_detected' if manages else 'unknown',
                    resource_id(binding),
                    'Referenced role declares viewer create/delete; effective caller access is not assessed.'
                    if manages else
                    'Viewer read permission does not establish create/delete permission; other roles may grant it.',
                    'Keep readers read-only; grant scoped viewer create/delete only to intended TensorBoard managers.',
                    'Test existing viewer reads and allowed/denied create/delete with representative credentials.'
                ))
    for failure in failures or []:
        findings.append(
            finding(
                'inventory.collection', 'unknown',
                failure.get('namespace', '_cluster') + '/' +
                failure['resource'], failure['reason'],
                'Check read permissions, connectivity and scope; raw server errors are not included.',
                'Rerun collection; missing data is never an empty successful assessment.'
            ))
    for gap in GAPS:
        findings.append(
            finding(
                'coverage.unassessed', 'unknown', 'installation', gap,
                'Complete this assessment before using the report for an upgrade decision.',
                'Follow the final-branch acceptance checklist in #14421.'))
    return dict(
        schema_version=1,
        ruleset=RULESET,
        generated_at=datetime.now(timezone.utc).isoformat(),
        target=dict(
            version='2.18',
            status='migration_plan_preview_not_release_certification',
            references=[
                'https://github.com/kubeflow/pipelines/issues/14421',
                'https://github.com/kubeflow/pipelines/pull/14362'
            ]),
        source=dict(
            version=source_version,
            version_evidence='operator_supplied_not_verified',
            collection=mode,
            namespaces=namespaces),
        assessment='incomplete',
        assessed_objects=len(items),
        counts=dict(Counter(f['status'] for f in findings)),
        findings=findings)


def markdown(report):
    lines = [
        '# KFP upgrade readiness — preview', '',
        '**Assessment: incomplete. This report does not certify readiness.**',
        '', 'Rules: ' + report['ruleset'] +
        '; source version supplied by operator: ' + report['source']['version'],
        ''
    ]
    for f in report['findings']:
        # Escape control characters and markup in inventory-controlled names.
        safe = lambda value: json.dumps(
            value, ensure_ascii=True)[1:-1].replace('<', '&lt;').replace(
                '>', '&gt;').replace('`', '\\`')
        lines += [
            '- **' + f['status'] + '** — ' + safe(f['rule']) + ' (`' +
            safe(f['resource']) + '`)', '  - Evidence: ' + safe(f['evidence']),
            '  - Action: ' + safe(f['action']),
            '  - Verify: ' + safe(f['verification']), ''
        ]
    return '\n'.join(lines)


class ArgumentParser(argparse.ArgumentParser):
    """Reserve status 2 for an emitted incomplete report, not usage errors."""

    def error(self, message):
        self.print_usage(sys.stderr)
        self.exit(1, f'{self.prog}: error: {message}\n')


def main(argv=None):
    parser = ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument(
        '--context',
        help='Explicit kubectl context; only get requests are issued.')
    source.add_argument(
        '--inventory',
        type=Path,
        help='Offline JSON items list; not assumed complete.')
    parser.add_argument('--system-namespace', required=True)
    parser.add_argument(
        '--namespace',
        action='append',
        default=[],
        help='Additional namespace to assess; repeatable.')
    parser.add_argument(
        '--source-version',
        required=True,
        choices=['2.17.0', '2.17.1', '2.17.2'])
    parser.add_argument('--ui-deployment', default='ml-pipeline-ui')
    parser.add_argument('--cache-deployment', default='cache-server')
    parser.add_argument(
        '--format', choices=['json', 'markdown'], default='markdown')
    args = parser.parse_args(argv)
    namespaces = sorted(set([args.system_namespace] + args.namespace))
    if len(namespaces) > 100 or any(
            not re.fullmatch(r'[a-z0-9]([-a-z0-9]*[a-z0-9])?', n) or len(n) > 63
            for n in namespaces):
        parser.error('Supply at most 100 valid Kubernetes namespace names.')
    try:
        inventory, failures = collect(
            args.context, namespaces) if args.context else (read_json(
                args.inventory), [])
        report = analyze(
            inventory,
            args.system_namespace,
            namespaces,
            args.ui_deployment,
            args.cache_deployment,
            args.source_version,
            failures,
            mode='live' if args.context else 'offline')
    except (OSError, ValueError, TypeError, AttributeError, KeyError):
        print(
            'Unable to assess inventory: check JSON structure, supported kinds, input size and scope. '
            'No readiness conclusion is available.',
            file=sys.stderr)
        return 1
    print(
        json.dumps(report, indent=2, sort_keys=True) if args.format ==
        'json' else markdown(report))
    # Zero means complete within the supported rule contract. Preview rules
    # deliberately leave coverage gaps; they must never silently signal green.
    return 2


if __name__ == '__main__':
    sys.exit(main())

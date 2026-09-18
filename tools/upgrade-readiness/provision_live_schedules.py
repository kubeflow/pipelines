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
"""MUTATING helper for disposable kind upgrade CI; never a readiness scan."""

import argparse
from datetime import datetime
from datetime import timezone
import http.client
import ipaddress
import json
import os
from pathlib import Path
import re
import subprocess
import time
import urllib.parse
import uuid

from readiness import kubectl_get

CONTEXT = 'kind-kfp-readiness'
NAMESPACE = 'kfp-readiness-test'
LABEL = 'pipelines.kubeflow.org/readiness-fixture'
OWNER = 'fixture-owner'
ACCOUNTS = ('pipeline-runner', 'readiness-granted', 'readiness-denied')
MAX_BYTES = 4 * 1024 * 1024


class FixtureError(ValueError):
    """A redacted fixture preparation error."""


def read_object(path):
    with open(path, 'rb') as stream:
        raw = stream.read(MAX_BYTES + 1)
    if len(raw) > MAX_BYTES:
        raise FixtureError('input_limit_exceeded')
    value = json.loads(raw)
    if not isinstance(value, dict):
        raise FixtureError('invalid_object')
    return value


def write_object(path, value):
    temporary = path.with_suffix('.tmp')
    with open(temporary, 'w', encoding='utf-8') as stream:
        os.chmod(temporary, 0o600)
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')
    temporary.replace(path)


class FixtureClient:
    """Authenticated POSTs to a literal loopback port-forward, without
    proxies."""

    def __init__(self, endpoint, token_file):
        try:
            parsed = urllib.parse.urlsplit(endpoint)
            if (parsed.scheme != 'http' or parsed.username is not None or
                    parsed.password is not None or
                    parsed.path not in ('', '/') or parsed.query or
                    parsed.fragment or not parsed.port or
                    not ipaddress.ip_address(parsed.hostname).is_loopback):
                raise ValueError()
            self.host, self.port = parsed.hostname, parsed.port
            with open(token_file, 'rb') as stream:
                raw = stream.read(16385)
            token = raw.decode('ascii').strip()
            if len(raw) > 16384 or not re.fullmatch(r'[A-Za-z0-9._~+/-]+=*',
                                                    token):
                raise ValueError()
            self.token = token
        except (OSError, ValueError, TypeError):
            raise FixtureError('invalid_loopback_endpoint_or_token') from None

    def post(self, path, body):
        if not re.fullmatch(
                r'/apis/v2beta1/(experiments|recurringruns)(/[a-zA-Z0-9_.-]+:(enable|disable))?',
                path):
            raise FixtureError('invalid_fixture_api_path')
        payload = json.dumps(body).encode('utf-8')
        if len(payload) > MAX_BYTES:
            raise FixtureError('request_limit_exceeded')
        connection = http.client.HTTPConnection(
            self.host, self.port, timeout=20)
        try:
            deadline = time.monotonic() + 20
            connection.request(
                'POST',
                path,
                body=payload,
                headers={
                    'Authorization': 'Bearer ' + self.token,
                    'Content-Type': 'application/json',
                    'Accept-Encoding': 'identity'
                })
            response = connection.getresponse()
            if response.status < 200 or response.status >= 300:
                raise FixtureError('fixture_api_request_rejected')
            chunks, size = [], 0
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise FixtureError('fixture_api_timeout')
                raw = getattr(getattr(response, 'fp', None), 'raw', None)
                sock = getattr(raw, '_sock', None) or connection.sock
                if sock is not None:
                    sock.settimeout(remaining)
                chunk = response.read1(min(65536, MAX_BYTES - size + 1))
                size += len(chunk)
                if size > MAX_BYTES or time.monotonic() >= deadline:
                    raise FixtureError('fixture_api_response_limit')
                if not chunk:
                    break
                chunks.append(chunk)
            value = json.loads(b''.join(chunks) or b'{}')
            if not isinstance(value, dict) or value.get('error'):
                raise FixtureError('invalid_fixture_api_response')
            return value
        except (OSError, http.client.HTTPException, ValueError):
            raise FixtureError('fixture_api_request_failed') from None
        finally:
            connection.close()


def kubectl_create(context, objects):
    try:
        result = subprocess.run([
            'kubectl', '--context', context, '--request-timeout=20s', 'create',
            '-f', '-', '-o', 'name'
        ],
                                input=json.dumps({
                                    'apiVersion': 'v1',
                                    'kind': 'List',
                                    'items': objects
                                }),
                                text=True,
                                stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL,
                                timeout=30,
                                check=False)
        if result.returncode:
            raise FixtureError('fixture_kubernetes_create_failed')
    except (OSError, subprocess.TimeoutExpired):
        raise FixtureError('fixture_kubernetes_create_failed') from None


def resource(kind, name, **fields):
    group = 'rbac.authorization.k8s.io/v1' if kind in ('Role',
                                                       'RoleBinding') else 'v1'
    return dict(
        apiVersion=group,
        kind=kind,
        metadata=dict(name=name, namespace=NAMESPACE),
        **fields)


def binding(name, role, names, namespace=NAMESPACE):
    return resource(
        'RoleBinding',
        name,
        roleRef=dict(
            apiGroup='rbac.authorization.k8s.io', kind='Role', name=role),
        subjects=[
            dict(kind='ServiceAccount', name=n, namespace=namespace)
            for n in names
        ])


def fixture_rbac(runner_rules):
    objects = [resource('ServiceAccount', name) for name in (OWNER,) + ACCOUNTS]
    objects += [
        resource('Role', 'pipeline-runner', rules=runner_rules),
        binding('fixture-runners', 'pipeline-runner', ACCOUNTS)
    ]
    api_resources = [
        'experiments', 'jobs', 'recurringruns', 'runs', 'pipelines',
        'pipelines/versions'
    ]
    objects += [
        resource(
            'Role',
            'fixture-owner',
            rules=[
                dict(
                    apiGroups=['pipelines.kubeflow.org'],
                    resources=api_resources,
                    verbs=[
                        'create', 'get', 'list', 'update', 'enable', 'disable'
                    ]),
                dict(
                    apiGroups=[''],
                    resources=['serviceaccounts'],
                    resourceNames=list(ACCOUNTS[1:]),
                    verbs=['use'])
            ]),
        binding('fixture-owner', 'fixture-owner', [OWNER]),
        resource(
            'Role',
            'fixture-controller',
            rules=[
                dict(
                    apiGroups=['pipelines.kubeflow.org'],
                    resources=['runs', 'pipelines', 'pipelines/versions'],
                    verbs=['create', 'get', 'list']),
                dict(
                    apiGroups=[''],
                    resources=['serviceaccounts'],
                    resourceNames=['readiness-granted'],
                    verbs=['use'])
            ]),
        binding('fixture-controller', 'fixture-controller',
                ['ml-pipeline-scheduledworkflow'], 'kubeflow')
    ]
    return objects


def source_role(context, name):
    role, error = kubectl_get(context, 'kubeflow', 'role/' + name)
    if error or not isinstance(role, dict) or not isinstance(
            role.get('rules'), list) or not role['rules']:
        raise FixtureError('fixture_source_role_unavailable')
    # A copied infrastructure role must not invalidate the denied-account case.
    for rule in role['rules']:
        if (any(g in ('', '*') for g in rule.get('apiGroups', [])) and any(
                r in ('serviceaccounts', '*')
                for r in rule.get('resources', [])) and
                any(v in ('use', '*') for v in rule.get('verbs', []))):
            raise FixtureError('fixture_source_role_grants_account_use')
    return role['rules']


def fixture_runtime_config(context):
    """Copy only disposable installation configuration needed by Argo
    runners."""
    objects = []
    for kind, name in (('Secret', 'mlpipeline-minio-artifact'),
                       ('ConfigMap', 'kfp-launcher')):
        value, error = kubectl_get(context, 'kubeflow',
                                   kind.lower() + '/' + name)
        if error or not isinstance(value, dict):
            raise FixtureError('fixture_runtime_config_unavailable')
        data = value.get('data', {})
        if not isinstance(data, dict) or not all(
                isinstance(k, str) and isinstance(v, str)
                for k, v in data.items()):
            raise FixtureError('invalid_fixture_runtime_config')
        if len(json.dumps(data).encode('utf-8')) > MAX_BYTES:
            raise FixtureError('fixture_runtime_config_limit')
        if kind == 'Secret':
            if value.get(
                    'type',
                    'Opaque') != 'Opaque' or not {'accesskey', 'secretkey'
                                                 } <= set(data):
                raise FixtureError('invalid_fixture_artifact_secret')
            objects.append(resource(kind, name, data=data, type='Opaque'))
        else:
            objects.append(resource(kind, name, data=data))
    return objects


def infrastructure_rbac(context):
    """Copy installed namespace roles; grant API authentication reviews
    separately."""
    objects = []
    for role, account in (('ml-pipeline', 'ml-pipeline'),
                          ('ml-pipeline-scheduledworkflow-role',
                           'ml-pipeline-scheduledworkflow'),
                          ('ml-pipeline-persistenceagent-role',
                           'ml-pipeline-persistenceagent')):
        name = 'fixture-' + account + '-infrastructure'
        objects += [
            resource('Role', name, rules=source_role(context, role)),
            binding(name, name, [account], 'kubeflow')
        ]
    deployment, error = kubectl_get(context, 'kubeflow',
                                    'deployment/workflow-controller')
    if error or not isinstance(deployment, dict):
        raise FixtureError('fixture_argo_deployment_unavailable')
    account = api_identifier(
        deployment.get('spec', {}).get('template',
                                       {}).get('spec',
                                               {}).get('serviceAccountName'))
    bindings, error = kubectl_get(context, 'kubeflow', 'rolebindings')
    if error or not isinstance(bindings, dict) or not isinstance(
            bindings.get('items'), list):
        raise FixtureError('fixture_argo_roles_unavailable')
    roles = set()
    for item in bindings['items']:
        subjects = item.get('subjects', [])
        if not any(
                subject.get('kind') == 'ServiceAccount' and
                subject.get('name') == account and
                subject.get('namespace', 'kubeflow') == 'kubeflow'
                for subject in subjects):
            continue
        reference = item.get('roleRef', {})
        if reference.get('kind') != 'Role' or reference.get(
                'apiGroup') != 'rbac.authorization.k8s.io':
            raise FixtureError('fixture_requires_namespaced_argo_roles')
        roles.add(api_identifier(reference.get('name')))
    if not roles or len(roles) > 10:
        raise FixtureError('fixture_argo_roles_unavailable')
    for index, role in enumerate(sorted(roles)):
        name = 'fixture-argo-infrastructure-' + str(index)
        objects += [
            resource('Role', name, rules=source_role(context, role)),
            binding(name, name, [account], 'kubeflow')
        ]
    # TokenReview and SubjectAccessReview are cluster-scoped even when the
    # installation's ordinary execution permissions are namespaced.
    name = 'kfp-readiness-fixture-authentication'
    objects += [
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='ClusterRole',
            metadata=dict(name=name),
            rules=[
                dict(
                    apiGroups=['authentication.k8s.io'],
                    resources=['tokenreviews'],
                    verbs=['create']),
                dict(
                    apiGroups=['authorization.k8s.io'],
                    resources=['subjectaccessreviews'],
                    verbs=['create'])
            ]),
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='ClusterRoleBinding',
            metadata=dict(name=name),
            roleRef=dict(
                apiGroup='rbac.authorization.k8s.io',
                kind='ClusterRole',
                name=name),
            subjects=[
                dict(
                    kind='ServiceAccount',
                    name='ml-pipeline',
                    namespace='kubeflow')
            ])
    ]
    return objects


def verify_state(context, state):
    marker = state.get('owner_marker', '')
    if state.get('context') != CONTEXT or state.get(
            'namespace') != NAMESPACE or not re.fullmatch(
                r'[a-f0-9]{32}', marker):
        raise FixtureError('invalid_fixture_state')
    obj, error = kubectl_get(context, None, 'namespace/' + NAMESPACE)
    if error or obj.get('metadata', {}).get('labels', {}).get(LABEL) != marker:
        raise FixtureError('fixture_namespace_ownership_mismatch')


def provision_rbac(context, state_dir):
    path = state_dir / 'state.json'
    if path.exists():
        raise FixtureError('fixture_state_already_exists_use_fresh_cluster')
    # A create (never apply) prevents adopting an existing namespace.
    role, error = kubectl_get(context, 'kubeflow', 'role/pipeline-runner')
    if error or not isinstance(role.get('rules'), list) or not role['rules']:
        raise FixtureError('pipeline_runner_role_unavailable')
    state = dict(
        context=CONTEXT,
        namespace=NAMESPACE,
        owner_marker=uuid.uuid4().hex,
        schedules=[])
    extra_roles = infrastructure_rbac(context) + fixture_runtime_config(context)
    write_object(path, state)
    kubectl_create(context, [
        dict(
            apiVersion='v1',
            kind='Namespace',
            metadata=dict(
                name=NAMESPACE, labels={LABEL: state['owner_marker']}))
    ])
    verify_state(context, state)
    kubectl_create(context, fixture_rbac(role['rules']) + extra_roles)
    state['rbac_ready'] = True
    write_object(path, state)


def api_identifier(value):
    if not isinstance(value, str) or not re.fullmatch(
            r'[a-zA-Z0-9][a-zA-Z0-9_.-]{0,252}', value):
        raise FixtureError('invalid_api_identifier')
    return value


def schedule_identity(context, recurring_id):
    for _ in range(10):
        data, error = kubectl_get(context, NAMESPACE,
                                  'scheduledworkflows.kubeflow.org')
        if error or not isinstance(data.get('items'), list):
            raise FixtureError('fixture_schedule_collection_failed')
        matches = [
            s for s in data['items']
            if s.get('metadata', {}).get('uid') == recurring_id
        ]
        if len(matches) == 1:
            metadata = matches[0]['metadata']
            if metadata.get('namespace') != NAMESPACE:
                raise FixtureError('fixture_schedule_namespace_mismatch')
            if matches[0].get('spec', {}).get('enabled', False) is not False:
                raise FixtureError('fixture_schedule_not_disabled')
            return api_identifier(metadata.get('name'))
        if len(matches) > 1:
            raise FixtureError('duplicate_fixture_schedule')
        time.sleep(1)
    raise FixtureError('fixture_schedule_not_found')


def prepare(context, state_dir, state, client, pipeline_spec):
    if not state.get('rbac_ready'):
        raise FixtureError('fixture_rbac_not_ready')
    if state.get('prepared') or state.get('schedules') or state.get(
            'experiment_id'):
        raise FixtureError(
            'fixture_already_or_partially_prepared_use_fresh_cluster')
    # Require compiled V2 IR; the CI supplies a reviewed trivial workload.
    if not pipeline_spec.get('pipelineInfo') or not pipeline_spec.get(
            'root') or not pipeline_spec.get('deploymentSpec'):
        raise FixtureError('compiled_v2_pipeline_required')
    experiment = client.post(
        '/apis/v2beta1/experiments',
        dict(
            display_name='readiness-' + state['owner_marker'],
            namespace=NAMESPACE))
    state['experiment_id'] = api_identifier(
        experiment.get('experiment_id', experiment.get('experimentId')))
    write_object(state_dir / 'state.json', state)
    for scenario, account in zip(('default', 'scoped', 'denied'), ACCOUNTS):
        payload = dict(
            display_name='readiness-' + scenario,
            experiment_id=state['experiment_id'],
            pipeline_spec=pipeline_spec,
            max_concurrency='1',
            mode='DISABLE',
            no_catchup=True,
            trigger=dict(periodic_schedule=dict(interval_second='30')))
        if scenario != 'default':
            payload['service_account'] = account
        response = client.post('/apis/v2beta1/recurringruns', payload)
        uid = api_identifier(
            response.get('recurring_run_id', response.get('recurringRunId')))
        record = dict(
            scenario=scenario, schedule_uid=uid, service_account=account)
        state['schedules'].append(record)
        # Preserve IDs immediately so disable remains possible after partial failure.
        write_object(state_dir / 'state.json', state)
        record['schedule_name'] = schedule_identity(context, uid)
        write_object(state_dir / 'state.json', state)
    state['prepared'] = True
    write_object(state_dir / 'state.json', state)
    cases = []
    for record in state['schedules']:
        denied = record['scenario'] == 'denied'
        cases.append(
            dict(
                record,
                expected_outcome='blocked' if denied else 'run_created',
                expected_prediction='policy_rejection'
                if denied else 'no_issue_detected'))
    write_object(state_dir / 'cases.json',
                 dict(namespace=NAMESPACE, cases=cases))


def set_enabled(state_dir, state, client, enabled):
    if enabled and not state.get('prepared'):
        raise FixtureError('fixture_not_prepared')
    records = state.get('schedules')
    if not isinstance(records, list) or not 1 <= len(records) <= 3:
        raise FixtureError('invalid_fixture_schedules')
    ids = [api_identifier(record.get('schedule_uid')) for record in records]
    start = datetime.now(timezone.utc).isoformat()
    failures = False
    for uid in ids:
        try:
            client.post(
                '/apis/v2beta1/recurringruns/' + uid +
                (':enable' if enabled else ':disable'), {})
        except FixtureError:
            failures = True
            if enabled:
                break
    if failures:
        # Enable is not atomic. Best-effort disable all fixtures on a partial failure.
        if enabled:
            for uid in ids:
                try:
                    client.post(
                        '/apis/v2beta1/recurringruns/' + uid + ':disable', {})
                except FixtureError:
                    pass
        raise FixtureError('fixture_activation_change_failed')
    state['enabled'] = enabled
    if enabled:
        state['activation_start'] = start
        (state_dir / 'activation-start.txt').write_text(
            start + '\n', encoding='utf-8')
    write_object(state_dir / 'state.json', state)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--context', required=True)
    parser.add_argument('--allow-test-cluster-mutations', action='store_true')
    parser.add_argument(
        '--phase',
        required=True,
        choices=('rbac', 'prepare', 'enable', 'disable'))
    parser.add_argument('--state-dir', required=True)
    parser.add_argument('--endpoint')
    parser.add_argument('--token-file')
    parser.add_argument('--pipeline-spec')
    args = parser.parse_args()
    try:
        if args.context != CONTEXT or not args.allow_test_cluster_mutations:
            raise FixtureError('explicit_isolated_cluster_consent_required')
        state_dir = Path(args.state_dir)
        state_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
        if args.phase == 'rbac':
            provision_rbac(args.context, state_dir)
        else:
            state = read_object(state_dir / 'state.json')
            verify_state(args.context, state)
            client = FixtureClient(args.endpoint, args.token_file)
            if args.phase == 'prepare':
                prepare(args.context, state_dir, state, client,
                        read_object(args.pipeline_spec))
            else:
                set_enabled(state_dir, state, client, args.phase == 'enable')
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        parser.exit(
            1,
            'Fixture operation failed; inspect the isolated cluster and fixture state before retrying.\n'
        )


if __name__ == '__main__':
    main()

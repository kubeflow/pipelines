# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Mutating, isolated Kind fixture for real API/UI and Kubernetes RBAC
checks."""
import argparse
import http.client
import json
from pathlib import Path
import subprocess
import time
from urllib.parse import urlencode

CONTEXT = 'kind-kfp-custom-roles'
TEAM = 'kfp-roles-team'
OTHER = 'kfp-roles-other'
LIMIT = 4 * 1024 * 1024


def kube(*args, value=None):
    result = subprocess.run(
        ['kubectl', '--context', CONTEXT, '--request-timeout=30s', *args],
        input=None if value is None else json.dumps(value),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=40,
        check=False)
    if result.returncode:
        raise RuntimeError('kubernetes_operation_failed')
    return result.stdout


def rule(group, resources, verbs):
    return dict(apiGroups=[group], resources=resources, verbs=verbs)


def role(namespace, user, rules):
    name = 'custom-role-' + user
    return [
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='Role',
            metadata=dict(name=name, namespace=namespace),
            rules=rules),
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='RoleBinding',
            metadata=dict(name=name, namespace=namespace),
            subjects=[
                dict(
                    kind='User',
                    name=user + '@fixture.invalid',
                    apiGroup='rbac.authorization.k8s.io')
            ],
            roleRef=dict(
                kind='Role', name=name, apiGroup='rbac.authorization.k8s.io'))
    ]


def resources():
    objects = [
        dict(apiVersion='v1', kind='Namespace', metadata=dict(name=n))
        for n in (TEAM, OTHER)
    ]
    for namespace in (TEAM, OTHER):
        objects.append(
            dict(
                apiVersion='v1',
                kind='ServiceAccount',
                metadata=dict(name='pipeline-runner', namespace=namespace)))
        base = [
            rule('pipelines.kubeflow.org', ['experiments', 'runs', 'jobs'],
                 ['create', 'get', 'list'])
        ]
        objects += role(
            namespace, 'author', base +
            [rule('pipelines.kubeflow.org', ['pipelines'], ['create', 'get'])])
        objects += role(namespace, 'no-reader', base)
    objects += role('kubeflow', 'publisher',
                    [rule('pipelines.kubeflow.org', ['pipelines'], ['create'])])
    objects += role(TEAM, 'log-reader',
                    [rule('pipelines.kubeflow.org', ['runs'], ['readLog'])])
    objects += role(TEAM, 'viewer',
                    [rule('kubeflow.org', ['viewers'], ['get'])])
    objects += role(
        TEAM, 'manager',
        [rule('kubeflow.org', ['viewers'], ['get', 'create', 'delete'])])
    # Service infrastructure grants are separate from end-user permissions.
    # These are the narrow operations this standalone fixture requires in its
    # two namespaces; no serviceaccounts/use or user impersonation is granted.
    for namespace in (TEAM, OTHER):
        service = role(namespace, 'api-service', [
            rule('argoproj.io', ['workflows'], ['create', 'get']),
            rule('kubeflow.org', ['scheduledworkflows'], ['create', 'get']),
            rule('', ['pods', 'pods/log', 'configmaps'], ['get']),
        ])
        service[1]['subjects'] = [
            dict(
                kind='ServiceAccount', name='ml-pipeline', namespace='kubeflow')
        ]
        objects += service
    service = role(
        TEAM, 'ui-service',
        [rule('kubeflow.org', ['viewers'], ['get', 'create', 'delete'])])
    service[1]['subjects'] = [
        dict(
            kind='ServiceAccount', name='ml-pipeline-ui', namespace='kubeflow')
    ]
    objects += service
    objects += [
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='ClusterRole',
            metadata=dict(name='custom-role-api-sar'),
            rules=[
                rule('authorization.k8s.io', ['subjectaccessreviews'],
                     ['create'])
            ]),
        dict(
            apiVersion='rbac.authorization.k8s.io/v1',
            kind='ClusterRoleBinding',
            metadata=dict(name='custom-role-api-sar'),
            subjects=[
                dict(
                    kind='ServiceAccount',
                    name='ml-pipeline',
                    namespace='kubeflow')
            ],
            roleRef=dict(
                kind='ClusterRole',
                name='custom-role-api-sar',
                apiGroup='rbac.authorization.k8s.io'))
    ]
    return objects


def request(port, user, method, path, value=None, upload=None):
    headers = {'kubeflow-userid': user + '@fixture.invalid'}
    body = None
    if upload is not None:
        boundary = 'kfp-custom-role-fixture'
        headers['Content-Type'] = 'multipart/form-data; boundary=' + boundary
        body = (
            f'--{boundary}\r\nContent-Disposition: form-data; name="uploadfile"; '
            'filename="pipeline.yaml"\r\nContent-Type: application/yaml\r\n\r\n'
        ).encode()
        body += upload + f'\r\n--{boundary}--\r\n'.encode()
    elif value is not None:
        headers['Content-Type'] = 'application/json'
        body = json.dumps(value).encode()
    connection = http.client.HTTPConnection('127.0.0.1', port, timeout=30)
    try:
        connection.request(method, path, body=body, headers=headers)
        response = connection.getresponse()
        raw = response.read(LIMIT + 1)
        if len(raw) > LIMIT:
            raise RuntimeError('response_limit_exceeded')
        return response.status, raw
    finally:
        connection.close()


class Matrix:

    def __init__(self):
        self.cases = []

    def check(self, name, user, method, path, expected=200, **kwargs):
        status, raw = request(8888, user, method, path, **kwargs)
        self.cases.append(
            dict(
                name=name,
                expected_status=expected,
                status=status,
                passed=status == expected))
        if status != expected:
            raise RuntimeError('unexpected_status_' + name)
        value = json.loads(raw) if raw else {}
        if expected == 400 and '/pipelines/upload' in path:
            message = value.get('error_message', '')
            if 'upload denied: permission to create pipelines.pipelines.kubeflow.org in namespace' not in message:
                self.cases[-1]['passed'] = False
                raise RuntimeError('missing_upload_authorization_diagnostic_' +
                                   name)
        return value

    def ui(self, name, user, method, namespace=TEAM, expected=200):
        path = '/apps/tensorboard?' + urlencode(
            dict(
                namespace=namespace,
                logdir='gs://fixture.invalid/tensorboard',
                image='python:3.12'))
        status, raw = request(3000, user, method, path)
        self.cases.append(
            dict(
                name=name,
                expected_status=expected,
                status=status,
                passed=status == expected))
        if status != expected:
            raise RuntimeError('unexpected_status_' + name)
        if expected == 401:
            verb = {'GET': 'GET', 'POST': 'CREATE', 'DELETE': 'DELETE'}[method]
            diagnostic = f'User is not authorized to {verb} VIEWERS in namespace {namespace}:'
            if diagnostic.encode() not in raw:
                self.cases[-1]['passed'] = False
                raise RuntimeError('missing_viewer_authorization_diagnostic_' +
                                   name)
        return raw

    def run(self):
        kube(
            'create',
            '-f',
            '-',
            value=dict(apiVersion='v1', kind='List', items=resources()))
        pipeline = Path(
            'test_data/sdk_compiled_pipelines/valid/hello_world.yaml'
        ).read_bytes()
        # Omission requests shared scope; an explicit namespace requests private scope.
        private = self.check(
            'private_upload',
            'author',
            'POST',
            '/apis/v2beta1/pipelines/upload?' +
            urlencode(dict(name='private', namespace=TEAM)),
            upload=pipeline)
        self.check(
            'foreign_upload_denied',
            'log-reader',
            'POST',
            '/apis/v2beta1/pipelines/upload?' +
            urlencode(dict(name='denied', namespace=OTHER)),
            expected=400,
            upload=pipeline)
        self.check(
            'omitted_namespace_shared_denied',
            'author',
            'POST',
            '/apis/v2beta1/pipelines/upload?name=denied-shared',
            expected=400,
            upload=pipeline)
        self.check(
            'empty_namespace_shared_denied',
            'author',
            'POST',
            '/apis/v2beta1/pipelines/upload?name=denied-empty&namespace=',
            expected=400,
            upload=pipeline)
        shared = self.check(
            'designated_shared_publisher',
            'publisher',
            'POST',
            '/apis/v2beta1/pipelines/upload?name=shared',
            upload=pipeline)
        if private.get('namespace') != TEAM or shared.get('namespace', ''):
            raise RuntimeError('uploaded_pipeline_scope_mismatch')
        private_id, shared_id = private['pipeline_id'], shared['pipeline_id']
        version = self.check(
            'private_version_parent_scope',
            'author',
            'POST',
            '/apis/v2beta1/pipelines/upload_version?' +
            urlencode(dict(name='version', pipelineid=private_id)),
            upload=pipeline)
        self.check(
            'private_version_no_create_denied',
            'log-reader',
            'POST',
            '/apis/v2beta1/pipelines/upload_version?' +
            urlencode(dict(name='denied-version', pipelineid=private_id)),
            expected=400,
            upload=pipeline)
        self.check(
            'shared_version_author_denied',
            'author',
            'POST',
            '/apis/v2beta1/pipelines/upload_version?' +
            urlencode(dict(name='denied-shared-version', pipelineid=shared_id)),
            expected=400,
            upload=pipeline)
        self.check(
            'shared_version_publisher',
            'publisher',
            'POST',
            '/apis/v2beta1/pipelines/upload_version?' +
            urlencode(dict(name='shared-version', pipelineid=shared_id)),
            upload=pipeline)
        experiments = {}
        for namespace in (TEAM, OTHER):
            experiments[namespace] = self.check(
                'experiment_' + namespace,
                'author',
                'POST',
                '/apis/v2beta1/experiments',
                value=dict(
                    display_name='roles-' + namespace,
                    namespace=namespace))['experiment_id']
        reference = dict(
            pipeline_id=private_id,
            pipeline_version_id=version['pipeline_version_id'])

        def body(namespace, ref=reference):
            return dict(
                display_name='role-test',
                experiment_id=experiments[namespace],
                pipeline_version_reference=ref)

        own_run = self.check(
            'private_reference_same_namespace',
            'author',
            'POST',
            '/apis/v2beta1/runs',
            value=body(TEAM))
        self.check(
            'private_reference_without_get_denied',
            'no-reader',
            'POST',
            '/apis/v2beta1/runs',
            value=body(TEAM),
            expected=403)
        self.check(
            'private_reference_cross_namespace_denied',
            'author',
            'POST',
            '/apis/v2beta1/runs',
            value=body(OTHER),
            expected=403)
        other_run = self.check(
            'shared_reference_other_namespace',
            'author',
            'POST',
            '/apis/v2beta1/runs',
            value=body(OTHER, dict(pipeline_id=shared_id)))
        for namespace, actor, expected, name in (
            (TEAM, 'author', 200, 'private_recurring_reference'),
            (TEAM, 'no-reader', 403, 'private_recurring_reference_without_get'),
            (OTHER, 'author', 403, 'private_recurring_cross_namespace')):
            schedule = body(namespace)
            schedule.update(
                max_concurrency='1',
                mode='DISABLE',
                no_catchup=True,
                trigger=dict(periodic_schedule=dict(interval_second='60')))
            self.check(
                name,
                actor,
                'POST',
                '/apis/v2beta1/recurringruns',
                expected=expected,
                value=schedule)
        run_id = own_run['run_id']
        # Controlled real Pod tests the log handler and run ownership, without
        # claiming this helper executes an Argo workload or tests archived logs.
        pod = dict(
            apiVersion='v1',
            kind='Pod',
            metadata=dict(
                name='role-log',
                namespace=TEAM,
                labels={'pipeline/runid': run_id}),
            spec=dict(
                restartPolicy='Never',
                containers=[
                    dict(
                        name='main',
                        image='python:3.12',
                        imagePullPolicy='IfNotPresent',
                        command=[
                            'python', '-c',
                            "print('custom-role-log-evidence', flush=True); import time; time.sleep(600)"
                        ])
                ]))
        kube('create', '-f', '-', value=pod)
        deadline = time.monotonic() + 120
        while True:
            pod_state = json.loads(
                kube('-n', TEAM, 'get', 'pod/role-log', '-o', 'json'))
            if any(
                    c.get('type') == 'Ready' and c.get('status') == 'True'
                    for c in pod_state.get('status', {}).get('conditions', [])):
                break
            if time.monotonic() >= deadline:
                raise RuntimeError('log_pod_not_ready')
            time.sleep(2)
        log_path = f'/apis/v1alpha1/runs/{run_id}/nodes/role-log/log'
        status, raw = request(8888, 'log-reader', 'GET', log_path)
        passed = status == 200 and b'custom-role-log-evidence' in raw
        self.cases.append(
            dict(
                name='readLog_allowed_actual_pod', status=status,
                passed=passed))
        if not passed:
            raise RuntimeError('log_positive_control_failed')
        self.check(
            'runs_get_is_not_readLog', 'author', 'GET', log_path, expected=403)
        self.check(
            'foreign_log_reader_denied',
            'publisher',
            'GET',
            log_path,
            expected=403)
        self.check(
            'readLog_cannot_cross_namespace',
            'log-reader',
            'GET',
            f"/apis/v1alpha1/runs/{other_run['run_id']}/nodes/role-log/log",
            expected=403)
        self.ui('manager_create', 'manager', 'POST')
        viewers = json.loads(kube('-n', TEAM, 'get', 'viewers', '-o',
                                  'json'))['items']
        if len(viewers) != 1:
            raise RuntimeError('viewer_create_evidence_failed')
        uid = viewers[0]['metadata']['uid']
        value = json.loads(self.ui('viewer_get', 'viewer', 'GET'))
        if not value.get('proxyPath'):
            raise RuntimeError('viewer_get_evidence_failed')
        self.ui('viewer_create_denied', 'viewer', 'POST', expected=401)
        self.ui('viewer_delete_denied', 'viewer', 'DELETE', expected=401)
        self.ui(
            'foreign_viewer_denied',
            'viewer',
            'GET',
            namespace=OTHER,
            expected=401)
        viewers = json.loads(kube('-n', TEAM, 'get', 'viewers', '-o',
                                  'json'))['items']
        if len(viewers) != 1 or viewers[0]['metadata']['uid'] != uid:
            raise RuntimeError('denied_request_mutated_viewer')
        self.ui('manager_delete', 'manager', 'DELETE')
        if json.loads(kube('-n', TEAM, 'get', 'viewers', '-o',
                           'json'))['items']:
            raise RuntimeError('viewer_delete_evidence_failed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        '--allow-test-cluster-mutations', action='store_true', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if kube('config', 'current-context').strip() != CONTEXT:
        raise SystemExit('This fixture requires its isolated Kind context.')
    matrix = Matrix()
    report = dict(
        scope='live_custom_role_api_ui_authorization',
        outcome='failed',
        revision=subprocess.check_output(['git', 'rev-parse', 'HEAD'],
                                         text=True).strip(),
        cases=matrix.cases,
        ingress_authentication_validated=False,
        tensorflow_execution_validated=False,
        argo_execution_validated=False)
    try:
        matrix.run()
        report['outcome'] = 'passed'
    finally:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
    print('Custom-role live authorization matrix passed.')


if __name__ == '__main__':
    main()

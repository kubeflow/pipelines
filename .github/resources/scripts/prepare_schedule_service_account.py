#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Provision the isolated multi-user recurring-run authorization fixture."""

import argparse
import json
import subprocess

ACCOUNT = 'kfp-schedule-test'


def fixture(namespace, controller_namespace, controller_account):
    """Namespace-scoped grants; no role grants wildcard resources or verbs."""

    def resource(kind, name, **fields):
        return dict(
            apiVersion='v1'
            if kind in ('ServiceAccount', 'Secret')
            else 'rbac.authorization.k8s.io/v1',
            kind=kind,
            metadata={'name': name, 'namespace': namespace},
            **fields,
        )

    def subject(name, ns=namespace):
        return {'kind': 'ServiceAccount', 'name': name, 'namespace': ns}

    def rule(group, resources, verbs, names=None):
        result = {'apiGroups': [group], 'resources': resources, 'verbs': verbs}
        if names:
            result['resourceNames'] = names
        return result

    def grant(name, rules, subjects):
        return [
            resource('Role', name, rules=rules),
            resource(
                'RoleBinding',
                name,
                subjects=subjects,
                roleRef={
                    'apiGroup': 'rbac.authorization.k8s.io',
                    'kind': 'Role',
                    'name': name,
                },
            ),
        ]

    account = resource('ServiceAccount', ACCOUNT)
    secret = resource(
        'Secret',
        ACCOUNT + '.service-account-token',
        type='kubernetes.io/service-account-token',
    )
    secret['metadata']['annotations'] = {
        'kubernetes.io/service-account.name': ACCOUNT
    }
    items = [account, secret]
    items += grant(
        'kfp-schedule-test-use',
        [rule('', ['serviceaccounts'], ['use'], [ACCOUNT])],
        [
            subject('default-editor'),
            subject(controller_account, controller_namespace),
        ],
    )
    items += grant(
        'kfp-schedule-test-token',
        [rule('', ['serviceaccounts/token'], ['create'], [ACCOUNT])],
        [subject('ml-pipeline-driver-agent-executor-plugin')],
    )
    # The fixture needs API metadata, Argo agent tasks, and executor results.
    items += grant(
        'kfp-schedule-test-runtime',
        [
            rule('pipelines.kubeflow.org', ['runs'], ['get', 'list', 'update']),
            rule(
                'pipelines.kubeflow.org',
                ['artifacts'],
                ['get', 'list', 'create'],
            ),
            rule('argoproj.io', ['workflowtasksets'], ['watch']),
            rule('argoproj.io', ['workflowtasksets/status'], ['patch']),
            rule('argoproj.io', ['workflowtaskresults'], ['create', 'patch']),
            rule('', ['secrets'], ['get'], ['mlpipeline-minio-artifact']),
            rule('', ['configmaps'], ['get'], ['kfp-launcher']),
        ],
        [subject(ACCOUNT)],
    )
    return {'apiVersion': 'v1', 'kind': 'List', 'items': items}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', required=True)
    parser.add_argument('--system-namespace', default='kubeflow')
    args = parser.parse_args()
    controller = subprocess.check_output(
        [
            'kubectl',
            '-n',
            args.system_namespace,
            'get',
            'deployment',
            'ml-pipeline-scheduledworkflow',
            '-o',
            'jsonpath={.spec.template.spec.serviceAccountName}',
        ],
        text=True,
    ).strip()
    if not controller:
        raise SystemExit(
            'Scheduled workflow controller has no explicit ServiceAccount'
        )
    subprocess.run(
        ['kubectl', 'apply', '-f', '-'],
        input=json.dumps(
            fixture(args.namespace, args.system_namespace, controller)
        ),
        text=True,
        check=True,
    )
    # Keep enforcement enabled; only explicitly authorize this fixture account.
    subprocess.run(
        [
            'kubectl',
            '-n',
            args.system_namespace,
            'set',
            'env',
            'deployment/ml-pipeline',
            'ALLOWEDSERVICEACCOUNTS=' + ACCOUNT,
            'KFP_SECURITY_SERVICE_ACCOUNT_MODE=enforce',
        ],
        check=True,
    )
    subprocess.run(
        [
            'kubectl',
            '-n',
            args.system_namespace,
            'rollout',
            'status',
            'deployment/ml-pipeline',
            '--timeout=180s',
        ],
        check=True,
    )


if __name__ == '__main__':
    main()

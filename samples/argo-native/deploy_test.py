# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

import importlib.util
from pathlib import Path

import yaml


def test_isolated_client_auth_and_persistent_services():
    spec = importlib.util.spec_from_file_location(
        'deployment', Path(__file__).with_name('deploy.py')
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    upstream = []
    for name in (
        'argo-server',
        'workflow-controller',
        'minio',
        'postgres',
        'httpbin',
    ):
        container = {
            'name': name,
            'image': 'quay.io/argoproj/example:latest',
            'lifecycle': {
                'postStart': {
                    'exec': {'command': ['mkdir', '-p', '/data/my-bucket']}
                }
            },
        }
        upstream.append(
            {
                'apiVersion': 'apps/v1',
                'kind': 'Deployment',
                'metadata': {'name': name, 'namespace': 'argo'},
                'spec': {'template': {'spec': {'containers': [container]}}},
            }
        )
    result = module.resources(upstream, 'isolated', 'ui:test')
    deployments = {
        item['metadata']['name']: item
        for item in result
        if item['kind'] == 'Deployment'
    }
    assert 'httpbin' not in deployments
    args = deployments['argo-server']['spec']['template']['spec']['containers'][
        0
    ]['args']
    assert args == [
        'server',
        '--auth-mode',
        'client',
        '--namespaced',
        '--managed-namespace',
        'isolated',
    ]
    assert (
        len(
            [item for item in result if item['kind'] == 'PersistentVolumeClaim']
        )
        == 2
    )
    for name in ('minio', 'postgres'):
        assert (
            deployments[name]['spec']['template']['spec']['volumes'][0][
                'persistentVolumeClaim'
            ]['claimName']
            == name + '-data'
        )
    proxy = deployments['kfp-argo-ui']['spec']['template']['spec']
    assert proxy['serviceAccountName'] == 'argo-ui-proxy'
    assert not any(
        item['kind'] == 'RoleBinding'
        and any(
            subject['name'] == 'argo-ui-proxy' for subject in item['subjects']
        )
        for item in result
    )
    assert all(
        item['metadata'].get('namespace', 'isolated') == 'isolated'
        for item in result
    )


def test_reader_cannot_mutate_and_no_cluster_admin_browser_binding():
    resources = list(
        yaml.safe_load_all(
            Path(__file__).with_name('native-rbac.yaml').read_text()
        )
    )
    assert not any(item['kind'].startswith('ClusterRole') for item in resources)
    read = next(
        item
        for item in resources
        if item['kind'] == 'Role' and item['metadata']['name'] == 'argo-ui-read'
    )
    assert all(
        set(rule['verbs']) <= {'get', 'list', 'watch'} for rule in read['rules']
    )
    assert (
        'watch'
        in next(rule for rule in read['rules'] if 'pods' in rule['resources'])[
            'verbs'
        ]
    )
    edit = next(
        item
        for item in resources
        if item['kind'] == 'Role' and item['metadata']['name'] == 'argo-ui-edit'
    )
    templates = next(
        rule
        for rule in edit['rules']
        if rule['resources'] == ['workflowtemplates']
    )
    assert (
        'update' not in templates['verbs'] and 'patch' not in templates['verbs']
    )

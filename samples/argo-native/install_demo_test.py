# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import io
import runpy
from pathlib import Path
from unittest.mock import patch

import yaml


def test_controller_can_update_only_demo_cache():
    resources = list(
        yaml.safe_load_all(Path(__file__).with_name('rbac.yaml').read_text())
    )
    role = next(
        (
            r
            for r in resources
            if r['kind'] == 'Role'
            and r['metadata']['name'] == 'argo-demo-cache'
        ),
        None,
    )
    assert role is not None
    assert role['rules'] == [
        {
            'apiGroups': [''],
            'resources': ['configmaps'],
            'resourceNames': ['argo-demo-cache'],
            'verbs': ['get', 'update'],
        }
    ]
    binding = next(
        r
        for r in resources
        if r['kind'] == 'RoleBinding'
        and r['metadata']['name'] == 'argo-demo-cache'
    )
    assert binding['roleRef']['name'] == role['metadata']['name']
    assert binding['subjects'] == [{'kind': 'ServiceAccount', 'name': 'argo'}]


def test_demo_replaces_retired_minio_image(capsys):
    deployment = {
        'kind': 'Deployment',
        'spec': {
            'template': {
                'spec': {
                    'containers': [
                        {
                            'image': 'quay.io/minio/minio:retired',
                            'command': ['minio', 'server', '/data'],
                            'env': [
                                {'name': 'MINIO_ACCESS_KEY', 'value': 'admin'},
                                {
                                    'name': 'MINIO_SECRET_KEY',
                                    'value': 'password',
                                },
                            ],
                            'lifecycle': {
                                'postStart': {
                                    'exec': {
                                        'command': [
                                            'mkdir',
                                            '-p',
                                            '/data/my-bucket',
                                        ]
                                    }
                                }
                            },
                        }
                    ]
                }
            }
        },
    }
    with patch(
        'urllib.request.urlopen',
        return_value=io.BytesIO(yaml.safe_dump(deployment).encode()),
    ):
        runpy.run_path(str(Path(__file__).with_name('install-demo.py')))
    result = next(yaml.safe_load_all(capsys.readouterr().out))
    container = result['spec']['template']['spec']['containers'][0]
    assert container['image'].startswith('docker.io/pgsty/silo:')
    assert '@sha256:' in container['image']
    assert container['command'][0] == 'silo'
    assert [entry['name'] for entry in container['env']] == [
        'MINIO_ROOT_USER',
        'MINIO_ROOT_PASSWORD',
    ]
    assert (
        container['lifecycle']['postStart']['exec']['command'][-1]
        == '/data/argo-artifacts'
    )

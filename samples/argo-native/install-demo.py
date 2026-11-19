# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Emit a disposable, single-user Argo installation.

Never use in production.
"""

import urllib.request

import yaml

VERSION = 'v3.7.1'
url = f'https://raw.githubusercontent.com/argoproj/argo-workflows/{VERSION}/manifests/quick-start-minimal.yaml'
with urllib.request.urlopen(url) as response:
    resources = list(yaml.safe_load_all(response.read()))
for resource in resources:
    if resource is None:
        continue
    if resource['kind'] == 'Deployment':
        for container in resource['spec']['template']['spec']['containers']:
            image = container['image']
            if image.startswith('quay.io/argoproj/'):
                container['image'] = image.replace(':latest', f':{VERSION}')
            if image.startswith('quay.io/minio/'):
                # Use the pinned replacement from Argo's current quick-start;
                # the old public MinIO image is no longer available.
                container['image'] = (
                    'docker.io/pgsty/silo:RELEASE.2026-09-16T00-00-00Z'
                    '@sha256:635197cb9f36d01bee221d34d1c7d7960f6a95c48b0b6c01d99cd13bdae51a46'
                )
                container['command'][0] = 'silo'
                for variable in container['env']:
                    variable['name'] = {
                        'MINIO_ACCESS_KEY': 'MINIO_ROOT_USER',
                        'MINIO_SECRET_KEY': 'MINIO_ROOT_PASSWORD',
                    }.get(variable['name'], variable['name'])
                container['lifecycle']['postStart']['exec']['command'][-1] = (
                    '/data/argo-artifacts'
                )
    if resource['kind'] == 'ConfigMap':
        for key, value in resource.get('data', {}).items():
            resource['data'][key] = value.replace(
                'bucket: my-bucket', 'bucket: argo-artifacts'
            )
        if resource['metadata']['name'] == 'workflow-controller-configmap':
            resource['data']['executor'] = yaml.safe_dump(
                {'image': f'quay.io/argoproj/argoexec:{VERSION}'}
            )
    print(yaml.safe_dump(resource), end='---\n')

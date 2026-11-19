# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Emit the isolated local prototype: Argo, persistent archive/artifacts and static UI.

Uses upstream quick-start development credentials. Bind port-forwards to loopback;
this is not a production authentication or multi-tenant installation.
"""

import argparse
import copy
import urllib.request
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
VERSION = (ROOT / 'third_party/argo/VERSION').read_text().strip()


def resources(upstream, namespace, ui_image):
    result = [
        {
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'metadata': {'name': namespace},
        }
    ]
    for original in upstream:
        if not original:
            continue
        resource = copy.deepcopy(original)
        metadata = resource.get('metadata', {})
        if resource['kind'] == 'Namespace':
            continue
        if metadata.get('namespace') == 'argo':
            metadata['namespace'] = namespace
        for subject in resource.get('subjects', []):
            if subject.get('namespace') == 'argo':
                subject['namespace'] = namespace
        if resource['kind'] == 'ConfigMap':
            for key, value in resource.get('data', {}).items():
                resource['data'][key] = value.replace(
                    'bucket: my-bucket', 'bucket: argo-artifacts'
                )
            if metadata['name'] == 'workflow-controller-configmap':
                resource['data']['executor'] = yaml.safe_dump(
                    {'image': f'quay.io/argoproj/argoexec:{VERSION}'}
                )
                # Retention is explicit in the test, not racing automatic deletion.
                resource['data'].pop('retentionPolicy', None)
        if resource['kind'] == 'Deployment':
            pod = resource['spec']['template']['spec']
            for container in pod['containers']:
                if container['image'].startswith('quay.io/argoproj/'):
                    container['image'] = container['image'].replace(
                        ':latest', ':' + VERSION
                    )
                if metadata['name'] == 'argo-server':
                    container['args'] = [
                        'server',
                        '--auth-mode',
                        'client',
                        '--namespaced',
                        '--managed-namespace',
                        namespace,
                    ]
                elif metadata['name'] == 'workflow-controller':
                    container['args'] = [
                        '--namespaced',
                        '--managed-namespace',
                        namespace,
                    ]
                    container.setdefault('env', []).append(
                        {'name': 'ALWAYS_OFFLOAD_NODE_STATUS', 'value': 'true'}
                    )
                elif metadata['name'] in ('minio', 'postgres'):
                    mount = (
                        '/data'
                        if metadata['name'] == 'minio'
                        else '/var/lib/pgsql/data'
                    )
                    container.setdefault('volumeMounts', []).append(
                        {'name': 'data', 'mountPath': mount}
                    )
                    pod['volumes'] = [
                        {
                            'name': 'data',
                            'persistentVolumeClaim': {
                                'claimName': metadata['name'] + '-data'
                            },
                        }
                    ]
                    resource['spec']['strategy'] = {'type': 'Recreate'}
                    if metadata['name'] == 'minio':
                        container['lifecycle']['postStart']['exec']['command'][
                            -1
                        ] = '/data/argo-artifacts'
                    else:
                        pod['securityContext'] = {'fsGroup': 26}
        result.append(resource)
    for name in ('minio-data', 'postgres-data'):
        result.append(
            {
                'apiVersion': 'v1',
                'kind': 'PersistentVolumeClaim',
                'metadata': {'name': name, 'namespace': namespace},
                'spec': {
                    'accessModes': ['ReadWriteOnce'],
                    'resources': {'requests': {'storage': '2Gi'}},
                },
            }
        )
    for resource in yaml.safe_load_all(
        Path(__file__).with_name('native-rbac.yaml').read_text()
    ):
        if resource:
            resource['metadata']['namespace'] = namespace
            result.append(resource)
    nginx = Path(__file__).with_name('nginx.conf').read_text()
    result.append(
        {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'metadata': {'name': 'kfp-argo-ui-nginx', 'namespace': namespace},
            'data': {'default.conf': nginx},
        }
    )
    result.append(
        {
            'apiVersion': 'apps/v1',
            'kind': 'Deployment',
            'metadata': {'name': 'kfp-argo-ui', 'namespace': namespace},
            'spec': {
                'selector': {'matchLabels': {'app': 'kfp-argo-ui'}},
                'template': {
                    'metadata': {'labels': {'app': 'kfp-argo-ui'}},
                    'spec': {
                        'serviceAccountName': 'argo-ui-proxy',
                        'containers': [
                            {
                                'name': 'ui',
                                'image': ui_image,
                                'imagePullPolicy': 'IfNotPresent',
                                'ports': [{'containerPort': 8080}],
                                'readinessProbe': {
                                    'httpGet': {
                                        'path': '/argo.html',
                                        'port': 8080,
                                    }
                                },
                                'volumeMounts': [
                                    {
                                        'name': 'nginx',
                                        'mountPath': '/etc/nginx/conf.d',
                                        'readOnly': True,
                                    }
                                ],
                            }
                        ],
                        'volumes': [
                            {
                                'name': 'nginx',
                                'configMap': {'name': 'kfp-argo-ui-nginx'},
                            }
                        ],
                    },
                },
            },
        }
    )
    result.append(
        {
            'apiVersion': 'v1',
            'kind': 'Service',
            'metadata': {'name': 'kfp-argo-ui', 'namespace': namespace},
            'spec': {
                'selector': {'app': 'kfp-argo-ui'},
                'ports': [{'port': 80, 'targetPort': 8080}],
            },
        }
    )
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', default='kfp-argo')
    parser.add_argument('--ui-image', default='kfp-argo-ui:dev')
    args = parser.parse_args()
    url = f'https://raw.githubusercontent.com/argoproj/argo-workflows/{VERSION}/manifests/quick-start-postgres.yaml'
    with urllib.request.urlopen(url, timeout=60) as response:
        upstream = list(yaml.safe_load_all(response.read()))
    print(
        yaml.safe_dump_all(
            resources(upstream, args.namespace, args.ui_image), sort_keys=False
        )
    )


if __name__ == '__main__':
    main()

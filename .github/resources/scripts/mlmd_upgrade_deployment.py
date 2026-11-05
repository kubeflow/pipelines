# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Select the target API and its DNS configuration for the upgrade fixture."""

import argparse
import sys

import yaml


def select_resources(rendered, image, mode):
    resources = [resource for resource in rendered if resource]

    def require(kind, name):
        matches = [
            resource
            for resource in resources
            if resource['kind'] == kind
            and resource['metadata']['name'] == name
            and resource['metadata'].get('namespace') == 'kubeflow'
        ]
        if len(matches) != 1:
            raise ValueError(f'expected one kubeflow {kind} named {name}')
        return matches[0]

    dns = require('ConfigMap', 'dns-config')
    if not dns.get('data', {}).get('dbHost'):
        raise ValueError('kubeflow/dns-config needs a nonempty dbHost')
    deployment = require('Deployment', 'ml-pipeline')
    deployment['spec']['replicas'] = 0
    pod = deployment['spec']['template']['spec']
    for container in pod.get('initContainers', []) + pod['containers']:
        for entry in container.get('env', []):
            ref = entry.get('valueFrom', {}).get('configMapKeyRef', {})
            if ref.get('name') == 'dns-config' and not ref.get('optional'):
                if ref['key'] not in dns['data']:
                    raise ValueError(
                        f'kubeflow/dns-config needs {ref["key"]} for {entry["name"]}'
                    )
    api = next(
        container
        for container in pod['containers']
        if container['name'] == 'ml-pipeline-api-server'
    )
    api['image'] = image
    if mode == 'startup-only':
        api['env'].extend(
            {'name': key, 'value': value}
            for key, value in {
                'METADATA_GRPC_SERVICE_HOST': 'metadata-grpc-service',
                'METADATA_GRPC_SERVICE_PORT': '8080',
                'MLMD_MIGRATION_AVAILABLE_DISK_BYTES': '1073741824',
                'MLMD_MIGRATION_DATABASE_BYTES': '1048576',
            }.items()
        )
    return dns, deployment


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('image')
    parser.add_argument('mode', choices=('cli-first', 'startup-only'))
    args = parser.parse_args()
    try:
        resources = select_resources(
            yaml.safe_load_all(sys.stdin), args.image, args.mode
        )
    except ValueError as error:
        parser.exit(1, f'{error}\n')
    yaml.safe_dump_all(resources, sys.stdout)


if __name__ == '__main__':
    main()

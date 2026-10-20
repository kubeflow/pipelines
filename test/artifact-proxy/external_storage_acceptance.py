# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Full-cluster acceptance of external storage and rotating brokered
credentials.

Run only against a disposable multi-user installation. The broker
exchanges real projected Kubernetes ServiceAccount tokens for expiring
SeaweedFS STS sessions.
"""

import argparse
import base64
import copy
import hashlib
import json
import re
import secrets
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime
from pathlib import Path

import boto3
from artifact_authorization_acceptance import BUCKET as MANAGED_BUCKET
from artifact_authorization_acceptance import (
    CONTROLLER,
    RECEIVER,
    Acceptance,
    apply,
    decode_secret,
    get,
    kubectl,
    require,
    wait_for,
)
from botocore.config import Config

APPROVALS = 'kfp-artifact-storage-config'
POLICIES_KEY = 'adoption-policies.json'
SOURCE_LABEL = 'pipelines.kubeflow.org/artifact-storage-policy-source'
CLASS_LABEL = 'pipelines.kubeflow.org/artifact-storage-class'
BUCKET = 'external.artifacts'
CREDENTIAL_DIRECTORY = '/var/run/kfp-artifacts'
PYTHON_IMAGE = 'python:3.12-slim'
STORAGE_IMAGE = 'ghcr.io/chrislusf/seaweedfs:4.48'

CONTROLLER_LOG_LIMIT = 65536
PENDING_REASON_LIMIT = 10
READINESS_STAGES = {
    'resource-deleting',
    'deployment-template',
    'deployment-rollout',
    'network-policy',
    'service-type',
    'service-spec',
    'pod-identity',
    'credential-wiring',
}
POLICY_PENDING_REASONS = {
    'Duplicate external storage configuration key',
    'Enrollment requires a UTC timestamp ending in Z',
    'Storage policy sources must be in the installation namespace',
    'Storage policy source is missing or lacks its discovery label',
    'Storage configuration must come from the installation namespace',
    'Invalid external storage adoption policy document',
    'Invalid external storage namespace selector',
    'Invalid or duplicate external storage adoption policy',
    'Overlapping external storage approvals or adoption policies',
    'Matching storage policy excludes this namespace from enrollment',
    'Storage policy needs a current UID-bound identity and trusted receiver template',
    "verify the installation administrator's storage approval, adoption policy, protected sources, and namespace UID",
    'the existing artifact receiver resources belong to another controller; retain that owner and configure a compatible managed receiver',
}
# Structural field names only. Unknown/custom map keys cannot reach artifacts.
READINESS_PATH_FIELDS = set(
    """
metadata spec template containers initContainers volumes name image imagePullPolicy
labels annotations env envFrom command args value valueFrom workingDir lifecycle
postStart preStop exec httpGet tcpSocket restartPolicy securityContext runAsUser
runAsGroup runAsNonRoot fsGroup privileged readOnlyRootFilesystem procMount
volumeMounts volumeDevices mountPath readOnly subPath subPathExpr mountPropagation
emptyDir projected sources secret configMap downwardAPI serviceAccountToken items
key path defaultMode optional expirationSeconds audience fieldRef fieldPath
apiVersion resourceFieldRef containerName resource divisor secretKeyRef configMapKeyRef
secretRef configMapRef prefix serviceAccountName automountServiceAccountToken
hostNetwork hostPID hostIPC ports port protocol targetPort selector matchLabels
namespaceSelector podSelector policyTypes ingress egress from to type externalName
readinessProbe livenessProbe startupProbe periodSeconds timeoutSeconds replicas

Deployment Service NetworkPolicy status generation observedGeneration updatedReplicas availableReplicas readyReplicas unavailableReplicas deletionTimestamp
""".split()
)


def safe_readiness_path(path):
    if not isinstance(path, str) or not path or len(path) > 192:
        return '<redacted-path>'
    if path.split('.')[0] not in {'Deployment', 'Service', 'NetworkPolicy'}:
        return '<redacted-path>'
    for segment in path.split('.'):
        match = re.fullmatch(
            r'([A-Za-z][A-Za-z0-9]*)(?:\[(?:[0-9]{1,3}|key|\*)\])*', segment
        )
        if not match or match[1] not in READINESS_PATH_FIELDS:
            return '<redacted-path>'
    return path


def receiver_ownership_shape(resource, namespace_uid):
    """Record ownership compatibility without names, UIDs or annotations."""
    owners = (resource or {}).get('metadata', {}).get('ownerReferences', [])
    known_kinds = {
        'Namespace',
        'Profile',
        'Deployment',
        'ReplicaSet',
        'StatefulSet',
        'DaemonSet',
        'Job',
        'CronJob',
    }
    return {
        'exists': resource is not None,
        'ownerCount': len(owners),
        'owners': [
            {
                'kind': owner.get('kind')
                if owner.get('kind') in known_kinds
                else 'Other',
                'matchesNamespace': bool(namespace_uid)
                and owner.get('kind') == 'Namespace'
                and owner.get('uid') == namespace_uid,
                'controller': owner.get('controller') is True,
            }
            for owner in owners[:5]
        ],
    }


def controller_pending_reasons(output, namespaces):
    """Keep only bounded, authored readiness diagnostics for fixture
    tenants."""
    result = {namespace: [] for namespace in namespaces}
    for line in output[-CONTROLLER_LOG_LIMIT:].splitlines():
        for namespace in namespaces:
            entry = None
            prefix = (
                'External artifact storage authorization for '
                + namespace
                + ' is pending: '
            )
            if line.startswith(prefix):
                reason = line[len(prefix) :]
                if (
                    reason.endswith('.')
                    and reason[:-1] in POLICY_PENDING_REASONS
                ):
                    entry = {'reason': reason[:-1]}
            prefix = (
                'External artifact storage readiness for '
                + namespace
                + ' is pending: '
            )
            if line.startswith(prefix):
                try:
                    value = json.loads(line[len(prefix) :])
                except (ValueError, TypeError, RecursionError):
                    continue
                if (
                    isinstance(value, dict)
                    and isinstance(value.get('stage'), str)
                    and value['stage'] in READINESS_STAGES
                    and isinstance(value.get('paths'), list)
                ):
                    entry = {
                        'stage': value['stage'],
                        'paths': [
                            safe_readiness_path(path)
                            for path in value['paths'][:5]
                        ],
                    }
            if entry is not None and (
                not result[namespace] or result[namespace][-1] != entry
            ):
                result[namespace].append(entry)
                result[namespace] = result[namespace][-PENDING_REASON_LIMIT:]
    return result


LIST_PROBE = r"""
const fs = require('fs');
const prefix = JSON.parse(fs.readFileSync(0, 'utf8')).prefix;
(async () => {
  const {createExternalArtifactClient} = await import('./dist/helpers/external-artifact-storage.js');
  const config = JSON.parse(process.env.KFP_ARTIFACT_EXTERNAL_STORAGE_CONFIG);
  const client = await createExternalArtifactClient(config);
  const objects = [];
  try {
    for await (const object of client.listObjectsV2(config.storage.bucket, prefix, true)) {
      objects.push(object.name);
    }
    console.log(JSON.stringify({objects}));
  } catch (error) { console.log(JSON.stringify({denied: true, code: error.code})); }
})().catch(() => process.exit(1));
"""


class ExternalAcceptance(Acceptance):
    def __init__(self, args):
        super().__init__(args)
        self.store_namespace = self.run_id + '-storage'
        self.approval_snapshot = None
        self.approvals_changed = False
        self.binding_source = self.run_id + '-bindings'
        self.template_source = self.run_id + '-template'
        self.source_names = []
        self.trusted_bindings = {}
        self.receiver_template = None
        self.bundled_history = {}
        self.gateway_created = False
        self.broker_code = (
            Path(__file__).with_name('external_storage_broker.py').read_text()
        )
        self.prefixes = {
            name: 'model/customer-' + suffix + '-training/'
            for name, suffix in zip(self.tenants, ['17', '23'])
        }
        self.external_endpoint = (
            'http://storage.' + self.store_namespace + '.svc:8333'
        )

    def forward_external_storage(self):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        process = subprocess.Popen(
            [
                'kubectl',
                '-n',
                self.store_namespace,
                'port-forward',
                'service/storage',
                '--address',
                '127.0.0.1',
                str(port) + ':8333',
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        self.forwarders.append(process)

        def connected():
            require(
                process.poll() is None, 'External storage port-forward exited'
            )
            try:
                with socket.create_connection(('127.0.0.1', port), timeout=1):
                    return True
            except OSError:
                return False

        wait_for('external storage port-forward', connected, 30)
        return 'http://127.0.0.1:' + str(port)

    def prepare_external_store(self):
        discovery = json.loads(
            kubectl('get', '--raw', '/.well-known/openid-configuration').stdout
        )
        jwks = json.loads(kubectl('get', '--raw', '/openid/v1/jwks').stdout)
        issuer = discovery['issuer']
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'Namespace',
                'metadata': {'name': self.store_namespace},
            }
        )
        self.gateway_created = True
        policies, roles = [], []
        for namespace in self.tenants:
            prefix = self.prefixes[namespace]
            policies.append(
                {
                    'name': namespace,
                    'document': {
                        'Version': '2012-10-17',
                        'Statement': [
                            {
                                'Effect': 'Allow',
                                'Action': ['s3:GetObject'],
                                'Resource': [
                                    'arn:aws:s3:::'
                                    + BUCKET
                                    + '/'
                                    + prefix
                                    + '*'
                                ],
                            },
                            {
                                'Effect': 'Allow',
                                'Action': ['s3:ListBucket'],
                                'Resource': ['arn:aws:s3:::' + BUCKET],
                                'Condition': {
                                    'StringLike': {'s3:prefix': prefix + '*'}
                                },
                            },
                        ],
                    },
                }
            )
            roles.append(
                {
                    'roleName': namespace,
                    'roleArn': 'arn:aws:iam::role/' + namespace,
                    'attachedPolicies': [namespace],
                    'trustPolicy': {
                        'Version': '2012-10-17',
                        'Statement': [
                            {
                                'Effect': 'Allow',
                                'Principal': {'Federated': 'kind'},
                                'Action': ['sts:AssumeRoleWithWebIdentity'],
                                'Condition': {
                                    'StringEquals': {
                                        'oidc:sub': 'system:serviceaccount:'
                                        + namespace
                                        + ':artifact-broker'
                                    }
                                },
                            }
                        ],
                    },
                }
            )
        configuration = {
            'sts': {
                'tokenDuration': '12s',
                'maxSessionLength': '1h',
                'issuer': 'kfp-external-acceptance',
                'signingKey': base64.b64encode(
                    secrets.token_bytes(48)
                ).decode(),
            },
            'providers': [
                {
                    'name': 'kind',
                    'type': 'oidc',
                    'enabled': True,
                    'config': {
                        'issuer': issuer,
                        'clientId': 'artifact-storage',
                        'jwksUri': 'http://jwks.'
                        + self.store_namespace
                        + '.svc:8080/jwks',
                    },
                }
            ],
            'policy': {'defaultEffect': 'Deny'},
            'policies': policies,
            'roles': roles,
        }
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ConfigMap',
                'metadata': {'name': 'jwks', 'namespace': self.store_namespace},
                'data': {'jwks': json.dumps(jwks)},
            }
        )
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'Secret',
                'metadata': {
                    'name': 'storage-config',
                    'namespace': self.store_namespace,
                },
                'stringData': {
                    'iam.json': json.dumps(configuration),
                    's3.json': json.dumps(
                        {
                            'identities': [
                                {
                                    'name': 'fixture-admin',
                                    'actions': ['Admin'],
                                    'credentials': [
                                        {
                                            'accessKey': 'fixture-admin',
                                            'secretKey': 'fixture-admin-secret',
                                        }
                                    ],
                                }
                            ]
                        }
                    ),
                },
            }
        )
        for name, ports, containers, volumes in [
            (
                'jwks',
                [{'port': 8080}],
                [
                    {
                        'name': 'jwks',
                        'image': PYTHON_IMAGE,
                        'command': [
                            'python',
                            '-m',
                            'http.server',
                            '8080',
                            '--directory',
                            '/data',
                        ],
                        'volumeMounts': [
                            {
                                'name': 'config',
                                'mountPath': '/data',
                                'readOnly': True,
                            }
                        ],
                        'readinessProbe': {
                            'httpGet': {'path': '/jwks', 'port': 8080}
                        },
                        'resources': {
                            'requests': {'cpu': '10m', 'memory': '32Mi'}
                        },
                    }
                ],
                [{'name': 'config', 'configMap': {'name': 'jwks'}}],
            ),
            (
                'storage',
                [{'port': 8333}],
                [
                    {
                        'name': 'storage',
                        'image': STORAGE_IMAGE,
                        'args': [
                            'server',
                            '-dir=/data',
                            '-s3',
                            '-iam',
                            '-filer',
                            '-master.volumePreallocate=false',
                            '-s3.config=/config/s3.json',
                            '-s3.iam.config=/config/iam.json',
                        ],
                        'volumeMounts': [
                            {
                                'name': 'config',
                                'mountPath': '/config',
                                'readOnly': True,
                            },
                            {'name': 'data', 'mountPath': '/data'},
                        ],
                        'readinessProbe': {
                            'httpGet': {'path': '/status', 'port': 8333},
                            'initialDelaySeconds': 5,
                            'periodSeconds': 2,
                        },
                        'resources': {
                            'requests': {'cpu': '100m', 'memory': '128Mi'}
                        },
                    }
                ],
                [
                    {
                        'name': 'config',
                        'secret': {'secretName': 'storage-config'},
                    },
                    {'name': 'data', 'emptyDir': {}},
                ],
            ),
        ]:
            apply(
                {
                    'apiVersion': 'v1',
                    'kind': 'Service',
                    'metadata': {
                        'name': name,
                        'namespace': self.store_namespace,
                    },
                    'spec': {'selector': {'app': name}, 'ports': ports},
                }
            )
            apply(
                {
                    'apiVersion': 'apps/v1',
                    'kind': 'Deployment',
                    'metadata': {
                        'name': name,
                        'namespace': self.store_namespace,
                    },
                    'spec': {
                        'replicas': 1,
                        'selector': {'matchLabels': {'app': name}},
                        'template': {
                            'metadata': {
                                'labels': {
                                    'app': name,
                                    'sidecar.istio.io/inject': 'false',
                                }
                            },
                            'spec': {
                                'containers': containers,
                                'volumes': volumes,
                            },
                        },
                    },
                }
            )
            kubectl(
                'rollout',
                'status',
                'deployment/' + name,
                '-n',
                self.store_namespace,
                '--timeout=' + str(self.timeout) + 's',
                timeout=self.timeout + 10,
            )
        endpoint = self.forward_external_storage()
        self.external_admin = boto3.client(
            's3',
            endpoint_url=endpoint,
            region_name='us-east-1',
            aws_access_key_id='fixture-admin',
            aws_secret_access_key='fixture-admin-secret',
            config=Config(
                connect_timeout=5, read_timeout=15, retries={'max_attempts': 2}
            ),
        )
        self.external_admin.create_bucket(Bucket=BUCKET)
        for namespace in self.tenants:
            for suffix in ['model.txt', 'directory/part.txt']:
                self.external_admin.put_object(
                    Bucket=BUCKET,
                    Key=self.prefixes[namespace] + suffix,
                    Body=(namespace + ' historical artifact').encode(),
                )
        self.capture_images(self.store_namespace, 'app=storage')
        self.capture_images(self.store_namespace, 'app=jwks')
        self.passed(
            'external bucket seeded before controller adoption; real Kubernetes OIDC configured'
        )

    def source_configmap(self, name, key, value):
        return {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'metadata': {
                'name': name,
                'namespace': self.namespace,
                'labels': {SOURCE_LABEL: 'true'},
            },
            'data': {key: json.dumps(value)},
        }

    def publish_source(self, name, key, value):
        if name not in self.source_names:
            self.source_names.append(name)
        apply(self.source_configmap(name, key, value))

    def publish_bindings(self):
        self.publish_source(
            self.binding_source, 'bindings.json', self.trusted_bindings
        )

    def publish_template(self):
        self.publish_source(
            self.template_source, 'receiver.json', self.receiver_template
        )

    def adoption_policy(self):
        return {
            'name': self.run_id,
            'namespaceSelector': {'matchLabels': {CLASS_LABEL: self.run_id}},
            'bindingsSource': {
                'namespace': self.namespace,
                'configMap': self.binding_source,
                'key': 'bindings.json',
            },
            'allowedStorage': {
                'source': 's3',
                'bucket': BUCKET,
                'endpoint': self.external_endpoint,
                'region': 'us-east-1',
            },
            'receiverTemplateRef': {
                'namespace': self.namespace,
                'configMap': self.template_source,
                'key': 'receiver.json',
            },
            'adoptExistingNamespaces': True,
            'enrollNewNamespaces': True,
        }

    def publish_adoption_policy(self):
        current = (
            get('configmap', APPROVALS, self.namespace, optional=True) or {}
        )
        data = dict(current.get('data', {}))
        policies = json.loads(
            data.get(POLICIES_KEY, '{"version":1,"policies":[]}')
        )
        require(
            policies.get('version') == 1
            and isinstance(policies.get('policies'), list),
            'Existing adoption policy configuration is not valid',
        )
        policies['policies'].append(self.adoption_policy())
        data[POLICIES_KEY] = json.dumps(policies)
        self.approvals_changed = True
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ConfigMap',
                'metadata': {
                    'name': APPROVALS,
                    'namespace': self.namespace,
                    'labels': {
                        **current.get('metadata', {}).get('labels', {}),
                        SOURCE_LABEL: 'true',
                    },
                },
                'data': data,
            }
        )
        self.assert_no_manual_approvals()

    def assert_no_manual_approvals(self):
        current = get('configmap', APPROVALS, self.namespace)
        original = (
            (self.approval_snapshot or {}).get('data', {}).get('approvals.json')
        )
        actual = current.get('data', {}).get('approvals.json')
        require(
            actual == original,
            'Fixture or controller wrote manual approvals.json entries',
        )
        require(
            not any(
                namespace in json.loads(actual or '{}')
                for namespace in self.tenants
            ),
            'Fixture namespace has a manual approval',
        )

    def prepare_namespace(self, namespace):
        apply(
            {
                'apiVersion': 'kubeflow.org/v1',
                'kind': 'Profile',
                'metadata': {'name': namespace},
                'spec': {
                    'owner': {'kind': 'User', 'name': self.users[namespace]}
                },
            }
        )
        wait_for(
            'external fixture Profile namespace',
            lambda: get('namespace', namespace, optional=True),
            self.timeout,
        )
        kubectl(
            'label',
            'namespace',
            namespace,
            CLASS_LABEL + '=' + self.run_id,
            '--overwrite',
        )
        apply(
            {
                'apiVersion': 'rbac.authorization.k8s.io/v1',
                'kind': 'Role',
                'metadata': {'name': 'artifact-viewer', 'namespace': namespace},
                'rules': [
                    {
                        'apiGroups': ['kubeflow.org'],
                        'resources': ['viewers'],
                        'verbs': ['get'],
                    },
                    {
                        'apiGroups': [''],
                        'resources': ['configmaps'],
                        'verbs': ['get', 'create', 'update', 'patch'],
                    },
                ],
            }
        )
        apply(
            {
                'apiVersion': 'rbac.authorization.k8s.io/v1',
                'kind': 'RoleBinding',
                'metadata': {'name': 'artifact-viewer', 'namespace': namespace},
                'roleRef': {
                    'apiGroup': 'rbac.authorization.k8s.io',
                    'kind': 'Role',
                    'name': 'artifact-viewer',
                },
                'subjects': [
                    {
                        'apiGroup': 'rbac.authorization.k8s.io',
                        'kind': 'User',
                        'name': self.users[namespace],
                    }
                ],
            }
        )
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ServiceAccount',
                'metadata': {'name': 'artifact-broker', 'namespace': namespace},
            }
        )
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ConfigMap',
                'metadata': {'name': 'artifact-broker', 'namespace': namespace},
                'data': {'broker.py': self.broker_code},
            }
        )

    def binding(self, namespace):
        return {
            'namespaceUID': get('namespace', namespace)['metadata']['uid'],
            'principal': 'arn:aws:iam::role/' + namespace,
        }

    def shared_template(self, namespace):
        approval = self.approval(namespace)
        template = approval['podTemplate']
        broker = next(
            container
            for container in template['spec']['containers']
            if container['name'] == 'broker'
        )
        broker['env'] = [
            {'name': 'STS_ENDPOINT', 'value': self.external_endpoint},
            {
                'name': 'POD_NAMESPACE',
                'valueFrom': {'fieldRef': {'fieldPath': 'metadata.namespace'}},
            },
            {
                'name': 'STORAGE_ROLE',
                'value': 'arn:aws:iam::role/$(POD_NAMESPACE)',
            },
        ]
        return {
            'identity': {
                key: approval['identity'][key]
                for key in ('kind', 'credentialFile', 'profile')
            },
            'podTemplate': template,
        }

    def wait_unavailable(self, namespace):
        self.wait_state(namespace, 'pending')

        def rejected():
            response = self.external_request(namespace)
            require(
                response['status'] != 200,
                'Unapproved external identity served an artifact',
            )
            return response if response['status'] == 503 else None

        response = wait_for(
            'external pending receiver and real owner SAR',
            rejected,
            self.timeout,
        )
        require(
            response['retryAfter'] == '5',
            'External readiness failure is not retryable',
        )

    def assert_receiver_ownership(self, namespace):
        namespace_uid = get('namespace', namespace)['metadata']['uid']
        for kind in ['Deployment', 'Service', 'NetworkPolicy']:
            resource = get(kind.lower(), RECEIVER, namespace, optional=True)
            shape = receiver_ownership_shape(resource, namespace_uid)
            require(
                shape['exists']
                and shape['ownerCount'] > 0
                and shape['ownerCount'] == len(shape['owners'])
                and all(owner['matchesNamespace'] for owner in shape['owners'])
                and any(owner['controller'] for owner in shape['owners']),
                'Enrolled external ' + kind + ' lost its Namespace ownership',
            )

    def wait_enrolled(self, namespace, superseded_uids=None):
        record = self.wait_state(namespace, 'ready')
        require(
            record.get('adoptionPolicy', {}).get('name') == self.run_id,
            'Ready record was not automatically derived from the fixture policy',
        )
        wait_for(
            'initial broker session',
            lambda: self.broker_status(namespace),
            self.timeout,
        )
        if superseded_uids is not None:
            self.wait_receiver_pods(namespace, self.ui_image, superseded_uids)
        self.assert_receiver_ownership(namespace)

        def readable():
            response = self.external_request(namespace)
            if response['status'] == 503:
                require(
                    response['retryAfter'] == '5',
                    'External rollout rejection is not retryable',
                )
                return None
            require(
                response['status'] == 200
                and response['body'] == namespace + ' historical artifact',
                'Enrolled external receiver did not serve the exact historical artifact',
            )
            return response

        wait_for(
            'enrolled receiver serves historical artifact',
            readable,
            self.timeout,
        )
        self.capture_images(namespace, 'app=' + RECEIVER)
        self.assert_no_manual_approvals()

    def apply_approval_configuration(self, snapshot):
        self.approvals_changed = True
        if snapshot is None:
            kubectl(
                'delete',
                'configmap',
                APPROVALS,
                '-n',
                self.namespace,
                '--ignore-not-found',
                timeout=20,
            )
            return
        resource = copy.deepcopy(snapshot)
        resource['metadata'] = {
            key: value
            for key, value in resource['metadata'].items()
            if key
            in ('name', 'namespace', 'labels', 'annotations', 'ownerReferences')
        }
        apply(resource)

    def authorization_snapshot(self, namespace, secret_uid=None):
        secret = get(
            'secret',
            'kfp-artifact-storage-' + namespace,
            self.namespace,
            optional=True,
        )
        if secret_uid is not None:
            require(
                secret and secret.get('metadata', {}).get('uid') == secret_uid,
                'Authorization record was deleted or replaced during recovery',
            )
        if not secret:
            return None
        record = json.loads(decode_secret(secret, 'authorization.json'))
        namespace_uid = get('namespace', namespace)['metadata']['uid']
        require(
            record.get('namespace') == namespace
            and record.get('namespaceUID') == namespace_uid
            and any(
                owner.get('kind') == 'Namespace'
                and owner.get('uid') == namespace_uid
                for owner in secret['metadata'].get('ownerReferences', [])
            ),
            'Recovery record does not identify and belong to the actual namespace',
        )
        require(
            bool(secret['metadata'].get('uid')),
            'Authorization record has no UID',
        )
        return secret, record

    def wait_authorization(self, namespace, version, state, secret_uid=None):

        def observed():
            snapshot = self.authorization_snapshot(namespace, secret_uid)
            if not snapshot:
                return None
            secret, record = snapshot
            return (
                secret
                if record.get('version') == version
                and record.get('state') == state
                else None
            )

        return wait_for('authorization record recovery', observed, self.timeout)

    def wait_bundled_history(
        self, namespace, secret_uid=None, managed_children=None
    ):
        history = self.bundled_history[namespace]

        def readable():
            if managed_children is not None:
                self.assert_managed_children(namespace, managed_children)
            snapshot = self.authorization_snapshot(namespace, secret_uid)
            if not snapshot:
                return None
            secret, record = snapshot
            if record.get('version') != 1 or record.get('state') != 'ready':
                return None
            response = self.request(namespace, history['key'])
            if response['status'] == 503:
                require(
                    response['retryAfter'] == '5',
                    'Managed recovery rejection is not retryable',
                )
                return None
            require(
                response['status'] == 200
                and response['body'] == history['body'],
                'Managed recovery did not serve the exact bundled historical artifact',
            )
            return secret

        return wait_for(
            'bundled historical artifact recovery', readable, self.timeout
        )

    def managed_children(self, namespace):
        kinds = 'secret,configmap,deployment,service,serviceaccount,role,rolebinding,networkpolicy'
        return {
            (resource['kind'], resource['metadata']['name']): resource
            for resource in get(kinds, namespace=namespace)['items']
        }

    def managed_children_snapshot(self, namespace):
        namespace_uid = get('namespace', namespace)['metadata']['uid']
        snapshot = {}
        for key, resource in self.managed_children(namespace).items():
            metadata = resource['metadata']
            owners = metadata.get('ownerReferences', [])
            if not any(
                owner.get('kind') == 'Namespace'
                and owner.get('uid') == namespace_uid
                for owner in owners
            ):
                continue
            require(
                metadata.get('uid')
                and all(
                    owner.get('apiVersion') == 'v1'
                    and owner.get('kind') == 'Namespace'
                    and owner.get('name') == namespace
                    and owner.get('uid') == namespace_uid
                    for owner in owners
                )
                and any(owner.get('controller') is True for owner in owners),
                'Managed child does not have exclusive current Namespace ownership',
            )
            snapshot[key] = {
                'uid': metadata['uid'],
                'owners': copy.deepcopy(owners),
            }
            if resource['kind'] == 'Secret' and resource.get('data'):
                # Compare populated credentials without retaining or reporting bytes.
                # An empty service-account token may still be populated by Kubernetes.
                snapshot[key]['secretDataHash'] = hashlib.sha256(
                    json.dumps(
                        resource.get('data', {}), sort_keys=True
                    ).encode()
                ).hexdigest()
        require(
            {
                ('Secret', 'mlpipeline-minio-artifact'),
                ('ConfigMap', 'kfp-launcher'),
                ('ConfigMap', 'artifact-repositories'),
                ('Deployment', RECEIVER),
                ('Service', RECEIVER),
                ('NetworkPolicy', RECEIVER),
            }.issubset(snapshot),
            'Managed recovery baseline lacks required namespace-owned children',
        )
        return snapshot

    def assert_managed_children(self, namespace, snapshot):
        observed = self.managed_children(namespace)
        for key, original in snapshot.items():
            resource = observed.get(key, {})
            metadata = resource.get('metadata', {})
            require(
                metadata.get('uid') == original['uid']
                and metadata.get('ownerReferences') == original['owners'],
                'Managed child was removed, replaced or detached during recovery: '
                + key[0]
                + '/'
                + key[1],
            )
            if 'secretDataHash' in original:
                digest = hashlib.sha256(
                    json.dumps(
                        resource.get('data', {}), sort_keys=True
                    ).encode()
                ).hexdigest()
                require(
                    digest == original['secretDataHash'],
                    'Managed Secret data changed during recovery: ' + key[1],
                )

    def check_unadopted_config_recovery(self, namespace):
        history = {
            'key': 'private-artifacts/'
            + namespace
            + '/'
            + self.run_id
            + '/before-external.txt',
            'body': namespace + ' bundled history before external adoption',
        }
        self.bundled_history[namespace] = history
        self.objects.append(history['key'])
        self.s3.put_object(
            Bucket=MANAGED_BUCKET,
            Key=history['key'],
            Body=history['body'].encode(),
        )
        secret_uid = self.wait_bundled_history(namespace)['metadata']['uid']
        managed_children = self.managed_children_snapshot(namespace)
        valid = (
            copy.deepcopy(self.approval_snapshot)
            if self.approval_snapshot
            else {
                'apiVersion': 'v1',
                'kind': 'ConfigMap',
                'metadata': {'name': APPROVALS, 'namespace': self.namespace},
                'data': {},
            }
        )
        for field, recovery in [
            (POLICIES_KEY, 'corrected'),
            ('approvals.json', 'deleted'),
        ]:
            malformed = copy.deepcopy(valid)
            malformed.setdefault('data', {})[field] = '{malformed'
            self.apply_approval_configuration(malformed)
            self.wait_authorization(namespace, 2, 'pending', secret_uid)
            denied = self.request(namespace, history['key'])
            require(
                denied['status'] == 503 and denied['retryAfter'] == '5',
                'Malformed shared configuration did not block the real bundled artifact read',
            )
            self.apply_approval_configuration(
                valid if recovery == 'corrected' else None
            )
            self.wait_bundled_history(namespace, secret_uid, managed_children)
            self.assert_managed_children(namespace, managed_children)
            self.passed(
                'unadopted namespace recovers real bundled artifact access without replacing its authorization Secret',
                malformedField=field,
                recovery=recovery,
                managedChildrenPreserved=len(managed_children),
            )
        # Restore unrelated administrator configuration before publishing the fixture policy.
        self.apply_approval_configuration(self.approval_snapshot)
        self.wait_bundled_history(namespace, secret_uid, managed_children)
        self.assert_managed_children(namespace, managed_children)

    def check_bundled_history_cutover(self, namespace):
        history = self.bundled_history[namespace]

        def checked():
            response = self.request(namespace, history['key'])
            if response['status'] == 503:
                require(
                    response.get('retryAfter') == '5',
                    'External cutover rejection is not retryable',
                )
                return False
            require(
                response['status'] == 403
                and history['body'] not in response['body'],
                'External cutover unexpectedly served earlier bundled-store run history',
            )
            external = self.external_request(namespace)
            if external['status'] == 503:
                require(
                    external.get('retryAfter') == '5',
                    'External cutover rejection is not retryable',
                )
                return False
            require(
                external['status'] == 200
                and external['body'] == namespace + ' historical artifact',
                'External cutover did not preserve the approved historical artifact',
            )
            return True

        wait_for(
            'bundled artifact denial after external cutover',
            checked,
            self.timeout,
        )
        self.passed(
            'intentional external cutover denies earlier bundled-store run artifacts without destination fallback'
        )

    def prepare_profiles(self):
        self.approval_snapshot = get(
            'configmap', APPROVALS, self.namespace, optional=True
        )
        existing, new = self.tenants
        self.prepare_namespace(existing)

        def stock_receiver_ready():
            deployment = get('deployment', RECEIVER, existing, optional=True)
            return (
                deployment
                and deployment.get('status', {}).get('availableReplicas') == 1
            )

        wait_for(
            'existing namespace receiver before policy publication',
            stock_receiver_ready,
            self.timeout,
        )
        self.check_unadopted_config_recovery(existing)
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ResourceQuota',
                'metadata': {
                    'name': 'hold-external-rollout',
                    'namespace': existing,
                },
                'spec': {'hard': {'pods': '0'}},
            }
        )
        previous_existing_pods = self.receiver_pod_uids(existing)
        self.receiver_template = self.shared_template(existing)
        self.trusted_bindings[existing] = self.binding(existing)
        self.publish_template()
        self.publish_bindings()
        self.publish_adoption_policy()
        self.wait_unavailable(existing)
        kubectl(
            'delete', 'resourcequota', 'hold-external-rollout', '-n', existing
        )
        self.wait_enrolled(existing, previous_existing_pods)
        self.check_bundled_history_cutover(existing)
        self.passed(
            'existing namespace automatically adopted from central policy and UID-bound inventory'
        )

        existing_pod_uid = self.pod(existing)['metadata']['uid']
        existing_record = self.record(existing)
        self.prepare_namespace(new)
        self.wait_unavailable(new)
        # Simulate the trusted external provisioner publishing the newly observed
        # namespace identity; no per-namespace KFP approval is manufactured.
        previous_new_pods = self.receiver_pod_uids(new)
        self.trusted_bindings[new] = self.binding(new)
        self.publish_bindings()
        self.wait_enrolled(new, previous_new_pods)
        require(
            self.pod(existing)['metadata']['uid'] == existing_pod_uid
            and self.record(existing) == existing_record,
            'Enrolling another namespace changed the existing receiver or grant',
        )
        self.passed(
            'new namespace automatically enrolled without manual approvals or disturbing the existing receiver'
        )

    def approval(self, namespace):
        uid = get('namespace', namespace)['metadata']['uid']
        mounts = [{'name': 'credentials', 'mountPath': CREDENTIAL_DIRECTORY}]
        # CI loads the receiver image into Kind without publishing it.
        return {
            'namespaceUID': uid,
            'storage': {
                'source': 's3',
                'bucket': BUCKET,
                'endpoint': self.external_endpoint,
                'region': 'us-east-1',
            },
            'identity': {
                'kind': 'shared-credentials-file',
                'principal': 'arn:aws:iam::role/' + namespace,
                'credentialFile': CREDENTIAL_DIRECTORY + '/credentials',
                'profile': 'artifact-reader',
            },
            'podTemplate': {
                'metadata': {'labels': {'sidecar.istio.io/inject': 'false'}},
                'spec': {
                    'serviceAccountName': 'artifact-broker',
                    'securityContext': {'fsGroup': 1000},
                    'containers': [
                        {
                            'name': RECEIVER,
                            'image': self.ui_image,
                            'imagePullPolicy': 'IfNotPresent',
                            'env': [],
                            'securityContext': {
                                'runAsUser': 1000,
                                'runAsNonRoot': True,
                            },
                            'volumeMounts': [{**mounts[0], 'readOnly': True}],
                        },
                        {
                            'name': 'broker',
                            'image': PYTHON_IMAGE,
                            'command': ['python', '/fixture/broker.py'],
                            'securityContext': {
                                'runAsUser': 1000,
                                'runAsNonRoot': True,
                            },
                            'env': [
                                {
                                    'name': 'STS_ENDPOINT',
                                    'value': self.external_endpoint,
                                },
                                {
                                    'name': 'STORAGE_ROLE',
                                    'value': 'arn:aws:iam::role/' + namespace,
                                },
                            ],
                            'volumeMounts': mounts
                            + [
                                {
                                    'name': 'broker',
                                    'mountPath': '/fixture',
                                    'readOnly': True,
                                },
                                {
                                    'name': 'identity',
                                    'mountPath': '/var/run/secrets/artifact-storage',
                                    'readOnly': True,
                                },
                            ],
                            'resources': {
                                'requests': {'cpu': '10m', 'memory': '32Mi'}
                            },
                        },
                    ],
                    'volumes': [
                        {'name': 'credentials', 'emptyDir': {}},
                        {
                            'name': 'broker',
                            'configMap': {'name': 'artifact-broker'},
                        },
                        {
                            'name': 'identity',
                            'projected': {
                                'sources': [
                                    {
                                        'serviceAccountToken': {
                                            'audience': 'artifact-storage',
                                            'expirationSeconds': 600,
                                            'path': 'token',
                                        }
                                    }
                                ]
                            },
                        },
                    ],
                },
            },
        }

    def record(self, namespace):
        secret = get(
            'secret',
            'kfp-artifact-storage-' + namespace,
            self.namespace,
            optional=True,
        )
        if not secret:
            return None
        record = json.loads(decode_secret(secret, 'authorization.json'))
        uid = get('namespace', namespace)['metadata']['uid']
        if record.get('version') != 2:
            return None
        require(
            record['namespace'] == namespace and record['namespaceUID'] == uid,
            'External readiness record names the wrong actual namespace',
        )
        require(
            any(
                owner.get('kind') == 'Namespace' and owner.get('uid') == uid
                for owner in secret['metadata'].get('ownerReferences', [])
            ),
            'External readiness record lacks the actual Namespace owner',
        )
        if record['state'] == 'ready':
            require(
                record['storage']['endpoint'] == self.external_endpoint
                and record['storage']['bucket'] == BUCKET,
                'Controller rewrote the external destination',
            )
        return record

    def pod(self, namespace):
        pods = get('pods', namespace=namespace, selector='app=' + RECEIVER)[
            'items'
        ]
        ready = [
            pod
            for pod in pods
            if not pod['metadata'].get('deletionTimestamp')
            and pod.get('status', {}).get('phase') == 'Running'
        ]
        return ready[0] if len(ready) == 1 else None

    def broker_status(self, namespace):
        pod = self.pod(namespace)
        if not pod:
            return None
        result = kubectl(
            'exec',
            '-n',
            namespace,
            pod['metadata']['name'],
            '-c',
            'broker',
            '--',
            'python',
            '-c',
            'import pathlib; print(pathlib.Path("/var/run/kfp-artifacts/status.json").read_text())',
            optional=True,
            timeout=20,
        )
        if result.returncode:
            return None
        return json.loads(result.stdout)

    def external_request(self, namespace, key=None, user=None):
        query = urllib.parse.urlencode(
            {
                'source': 's3',
                'bucket': BUCKET,
                'namespace': namespace,
                'key': key or self.prefixes[namespace] + 'model.txt',
                'providerInfo': json.dumps(
                    {
                        'Provider': 's3',
                        'Params': {
                            'fromEnv': 'true',
                            'endpoint': 'http://127.0.0.1:1',
                        },
                    }
                ),
            }
        )
        request = urllib.request.Request(
            self.ui + '/artifacts/get?' + query,
            headers={'kubeflow-userid': user or self.users[namespace]},
        )
        try:
            response = urllib.request.urlopen(request, timeout=30)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            result = {
                'status': response.status,
                'body': response.read(4096).decode(errors='replace'),
                'retryAfter': response.headers.get('Retry-After'),
            }
            self.last_response = {
                key: result[key] for key in ('status', 'retryAfter')
            }
            return result

    def list_prefix(self, namespace, prefix):
        pod = self.pod(namespace)
        require(pod is not None, 'External receiver Pod is unavailable')
        result = subprocess.run(
            [
                'kubectl',
                '-n',
                namespace,
                'exec',
                '-i',
                pod['metadata']['name'],
                '-c',
                RECEIVER,
                '--',
                'node',
                '-e',
                LIST_PROBE,
            ],
            input=json.dumps({'prefix': prefix}),
            capture_output=True,
            text=True,
            timeout=30,
        )
        require(result.returncode == 0, 'External directory probe failed')
        return json.loads(result.stdout)

    def check_external_access(self, *, recovering=False):

        def checked():
            for namespace, other in [
                self.tenants,
                list(reversed(self.tenants)),
            ]:
                response = self.external_request(namespace)
                if recovering and response['status'] == 503:
                    require(
                        response['retryAfter'] == '5',
                        'External policy recovery rejection is not retryable',
                    )
                    return False
                require(
                    response['status'] == 200
                    and response['body'] == namespace + ' historical artifact',
                    'Authorized historical external artifact was not served',
                )
                denied = self.external_request(
                    namespace, self.prefixes[other] + 'model.txt'
                )
                if recovering and denied['status'] == 503:
                    require(
                        denied['retryAfter'] == '5',
                        'External policy recovery rejection is not retryable',
                    )
                    return False
                require(
                    denied['status'] == 500
                    and other + ' historical artifact' not in denied['body'],
                    'External identity did not deny another tenant object',
                )
                impersonated = self.external_request(
                    other, user=self.users[namespace]
                )
                require(
                    impersonated['status'] == 403,
                    'Real SAR allowed namespace impersonation',
                )
                listing = self.list_prefix(
                    namespace, self.prefixes[namespace] + 'directory/'
                )
                require(
                    listing.get('objects')
                    == [self.prefixes[namespace] + 'directory/part.txt'],
                    'Authorized external directory listing failed',
                )
                for prefix in [self.prefixes[other], '']:
                    listing = self.list_prefix(namespace, prefix)
                    require(
                        listing == {'denied': True, 'code': 'AccessDenied'},
                        'External identity did not deny a foreign or bucket-wide listing',
                    )
            return True

        if recovering:
            # Restoring policy sources can update Metacontroller's Deployment
            # annotation after a ready observation. Retry only the explicit
            # pending protocol while the controller observes that generation.
            wait_for(
                'external access after policy restoration',
                checked,
                self.timeout,
            )
        else:
            checked()
        self.passed(
            'existing external roots readable; cross-tenant object/list/SAR requests denied'
        )

    def check_role_trust(self):
        namespace, other = self.tenants
        pod = self.pod(namespace)
        script = """
import pathlib, sys, tempfile, urllib.error
sys.path.insert(0, '/fixture')
import broker
try:
    with tempfile.TemporaryDirectory() as directory:
        broker.refresh(sys.argv[1], sys.argv[2], pathlib.Path('/var/run/secrets/artifact-storage/token'), pathlib.Path(directory), 1)
except urllib.error.HTTPError as error:
    print(error.code)
else:
    print('ALLOWED')
"""
        result = kubectl(
            'exec',
            '-n',
            namespace,
            pod['metadata']['name'],
            '-c',
            'broker',
            '--',
            'python',
            '-c',
            script,
            self.external_endpoint,
            'arn:aws:iam::role/' + other,
            timeout=20,
        )
        require(
            result.stdout.strip() == '403',
            'Projected namespace ServiceAccount could assume another tenant storage role',
        )
        self.passed(
            'real projected ServiceAccount cannot assume the other tenant storage role'
        )

    def check_rotation(self):
        namespace = self.tenants[0]
        pod_uid = self.pod(namespace)['metadata']['uid']
        original_record = self.record(namespace)
        initial = self.broker_status(namespace)
        fingerprints = {initial['fingerprint']}
        previous = initial
        for _ in range(3):

            def changed():
                current = self.broker_status(namespace)
                return (
                    current
                    if current
                    and current['generation'] > previous['generation']
                    else None
                )

            current = wait_for('real broker credential renewal', changed, 30)
            fingerprints.add(current['fingerprint'])
            require(
                self.pod(namespace)['metadata']['uid'] == pod_uid,
                'Credential renewal restarted the receiver Pod',
            )
            require(
                self.record(namespace) == original_record,
                'Credential renewal changed the stable identity readiness binding',
            )
            self.check_external_access()
            previous = current
        require(
            len(fingerprints) == 4, 'Broker did not issue distinct STS sessions'
        )
        self.passed(
            'three real STS renewals preserve receiver Pod and controller identity binding',
            namespace=namespace,
            receiverPodUID=pod_uid,
        )

    def check_expiry_and_recovery(self):
        namespace = self.tenants[0]
        pod = self.pod(namespace)
        name, uid = pod['metadata']['name'], pod['metadata']['uid']
        kubectl(
            'exec',
            '-n',
            namespace,
            name,
            '-c',
            'broker',
            '--',
            'python',
            '-c',
            'import pathlib; pathlib.Path("/var/run/kfp-artifacts/pause").touch()',
        )
        time.sleep(5)
        frozen = self.broker_status(namespace)
        expires = datetime.fromisoformat(
            frozen['expiration'].replace('Z', '+00:00')
        ).timestamp()
        time.sleep(max(0, expires - time.time()) + 1)
        response = self.external_request(namespace)
        require(
            response['status'] == 500
            and namespace + ' historical artifact' not in response['body'],
            'Expired external session was served through broader credentials',
        )
        listing = self.list_prefix(namespace, self.prefixes[namespace])
        require(
            listing.get('denied')
            and listing.get('code') in ['InvalidAccessKeyId', 'ExpiredToken'],
            'External storage accepted the expired session',
        )
        kubectl(
            'exec',
            '-n',
            namespace,
            name,
            '-c',
            'broker',
            '--',
            'python',
            '-c',
            'import pathlib; pathlib.Path("/var/run/kfp-artifacts/pause").unlink()',
        )
        wait_for(
            'broker refresh after expiry',
            lambda: (
                (self.broker_status(namespace) or {}).get('generation', 0)
                > frozen['generation']
            ),
            30,
        )
        self.check_external_access()
        require(
            self.pod(namespace)['metadata']['uid'] == uid,
            'Expiry recovery restarted the receiver',
        )
        self.passed(
            'real expired session denied; broker renewal restores reads without Pod restart'
        )

    def check_adoption_uid_mismatch(self):
        namespace = self.tenants[0]
        original = self.trusted_bindings[namespace]['namespaceUID']
        self.trusted_bindings[namespace]['namespaceUID'] = (
            '00000000-0000-0000-0000-000000000000'
        )
        self.publish_bindings()
        self.wait_unavailable(namespace)
        self.trusted_bindings[namespace]['namespaceUID'] = original
        self.publish_bindings()
        self.wait_enrolled(namespace)
        self.check_external_access(recovering=True)
        self.passed(
            'stale namespace UID inventory is denied; current UID automatically recovers'
        )

    def check_adoption_forgery(self):
        namespace, other = self.tenants
        original = self.trusted_bindings.pop(namespace)
        self.publish_bindings()
        self.wait_unavailable(namespace)
        forged = {
            namespace: {**original, 'principal': 'arn:aws:iam::role/' + other}
        }
        for name, key, value in [
            (self.binding_source, 'bindings.json', forged),
            (self.template_source, 'receiver.json', self.receiver_template),
        ]:
            resource = self.source_configmap(name, key, value)
            resource['metadata']['namespace'] = namespace
            kubectl(
                'apply',
                '-f',
                '-',
                '--validate=false',
                '--as=' + self.users[namespace],
                payload=resource,
            )
            mutation = kubectl(
                'patch',
                'configmap',
                name,
                '-n',
                self.namespace,
                '--as=' + self.users[namespace],
                '--type=merge',
                '--patch-file=/dev/stdin',
                payload={'data': {key: json.dumps(value)}},
                optional=True,
            )
            require(
                mutation.returncode != 0 and '(Forbidden)' in mutation.stderr,
                'Tenant could change a central storage policy source',
            )
        self.wait_unavailable(namespace)
        central = get('configmap', self.binding_source, self.namespace)
        require(
            json.loads(central['data']['bindings.json'])
            == self.trusted_bindings,
            'Tenant mutation changed central identity inventory',
        )
        self.trusted_bindings[namespace] = original
        self.publish_bindings()
        self.wait_enrolled(namespace)
        self.check_external_access(recovering=True)
        self.passed(
            'matching labels and tenant-local source forgeries confer no grant; central writes are Forbidden'
        )

    def check_adoption_revocation(self):
        for name, restore in [
            (self.binding_source, self.publish_bindings),
            (self.template_source, self.publish_template),
        ]:
            kubectl('delete', 'configmap', name, '-n', self.namespace)
            for namespace in self.tenants:
                self.wait_unavailable(namespace)
            restore()
            for namespace in self.tenants:
                self.wait_enrolled(namespace)
            self.check_external_access(recovering=True)
            self.passed(
                'trusted source removal blocks requests; restoration automatically recovers',
                source=name,
            )
        self.check_adopted_config_recovery()
        self.assert_no_manual_approvals()

    def check_adopted_config_recovery(self):
        valid = get('configmap', APPROVALS, self.namespace)
        malformed = copy.deepcopy(valid)
        malformed['data'][POLICIES_KEY] = '{malformed'
        secret_uids = {
            namespace: self.authorization_snapshot(namespace)[0]['metadata'][
                'uid'
            ]
            for namespace in self.tenants
        }
        self.apply_approval_configuration(malformed)
        for namespace in self.tenants:
            self.wait_authorization(
                namespace, 2, 'pending', secret_uids[namespace]
            )
            self.wait_unavailable(namespace)
        self.apply_approval_configuration(valid)
        for namespace in self.tenants:
            self.wait_enrolled(namespace)
            self.wait_authorization(
                namespace, 2, 'ready', secret_uids[namespace]
            )
        self.check_external_access(recovering=True)
        self.passed(
            'adopted namespaces recover after malformed shared policy without grant or authorization Secret replacement'
        )

    def diagnostics(self):
        super().diagnostics()
        diagnostic = {
            'storagePods': [],
            'receiverStartup': {},
            'brokerErrors': {},
            'controllerPendingReasons': {},
            'authorizationShape': {},
            'receiverOwnership': {},
        }
        controller_logs = kubectl(
            'logs',
            '-n',
            self.namespace,
            'deployment/' + CONTROLLER,
            '-c',
            'profile-controller',
            '--tail=1000',
            '--limit-bytes=' + str(CONTROLLER_LOG_LIMIT),
            optional=True,
            timeout=20,
        )
        diagnostic['controllerLogUnavailable'] = controller_logs.returncode != 0
        if controller_logs.returncode == 0:
            diagnostic['controllerPendingReasons'] = controller_pending_reasons(
                controller_logs.stdout, self.tenants
            )
        if self.gateway_created:
            for pod in get('pods', namespace=self.store_namespace)['items']:
                diagnostic['storagePods'].append(
                    {
                        'name': pod['metadata']['name'],
                        'phase': pod.get('status', {}).get('phase'),
                        'containers': [
                            {
                                key: status.get(key)
                                for key in (
                                    'name',
                                    'ready',
                                    'restartCount',
                                    'state',
                                )
                            }
                            for status in pod.get('status', {}).get(
                                'containerStatuses', []
                            )
                        ],
                    }
                )
        for namespace in self.tenants:
            namespace_resource = get('namespace', namespace, optional=True)
            if namespace_resource is None:
                continue
            namespace_uid = namespace_resource.get('metadata', {}).get('uid')
            diagnostic['receiverOwnership'][namespace] = {
                kind: receiver_ownership_shape(
                    get(kind.lower(), RECEIVER, namespace, optional=True),
                    namespace_uid,
                )
                for kind in ['Deployment', 'Service', 'NetworkPolicy']
            }
            secret = get(
                'secret',
                'kfp-artifact-storage-' + namespace,
                self.namespace,
                optional=True,
            )
            if secret:
                record = json.loads(decode_secret(secret, 'authorization.json'))
                diagnostic['authorizationShape'][namespace] = {
                    'hasStorage': isinstance(record.get('storage'), dict),
                    'hasIdentity': isinstance(record.get('identity'), dict),
                    'hasBinding': bool(record.get('credentialBinding')),
                    'hasAdoptionPolicy': isinstance(
                        record.get('adoptionPolicy'), dict
                    ),
                }
            pods = get('pods', namespace=namespace, selector='app=' + RECEIVER)[
                'items'
            ]
            diagnostic['receiverStartup'][namespace] = []
            diagnostic['brokerErrors'][namespace] = []
            for pod in pods:
                if pod['metadata'].get('deletionTimestamp'):
                    continue
                containers = {
                    item['name']: item for item in pod['spec']['containers']
                }
                diagnostic['receiverStartup'][namespace].extend(
                    [
                        {
                            'pod': pod['metadata']['name'],
                            'name': status['name'],
                            'ready': status.get('ready', False),
                            'imagePullPolicy': containers[status['name']].get(
                                'imagePullPolicy'
                            ),
                            'waitingReason': status.get('state', {})
                            .get('waiting', {})
                            .get('reason'),
                            'waitingMessage': status.get('state', {})
                            .get('waiting', {})
                            .get('message', '')[:1000],
                        }
                        for status in pod.get('status', {}).get(
                            'containerStatuses', []
                        )
                    ]
                )
                logs = kubectl(
                    'logs',
                    '-n',
                    namespace,
                    pod['metadata']['name'],
                    '-c',
                    'broker',
                    '--tail=20',
                    optional=True,
                    timeout=20,
                )
                diagnostic['brokerErrors'][namespace].extend(
                    [
                        line
                        for line in logs.stdout.splitlines()
                        if line.startswith('Credential refresh failed: ')
                    ]
                )
        (self.output / 'external-diagnostics.json').write_text(
            json.dumps(diagnostic, indent=2) + '\n'
        )

    def cleanup(self):
        errors = []
        if self.approvals_changed:
            try:
                if self.approval_snapshot is None:
                    kubectl(
                        'delete',
                        'configmap',
                        APPROVALS,
                        '-n',
                        self.namespace,
                        '--ignore-not-found',
                        timeout=20,
                    )
                else:
                    snapshot = copy.deepcopy(self.approval_snapshot)
                    snapshot['metadata'] = {
                        key: value
                        for key, value in snapshot['metadata'].items()
                        if key
                        in (
                            'name',
                            'namespace',
                            'labels',
                            'annotations',
                            'ownerReferences',
                        )
                    }
                    apply(snapshot)
            except Exception:
                errors.append(
                    'Restore original external-storage approval ConfigMap'
                )
        for name in self.source_names:
            try:
                kubectl(
                    'delete',
                    'configmap',
                    name,
                    '-n',
                    self.namespace,
                    '--ignore-not-found',
                    timeout=20,
                )
            except Exception:
                errors.append('Remove fixture policy source ConfigMap ' + name)
        errors.extend(super().cleanup())
        if self.gateway_created:
            try:
                kubectl(
                    'delete',
                    'namespace',
                    self.store_namespace,
                    '--wait=false',
                    timeout=20,
                )
                wait_for(
                    'external fixture namespace cleanup',
                    lambda: (
                        get('namespace', self.store_namespace, optional=True)
                        is None
                    ),
                    90,
                )
            except Exception:
                errors.append(
                    'Remove external-storage fixture namespace '
                    + self.store_namespace
                )
        return errors

    def run(self):
        success, failure = False, None
        try:
            for phase in [
                self.prepare,
                self.prepare_external_store,
                self.prepare_profiles,
                self.check_external_access,
                self.check_role_trust,
                self.check_rotation,
                self.check_expiry_and_recovery,
                self.check_adoption_uid_mismatch,
                self.check_adoption_forgery,
                self.check_adoption_revocation,
            ]:
                self.phase = phase.__name__
                print('PHASE: ' + self.phase, flush=True)
                phase()
            success = True
        except Exception as error:
            failure = (
                str(error)
                if isinstance(error, AssertionError)
                else type(error).__name__
            )
            print('FAIL: ' + failure, file=sys.stderr)
        finally:
            try:
                self.diagnostics()
            except Exception:
                print(
                    'Could not collect external receiver diagnostics',
                    file=sys.stderr,
                )
            try:
                cleanup_errors = self.cleanup()
            except Exception:
                cleanup_errors = [
                    'Cleanup failed; restore external approval ConfigMap and remove fixture namespaces'
                ]
            report = {
                'passed': success and not cleanup_errors,
                'failure': failure,
                'sourceHead': self.source_head,
                'phase': self.phase,
                'checks': self.results,
                'runtimeImages': self.runtime_images,
                'cleanupErrors': cleanup_errors,
                'scope': 'Real central-policy adoption, controller readiness, Kubernetes SAR and projected ServiceAccount identity, external STS/store enforcement and receiver rollout. Trusted UI identity headers; ingress login is not exercised.',
            }
            (self.output / 'results.json').write_text(
                json.dumps(report, indent=2) + '\n'
            )
        return 0 if report['passed'] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', default='kubeflow')
    parser.add_argument(
        '--output-dir', default='artifacts/external-artifact-authorization'
    )
    parser.add_argument('--timeout', type=int, default=600)
    args = parser.parse_args()
    args.legacy_frontend_image = ''
    return ExternalAcceptance(args).run()


if __name__ == '__main__':
    sys.exit(main())

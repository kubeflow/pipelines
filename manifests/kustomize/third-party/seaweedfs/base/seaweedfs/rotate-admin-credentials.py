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
"""Resumable rotation of the bundled SeaweedFS administrator credentials.

The generated Secret is the durable source of truth. The legacy Secret
remains at its original name for existing workflow references. No tenant
Secret or policy is modified. Errors intentionally exclude API bodies
and credential data.
"""

import base64
import hashlib
import json
import os
import secrets
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

ADMIN_SECRET = 'seaweedfs-admin-credentials'
LEGACY_SECRET = 'mlpipeline-minio-artifact'
REVISION = 'pipelines.kubeflow.org/storage-credential-revision'
PHASE = 'pipelines.kubeflow.org/storage-credential-phase'
CONSUMERS = (
    'ml-pipeline',
    'ml-pipeline-ui',
    'kubeflow-pipelines-profile-controller',
)
# Public access-key identifier and a fingerprint, never the shipped secret key.
DEFAULT_ACCESS_KEY = 'minio'
# This publicly shipped value is used only for a negative authentication probe.
# It is never used to provision or authenticate the replacement administrator.
DEFAULT_SECRET_KEY = 'minio123'
DEFAULT_FINGERPRINT = (
    '0c9d72cb381b9afa7023c36f330317ee8783fc8becf72e2c0610688e9afb544d'
)


class Retry(Exception):
    """Safe, non-secret progress or failure message."""


class APIError(Exception):
    def __init__(self, status):
        self.status = status


class Kubernetes:
    def __init__(self, namespace):
        self.namespace = namespace
        self.base = (
            'https://'
            + os.environ['KUBERNETES_SERVICE_HOST']
            + ':'
            + os.environ.get('KUBERNETES_SERVICE_PORT', '443')
        )
        self.account = Path('/var/run/secrets/kubernetes.io/serviceaccount')
        self.context = ssl.create_default_context(
            cafile=str(self.account / 'ca.crt')
        )

    def request(self, method, kind, name='', body=None, query=''):
        group = '/apis/apps/v1' if kind == 'deployments' else '/api/v1'
        path = f'{group}/namespaces/{self.namespace}/{kind}'
        if name:
            path += '/' + name
        if query:
            path += '?' + query
        headers = {
            'Authorization': 'Bearer '
            + (self.account / 'token').read_text().strip()
        }
        if body is not None:
            headers['Content-Type'] = (
                'application/merge-patch+json'
                if method == 'PATCH'
                else 'application/json'
            )
        request = urllib.request.Request(
            self.base + path,
            method=method,
            headers=headers,
            data=None if body is None else json.dumps(body).encode(),
        )
        try:
            with urllib.request.urlopen(
                request, context=self.context, timeout=20
            ) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise APIError(error.code) from None
        except (OSError, ValueError):
            raise Retry(
                'Kubernetes API unavailable; retrying without changing credentials'
            ) from None

    def get(self, kind, name):
        try:
            return self.request('GET', kind, name)
        except APIError as error:
            if error.status == 404:
                return None
            raise


def credentials(secret):
    data = (secret or {}).get('data', {})
    if 'accesskey' not in data and 'secretkey' not in data:
        return None
    try:
        pair = tuple(
            base64.b64decode(data[key], validate=True).decode()
            for key in ('accesskey', 'secretkey')
        )
        # The storage startup shell passes these as weed-shell arguments. Do not
        # accept whitespace/control characters or shell-command delimiters.
        if not all(
            value
            and all(
                c.isascii() and (c.isalnum() or c in '/+=_.-') for c in value
            )
            for value in pair
        ):
            raise ValueError()
        return pair
    except (KeyError, ValueError, UnicodeError):
        raise Retry(
            'Credential Secret is incomplete or contains unsupported characters; preserve and repair it'
        ) from None


def encoded(pair):
    return {
        key: base64.b64encode(value.encode()).decode()
        for key, value in zip(('accesskey', 'secretkey'), pair)
    }


def is_default(pair):
    return (
        pair is not None
        and hashlib.sha256(('\0'.join(pair)).encode()).hexdigest()
        == DEFAULT_FINGERPRINT
    )


def revision(secret):
    return secret['metadata']['annotations'][REVISION]


def uses_secret(deployment, name):
    spec = deployment['spec']['template']['spec']
    for container in spec.get('containers', []) + spec.get(
        'initContainers', []
    ):
        if any(
            item.get('secretRef', {}).get('name') == name
            for item in container.get('envFrom', [])
        ):
            return True
        if any(
            item.get('valueFrom', {}).get('secretKeyRef', {}).get('name')
            == name
            for item in container.get('env', [])
        ):
            return True
    return any(
        item.get('secret', {}).get('secretName') == name
        for item in spec.get('volumes', [])
    )


class Migration:
    def __init__(self, api, storage_factory):
        self.api = api
        self.storage_factory = storage_factory
        self.storage_pair = None
        self.storage = None

    def phase(self, state, value):
        if state['metadata'].get('annotations', {}).get(PHASE) != value:
            self.api.request(
                'PATCH',
                'secrets',
                ADMIN_SECRET,
                {
                    'metadata': {
                        'resourceVersion': state['metadata']['resourceVersion'],
                        'annotations': {PHASE: value},
                    }
                },
            )
            print(value, flush=True)

    def prepare(self):
        state = self.api.get('secrets', ADMIN_SECRET)
        if state is None:
            raise Retry(
                'Waiting for the seaweedfs-admin-credentials Secret manifest'
            )
        current = credentials(state)
        annotations = state['metadata'].get('annotations', {})
        if current is not None and REVISION in annotations:
            if is_default(current):
                raise Retry(
                    'Generated admin Secret contains the shipped default; refusing to use it'
                )
            old = credentials(self.api.get('secrets', LEGACY_SECRET))
            if (
                old != current
                and (old is None or is_default(old))
                and annotations.get(PHASE) == 'DefaultRevoked'
            ):
                # A reconciler may restore stale declarative defaults after the
                # first migration. Roll consumers again, even with the same key.
                return self.api.request(
                    'PATCH',
                    'secrets',
                    ADMIN_SECRET,
                    {
                        'metadata': {
                            'resourceVersion': state['metadata'][
                                'resourceVersion'
                            ],
                            'annotations': {
                                REVISION: secrets.token_hex(16),
                                PHASE: 'ReplacementPrepared',
                            },
                        }
                    },
                )
            return state
        old = credentials(self.api.get('secrets', LEGACY_SECRET))
        if current is None:
            current = (
                old
                if old is not None and not is_default(old)
                else ('kfp-' + secrets.token_hex(16), secrets.token_hex(32))
            )
        if is_default(current):
            raise Retry(
                'Admin Secret contains the shipped default; refusing to use it'
            )
        return self.api.request(
            'PATCH',
            'secrets',
            ADMIN_SECRET,
            {
                'metadata': {
                    'resourceVersion': state['metadata']['resourceVersion'],
                    'annotations': {
                        REVISION: secrets.token_hex(16),
                        PHASE: 'ReplacementPrepared',
                    },
                },
                'data': encoded(current),
            },
        )

    def storage_ready(self):
        deployment = self.api.get('deployments', 'seaweedfs')
        if not deployment or not uses_secret(deployment, ADMIN_SECRET):
            raise Retry(
                'Waiting for the SeaweedFS deployment to use the replacement Secret'
            )
        self.rollout_ready(deployment)
        pods = self.api.request(
            'GET', 'pods', query='labelSelector=app%3Dseaweedfs'
        )['items']
        # An old container restart can re-register the default through postStart.
        if not pods or any(
            not uses_secret({'spec': {'template': pod}}, ADMIN_SECRET)
            for pod in pods
        ):
            raise Retry(
                'Waiting for all old SeaweedFS pods to terminate before revocation'
            )

    def rollout_ready(self, deployment, expected_revision=None):
        desired = deployment['spec'].get('replicas', 1)
        status = deployment.get('status', {})
        if (
            status.get('observedGeneration', 0)
            < deployment['metadata']['generation']
            or status.get('updatedReplicas', 0) != desired
            or status.get('replicas', 0) != desired
            or status.get('availableReplicas', 0) != desired
        ):
            raise Retry(
                'Waiting for deployment '
                + deployment['metadata']['name']
                + ' to finish rolling out'
            )
        if expected_revision is not None:
            selector = deployment['spec']['selector']
            if selector.get('matchExpressions'):
                raise Retry(
                    'Custom deployment selectors require explicit migration review'
                )
            query = urllib.parse.urlencode(
                {
                    'labelSelector': ','.join(
                        key + '=' + value
                        for key, value in selector['matchLabels'].items()
                    )
                }
            )
            pods = self.api.request('GET', 'pods', query=query)['items']
            if any(
                pod.get('status', {}).get('phase')
                not in ('Succeeded', 'Failed')
                and pod['metadata'].get('annotations', {}).get(REVISION)
                != expected_revision
                for pod in pods
            ):
                raise Retry('Waiting for old consumer pods to terminate')

    def publish(self, pair):
        old = self.api.get('secrets', LEGACY_SECRET)
        existing = credentials(old)
        if existing == pair:
            return
        if existing is not None and not is_default(existing):
            raise Retry(
                'Legacy Secret changed to custom credentials; reconcile the two Secrets explicitly rather than overwrite it'
            )
        if old is None:
            self.api.request(
                'POST',
                'secrets',
                body={
                    'apiVersion': 'v1',
                    'kind': 'Secret',
                    'type': 'Opaque',
                    'metadata': {'name': LEGACY_SECRET},
                    'data': encoded(pair),
                },
            )
        else:
            self.api.request(
                'PATCH',
                'secrets',
                LEGACY_SECRET,
                {
                    'metadata': {
                        'resourceVersion': old['metadata']['resourceVersion']
                    },
                    'data': encoded(pair),
                },
            )

    def roll_deployment(self, name, value):
        deployment = self.api.get('deployments', name)
        if (
            deployment['spec']['template']
            .get('metadata', {})
            .get('annotations', {})
            .get(REVISION)
            != value
        ):
            self.api.request(
                'PATCH',
                'deployments',
                name,
                {
                    'metadata': {
                        'resourceVersion': deployment['metadata'][
                            'resourceVersion'
                        ]
                    },
                    'spec': {
                        'template': {
                            'metadata': {'annotations': {REVISION: value}}
                        }
                    },
                },
            )
            raise Retry(
                'Rolling deployment '
                + name
                + ' onto the replacement credential'
            )
        self.rollout_ready(deployment, value)

    def roll_consumers(self, value):
        for name in CONSUMERS:
            deployment = self.api.get('deployments', name)
            if deployment is not None and uses_secret(
                deployment, LEGACY_SECRET
            ):
                self.roll_deployment(name, value)

    def reconcile(self):
        state = self.prepare()
        pair = credentials(state)
        self.storage_ready()
        if self.storage_pair != pair:
            if self.storage is not None:
                self.storage.close()
            self.storage = self.storage_factory(pair)
            self.storage_pair = pair
        storage = self.storage
        storage.verify_registration()
        # The separate IAM listener does not refresh its authentication cache
        # when postStart adds a key through the filer gRPC API. Restart once
        # after S3 confirms registration, before changing any consumer Secret.
        # Secret UID is durable and independent of later consumer-only rollouts.
        self.roll_deployment('seaweedfs', state['metadata']['uid'])
        storage.verify()
        self.publish(pair)
        self.roll_consumers(revision(state))
        # Re-read to detect a concurrent operator edit before revoking a key.
        if (
            credentials(self.api.get('secrets', ADMIN_SECRET)) != pair
            or credentials(self.api.get('secrets', LEGACY_SECRET)) != pair
        ):
            raise Retry(
                'Credentials changed during rollout; restarting verification'
            )
        storage.revoke_default()
        storage.verify()
        self.phase(state, 'DefaultRevoked')


class Storage:
    def __init__(self, pair):
        try:
            import botocore.session
        except ModuleNotFoundError as error:
            if error.name != 'botocore':
                raise
            import awscli  # noqa: F401
            import botocore.session
        from botocore.config import Config

        session = botocore.session.get_session()
        common = {
            'region_name': 'us-east-1',
            'aws_access_key_id': pair[0],
            'aws_secret_access_key': pair[1],
            'config': Config(
                connect_timeout=5, read_timeout=10, retries={'max_attempts': 2}
            ),
        }
        self.pair = pair
        self.default_s3 = session.create_client(
            's3',
            endpoint_url='http://seaweedfs:8333',
            **{
                **common,
                'aws_access_key_id': DEFAULT_ACCESS_KEY,
                'aws_secret_access_key': DEFAULT_SECRET_KEY,
            },
        )
        self.s3 = session.create_client(
            's3', endpoint_url='http://seaweedfs:8333', **common
        )
        self.iam = session.create_client(
            'iam', endpoint_url='http://seaweedfs:8111', **common
        )

    def verify_registration(self):
        self.s3.head_bucket(Bucket='mlpipeline')

    def verify(self):
        self.verify_registration()
        self.iam.get_user(UserName='kubeflow-admin')

    def close(self):
        for client in (self.s3, self.iam, self.default_s3):
            client.close()

    def default_accepted(self):
        from botocore.exceptions import ClientError

        try:
            self.default_s3.head_bucket(Bucket='mlpipeline')
            return True
        except ClientError as error:
            # HEAD responses can omit the XML error code. Only an explicit
            # authentication/authorization rejection is a successful negative
            # probe; transport errors, 404s and 5xx must retry.
            if error.response.get('ResponseMetadata', {}).get(
                'HTTPStatusCode'
            ) in (401, 403):
                return False
            raise

    def revoke_default(self):
        if not self.default_accepted():
            return
        if self.pair[0] == DEFAULT_ACCESS_KEY:
            raise Retry(
                'Default remains accepted under an operator-owned access-key ID; inspect storage identities'
            )
        self.iam.delete_access_key(
            UserName='kubeflow-admin', AccessKeyId=DEFAULT_ACCESS_KEY
        )
        if self.default_accepted():
            raise Retry(
                'Waiting for storage to reject the default credential after revocation'
            )


def main():
    migration = Migration(Kubernetes(os.environ['POD_NAMESPACE']), Storage)
    ready = Path('/tmp/credentials-ready')
    while True:
        try:
            migration.reconcile()
            ready.touch()
        except Retry as error:
            ready.unlink(missing_ok=True)
            print(str(error), flush=True)
        except Exception:
            ready.unlink(missing_ok=True)
            # API/SDK exceptions can include credentials or entire Secret bodies.
            print(
                'Credential reconciliation failed; inspect RBAC, storage health and deployment readiness. Retrying.',
                flush=True,
            )
        time.sleep(10)


if __name__ == '__main__':
    main()

# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Acceptance against a disposable, deployed multi-user KFP installation.

Uses real Profiles, IAM, Kubernetes reconciliation, and the KFP
authorization API. Port-forwarding supplies the trusted user header
normally set by ingress; this tests authorization, not ingress login.
Never run against production.
"""

import argparse
import base64
import copy
import hashlib
import json
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

import boto3
from botocore.config import Config

BUCKET = 'mlpipeline'
CREDENTIAL_SECRET = 'mlpipeline-minio-artifact'
CONTROLLER = 'kubeflow-pipelines-profile-controller'
RECEIVER = 'ml-pipeline-ui-artifact'
QUOTA = 'artifact-acceptance-hold-rollout'
NAMESPACE_HEADER = 'x-kfp-artifact-namespace'
BINDING_HEADER = 'x-kfp-artifact-credential-binding'
RECEIVER_UPGRADE_MESSAGE = (
    'Namespace artifact service upgrade required. Wait for its rollout to complete; '
    'if this persists, check the profile controller and namespace artifact service '
    'deployment. No broader credentials were used.'
)

# Run from a real allowed UI Pod or an unprivileged network probe. The request
# (including credential binding) is stdin, never a command argument or log line.
NODE_REQUEST = r"""
const fs = require('fs');
const request = JSON.parse(fs.readFileSync(0, 'utf8'));
fetch(request.url, {
  headers: request.headers || {},
  signal: AbortSignal.timeout(request.timeout || 10000),
}).then(async response => {
  console.log(JSON.stringify({status: response.status,
    body: (await response.text()).slice(0, 4096),
    version: response.headers.get('x-kfp-artifact-proxy-version'),
    retryAfter: response.headers.get('retry-after')}));
}).catch(error => {
  console.log(JSON.stringify({networkError: error.name,
    code: error.cause?.code || null}));
});
"""


def kubectl(*args, payload=None, optional=False, timeout=90):
    result = subprocess.run(
        ['kubectl', *args],
        input=json.dumps(payload) if payload else None,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    if result.returncode and not optional:
        # kubectl failures can echo submitted Secret content. Do not print stderr.
        raise RuntimeError(
            f'kubectl {args[0]} failed (exit {result.returncode})'
        )
    return result


def get(kind, name=None, namespace=None, optional=False, selector=None):
    args = ['get', kind]
    if name:
        args.append(name)
    if namespace:
        args += ['-n', namespace]
    if selector:
        args += ['-l', selector]
    if optional:
        args.append('--ignore-not-found')
    result = kubectl(*args, '-o', 'json')
    return json.loads(result.stdout) if result.stdout.strip() else None


def apply(resource):
    kubectl('apply', '-f', '-', payload=resource)


def wait_for(description, condition, timeout=300):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        value = condition()
        if value:
            return value
        time.sleep(2)
    raise AssertionError(f'Timed out waiting for {description}')


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def decode_secret(secret, key):
    return base64.b64decode(secret['data'][key], validate=True).decode()


def document(value):
    return (
        json.loads(urllib.parse.unquote(value))
        if isinstance(value, str)
        else value
    )


def receiver_pods_converged(pods, image, superseded_uids):
    """Deployment readiness does not imply draining Pods have disappeared."""
    if not pods:
        return False
    for pod in pods:
        metadata = pod['metadata']
        if (
            not metadata.get('uid')
            or metadata['uid'] in superseded_uids
            or metadata.get('deletionTimestamp')
        ):
            return False
        receiver = next(
            (
                container
                for container in pod['spec']['containers']
                if container['name'] == RECEIVER
            ),
            None,
        )
        status = next(
            (
                container
                for container in pod.get('status', {}).get(
                    'containerStatuses', []
                )
                if container['name'] == RECEIVER
            ),
            {},
        )
        if (
            not receiver
            or receiver['image'] != image
            or not status.get('ready')
        ):
            return False
    return True


class Acceptance:
    def __init__(self, args):
        self.namespace = args.namespace
        self.timeout = args.timeout
        self.output = Path(args.output_dir)
        self.output.mkdir(parents=True, exist_ok=True)
        self.run_id = 'artifact-accept-' + uuid.uuid4().hex[:8]
        self.tenants = [self.run_id + '-a', self.run_id + '-b']
        self.users = {name: name + '@example.com' for name in self.tenants}
        self.shared_viewer = self.run_id + '-viewer@example.com'
        self.results = []
        self.last_response = None
        self.last_probe_response = None
        self.runtime_images = []
        self.phase = 'initialization'
        self.source_head = subprocess.run(
            ['git', 'rev-parse', 'HEAD'],
            cwd=Path(__file__).resolve().parents[2],
            capture_output=True,
            text=True,
            check=True,
        ).stdout.strip()
        self.forwarders = []
        self.objects = []
        self.probes = []
        self.controller_replicas = None
        self.iam = None
        self.s3 = None
        self.recovery_bucket_policy = None
        self.ui_pod = None
        self.ui_container = None
        self.controller_paused = False
        self.legacy_image = args.legacy_frontend_image
        self.supports_legacy_ui = (
            Path(__file__).resolve().parents[2]
            / 'backend/api/v1beta1/swagger/auth.swagger.json'
        ).exists()

    def passed(self, name, **evidence):
        self.results.append({'name': name, 'passed': True, **evidence})
        print('PASS: ' + name, flush=True)

    def capture_images(self, namespace, selector):
        pods = get('pods', namespace=namespace, selector=selector)['items']
        for pod in pods:
            for status in pod.get('status', {}).get('containerStatuses', []):
                if status.get('imageID'):
                    evidence = {
                        'namespace': namespace,
                        'pod': pod['metadata']['name'],
                        'container': status['name'],
                        'image': status['image'],
                        'imageID': status['imageID'],
                    }
                    if evidence not in self.runtime_images:
                        self.runtime_images.append(evidence)

    def receiver_pod_uids(self, namespace):
        return {
            pod['metadata']['uid']
            for pod in get(
                'pods', namespace=namespace, selector='app=' + RECEIVER
            )['items']
        }

    def wait_receiver_pods(self, namespace, image, superseded_uids):
        wait_for(
            'superseded receiver Pods to disappear',
            lambda: receiver_pods_converged(
                get('pods', namespace=namespace, selector='app=' + RECEIVER)[
                    'items'
                ],
                image,
                superseded_uids,
            ),
            min(self.timeout, 60),
        )

    def wait_artifact_response(
        self, namespace, key, description, status, body=None, retry_after=None
    ):

        def received():
            try:
                response = self.request(namespace, key)
            except (
                urllib.error.URLError,
                TimeoutError,
                ConnectionError,
            ) as error:
                self.last_response = {
                    'namespace': namespace,
                    'networkError': type(error).__name__,
                }
                return False
            if (
                response['status'] != status
                or (body is not None and response['body'] != body)
                or (
                    retry_after is not None
                    and response['retryAfter'] != retry_after
                )
            ):
                return False
            return response

        return wait_for(description, received, min(self.timeout, 60))

    def wait_receiver_upgrade_required(self, namespace, key):
        response = self.wait_artifact_response(
            namespace,
            key,
            'shared UI to replace the old receiver response with an upgrade-required error',
            status=503,
            body=RECEIVER_UPGRADE_MESSAGE,
            retry_after='5',
        )
        require(
            response.get('contentType') == 'text/plain; charset=utf-8'
            and response.get('contentDisposition') == 'attachment'
            and response.get('contentTypeOptions') == 'nosniff'
            and response.get('proxyError') == 'receiver-upgrade-required',
            'Old receiver rejection must be a hardened plain-text upgrade-required error',
        )
        return response

    def port_forward(self, resource, remote_port):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        process = subprocess.Popen(
            [
                'kubectl',
                '-n',
                self.namespace,
                'port-forward',
                resource,
                '--address',
                '127.0.0.1',
                f'{port}:{remote_port}',
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        self.forwarders.append(process)

        def connected():
            if process.poll() is not None:
                raise RuntimeError('kubectl port-forward exited')
            try:
                with socket.create_connection(('127.0.0.1', port), timeout=1):
                    return True
            except OSError:
                return False

        wait_for('port-forward', connected, 30)
        return f'http://127.0.0.1:{port}'

    def request(
        self,
        namespace,
        key,
        user=None,
        headers=None,
        base_url=None,
        raw=False,
        peek=None,
    ):
        parameters = {'namespace': namespace}
        if raw:
            path = (
                '/artifacts/minio/'
                + BUCKET
                + '/'
                + urllib.parse.quote(key, safe='/')
            )
        else:
            path = '/artifacts/get'
            parameters.update(source='minio', bucket=BUCKET, key=key)
        if peek is not None:
            parameters['peek'] = peek
        query = urllib.parse.urlencode(parameters)
        request_headers = {'kubeflow-userid': user or self.users[namespace]}
        request_headers.update(headers or {})
        request = urllib.request.Request(
            (base_url or self.ui) + path + '?' + query,
            headers=request_headers,
        )
        try:
            response = urllib.request.urlopen(request, timeout=30)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            self.last_response = {
                'namespace': namespace,
                'status': response.status,
                'retryAfter': response.headers.get('Retry-After'),
            }
            return {
                'status': response.status,
                'body': response.read(4096).decode(errors='replace'),
                'contentType': response.headers.get('Content-Type'),
                'contentDisposition': response.headers.get(
                    'Content-Disposition'
                ),
                'contentTypeOptions': response.headers.get(
                    'X-Content-Type-Options'
                ),
                'proxyError': response.headers.get(
                    'X-Kfp-Artifact-Proxy-Error'
                ),
                'retryAfter': response.headers.get('Retry-After'),
            }

    def exec_request(self, namespace, pod, container, url, headers=None):
        payload = json.dumps(
            {'url': url, 'headers': headers or {}, 'timeout': 8000}
        )
        result = subprocess.run(
            [
                'kubectl',
                '-n',
                namespace,
                'exec',
                '-i',
                pod,
                '-c',
                container,
                '--',
                'node',
                '-e',
                NODE_REQUEST,
            ],
            input=payload,
            capture_output=True,
            text=True,
            timeout=30,
        )
        require(result.returncode == 0, 'Network probe process failed')
        response = json.loads(result.stdout)
        self.last_probe_response = {
            key: response.get(key)
            for key in (
                'status',
                'version',
                'retryAfter',
                'networkError',
                'code',
            )
        }
        return response

    def receiver_request(self, namespace, key, headers=None, legacy=False):
        path = '' if legacy else '/artifact-proxy/v1'
        query = urllib.parse.urlencode(
            {
                'source': 'minio',
                'bucket': BUCKET,
                'key': key,
            }
        )
        url = f'http://{RECEIVER}.{namespace}:80{path}/artifacts/get?{query}'
        return self.exec_request(
            self.namespace, self.ui_pod, self.ui_container, url, headers
        )

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
        actual_namespace = get('namespace', namespace)
        uid = actual_namespace['metadata']['uid']
        require(
            record['namespace'] == namespace
            and record['namespaceUID'] == uid
            and record['version'] == 1
            and record['bucket'] == BUCKET,
            'Central authorization record does not identify the real namespace',
        )
        require(
            any(
                owner.get('kind') == 'Namespace' and owner.get('uid') == uid
                for owner in secret['metadata'].get('ownerReferences', [])
            ),
            'Central authorization record lacks the actual Namespace owner',
        )
        return record

    def wait_state(self, namespace, state):

        def matches():
            record = self.record(namespace)
            return record if record and record['state'] == state else None

        return wait_for(
            f'{namespace} authorization {state}', matches, self.timeout
        )

    def scale_controller(self, replicas):
        kubectl(
            'scale',
            'deployment/' + CONTROLLER,
            '-n',
            self.namespace,
            '--replicas=' + str(replicas),
        )
        self.controller_paused = replicas == 0
        if replicas == 0:
            wait_for(
                'profile webhook to stop',
                lambda: (
                    not get(
                        'pods',
                        namespace=self.namespace,
                        selector='app=' + CONTROLLER,
                    )['items']
                ),
                self.timeout,
            )
        else:
            kubectl(
                'rollout',
                'status',
                'deployment/' + CONTROLLER,
                '-n',
                self.namespace,
                '--timeout=' + str(self.timeout) + 's',
                timeout=self.timeout + 10,
            )

    def prepare(self):
        controller = get('deployment', CONTROLLER, self.namespace)
        self.controller_replicas = controller['spec'].get('replicas', 1)
        require(
            self.controller_replicas > 0,
            'The deployed profile webhook must already be running',
        )
        deployment = get('deployment', 'ml-pipeline-ui', self.namespace)
        container = next(
            item
            for item in deployment['spec']['template']['spec']['containers']
            if item['name'] == 'ml-pipeline-ui'
        )
        self.ui_template = deployment['spec']['template']
        self.ui_container = container['name']
        self.ui_image = container['image']
        pods = get(
            'pods', namespace=self.namespace, selector='app=ml-pipeline-ui'
        )
        self.ui_pod = next(
            pod['metadata']['name']
            for pod in pods['items']
            if pod['status'].get('phase') == 'Running'
            and not pod['metadata'].get('deletionTimestamp')
        )
        self.ui = self.port_forward('service/ml-pipeline-ui', 80)
        s3_url = self.port_forward('deployment/seaweedfs', 8333)
        iam_url = self.port_forward('deployment/seaweedfs', 8111)
        secret = get('secret', CREDENTIAL_SECRET, self.namespace)
        session = boto3.Session(
            aws_access_key_id=decode_secret(secret, 'accesskey'),
            aws_secret_access_key=decode_secret(secret, 'secretkey'),
            region_name='us-east-1',
        )
        config = Config(
            connect_timeout=5, read_timeout=15, retries={'max_attempts': 2}
        )
        self.s3 = session.client('s3', endpoint_url=s3_url, config=config)
        self.iam = session.client('iam', endpoint_url=iam_url, config=config)
        self.s3.head_bucket(Bucket=BUCKET)
        self.capture_images(self.namespace, 'app=ml-pipeline-ui')
        self.capture_images(self.namespace, 'app=seaweedfs')
        self.passed(
            'deployed multi-user stack reachable', uiImage=self.ui_image
        )

    def pending_rollout(self):
        self.scale_controller(0)
        for namespace in self.tenants:
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
                'Profile namespace',
                lambda: get('namespace', namespace, optional=True),
                self.timeout,
            )
            # The standalone CI Profile controller does not install the dashboard's
            # kubeflow-admin ClusterRole. Grant only artifact viewing to this User;
            # the real API SAR must still deny access to the other namespace.
            apply(
                {
                    'apiVersion': 'rbac.authorization.k8s.io/v1',
                    'kind': 'Role',
                    'metadata': {
                        'name': 'artifact-acceptance-viewer',
                        'namespace': namespace,
                    },
                    'rules': [
                        {
                            'apiGroups': ['kubeflow.org'],
                            'resources': ['viewers'],
                            'verbs': ['get'],
                        }
                    ],
                }
            )
            apply(
                {
                    'apiVersion': 'rbac.authorization.k8s.io/v1',
                    'kind': 'RoleBinding',
                    'metadata': {
                        'name': 'artifact-acceptance-viewer',
                        'namespace': namespace,
                    },
                    'roleRef': {
                        'apiGroup': 'rbac.authorization.k8s.io',
                        'kind': 'Role',
                        'name': 'artifact-acceptance-viewer',
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
        self.configure_shared_viewer()
        # Quota is a real scheduling blocker: neither readiness nor the controller
        # record is fabricated. Releasing it must let normal reconciliation finish.
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'ResourceQuota',
                'metadata': {'name': QUOTA, 'namespace': self.tenants[0]},
                'spec': {'hard': {'pods': '0'}},
            }
        )
        self.scale_controller(self.controller_replicas)
        self.wait_state(self.tenants[0], 'pending')

        def pending_response():
            response = self.request(self.tenants[0], 'model/historical.txt')
            require(
                response['status'] != 200,
                'Pending rollout incorrectly returned an artifact',
            )
            # API authorization can lag behind the fixture RoleBinding.
            return response if response['status'] == 503 else None

        response = wait_for(
            'Profile owner authorization and pending response',
            pending_response,
            self.timeout,
        )
        require(
            response['retryAfter'] == '5',
            'Pending receiver rollout must return retryable HTTP503',
        )
        self.passed('genuine pending rollout is retryable and fails closed')
        kubectl('delete', 'resourcequota', QUOTA, '-n', self.tenants[0])
        for namespace in self.tenants:
            record = self.wait_state(namespace, 'ready')
            self.capture_images(namespace, 'app=' + RECEIVER)
            self.passed(
                'controller publishes ready for actual namespace',
                namespace=namespace,
                namespaceUID=record['namespaceUID'],
            )

    def configure_shared_viewer(self):
        apply(
            {
                'apiVersion': 'rbac.authorization.k8s.io/v1',
                'kind': 'RoleBinding',
                'metadata': {
                    'name': 'artifact-acceptance-readonly-viewer',
                    'namespace': self.tenants[0],
                },
                'roleRef': {
                    'apiGroup': 'rbac.authorization.k8s.io',
                    'kind': 'Role',
                    'name': 'artifact-acceptance-viewer',
                },
                'subjects': [
                    {
                        'apiGroup': 'rbac.authorization.k8s.io',
                        'kind': 'User',
                        'name': self.shared_viewer,
                    }
                ],
            }
        )

    def seed_existing_roots(self):
        self.keys = {}
        self.expected_policies = {}
        for namespace in self.tenants:
            historical = (
                f'model/{self.run_id}/{namespace[-1]}-training/model.txt'
            )
            private = f'private-artifacts/{namespace}/{self.run_id}/model.txt'
            old_private = f'private/{namespace}/{self.run_id}/model.txt'
            self.keys[namespace] = {
                'historical': historical,
                'private': private,
                'old_private': old_private,
            }
            policy_name = 'KubeflowProject' + namespace
            policy = document(
                self.iam.get_user_policy(
                    UserName=namespace, PolicyName=policy_name
                )['PolicyDocument']
            )
            policy['Statement'].append(
                {
                    'Sid': 'ExistingArtifactRoot',
                    'Effect': 'Allow',
                    'Action': ['s3:GetObject'],
                    'Resource': ['arn:aws:s3:::' + BUCKET + '/' + historical],
                }
            )
            self.iam.put_user_policy(
                UserName=namespace,
                PolicyName=policy_name,
                PolicyDocument=json.dumps(policy),
            )
            # SeaweedFS serializes singleton Action/Resource lists as strings.
            # Preserve the stored policy, not the submitted JSON representation.
            stored_policy = document(
                self.iam.get_user_policy(
                    UserName=namespace, PolicyName=policy_name
                )['PolicyDocument']
            )
            require(
                any(
                    statement.get('Sid') == 'ExistingArtifactRoot'
                    and statement.get('Effect') == 'Allow'
                    and statement.get('Action')
                    in ('s3:GetObject', ['s3:GetObject'])
                    and statement.get('Resource')
                    in (
                        'arn:aws:s3:::' + BUCKET + '/' + historical,
                        ['arn:aws:s3:::' + BUCKET + '/' + historical],
                    )
                    and not statement.get('Condition')
                    for statement in stored_policy.get('Statement', [])
                ),
                'Storage did not retain the administrator historical-root grant',
            )
            self.expected_policies[namespace] = stored_policy
            for name, key in self.keys[namespace].items():
                content = namespace + ':' + name
                self.s3.put_object(
                    Bucket=BUCKET, Key=key, Body=content.encode()
                )
                self.objects.append(key)
        self.seed_shared_root()
        self.seed_legacy_shared_root()
        # Reconciliation is triggered again during mixed-version recovery and
        # credential rotation below; both must preserve these existing grants.

    def seed_shared_root(self):
        self.shared = f'shared/{self.run_id}/model.txt'
        # shared/ is writable by namespace credentials, not administrator-curated.
        self.namespace_storage_client(self.tenants[1]).put_object(
            Bucket=BUCKET,
            Key=self.shared,
            Body=b'tenant-written shared artifact',
        )
        self.objects.append(self.shared)

    def seed_legacy_shared_root(self):
        # Before #11965 the shared Argo repository stored tenant workflow data
        # under artifacts/<workflow>/<date>/<pod>, without a namespace segment.
        self.legacy_keys = {}
        for namespace in self.tenants:
            suffix = namespace[-1]
            key = (
                f'artifacts/{self.run_id}-workflow-{suffix}/2025/08/19/'
                f'{self.run_id}-pod-{suffix}/model.txt'
            )
            self.legacy_keys[namespace] = key
            self.s3.put_object(
                Bucket=BUCKET,
                Key=key,
                Body=(namespace + ':legacy-shared-root').encode(),
            )
            self.objects.append(key)

    def namespace_storage_client(self, namespace):
        secret = get('secret', CREDENTIAL_SECRET, namespace)
        return boto3.Session(
            aws_access_key_id=decode_secret(secret, 'accesskey'),
            aws_secret_access_key=decode_secret(secret, 'secretkey'),
            region_name='us-east-1',
        ).client(
            's3',
            endpoint_url=self.s3.meta.endpoint_url,
            config=Config(
                connect_timeout=5, read_timeout=15, retries={'max_attempts': 2}
            ),
        )

    def check_legacy_shared_root(self, base_url=None, mode='managed auto'):
        endpoint = {'base_url': base_url} if base_url is not None else {}
        for namespace in self.tenants:
            record = self.record(namespace)
            require(
                record
                and record.get('version') == 1
                and record.get('state') == 'ready',
                'Legacy-root isolation requires an actually ready managed-v1 record',
            )
            client = self.namespace_storage_client(namespace)
            for owner, key in self.legacy_keys.items():
                with client.get_object(Bucket=BUCKET, Key=key)['Body'] as body:
                    require(
                        body.read() == (owner + ':legacy-shared-root').encode(),
                        'Stock namespace credentials no longer read the real legacy fixture object',
                    )
                listing = client.list_objects_v2(
                    Bucket=BUCKET, Prefix=key.rsplit('/', 1)[0] + '/'
                )
                require(
                    any(
                        item.get('Key') == key
                        for item in listing.get('Contents', [])
                    ),
                    'Stock namespace credentials no longer list the real legacy fixture object',
                )
        self.passed(
            'stock namespace credentials retain shared legacy workflow object and listing access'
        )

        for namespace in self.tenants:
            for owner, key in self.legacy_keys.items():
                directory = key.rsplit('/', 1)[0] + '/'
                for route, target, kwargs in [
                    ('preview', key, {}),
                    ('raw', key, {'raw': True}),
                    ('directory-preview', directory, {'peek': 4096}),
                    ('directory-download', directory, {'raw': True}),
                ]:
                    response = self.request(
                        namespace, target, **endpoint, **kwargs
                    )
                    require(
                        response['status'] == 403
                        and owner + ':legacy-shared-root'
                        not in response['body'],
                        mode.capitalize()
                        + ' exposed or failed to authorize the shared legacy workflow '
                        + route,
                    )
            for key in ['artifacts', 'artifacts/']:
                response = self.request(namespace, key, peek=4096, **endpoint)
                require(
                    response['status'] == 403
                    and all(
                        tenant + ':legacy-shared-root' not in response['body']
                        for tenant in self.tenants
                    ),
                    mode.capitalize()
                    + ' did not retain ownership validation for the legacy root itself',
                )
        self.passed(
            mode
            + ' blocks ambiguous legacy workflow previews, downloads and directory listings'
        )

    def check_explicit_storage(self):
        name = self.run_id + '-storage-ui'
        self.probes.append((self.namespace, name))
        spec = copy.deepcopy(self.ui_template['spec'])
        # Keep the allowed UI network identity without joining shared Service
        # endpoints. The probe uses a separate Pod port-forward throughout.
        spec['readinessGates'] = [
            {'conditionType': 'artifact-acceptance/not-a-service-endpoint'}
        ]
        spec['restartPolicy'] = 'Never'
        container = next(
            item
            for item in spec['containers']
            if item['name'] == self.ui_container
        )
        mode_name = 'KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE'
        container['env'] = [
            item
            for item in container.get('env', [])
            if item['name'] != mode_name
        ]
        container['env'].append({'name': mode_name, 'value': 'storage'})
        container['imagePullPolicy'] = 'IfNotPresent'
        apply(
            {
                'apiVersion': 'v1',
                'kind': 'Pod',
                'metadata': {
                    'name': name,
                    'namespace': self.namespace,
                    'labels': {'app': 'ml-pipeline-ui'},
                    'annotations': {'sidecar.istio.io/inject': 'false'},
                },
                'spec': spec,
            }
        )

        def running_probe():
            pod = get('pod', name, self.namespace)
            return (
                pod
                if any(
                    item['name'] == self.ui_container
                    and 'running' in item.get('state', {})
                    for item in pod.get('status', {}).get(
                        'containerStatuses', []
                    )
                )
                else None
            )

        pod = wait_for(
            'explicit storage UI container',
            running_probe,
            self.timeout,
        )
        observed = next(
            item
            for item in pod['spec']['containers']
            if item['name'] == self.ui_container
        )
        require(
            observed['image'] == self.ui_image
            and [
                item
                for item in observed.get('env', [])
                if item['name'] == mode_name
            ]
            == [{'name': mode_name, 'value': 'storage'}]
            and pod['spec'].get('serviceAccountName', 'default')
            == spec.get('serviceAccountName', 'default'),
            'Explicit storage probe must use the candidate UI image, storage mode and original ServiceAccount',
        )
        storage_ui = self.port_forward('pod/' + name, 3000)
        require(
            storage_ui != self.ui,
            'Explicit storage probe must not use the default UI endpoint',
        )
        first, second = self.tenants

        def private_read():
            try:
                response = self.request(
                    first, self.keys[first]['private'], base_url=storage_ui
                )
                return response if response['status'] == 200 else None
            except (OSError, urllib.error.URLError):
                return None

        response = wait_for(
            'explicit storage UI to read its namespace artifact',
            private_read,
            self.timeout,
        )
        require(
            response
            and response['status'] == 200
            and response['body'] == first + ':private',
            'Explicit storage UI returned unexpected private artifact bytes',
        )
        response = self.request(
            first, self.keys[second]['private'], base_url=storage_ui
        )
        require(
            response['status'] in (403, 500)
            and 'Access Denied' in response['body']
            and second + ':private' not in response['body'],
            'Explicit storage UI did not preserve cross-tenant storage denial',
        )
        response = self.request(
            second,
            self.keys[second]['private'],
            user=self.users[first],
            base_url=storage_ui,
        )
        require(
            response['status'] == 403
            and second + ':private' not in response['body'],
            'Explicit storage UI did not preserve claimed-namespace denial',
        )
        self.passed(
            'explicit storage UI preserves private access and cross-tenant namespace isolation'
        )
        self.check_legacy_shared_root(
            base_url=storage_ui, mode='explicit storage'
        )
        self.capture_images(self.namespace, 'app=ml-pipeline-ui')
        kubectl('delete', 'pod', name, '-n', self.namespace, '--wait=false')

    def check_shared_access(self):
        first, second = self.tenants

        def permitted(namespace, resource, verb, group='kubeflow.org'):
            review = kubectl(
                'create',
                '-f',
                '-',
                '-o',
                'json',
                payload={
                    'apiVersion': 'authorization.k8s.io/v1',
                    'kind': 'SubjectAccessReview',
                    'spec': {
                        'user': self.shared_viewer,
                        'resourceAttributes': {
                            'namespace': namespace,
                            'group': group,
                            'resource': resource,
                            'verb': verb,
                        },
                    },
                },
            )
            status = json.loads(review.stdout).get('status', {})
            require(
                not status.get('evaluationError'),
                'Shared viewer SAR evaluation failed',
            )
            return status.get('allowed') is True

        wait_for(
            'read-only shared viewer authorization',
            lambda: permitted(first, 'viewers', 'get'),
            min(self.timeout, 60),
        )
        require(
            not permitted(first, 'pods', 'create', group=''),
            'Dedicated shared viewer unexpectedly has Pod creation privileges',
        )
        require(
            not permitted(second, 'viewers', 'get'),
            'Dedicated shared viewer unexpectedly has access to the other namespace',
        )
        for raw in [False, True]:
            response = self.request(
                first, self.shared, user=self.shared_viewer, raw=raw
            )
            require(
                response['status'] == 200
                and response['body'] == 'tenant-written shared artifact',
                'Read-only viewer could not read another tenant-written shared object',
            )
        # SeaweedFS 4.34 returns an empty object for virtual directories. A GET
        # cannot prove listing permission; use the actual namespace identity.
        listing = self.namespace_storage_client(first).list_objects_v2(
            Bucket=BUCKET, Prefix=self.shared.rsplit('/', 1)[0] + '/'
        )
        require(
            listing.get('IsTruncated') is False
            and [item['Key'] for item in listing.get('Contents', [])]
            == [self.shared],
            'Shared storage listing did not return exactly the tenant-written object',
        )
        response = self.request(
            first, self.keys[first]['private'], user=self.shared_viewer
        )
        require(
            response['status'] == 200
            and response['body'] == first + ':private',
            'Read-only viewer could not read its own namespace private artifact',
        )
        response = self.request(
            first, self.keys[second]['private'], user=self.shared_viewer
        )
        require(
            response['status'] >= 400
            and 'Access Denied' in response['body']
            and second + ':private' not in response['body'],
            'Read-only viewer unexpectedly read another namespace private artifact',
        )
        response = self.request(
            second, self.keys[second]['private'], user=self.shared_viewer
        )
        # The authorization error names the namespace followed by a colon;
        # reject actual seeded artifact bytes, not that expected punctuation.
        require(
            response['status'] == 403
            and second + ':private' not in response['body'],
            'Read-only viewer could claim the other namespace',
        )
        self.passed(
            'namespace storage credentials list the tenant-written shared object'
        )
        self.passed(
            'read-only namespace viewer reads tenant-written shared artifacts while private isolation holds'
        )

    def check_access(self):
        first, second = self.tenants
        for namespace in self.tenants:
            for name, key in self.keys[namespace].items():
                response = self.request(namespace, key)
                require(
                    response['status'] == 200
                    and response['body'] == namespace + ':' + name,
                    'Shared UI could not read authorized ' + name + ' root',
                )
                self.passed(
                    'shared UI reads authorized ' + name, namespace=namespace
                )
            response = self.request(namespace, self.shared)
            require(
                response['status'] == 200
                and response['body'] == 'tenant-written shared artifact',
                'Shared UI could not read the tenant-written shared object',
            )
        for target in ['private', 'historical']:
            response = self.request(first, self.keys[second][target])
            require(
                response['status'] >= 400
                and 'Access Denied' in response['body']
                and second + ':' not in response['body'],
                'Cross-tenant object was not denied by storage',
            )
            self.passed('storage denies cross-tenant ' + target + ' object')
        response = self.request(
            second, self.keys[second]['private'], user=self.users[first]
        )
        require(
            response['status'] == 403,
            'Claimed victim namespace must be rejected by real API/SAR',
        )
        self.passed('real API/SAR rejects claimed victim namespace')
        response = self.request(
            first,
            self.keys[second]['private'],
            headers={
                NAMESPACE_HEADER: second,
                BINDING_HEADER: self.record(second)['credentialBinding'],
            },
        )
        require(
            response['status'] >= 400 and second + ':' not in response['body'],
            'Browser-supplied receiver headers bypassed authorization',
        )
        self.passed('shared UI ignores spoofed internal binding headers')
        for namespace in self.tenants:
            policy = document(
                self.iam.get_user_policy(
                    UserName=namespace, PolicyName='KubeflowProject' + namespace
                )['PolicyDocument']
            )
            require(
                policy == self.expected_policies[namespace],
                'Stored artifact policy changed during reconciliation',
            )
        self.passed(
            'reconciliation preserves administrator historical-root grants'
        )

        self.check_shared_access()
        self.check_legacy_shared_root()

    def check_protocol(self):
        namespace = self.tenants[0]
        key = self.keys[namespace]['historical']
        for headers in [{}, {NAMESPACE_HEADER: self.tenants[1]}]:
            response = self.receiver_request(namespace, key, headers)
            require(
                response.get('status') == 403
                and response.get('version') == '1',
                'Versioned receiver must reject absent/wrong namespace binding',
            )
        response = self.receiver_request(
            namespace,
            key,
            {
                NAMESPACE_HEADER: namespace,
                BINDING_HEADER: '0' * 64,
            },
        )
        require(
            response.get('status') == 503 and response.get('retryAfter') == '5',
            'Versioned receiver must reject stale credential binding',
        )
        response = self.receiver_request(
            namespace,
            key,
            {
                NAMESPACE_HEADER: namespace,
                BINDING_HEADER: self.record(namespace)['credentialBinding'],
            },
        )
        require(
            response.get('status') == 200
            and response.get('version') == '1'
            and response['body'] == namespace + ':historical',
            'Actual UI Pod cannot use certified versioned receiver route',
        )
        response = self.receiver_request(namespace, key, legacy=True)
        require(
            response.get('status') == 200,
            'Legacy receiver route must support old UI without a binding header',
        )
        response = self.receiver_request(
            namespace, key, {NAMESPACE_HEADER: self.tenants[1]}, legacy=True
        )
        require(
            response.get('status') == 403,
            'Legacy receiver must validate a supplied namespace binding',
        )
        self.passed(
            'real receiver validates namespace/protocol/credential bindings'
        )
        self.passed(
            'legacy request protocol remains compatible; supplied binding is checked'
        )

    def mixed_versions(self):
        namespace = self.tenants[0]
        key = self.keys[namespace]['private']
        if self.supports_legacy_ui:
            legacy_name = self.run_id + '-old-ui'
            self.probes.append((self.namespace, legacy_name))
            template = copy.deepcopy(self.ui_template)
            # An old UI must have the allowed UI label to reach the receiver, but
            # must never become a shared Service endpoint. This deliberately unset
            # readiness gate isolates only this test client, not receiver readiness.
            spec = template['spec']
            spec['readinessGates'] = [
                {'conditionType': 'artifact-acceptance/not-a-service-endpoint'}
            ]
            spec['restartPolicy'] = 'Never'
            container = next(
                item
                for item in spec['containers']
                if item['name'] == self.ui_container
            )
            container['image'] = self.legacy_image
            container['imagePullPolicy'] = 'IfNotPresent'
            apply(
                {
                    'apiVersion': 'v1',
                    'kind': 'Pod',
                    'metadata': {
                        'name': legacy_name,
                        'namespace': self.namespace,
                        'labels': {'app': 'ml-pipeline-ui'},
                        'annotations': {'sidecar.istio.io/inject': 'false'},
                    },
                    'spec': spec,
                }
            )
            wait_for(
                'old UI container',
                lambda: any(
                    item['name'] == self.ui_container
                    and 'running' in item.get('state', {})
                    for item in get('pod', legacy_name, self.namespace)
                    .get('status', {})
                    .get('containerStatuses', [])
                ),
                self.timeout,
            )
            legacy_ui = self.port_forward('pod/' + legacy_name, 3000)

            def legacy_read():
                try:
                    response = self.request(namespace, key, base_url=legacy_ui)
                    return response if response['status'] == 200 else None
                except (OSError, urllib.error.URLError):
                    return None

            response = wait_for(
                'old UI to read through new receiver', legacy_read, self.timeout
            )
            require(
                response['body'] == namespace + ':private',
                'Old UI returned unexpected artifact bytes',
            )
            self.capture_images(self.namespace, 'app=ml-pipeline-ui')
            self.passed(
                'old 2.17.2 UI reads through new receiver',
                image=self.legacy_image,
            )
            kubectl(
                'delete',
                'pod',
                legacy_name,
                '-n',
                self.namespace,
                '--wait=false',
            )
        else:
            self.results.append(
                {
                    'name': 'old 2.17.2 UI reads through new receiver',
                    'skipped': True,
                    'reason': 'This checkout removed the v1 AuthService required by the 2.17.2 UI. Old receiver compatibility is tested separately.',
                }
            )
            print(
                'SKIP: old UI requires v1 AuthService removed in master',
                flush=True,
            )

        self.scale_controller(0)
        superseded_uids = self.receiver_pod_uids(namespace)
        kubectl(
            'patch',
            'deployment',
            RECEIVER,
            '-n',
            namespace,
            '--type=strategic',
            '--patch-file=/dev/stdin',
            payload={
                'spec': {
                    'template': {
                        'spec': {
                            'containers': [
                                {
                                    'name': RECEIVER,
                                    'image': self.legacy_image,
                                }
                            ]
                        }
                    }
                },
            },
        )
        kubectl(
            'rollout',
            'status',
            'deployment/' + RECEIVER,
            '-n',
            namespace,
            '--timeout=' + str(self.timeout) + 's',
            timeout=self.timeout + 10,
        )
        self.wait_receiver_pods(namespace, self.legacy_image, superseded_uids)
        self.capture_images(namespace, 'app=' + RECEIVER)

        # Deployment readiness can precede Service endpoint convergence. Prove
        # that the old binary serves an artifact before testing its unsupported
        # protocol; a network outage must not count as an old receiver response.
        def old_receiver_available():
            legacy = self.receiver_request(namespace, key, legacy=True)
            if (
                legacy.get('status') != 200
                or legacy.get('body') != namespace + ':private'
            ):
                return False
            response = self.receiver_request(
                namespace,
                self.keys[namespace]['historical'],
                {
                    NAMESPACE_HEADER: namespace,
                    BINDING_HEADER: self.record(namespace)['credentialBinding'],
                },
            )
            return (
                response.get('status') in [200, 404]
                and response.get('version') is None
            )

        wait_for(
            'old receiver Service to serve the legacy artifact and reject the versioned route',
            old_receiver_available,
            min(self.timeout, 60),
        )
        self.passed(
            'real old receiver serves the artifact before protocol rejection'
        )
        self.wait_receiver_upgrade_required(
            namespace, self.keys[namespace]['historical']
        )
        self.passed(
            'new UI replaces the real old receiver response with a plain-text upgrade-required error'
        )
        superseded_uids = self.receiver_pod_uids(namespace)
        self.scale_controller(self.controller_replicas)
        expected_image = get('deployment', RECEIVER, self.tenants[1])['spec'][
            'template'
        ]['spec']['containers'][0]['image']

        def restored():
            deployment = get('deployment', RECEIVER, namespace)
            record = self.record(namespace)
            return (
                deployment['spec']['template']['spec']['containers'][0]['image']
                == expected_image
                and record
                and record['state'] == 'ready'
                and deployment.get('status', {}).get('observedGeneration', 0)
                >= deployment['metadata']['generation']
                and all(
                    deployment.get('status', {}).get(field, 0) == 1
                    for field in [
                        'replicas',
                        'updatedReplicas',
                        'availableReplicas',
                        'readyReplicas',
                    ]
                )
            )

        wait_for(
            'controller to restore current receiver', restored, self.timeout
        )
        self.wait_receiver_pods(namespace, expected_image, superseded_uids)
        self.wait_artifact_response(
            namespace,
            self.keys[namespace]['historical'],
            'shared UI to serve the historical artifact after receiver recovery',
            status=200,
            body=namespace + ':historical',
        )
        self.passed(
            'controller repairs mixed-version rollout and restores historical access'
        )

    def check_network_policy(self):
        namespace = self.tenants[0]
        policy = get('networkpolicy', RECEIVER, namespace)
        require(
            policy['spec']['podSelector']['matchLabels']['app'] == RECEIVER,
            'Generated NetworkPolicy does not select artifact receivers',
        )
        service_ip = get('service', RECEIVER, namespace)['spec']['clusterIP']
        pods = get('pods', namespace=namespace, selector='app=' + RECEIVER)[
            'items'
        ]
        pod_ip = next(
            pod['status']['podIP']
            for pod in pods
            if not pod['metadata'].get('deletionTimestamp')
            and any(
                condition['type'] == 'Ready' and condition['status'] == 'True'
                for condition in pod.get('status', {}).get('conditions', [])
            )
        )
        query = urllib.parse.urlencode(
            {
                'source': 'minio',
                'bucket': BUCKET,
                'key': self.keys[namespace]['private'],
            }
        )
        targets = [f'http://{service_ip}:80', f'http://{pod_ip}:3000']
        urls = [
            target + '/artifact-proxy/v1/artifacts/get?' + query
            for target in targets
        ]
        headers = {
            NAMESPACE_HEADER: namespace,
            BINDING_HEADER: self.record(namespace)['credentialBinding'],
        }
        for url in urls:
            response = self.exec_request(
                self.namespace, self.ui_pod, self.ui_container, url, headers
            )
            require(
                response.get('status') == 200
                and response['body'] == namespace + ':private',
                'Actual UI cannot reach receiver Pod/Service IP for network-policy baseline',
            )
        self.passed(
            'NetworkPolicy permits actual shared UI to receiver Pod and Service IPs'
        )
        for probe_namespace, label in [
            (self.namespace, self.run_id),
            (namespace, 'ml-pipeline-ui'),
        ]:
            name = self.run_id + '-probe'
            self.probes.append((probe_namespace, name))
            apply(
                {
                    'apiVersion': 'v1',
                    'kind': 'Pod',
                    'metadata': {
                        'name': name,
                        'namespace': probe_namespace,
                        'labels': {'app': label},
                        'annotations': {'sidecar.istio.io/inject': 'false'},
                    },
                    'spec': {
                        'automountServiceAccountToken': False,
                        'restartPolicy': 'Never',
                        'containers': [
                            {
                                'name': 'probe',
                                'image': self.ui_image,
                                'imagePullPolicy': 'IfNotPresent',
                                'command': [
                                    'node',
                                    '-e',
                                    'setInterval(() => {}, 60000)',
                                ],
                                'resources': {
                                    'requests': {'cpu': '5m', 'memory': '32Mi'},
                                    'limits': {'memory': '128Mi'},
                                },
                            }
                        ],
                    },
                }
            )
            kubectl(
                'wait',
                '--for=condition=Ready',
                'pod/' + name,
                '-n',
                probe_namespace,
                '--timeout=' + str(self.timeout) + 's',
                timeout=self.timeout + 10,
            )
            for url in urls:
                response = self.exec_request(
                    probe_namespace, name, 'probe', url, headers
                )
                require(
                    response.get('networkError') == 'TimeoutError'
                    or response.get('code')
                    in ['ETIMEDOUT', 'UND_ERR_CONNECT_TIMEOUT'],
                    'Unrelated Pod reached receiver or network probe failed for another reason',
                )
            # Recheck the allowed path so a receiver outage cannot masquerade
            # as successful NetworkPolicy enforcement.
            for url in urls:
                baseline = self.exec_request(
                    self.namespace, self.ui_pod, self.ui_container, url, headers
                )
                require(
                    baseline.get('status') == 200
                    and baseline['body'] == namespace + ':private',
                    'Receiver became unavailable during network-denial checks',
                )
            self.passed(
                'NetworkPolicy blocks unrelated Pod',
                namespace=probe_namespace,
                spoofedUILabel=label == 'ml-pipeline-ui',
            )
            kubectl(
                'delete', 'pod', name, '-n', probe_namespace, '--wait=false'
            )

    def recover_bucket_policy(self):
        credentials = {
            namespace: get('secret', CREDENTIAL_SECRET, namespace)['data']
            for namespace in self.tenants
        }
        original = document(self.s3.get_bucket_policy(Bucket=BUCKET)['Policy'])
        missing = copy.deepcopy(original)
        missing['Statement'] = [
            statement
            for statement in missing['Statement']
            if statement.get('Sid') != 'AllowHeadBucket'
        ]
        require(
            len(missing['Statement']) == len(original['Statement']) - 1,
            'Expected exactly one managed HeadBucket policy statement',
        )
        self.scale_controller(0)
        self.recovery_bucket_policy = original
        # Remove only a permission, retaining all cross-tenant Deny statements.
        # Existing credentials must trigger repair; no new Profile is created.
        self.s3.put_bucket_policy(Bucket=BUCKET, Policy=json.dumps(missing))
        stored = document(self.s3.get_bucket_policy(Bucket=BUCKET)['Policy'])
        require(
            not any(
                statement.get('Sid') == 'AllowHeadBucket'
                for statement in stored.get('Statement', [])
            ),
            'Policy recovery fixture did not remove HeadBucket permission',
        )
        require(
            all(
                statement in stored.get('Statement', [])
                for statement in original['Statement']
                if statement.get('Effect') == 'Deny'
            ),
            'Policy recovery fixture must preserve every Deny statement',
        )
        # These namespaces were already Ready before this external policy change.
        # Queue a watched event; real incomplete initialization already requests
        # a 10-second resync and does not need an operator to trigger it.
        kubectl(
            'annotate',
            'namespace',
            self.tenants[0],
            'artifact-acceptance/policy-recovery=' + self.run_id,
            '--overwrite',
        )
        self.scale_controller(self.controller_replicas)

        def repaired():
            current = document(
                self.s3.get_bucket_policy(Bucket=BUCKET)['Policy']
            )
            return all(
                statement in current.get('Statement', [])
                for statement in original['Statement']
            )

        wait_for(
            'existing namespace to recover missing managed bucket policy',
            repaired,
            min(self.timeout, 120),
        )
        for namespace in self.tenants:
            self.wait_state(namespace, 'ready')
            require(
                get('secret', CREDENTIAL_SECRET, namespace)['data']
                == credentials[namespace],
                'Bucket policy recovery replaced existing tenant credentials',
            )
            response = self.request(
                namespace, self.keys[namespace]['historical']
            )
            require(
                response['status'] == 200
                and response['body'] == namespace + ':historical',
                'Historical artifact access did not recover with bucket policy',
            )
        self.recovery_bucket_policy = None
        self.passed(
            'existing namespaces repair missing managed policy without replacing credentials'
        )

    def rotate_credentials(self):
        namespace = self.tenants[0]
        previous = self.record(namespace)
        old_secret = get('secret', CREDENTIAL_SECRET, namespace)
        old_key = decode_secret(old_secret, 'accesskey')
        receiver = get('deployment', RECEIVER, namespace)
        old_revision = receiver['spec']['template']['metadata']['annotations'][
            'kubeflow-pipelines/artifact-storage-revision'
        ]
        old_generation = receiver['metadata']['generation']
        superseded_uids = self.receiver_pod_uids(namespace)
        new_key = self.iam.create_access_key(UserName=namespace)['AccessKey']
        kubectl(
            'patch',
            'secret',
            CREDENTIAL_SECRET,
            '-n',
            namespace,
            '--type=merge',
            '--patch-file=/dev/stdin',
            payload={
                'data': {
                    'accesskey': base64.b64encode(
                        new_key['AccessKeyId'].encode()
                    ).decode(),
                    'secretkey': base64.b64encode(
                        new_key['SecretAccessKey'].encode()
                    ).decode(),
                }
            },
        )

        # The old receiver's env does not change with the Secret. The controller
        # must roll the Deployment, then publish readiness for the replacement.
        def rotated():
            record = self.record(namespace)
            if (
                not record
                or record['state'] != 'ready'
                or record['credentialBinding'] == previous['credentialBinding']
            ):
                return False
            current = get('deployment', RECEIVER, namespace)
            return (
                current['metadata']['generation'] > old_generation
                and current['spec']['template']['metadata']['annotations'][
                    'kubeflow-pipelines/artifact-storage-revision'
                ]
                != old_revision
            )

        wait_for(
            'automatic credential rotation and receiver readiness',
            rotated,
            self.timeout,
        )
        expected_image = next(
            container['image']
            for container in receiver['spec']['template']['spec']['containers']
            if container['name'] == RECEIVER
        )
        self.wait_receiver_pods(namespace, expected_image, superseded_uids)
        self.iam.delete_access_key(UserName=namespace, AccessKeyId=old_key)
        response = self.receiver_request(
            namespace,
            self.keys[namespace]['historical'],
            {
                NAMESPACE_HEADER: namespace,
                BINDING_HEADER: previous['credentialBinding'],
            },
        )
        require(
            response.get('status') == 503,
            'Replacement receiver accepted the old credential binding',
        )
        self.wait_artifact_response(
            namespace,
            self.keys[namespace]['historical'],
            'shared UI to serve the historical artifact after credential rotation',
            status=200,
            body=namespace + ':historical',
        )
        # Check the binding is to the actual deployed env credentials and UID.
        current = self.record(namespace)
        expected = hashlib.sha256(
            '\0'.join(
                [
                    namespace,
                    current['namespaceUID'],
                    f'http://seaweedfs.{self.namespace}:8333',
                    new_key['AccessKeyId'],
                    new_key['SecretAccessKey'],
                ]
            ).encode()
        ).hexdigest()
        require(
            current['credentialBinding'] == expected,
            'Ready record does not bind the actual rotated storage identity',
        )
        self.capture_images(namespace, 'app=' + RECEIVER)
        self.passed(
            'IAM credential rotation automatically rolls receiver and recovers reads'
        )

    def diagnostics(self):
        # Whitelist public fields; never serialize Secrets, raw commands, exception
        # bodies, or controller logs that could contain credential material.
        diagnostics = {
            'sourceHead': self.source_head,
            'phase': self.phase,
            'lastResponse': self.last_response,
            'lastProbeResponse': self.last_probe_response,
            'authorization': {},
            'deployments': {},
            'namespaces': {},
        }
        for namespace in [self.namespace, *self.tenants]:
            pods = get('pods', namespace=namespace, optional=True)
            diagnostics['namespaces'][namespace] = [
                {
                    'name': item['metadata']['name'],
                    'uid': item['metadata'].get('uid'),
                    'deletionTimestamp': item['metadata'].get(
                        'deletionTimestamp'
                    ),
                    'podTemplateHash': item['metadata']
                    .get('labels', {})
                    .get('pod-template-hash'),
                    'phase': item.get('status', {}).get('phase'),
                    'containers': [
                        {
                            'name': status['name'],
                            'ready': status.get('ready'),
                            'image': status.get('image'),
                            'imageID': status.get('imageID'),
                            'restarts': status.get('restartCount'),
                            'waitingReason': status.get('state', {})
                            .get('waiting', {})
                            .get('reason'),
                        }
                        for status in item.get('status', {}).get(
                            'containerStatuses', []
                        )
                    ],
                }
                for item in (pods or {}).get('items', [])
            ]
        for namespace in self.tenants:
            secret = get(
                'secret',
                'kfp-artifact-storage-' + namespace,
                self.namespace,
                optional=True,
            )
            if secret:
                record = json.loads(decode_secret(secret, 'authorization.json'))
                diagnostics['authorization'][namespace] = {
                    key: record.get(key)
                    for key in (
                        'version',
                        'namespace',
                        'namespaceUID',
                        'state',
                        'bucket',
                    )
                }
            receiver = get('deployment', RECEIVER, namespace, optional=True)
            if receiver:
                status = receiver.get('status', {})
                diagnostics['deployments'][namespace] = {
                    'generation': receiver['metadata'].get('generation'),
                    **{
                        key: status.get(key)
                        for key in (
                            'observedGeneration',
                            'replicas',
                            'updatedReplicas',
                            'availableReplicas',
                            'readyReplicas',
                        )
                    },
                    'conditions': [
                        {
                            key: condition.get(key)
                            for key in ('type', 'status', 'reason')
                        }
                        for condition in status.get('conditions', [])
                    ],
                }
        (self.output / 'diagnostics.json').write_text(
            json.dumps(diagnostics, indent=2) + '\n'
        )

    def cleanup(self):
        errors = []
        deadline = time.monotonic() + 240

        def attempt(description, operation):
            try:
                if time.monotonic() >= deadline:
                    raise TimeoutError('Cleanup budget exhausted')
                result = operation()
                if hasattr(result, 'returncode') and result.returncode:
                    raise RuntimeError('Cleanup command failed')
                return True
            except Exception:
                errors.append(description)
                return False

        def delete(kind, name, namespace=None):
            args = ['delete', kind, name, '--ignore-not-found', '--wait=false']
            if namespace:
                args += ['-n', namespace]
            return kubectl(*args, optional=True, timeout=20)

        try:
            if self.recovery_bucket_policy is not None and self.s3:
                attempt(
                    'Could not restore original bucket policy',
                    lambda: self.s3.put_bucket_policy(
                        Bucket=BUCKET,
                        Policy=json.dumps(self.recovery_bucket_policy),
                    ),
                )
            if self.controller_paused and self.controller_replicas is not None:
                attempt(
                    'Restore profile webhook replicas manually',
                    lambda: kubectl(
                        'scale',
                        'deployment/' + CONTROLLER,
                        '-n',
                        self.namespace,
                        '--replicas=' + str(self.controller_replicas),
                        timeout=20,
                    ),
                )
            for namespace, name in self.probes:
                attempt(
                    'Could not remove fixture Pod ' + name,
                    lambda: delete('pod', name, namespace),
                )
            for namespace in self.tenants:
                attempt(
                    'Could not remove fixture Profile ' + namespace,
                    lambda: delete('profile', namespace),
                )
            namespaces_deleted = attempt(
                'Fixture namespaces remain; remove them before retrying IAM cleanup',
                lambda: wait_for(
                    'fixture namespaces to be deleted',
                    lambda: all(
                        get('namespace', namespace, optional=True) is None
                        for namespace in self.tenants
                    ),
                    min(self.timeout, 90),
                ),
            )
            if self.s3:
                for key in self.objects:
                    attempt(
                        'Could not remove fixture object',
                        lambda: self.s3.delete_object(Bucket=BUCKET, Key=key),
                    )
            if self.iam and namespaces_deleted:
                for namespace in self.tenants:

                    def remove_identity():
                        try:
                            for key in self.iam.list_access_keys(
                                UserName=namespace
                            )['AccessKeyMetadata']:
                                self.iam.delete_access_key(
                                    UserName=namespace,
                                    AccessKeyId=key['AccessKeyId'],
                                )
                            for policy in self.iam.list_user_policies(
                                UserName=namespace
                            )['PolicyNames']:
                                self.iam.delete_user_policy(
                                    UserName=namespace, PolicyName=policy
                                )
                            self.iam.delete_user(UserName=namespace)
                        except Exception as error:
                            if (
                                getattr(error, 'response', {})
                                .get('Error', {})
                                .get('Code')
                                != 'NoSuchEntity'
                            ):
                                raise

                    attempt(
                        'Could not remove fixture IAM user ' + namespace,
                        remove_identity,
                    )
        finally:
            for process in self.forwarders:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
        return errors

    def run(self):
        success = False
        failure = None
        try:
            for phase in [
                self.prepare,
                self.pending_rollout,
                self.seed_existing_roots,
                self.check_access,
                self.check_explicit_storage,
                self.check_protocol,
                self.check_network_policy,
                self.mixed_versions,
                self.recover_bucket_policy,
                self.rotate_credentials,
                self.check_access,
            ]:
                self.phase = phase.__name__
                print('PHASE: ' + self.phase, flush=True)
                phase()
            success = True
        except Exception as error:
            # Assertions are our own fixed messages; dependency errors may contain
            # storage credentials, so report only their type.
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
                print('Could not collect Pod diagnostics', file=sys.stderr)
            try:
                cleanup_errors = self.cleanup()
            except Exception:
                cleanup_errors = [
                    'Cleanup failed; inspect fixture resources and restore profile webhook replicas'
                ]
            success = success and not cleanup_errors
            report = {
                'passed': success,
                'failure': failure,
                'sourceHead': self.source_head,
                'phase': self.phase,
                'checks': self.results,
                'runtimeImages': self.runtime_images,
                'cleanupErrors': cleanup_errors,
                'scope': 'Real deployed authorization/reconciliation and old-receiver recovery. Old-UI compatibility runs only where v1 AuthService remains. Ingress login is not exercised.',
            }
            (self.output / 'results.json').write_text(
                json.dumps(report, indent=2) + '\n'
            )
        return 0 if success else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', default='kubeflow')
    parser.add_argument(
        '--output-dir', default='artifacts/artifact-authorization'
    )
    parser.add_argument(
        '--legacy-frontend-image',
        default=os.environ.get(
            'LEGACY_FRONTEND_IMAGE',
            'ghcr.io/kubeflow/kfp-frontend:2.17.2@sha256:0da8b3f14a7bb2677834e8b68a68a2b2056bcc0c7a5c2a83c9722ea7fe2282a3',
        ),
    )
    parser.add_argument(
        '--timeout',
        type=int,
        default=600,
        help='Maximum seconds for each reconciliation/rollout phase',
    )
    args = parser.parse_args()
    return Acceptance(args).run()


if __name__ == '__main__':
    sys.exit(main())

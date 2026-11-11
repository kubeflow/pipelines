#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise rendered storage/profile manifests across a persistent-volume
upgrade."""

import argparse
import base64
from contextlib import contextmanager
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time
import urllib.request

import yaml

ROOT = Path(__file__).resolve().parents[3]
SEAWEED = ROOT / 'manifests/kustomize/third-party/seaweedfs/base'
PROFILE = (
    ROOT /
    'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller')
CLUSTER = 'third-party-upgrade'
BUCKET = 'mlpipeline'
S3_LOCAL_PORT = 18333
S3_PORT = 8333
IAM_LOCAL_PORT = 18111
IAM_PORT = 8111
PROFILE_LOCAL_PORT = 18080
PROFILE_PORT = 8080
ROLLOUT_TIMEOUT = '360s'
PAYLOAD = b'KFP persistent upgrade compatibility\x00\xff\n' * 128


def run(*args, data=None):
    """Run a command and retain its diagnostic output on failure."""
    try:
        return subprocess.run(
            args, input=data, text=True, capture_output=True, check=True).stdout
    except subprocess.CalledProcessError as error:
        print(error.stderr or error.stdout or str(error), file=sys.stderr)
        raise


def kubectl(*args, data=None):
    return run(
        'kubectl',
        '--context',
        f'kind-{CLUSTER}',
        '-n',
        'kubeflow',
        *args,
        data=data,
    )


def rendered(path):
    return list(yaml.safe_load_all(run('kustomize', 'build', str(path))))


def apply(objects):
    kubectl('apply', '-f', '-', data=yaml.safe_dump_all(objects))


def wait_ready(name):
    kubectl(
        'rollout',
        'status',
        f'deployment/{name}',
        f'--timeout={ROLLOUT_TIMEOUT}',
    )


@contextmanager
def forward(deployment, local, remote, evidence):
    with (evidence / f'port-forward-{deployment}-{local}.log').open('a') as log:
        process = subprocess.Popen(
            [
                'kubectl',
                '--context',
                f'kind-{CLUSTER}',
                '-n',
                'kubeflow',
                'port-forward',
                f'deployment/{deployment}',
                f'{local}:{remote}',
            ],
            stdout=log,
            stderr=log,
        )
        try:
            url = f'http://127.0.0.1:{local}'
            health = '/healthz' if remote == PROFILE_PORT else '/status'
            for _ in range(60):
                if process.poll() is not None:
                    raise RuntimeError(
                        f'Port-forward for {deployment} stopped; inspect its log'
                    )
                try:
                    with urllib.request.urlopen(url + health, timeout=2):
                        break
                except urllib.error.HTTPError:
                    if remote == IAM_PORT:
                        break  # IAM has no public health endpoint.
                    time.sleep(1)
                except OSError:
                    time.sleep(1)
            else:
                raise RuntimeError(f'{deployment} did not become reachable')
            yield url
        finally:
            process.terminate()
            process.wait(timeout=10)


def secret_credentials(secret):
    """Read Kubernetes Secret values using stringData-over-data precedence."""
    values = []
    for key in ('accesskey', 'secretkey'):
        try:
            if key in secret.get('stringData', {}):
                value = secret['stringData'][key]
            else:
                value = base64.b64decode(
                    secret['data'][key], validate=True).decode('utf-8')
            if not isinstance(value, str) or not value:
                raise ValueError('Empty or non-string credential')
        except (KeyError, TypeError, ValueError) as error:
            raise ValueError(f'Secret has invalid or missing {key}') from error
        values.append(value)
    return tuple(values)


def client(endpoint, access, secret):
    import boto3
    from botocore.config import Config

    return boto3.client(
        's3',
        endpoint_url=endpoint,
        aws_access_key_id=access,
        aws_secret_access_key=secret,
        region_name='us-east-1',
        config=Config(
            s3={'addressing_style': 'path'},
            retries={'max_attempts': 4},
            request_checksum_calculation='when_required',
            response_checksum_validation='when_required',
        ),
    )


def iam_client(endpoint, access, secret):
    import boto3
    from botocore.config import Config

    return boto3.client(
        'iam',
        endpoint_url=endpoint,
        aws_access_key_id=access,
        aws_secret_access_key=secret,
        region_name='us-east-1',
        config=Config(
            connect_timeout=5, read_timeout=10, retries={'max_attempts': 2}),
    )


def sync_profile(endpoint, namespace, secret=None):
    attachments = {
        kind: {} for kind in (
            'Secret.v1',
            'ConfigMap.v1',
            'Deployment.apps/v1',
            'Service.v1',
        )
    }
    if secret:
        attachments['Secret.v1'][f'{namespace}/mlpipeline-minio-artifact'] = (
            secret)
    body = {
        'object': {
            'metadata': {
                'name': namespace,
                'labels': {
                    'pipelines.kubeflow.org/enabled': 'true'
                },
            }
        },
        'attachments': attachments,
    }
    request = urllib.request.Request(
        endpoint + '/sync',
        data=json.dumps(body).encode(),
        headers={'Content-Type': 'application/json'},
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        result = json.load(response)
    credentials = [
        item for item in result['attachments']
        if item.get('kind') == 'Secret' and
        item.get('metadata', {}).get('name') == 'mlpipeline-minio-artifact'
    ]
    if len(credentials) != 1:
        raise ValueError(
            'Expected one artifact credential Secret; inspect profile reconciliation output'
        )
    return credentials[0]


def verify_payload(s3, key):
    actual = s3.get_object(Bucket=BUCKET, Key=key)['Body'].read()
    if actual != PAYLOAD:
        raise AssertionError(f'Persistent object {key} changed during upgrade')


def verify_state(s3):
    verify_payload(s3, 'shared/before-upgrade')
    rules = s3.get_bucket_lifecycle_configuration(Bucket=BUCKET)['Rules']
    if (len(rules) != 1 or rules[0].get('ID') != 'private-artifacts' or
            rules[0].get('Status') != 'Enabled' or rules[0].get('Filter') != {
                'Prefix': 'private-artifacts'
            } or rules[0].get('Expiration') != {
                'Days': 30
            }):
        raise AssertionError(
            'Lifecycle configuration changed; inspect the private-artifacts rule scope and retention'
        )


def seed_old_store(storage, baseline_image, evidence):
    """Seed retained objects and lifecycle configuration on the baseline
    image."""
    credentials = [
        obj for obj in storage if obj.get('kind') == 'Secret' and
        obj.get('metadata', {}).get('name') == 'mlpipeline-minio-artifact'
    ]
    if len(credentials) > 1:
        raise ValueError('Ambiguous legacy credential Secret')
    admin_credentials = (
        secret_credentials(credentials[0]) if credentials else None)
    deployment = next(
        obj for obj in storage if obj['kind'] == 'Deployment' and
        obj.get('metadata', {}).get('name', 'seaweedfs') == 'seaweedfs')
    deployment['spec']['template']['spec']['containers'][0]['image'] = (
        baseline_image)
    apply([
        {
            'apiVersion': 'v1',
            'kind': 'Namespace',
            'metadata': {
                'name': 'kubeflow'
            },
        },
        {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'metadata': {
                'name': 'pipeline-install-config'
            },
            'data': {
                'appVersion': 'compatibility',
                'ARTIFACT_RETENTION_DAYS': '30',
            },
        },
    ])
    apply(storage)
    wait_ready('seaweedfs')
    if admin_credentials is None:
        wait_migration_complete()
        admin_credentials = current_admin_credentials()
    (evidence / 'before-upgrade-pods.json').write_text(
        kubectl('get', 'pods', '-o', 'json'))
    pvc_uid = json.loads(kubectl('get', 'pvc', 'seaweedfs-pvc', '-o',
                                 'json'))['metadata']['uid']
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        s3 = client(endpoint, *admin_credentials)
        s3.put_object(Bucket=BUCKET, Key='shared/before-upgrade', Body=PAYLOAD)
        s3.put_bucket_lifecycle_configuration(
            Bucket=BUCKET,
            LifecycleConfiguration={
                'Rules': [{
                    'ID': 'private-artifacts',
                    'Status': 'Enabled',
                    'Filter': {
                        'Prefix': 'private-artifacts'
                    },
                    'Expiration': {
                        'Days': 30
                    },
                }]
            },
        )
        verify_state(s3)
    return admin_credentials, pvc_uid


def current_admin_credentials():
    return secret_credentials(
        json.loads(
            kubectl('get', 'secret', 'mlpipeline-minio-artifact', '-o',
                    'json')))


def upgrade_store(storage):
    """Apply the actual candidate manifests, including credential migration."""
    apply(storage)
    wait_ready('seaweedfs')
    if any(
            obj.get('kind') == 'Deployment' and
            obj['metadata']['name'] == 'seaweedfs-credential-rotation'
            for obj in storage):
        wait_migration_complete()


def wait_migration_complete():
    """Wait for revocation separately from controller readiness."""
    deadline = time.monotonic() + 360
    while time.monotonic() < deadline:
        state = json.loads(
            kubectl('get', 'secret', 'seaweedfs-admin-credentials', '-o',
                    'json'))
        if (state['metadata'].get(
                'annotations',
            {}).get('pipelines.kubeflow.org/storage-credential-phase') ==
                'DefaultRevoked'):
            return
        time.sleep(2)
    raise AssertionError('Credential migration did not complete')


def require_authentication_rejection(request):
    """Require an explicit authentication rejection."""
    try:
        with urllib.request.urlopen(request, timeout=10):
            pass
    except urllib.error.HTTPError as error:
        if error.code in (401, 403):
            return
        raise
    raise AssertionError('Unauthenticated storage request succeeded')


def verify_fresh_bootstrap(evidence):
    """Probe listener authentication before any dynamic identity is seeded."""
    from botocore.exceptions import ClientError

    storage = rendered(SEAWEED)
    unseeded = copy.deepcopy(storage)
    for obj in unseeded:
        if (obj['kind'] == 'Deployment' and
                obj['metadata']['name'] == 'seaweedfs'):
            container = obj['spec']['template']['spec']['containers'][0]
            # Test-only: keep the PVC empty to expose initial listener auth.
            container.pop('lifecycle')
            container.pop('startupProbe')
    apply([{
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'metadata': {
            'name': 'kubeflow'
        },
    }])
    apply(unseeded)
    wait_ready('seaweedfs')
    candidate = secret_credentials(
        json.loads(
            kubectl('get', 'secret', 'seaweedfs-admin-credentials', '-o',
                    'json')))
    with forward('seaweedfs', IAM_LOCAL_PORT, IAM_PORT, evidence) as endpoint:
        require_authentication_rejection(
            urllib.request.Request(
                endpoint,
                data=b'Action=ListUsers&Version=2010-05-08',
            ))

        with iam_client(endpoint, *candidate) as authorized:
            authorized.list_users()
        with iam_client(endpoint, 'invalid-bootstrap-key',
                        'invalid-bootstrap-secret') as invalid:
            try:
                invalid.list_users()
            except ClientError as error:
                if error.response.get('ResponseMetadata',
                                      {}).get('HTTPStatusCode') not in (401,
                                                                        403):
                    raise
            else:
                raise AssertionError('Wrong-key IAM request succeeded')
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        require_authentication_rejection(urllib.request.Request(endpoint))
    # Restore the exact manifest and complete a real first-install migration.
    upgrade_store(storage)
    if current_admin_credentials() != candidate:
        raise AssertionError(
            'Fresh installation changed its prepared credential')
    verify_manifest_reapply(storage)
    (evidence / 'fresh-bootstrap.json').write_text(
        json.dumps(
            {
                'result':
                    'passed',
                'checks': [
                    'unsigned-iam-denied-before-registration',
                    'wrong-key-iam-denied-before-registration',
                    'generated-key-iam-accepted-before-registration',
                    'unsigned-s3-denied-before-registration',
                    'fresh-install-migration-complete',
                ],
            },
            indent=2,
        ) + '\n')
    # This function runs only inside this script's disposable Kind cluster.
    kubectl('delete', 'namespace', 'kubeflow', '--wait=true', '--timeout=180s')


def verify_manifest_reapply(storage):
    """Check that manifest reapply preserves runtime credentials."""
    names = ('seaweedfs-admin-credentials', 'mlpipeline-minio-artifact')

    def snapshot():
        result = []
        for name in names:
            secret = json.loads(kubectl('get', 'secret', name, '-o', 'json'))
            result.append((secret['metadata']['uid'], secret['data']))
        return result

    expected = snapshot()
    apply(storage)
    if snapshot() != expected:
        raise AssertionError('Client-side apply changed runtime credentials')
    kubectl(
        'apply',
        '--server-side',
        '--field-manager=compatibility-gitops',
        '-f',
        '-',
        data=yaml.safe_dump_all(storage),
    )
    if snapshot() != expected:
        raise AssertionError('Server-side apply changed runtime credentials')
    wait_migration_complete()


def verify_workload_drain(storage, old, evidence):
    """Verify revocation waits for a live pipeline pod."""
    fingerprint = hashlib.sha256(('\0'.join(old)).encode()).hexdigest()
    if (fingerprint !=
            '0c9d72cb381b9afa7023c36f330317ee8783fc8becf72e2c0610688e9afb544d'):
        upgrade_store(storage)  # Already migrated or custom credentials.
        return
    name = 'credential-drain-fixture'
    apply([{
        'apiVersion': 'v1',
        'kind': 'Pod',
        'metadata': {
            'name': name,
            'labels': {
                'pipeline/runid': 'compatibility-drain'
            },
        },
        'spec': {
            'restartPolicy':
                'Never',
            'securityContext': {
                'runAsNonRoot': True,
                'runAsUser': 1000,
                'seccompProfile': {
                    'type': 'RuntimeDefault'
                },
            },
            'containers': [{
                'name': 'held-workload',
                'image': storage_image(storage),
                'command': ['/bin/sh', '-c', 'exec sleep 900'],
                'securityContext': {
                    'allowPrivilegeEscalation': False,
                    'readOnlyRootFilesystem': True,
                    'capabilities': {
                        'drop': ['ALL']
                    },
                },
                'resources': {
                    'requests': {
                        'cpu': '10m',
                        'memory': '16Mi'
                    },
                    'limits': {
                        'memory': '32Mi'
                    },
                },
            }],
        },
    }])
    kubectl('wait', '--for=condition=Ready', f'pod/{name}', '--timeout=120s')
    apply(storage)
    deadline = time.monotonic() + 300
    while time.monotonic() < deadline:
        state = json.loads(
            kubectl('get', 'secret', 'seaweedfs-admin-credentials', '-o',
                    'json'))
        phase = (
            state['metadata'].get(
                'annotations',
                {}).get('pipelines.kubeflow.org/storage-credential-phase'))
        if phase == 'DefaultRevoked':
            raise AssertionError(
                'Credential revoked while pipeline fixture is still running')
        if phase == 'DrainingWorkloads':
            break
        time.sleep(2)
    else:
        raise AssertionError('Migration did not reach the workload drain')
    wait_ready('seaweedfs-credential-rotation')
    if (json.loads(kubectl('get', 'pod', name, '-o', 'json'))['status']['phase']
            != 'Running'):
        raise AssertionError(
            'Drain fixture stopped before credential verification')
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        with client(endpoint, *old) as s3:
            s3.head_bucket(Bucket=BUCKET)
    with forward('seaweedfs', IAM_LOCAL_PORT, IAM_PORT, evidence) as endpoint:
        with iam_client(endpoint, *old) as iam:
            iam.list_users()
    kubectl('delete', 'pod', name, '--wait=true', '--timeout=60s')
    wait_migration_complete()
    (evidence / 'workload-drain.json').write_text(
        json.dumps(
            {
                'result':
                    'passed',
                'checks': [
                    'live-pipeline-pod-blocks-revocation',
                    'old-key-accepted-by-both-listeners-during-drain',
                    'migration-completes-after-pod-termination',
                ],
            },
            indent=2,
        ) + '\n')


def verify_old_credentials_rejected(old, current, evidence):
    from botocore.exceptions import ClientError

    if old == current:
        return  # Operator-provided credentials are preserved.
    for local, port, factory, method, arguments in (
        (S3_LOCAL_PORT, S3_PORT, client, 'head_bucket', {
            'Bucket': BUCKET
        }),
        (IAM_LOCAL_PORT, IAM_PORT, iam_client, 'list_users', {}),
    ):
        with forward('seaweedfs', local, port, evidence) as endpoint:
            with factory(endpoint, *old) as probe:
                try:
                    getattr(probe, method)(**arguments)
                except ClientError as error:
                    if error.response.get('ResponseMetadata',
                                          {}).get('HTTPStatusCode') in (401,
                                                                        403):
                        continue
                    raise
        raise AssertionError(
            f'Pre-upgrade default credential still accepted on port {port}')


def exercise_profile(admin_credentials, evidence, namespace='compat-user'):
    """Check retained data and issue reusable credentials through the
    webhook."""
    # The webhook is invoked directly; installing Metacontroller is unnecessary for this contract test.
    profile = [
        obj for obj in rendered(PROFILE) if obj['kind'] != 'DecoratorController'
    ]
    apply(profile)
    wait_ready('kubeflow-pipelines-profile-controller')
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        s3 = client(endpoint, *admin_credentials)
        verify_state(s3)
        with forward(
                'kubeflow-pipelines-profile-controller',
                PROFILE_LOCAL_PORT,
                PROFILE_PORT,
                evidence,
        ) as webhook:
            secret = sync_profile(webhook, namespace)
            repeated = sync_profile(webhook, namespace, secret)
            if repeated['data'] != secret['data']:
                raise AssertionError(
                    'Profile reconciliation replaced existing credentials')
        user = client(
            endpoint,
            *secret_credentials(secret),
        )
        user.put_object(
            Bucket=BUCKET,
            Key=f'private-artifacts/{namespace}/after-upgrade',
            Body=PAYLOAD,
        )
        verify_payload(user, f'private-artifacts/{namespace}/after-upgrade')
        verify_state(s3)
    return secret


def verify_after_restart(admin_credentials, secret, pvc_uid, evidence):
    """Require both credentials and data to survive a restart on the same
    PVC."""
    kubectl('rollout', 'restart', 'deployment/seaweedfs')
    wait_ready('seaweedfs')
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        verify_state(client(endpoint, *admin_credentials))
        user = client(
            endpoint,
            *secret_credentials(secret),
        )
        verify_payload(user, 'private-artifacts/compat-user/after-upgrade')
    final_uid = json.loads(
        kubectl('get', 'pvc', 'seaweedfs-pvc', '-o', 'json'))['metadata']['uid']
    if final_uid != pvc_uid:
        raise AssertionError('Upgrade replaced the PVC')


def write_result(evidence, baseline_image, target, pvc_uid):
    """Record the actual image pair and verified persistence boundaries."""
    (evidence / 'result.json').write_text(
        json.dumps(
            {
                'result':
                    'passed',
                'old_image':
                    baseline_image,
                'new_image':
                    target,
                'pvc_uid':
                    pvc_uid,
                'payload_sha256':
                    hashlib.sha256(PAYLOAD).hexdigest(),
                'checks': [
                    'old-image-write',
                    'upgrade-read',
                    'lifecycle-persistence',
                    'profile-health',
                    'profile-iam-credentials',
                    'profile-idempotence',
                    'new-image-write',
                    'restart-read',
                    'restart-credentials',
                    'same-pvc',
                    'pre-upgrade-tenant-key-preserved',
                    'post-upgrade-new-tenant-credentials',
                    'client-and-server-side-reapply-preserves-runtime-secrets',
                    'default-credential-rejected-after-restart',
                ],
            },
            indent=2,
        ) + '\n')


def storage_image(storage):
    """Select exactly one SeaweedFS container from rendered manifests."""
    images = [
        container['image'] for obj in storage
        if obj.get('kind') == 'Deployment' and
        obj.get('metadata', {}).get('name') == 'seaweedfs'
        for container in obj['spec']['template']['spec']['containers']
        if container.get('name') == 'seaweedfs'
    ]
    if len(images) != 1:
        raise ValueError(
            'Expected one rendered SeaweedFS image; check the baseline manifests'
        )
    return images[0]


def exercise(evidence, baseline):
    """Validate a base-ref to candidate store upgrade and profile contract."""
    storage = rendered(SEAWEED)
    target = storage_image(storage)
    baseline_storage = rendered(baseline / SEAWEED.relative_to(ROOT))
    baseline_image = storage_image(baseline_storage)
    old_credentials, pvc_uid = seed_old_store(baseline_storage, baseline_image,
                                              evidence)
    # Establish a tenant key and objects BEFORE rotation to prove preservation.
    secret = exercise_profile(old_credentials, evidence)
    verify_workload_drain(storage, old_credentials, evidence)
    credentials = current_admin_credentials()
    verify_manifest_reapply(storage)
    exercise_profile(credentials, evidence, namespace='compat-after-upgrade')
    verify_old_credentials_rejected(old_credentials, credentials, evidence)
    verify_after_restart(credentials, secret, pvc_uid, evidence)
    verify_old_credentials_rejected(old_credentials, credentials, evidence)
    write_result(evidence, baseline_image, target, pvc_uid)


def delete_cluster():
    """Attempt cleanup without replacing the original validation failure."""
    try:
        run('kind', 'delete', 'cluster', '--name', CLUSTER)
    except (OSError, subprocess.CalledProcessError) as error:
        print(
            f'Cluster cleanup failed; delete {CLUSTER} manually: {error}',
            file=sys.stderr,
        )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--baseline', type=Path, required=True)
    args = parser.parse_args()
    args.evidence.mkdir(parents=True, exist_ok=True)
    try:
        run(
            'kind',
            'create',
            'cluster',
            '--name',
            CLUSTER,
            '--image',
            'kindest/node:v1.36.1',
            '--wait',
            '120s',
        )
        verify_fresh_bootstrap(args.evidence)
        exercise(args.evidence, args.baseline)
    finally:
        for name, command in {
                'pods.json': ('get', 'pods', '-o', 'json'),
                'events.txt': (
                    'get',
                    'events',
                    '--sort-by=.metadata.creationTimestamp',
                ),
                'seaweedfs.log': (
                    'logs',
                    'deployment/seaweedfs',
                    '--all-containers=true',
                ),
                'credential-rotation.log': (
                    'logs',
                    'deployment/seaweedfs-credential-rotation',
                ),
                'profile.log': (
                    'logs',
                    'deployment/kubeflow-pipelines-profile-controller',
                ),
        }.items():
            try:
                (args.evidence / name).write_text(kubectl(*command))
            except (OSError, subprocess.CalledProcessError) as error:
                (args.evidence / name).write_text(
                    getattr(error, 'stderr', None) or str(error))
        delete_cluster()


if __name__ == '__main__':
    main()

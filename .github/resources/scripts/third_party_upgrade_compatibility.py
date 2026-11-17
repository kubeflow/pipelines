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
import hashlib
import json
import subprocess
import sys
import time
import urllib.request
from contextlib import closing, contextmanager
from pathlib import Path

import yaml
from third_party_upgrade_scan import build_profile_image, image_reference

ROOT = Path(__file__).resolve().parents[3]
SEAWEED = ROOT / 'manifests/kustomize/third-party/seaweedfs/base'
PROFILE = (
    ROOT
    / 'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller'
)
CLUSTER = 'third-party-upgrade'
BUCKET = 'mlpipeline'
S3_LOCAL_PORT = 18333
S3_PORT = 8333
PROFILE_LOCAL_PORT = 18080
PROFILE_PORT = 8080
ROLLOUT_TIMEOUT = '360s'
PAYLOAD = b'KFP persistent upgrade compatibility\x00\xff\n' * 128


def run(*args, data=None):
    """Run a command and retain its diagnostic output on failure."""
    try:
        return subprocess.run(
            args, input=data, text=True, capture_output=True, check=True
        ).stdout
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
                    secret['data'][key], validate=True
                ).decode('utf-8')
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


def sync_profile(endpoint, namespace, secret=None):
    attachments = {
        kind: {}
        for kind in (
            'Secret.v1',
            'ConfigMap.v1',
            'Deployment.apps/v1',
            'Service.v1',
        )
    }
    if secret:
        attachments['Secret.v1'][f'{namespace}/mlpipeline-minio-artifact'] = (
            secret
        )
    body = {
        'object': {
            'metadata': {
                'name': namespace,
                'labels': {'pipelines.kubeflow.org/enabled': 'true'},
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
        item
        for item in result['attachments']
        if item.get('kind') == 'Secret'
        and item.get('metadata', {}).get('name') == 'mlpipeline-minio-artifact'
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
    if not any(
        rule['Status'] == 'Enabled' and rule['Expiration']['Days'] == 30
        for rule in rules
    ):
        raise AssertionError('Lifecycle configuration did not survive upgrade')


def seed_old_store(storage, baseline_image, evidence):
    """Seed retained objects and lifecycle configuration on the baseline
    image."""
    credentials = [
        obj
        for obj in storage
        if obj.get('kind') == 'Secret'
        and obj.get('metadata', {}).get('name') == 'mlpipeline-minio-artifact'
    ]
    if len(credentials) != 1:
        raise ValueError(
            'Expected exactly one rendered mlpipeline-minio-artifact Secret'
        )
    admin_credentials = secret_credentials(credentials[0])
    deployment = next(obj for obj in storage if obj['kind'] == 'Deployment')
    deployment['spec']['template']['spec']['containers'][0]['image'] = (
        baseline_image
    )
    apply(
        [
            {
                'apiVersion': 'v1',
                'kind': 'Namespace',
                'metadata': {'name': 'kubeflow'},
            },
            {
                'apiVersion': 'v1',
                'kind': 'ConfigMap',
                'metadata': {'name': 'pipeline-install-config'},
                'data': {
                    'appVersion': 'compatibility',
                    'ARTIFACT_RETENTION_DAYS': '30',
                },
            },
        ]
    )
    apply(storage)
    wait_ready('seaweedfs')
    (evidence / 'before-upgrade-pods.json').write_text(
        kubectl('get', 'pods', '-o', 'json')
    )
    pvc_uid = json.loads(kubectl('get', 'pvc', 'seaweedfs-pvc', '-o', 'json'))[
        'metadata'
    ]['uid']
    with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
        s3 = client(endpoint, *admin_credentials)
        s3.put_object(Bucket=BUCKET, Key='shared/before-upgrade', Body=PAYLOAD)
        s3.put_bucket_lifecycle_configuration(
            Bucket=BUCKET,
            LifecycleConfiguration={
                'Rules': [
                    {
                        'ID': 'private-artifacts',
                        'Status': 'Enabled',
                        'Filter': {'Prefix': 'private-artifacts'},
                        'Expiration': {'Days': 30},
                    }
                ]
            },
        )
        verify_state(s3)
    return admin_credentials, pvc_uid


def upgrade_store(target):
    """Upgrade the deployed store while retaining its PVC."""
    kubectl('set', 'image', 'deployment/seaweedfs', f'seaweedfs={target}')
    wait_ready('seaweedfs')


PROFILE_DEPLOYMENT = 'kubeflow-pipelines-profile-controller'
PROFILE_ROLLBACK_IMAGE = 'docker.io/alpine/k8s:1.37.2'


def profile_running_image(expected: str) -> dict:
    """Require a ready running pod for the requested runtime, not an old
    replica."""
    pods = json.loads(
        kubectl('get', 'pods', '-l', f'app={PROFILE_DEPLOYMENT}', '-o', 'json')
    )['items']
    active = [
        pod for pod in pods if not pod['metadata'].get('deletionTimestamp')
    ]
    if len(active) != 1:
        raise AssertionError('Expected one active profile controller pod')
    pod = active[0]
    container = next(
        item
        for item in pod['spec']['containers']
        if item['name'] == 'profile-controller'
    )
    status = next(
        item
        for item in pod['status']['containerStatuses']
        if item['name'] == 'profile-controller'
    )
    if (
        container['image'] != expected
        or not status.get('ready')
        or 'running' not in status.get('state', {})
        or not status.get('imageID')
    ):
        raise AssertionError(
            'Profile controller is not running the requested image'
        )
    return {
        'image': expected,
        'image_id': status['imageID'],
        'pod_uid': pod['metadata']['uid'],
    }


def exercise_profile_rollback(
    candidate: str,
    secret: dict,
    admin_credentials: tuple[str, str],
    evidence: Path,
) -> None:
    """Change only the runtime image and prove data/credential continuity both
    ways."""
    run(
        'bash',
        str(ROOT / '.github/resources/scripts/docker-pull-with-retry.sh'),
        PROFILE_ROLLBACK_IMAGE,
    )
    run(
        'kind',
        'load',
        'docker-image',
        '--name',
        CLUSTER,
        PROFILE_ROLLBACK_IMAGE,
    )
    receipts = []
    keys = ['private-artifacts/compat-user/after-upgrade']
    for phase, image in [
        ('rollback', PROFILE_ROLLBACK_IMAGE),
        ('restore', candidate),
    ]:
        kubectl(
            'set',
            'image',
            f'deployment/{PROFILE_DEPLOYMENT}',
            f'profile-controller={image}',
        )
        wait_ready(PROFILE_DEPLOYMENT)
        receipt = profile_running_image(image)
        with forward(
            PROFILE_DEPLOYMENT, PROFILE_LOCAL_PORT, PROFILE_PORT, evidence
        ) as webhook:
            for _ in range(2):
                if (
                    sync_profile(webhook, 'compat-user', secret)['data']
                    != secret['data']
                ):
                    raise AssertionError(
                        f'Profile {phase} replaced existing credentials'
                    )
        with forward('seaweedfs', S3_LOCAL_PORT, S3_PORT, evidence) as endpoint:
            with closing(client(endpoint, *admin_credentials)) as admin:
                verify_state(admin)
            with closing(client(endpoint, *secret_credentials(secret))) as user:
                for key in keys:
                    verify_payload(user, key)
                key = f'private-artifacts/compat-user/after-{phase}'
                user.put_object(Bucket=BUCKET, Key=key, Body=PAYLOAD)
                verify_payload(user, key)
                keys.append(key)
        receipts.append({'phase': phase, **receipt})
    (evidence / 'profile-rollback.json').write_text(
        json.dumps(receipts, indent=2) + '\n'
    )


def exercise_profile(admin_credentials, evidence):
    """Check retained data and issue reusable credentials through the
    webhook."""
    # The webhook is invoked directly; installing Metacontroller is unnecessary for this contract test.
    profile = [
        obj for obj in rendered(PROFILE) if obj['kind'] != 'DecoratorController'
    ]
    reference = image_reference(ROOT, 'profile-controller')
    sha = run('git', '-C', str(ROOT), 'rev-parse', 'HEAD').strip()
    build = build_profile_image(ROOT, reference, sha)
    if build:
        run('kind', 'load', 'docker-image', '--name', CLUSTER, build['image'])
        deployment = next(
            obj
            for obj in profile
            if obj['kind'] == 'Deployment'
            and obj['metadata']['name']
            == 'kubeflow-pipelines-profile-controller'
        )
        container = next(
            item
            for item in deployment['spec']['template']['spec']['containers']
            if item['name'] == 'profile-controller'
        )
        container['image'] = build['image']
        container['imagePullPolicy'] = 'Never'
        image = json.loads(run('docker', 'image', 'inspect', build['image']))[0]
        build['image_id'] = image['Id']
        (evidence / 'profile-runtime.json').write_text(
            json.dumps(build, indent=2) + '\n'
        )
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
            secret = sync_profile(webhook, 'compat-user')
            repeated = sync_profile(webhook, 'compat-user', secret)
            if repeated['data'] != secret['data']:
                raise AssertionError(
                    'Profile reconciliation replaced existing credentials'
                )
        user = client(
            endpoint,
            *secret_credentials(secret),
        )
        user.put_object(
            Bucket=BUCKET,
            Key='private-artifacts/compat-user/after-upgrade',
            Body=PAYLOAD,
        )
        verify_payload(user, 'private-artifacts/compat-user/after-upgrade')
        verify_state(s3)
    kubectl(
        'rollout', 'restart', 'deployment/kubeflow-pipelines-profile-controller'
    )
    wait_ready('kubeflow-pipelines-profile-controller')
    with forward(
        'kubeflow-pipelines-profile-controller',
        PROFILE_LOCAL_PORT,
        PROFILE_PORT,
        evidence,
    ) as webhook:
        if (
            sync_profile(webhook, 'compat-user', secret)['data']
            != secret['data']
        ):
            raise AssertionError(
                'Profile restart replaced existing credentials'
            )
    exercise_profile_rollback(
        build['image'] if build else reference,
        secret,
        admin_credentials,
        evidence,
    )
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
        kubectl('get', 'pvc', 'seaweedfs-pvc', '-o', 'json')
    )['metadata']['uid']
    if final_uid != pvc_uid:
        raise AssertionError('Upgrade replaced the PVC')


def write_result(evidence, baseline_image, target, pvc_uid):
    """Record the actual image pair and verified persistence boundaries."""
    (evidence / 'result.json').write_text(
        json.dumps(
            {
                'result': 'passed',
                'old_image': baseline_image,
                'new_image': target,
                'pvc_uid': pvc_uid,
                'payload_sha256': hashlib.sha256(PAYLOAD).hexdigest(),
                'checks': [
                    'old-image-write',
                    'upgrade-read',
                    'lifecycle-persistence',
                    'profile-health',
                    'profile-iam-credentials',
                    'profile-idempotence',
                    'profile-restart-credentials',
                    'profile-rollback-running-image',
                    'profile-rollback-health-reconcile-idempotence',
                    'profile-rollback-credentials-and-object-continuity',
                    'profile-restore-running-image',
                    'profile-restore-health-reconcile-idempotence',
                    'profile-restore-credentials-and-object-continuity',
                    'new-image-write',
                    'restart-read',
                    'restart-credentials',
                    'same-pvc',
                ],
            },
            indent=2,
        )
        + '\n'
    )


def storage_image(storage):
    """Select exactly one SeaweedFS container from rendered manifests."""
    images = [
        container['image']
        for obj in storage
        if obj.get('kind') == 'Deployment'
        and obj.get('metadata', {}).get('name') == 'seaweedfs'
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
    baseline_image = storage_image(
        rendered(baseline / SEAWEED.relative_to(ROOT))
    )
    credentials, pvc_uid = seed_old_store(storage, baseline_image, evidence)
    upgrade_store(target)
    secret = exercise_profile(credentials, evidence)
    verify_after_restart(credentials, secret, pvc_uid, evidence)
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
            'profile.log': (
                'logs',
                'deployment/kubeflow-pipelines-profile-controller',
            ),
        }.items():
            try:
                (args.evidence / name).write_text(kubectl(*command))
            except (OSError, subprocess.CalledProcessError) as error:
                (args.evidence / name).write_text(
                    getattr(error, 'stderr', None) or str(error)
                )
        delete_cluster()


if __name__ == '__main__':
    main()

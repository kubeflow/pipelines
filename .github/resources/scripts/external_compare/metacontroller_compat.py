#!/usr/bin/env python3
"""Exercise real KFP namespace decoration across a Metacontroller upgrade."""

import argparse
import hashlib
import importlib
import json
import subprocess
import sys
import time
from pathlib import Path

NAMESPACES = ('meta-before', 'meta-after')
CONFIGMAPS = (
    'kfp-launcher',
    'metadata-grpc-configmap',
    'artifact-repositories',
)
SECRET = 'mlpipeline-minio-artifact'
META_PATH = 'manifests/kustomize/third-party/metacontroller/base'
ISTIO_PATH = 'manifests/kustomize/third-party/istio/install'


def credential_fingerprint(secret):
    values = secret.get('data', {})
    if not all(
        isinstance(values.get(key), str) and values[key]
        for key in ('accesskey', 'secretkey')
    ):
        raise AssertionError(
            'Actual reconciled credential Secret is incomplete'
        )
    return hashlib.sha256(
        json.dumps(values, sort_keys=True).encode()
    ).hexdigest()


def require_unchanged_secret(before, after):
    if before['metadata']['uid'] != after['metadata']['uid']:
        raise AssertionError('Reconciliation replaced the credential Secret')
    if credential_fingerprint(before) != credential_fingerprint(after):
        raise AssertionError('Reconciliation changed existing credentials')


def select_statefulset(objects):
    matches = [
        obj
        for obj in objects
        if obj
        and obj.get('kind') == 'StatefulSet'
        and obj.get('metadata', {}).get('name') == 'metacontroller'
    ]
    if len(matches) != 1:
        raise AssertionError(
            'Expected exactly one production Metacontroller StatefulSet'
        )
    return matches[0]


def set_metacontroller_image(objects, image):
    if '@sha256:' not in image:
        raise ValueError(
            'Metacontroller compatibility requires an immutable image reference'
        )
    workload = select_statefulset(objects)
    containers = [
        c
        for c in workload['spec']['template']['spec']['containers']
        if c['name'] == 'metacontroller'
    ]
    if len(containers) != 1:
        raise AssertionError('Expected one Metacontroller container')
    containers[0]['image'] = image


def require_live_images(pods, expected):
    """Record ready pods requesting the exact immutable reference, not just a tag."""
    found = []
    for pod in pods.get('items', []):
        for container in pod['spec']['containers']:
            if container.get('name') != 'metacontroller':
                continue
            statuses = [
                s
                for s in pod.get('status', {}).get('containerStatuses', [])
                if s['name'] == 'metacontroller'
            ]
            if (
                container['image'] != expected
                or len(statuses) != 1
                or not statuses[0].get('ready')
                or not statuses[0].get('imageID')
            ):
                raise AssertionError(
                    'Metacontroller pod does not run the expected ready immutable image'
                )
            found.append(
                {
                    'pod': pod['metadata']['name'],
                    'pod_uid': pod['metadata']['uid'],
                    'requested_image': container['image'],
                    'image_id': statuses[0]['imageID'],
                }
            )
    if len(found) != 1:
        raise AssertionError('Expected exactly one ready Metacontroller pod')
    return found


class Exercise:
    def __init__(self, source, pairs, evidence, seaweed):
        self.source = source.resolve()
        self.evidence = evidence.resolve()
        self.pairs = pairs
        self.seaweed = seaweed
        sys.path.insert(0, str(self.source / '.github/resources/scripts'))
        self.c = importlib.import_module('third_party_upgrade_compatibility')
        self.c.CLUSTER = 'metacontroller-upgrade'
        self.c.ROOT = self.source
        self.c.PROFILE = (
            self.source
            / 'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller'
        )
        self.c.SEAWEED = (
            self.source / 'manifests/kustomize/third-party/seaweedfs/base'
        )
        self.checks = []

    def kubectl(self, namespace, *args):
        return self.c.run(
            'kubectl',
            '--context',
            f'kind-{self.c.CLUSTER}',
            '-n',
            namespace,
            *args,
        )

    def get(self, namespace, resource, name):
        output = self.kubectl(
            namespace, 'get', resource, name, '--ignore-not-found', '-o', 'json'
        )
        return json.loads(output) if output.strip() else None

    def wait(self, predicate, description, seconds=180):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            result = predicate()
            if result:
                return result
            time.sleep(2)
        raise AssertionError(f'Timed out waiting for {description}')

    def wait_attachments(self, namespace):
        def complete():
            secret = self.get(namespace, 'secret', SECRET)
            maps = [
                self.get(namespace, 'configmap', name) for name in CONFIGMAPS
            ]
            if not secret or not all(maps):
                return None
            credential_fingerprint(secret)
            if namespace not in maps[0].get('data', {}).get(
                'defaultPipelineRoot', ''
            ):
                raise AssertionError(
                    'Reconciled launcher config has wrong namespace scope'
                )
            return secret

        return self.wait(
            complete, f'actual Secret and three ConfigMaps in {namespace}'
        )

    def create_namespace(self, namespace):
        self.c.apply(
            [
                {
                    'apiVersion': 'v1',
                    'kind': 'Namespace',
                    'metadata': {
                        'name': namespace,
                        'labels': {'pipelines.kubeflow.org/enabled': 'true'},
                    },
                }
            ]
        )
        return self.wait_attachments(namespace)

    def repair_attachment(self, namespace, name):
        before = self.get(namespace, 'configmap', name)
        self.kubectl(namespace, 'delete', 'configmap', name)
        after = self.wait(
            lambda: self.get(namespace, 'configmap', name),
            f'{name} recreation by Metacontroller',
        )
        if after['metadata']['uid'] == before['metadata']['uid'] or after.get(
            'data'
        ) != before.get('data'):
            raise AssertionError(
                'Attachment was not recreated with unchanged desired content'
            )
        self.checks.append(f'{namespace}-{name}-recreated')

    def meta_ready(self, expected):
        self.kubectl(
            'kubeflow',
            'rollout',
            'status',
            'statefulset/metacontroller',
            '--timeout=360s',
        )
        pods = json.loads(
            self.kubectl(
                'kubeflow',
                'get',
                'pods',
                '-l',
                'app=metacontroller',
                '-o',
                'json',
            )
        )
        return require_live_images(pods, expected)

    def image_receipt(self, image):
        self.c.run('docker', 'pull', '--platform', 'linux/amd64', image)
        metadata = json.loads(self.c.run('docker', 'image', 'inspect', image))[
            0
        ]
        if metadata['Os'] != 'linux' or metadata['Architecture'] != 'amd64':
            raise AssertionError('Unexpected compatibility image platform')
        return {
            'reference': image,
            'image_id': metadata['Id'],
            'repo_digests': metadata.get('RepoDigests', []),
            'platform': 'linux/amd64',
        }

    def check_s3(self, secret, namespace, key):
        with self.c.forward(
            'seaweedfs', self.c.S3_LOCAL_PORT, self.c.S3_PORT, self.evidence
        ) as endpoint:
            client = self.c.client(endpoint, *self.c.secret_credentials(secret))
            path = f'private-artifacts/{namespace}/{key}'
            client.put_object(
                Bucket=self.c.BUCKET, Key=path, Body=self.c.PAYLOAD
            )
            self.c.verify_payload(client, path)
        self.checks.append(f'{namespace}-iam-s3-write-read')

    def require_old_object(self, secret):
        with self.c.forward(
            'seaweedfs', self.c.S3_LOCAL_PORT, self.c.S3_PORT, self.evidence
        ) as endpoint:
            self.c.verify_payload(
                self.c.client(endpoint, *self.c.secret_credentials(secret)),
                'private-artifacts/meta-before/before-upgrade',
            )
        self.checks.append('existing-credentials-read-before-upgrade-object')

    def check_isolation(self, secret):
        from botocore.exceptions import ClientError

        with self.c.forward(
            'seaweedfs', self.c.S3_LOCAL_PORT, self.c.S3_PORT, self.evidence
        ) as endpoint:
            client = self.c.client(endpoint, *self.c.secret_credentials(secret))
            try:
                client.get_object(
                    Bucket=self.c.BUCKET,
                    Key='private-artifacts/meta-before/before-upgrade',
                )
            except ClientError as error:
                if (
                    error.response.get('ResponseMetadata', {}).get(
                        'HTTPStatusCode'
                    )
                    != 403
                ):
                    raise
            else:
                raise AssertionError(
                    'New namespace credentials can read another namespace private object'
                )
        self.checks.append('cross-namespace-private-read-denied')

    def delete_namespace(self, namespace):
        self.kubectl(
            namespace, 'delete', 'namespace', namespace, '--wait=false'
        )
        self.wait(
            lambda: self.get(namespace, 'namespace', namespace) is None,
            f'{namespace} deletion without stuck finalizers',
        )
        self.checks.append(f'{namespace}-deleted-without-stuck-finalizers')

    def exercise(self):
        sha = self.c.run(
            'git', '-C', str(self.source), 'rev-parse', 'HEAD'
        ).strip()
        if sha != self.pairs['source_sha']:
            raise AssertionError(
                'Source differs from the requested release candidate'
            )
        images = self.pairs['pairs']
        old, new = (
            images['metacontroller']['baseline'],
            images['metacontroller']['candidate'],
        )
        seaweed = images['seaweedfs'][self.seaweed]
        receipts = {
            key: self.image_receipt(value)
            for key, value in [
                ('metacontroller_before', old),
                ('metacontroller_after', new),
                ('seaweedfs', seaweed),
            ]
        }
        self.c.run(
            'kind',
            'create',
            'cluster',
            '--name',
            self.c.CLUSTER,
            '--image',
            'kindest/node:v1.36.1',
            '--wait',
            '120s',
        )
        storage = self.c.rendered(self.c.SEAWEED)
        credentials, pvc_uid = self.c.seed_old_store(
            storage, seaweed, self.evidence
        )
        self.c.exercise_profile(credentials, self.evidence)
        istio = self.c.rendered(self.source / ISTIO_PATH)
        crds = [
            obj
            for obj in istio
            if obj and obj.get('kind') == 'CustomResourceDefinition'
        ]
        required = {
            'destinationrules.networking.istio.io',
            'authorizationpolicies.security.istio.io',
        }
        selected = [obj for obj in crds if obj['metadata']['name'] in required]
        if {obj['metadata']['name'] for obj in selected} != required:
            raise AssertionError(
                'Pinned Istio manifests lack required attachment discovery CRDs'
            )
        self.c.apply(selected)
        for name in sorted(required):
            self.kubectl(
                'kubeflow',
                'wait',
                '--for=condition=Established',
                f'crd/{name}',
                '--timeout=120s',
            )
        meta = self.c.rendered(self.source / META_PATH)
        production_image = next(
            c['image']
            for c in select_statefulset(meta)['spec']['template']['spec'][
                'containers'
            ]
            if c['name'] == 'metacontroller'
        )
        if production_image != old:
            raise AssertionError(
                'Baseline Metacontroller reference differs from the exact release pin'
            )
        set_metacontroller_image(meta, old)
        self.c.apply(meta)
        self.kubectl(
            'kubeflow',
            'wait',
            '--for=condition=Established',
            'crd/decoratorcontrollers.metacontroller.k8s.io',
            '--timeout=120s',
        )
        receipts['baseline_pods'] = self.meta_ready(old)
        decorator = [
            obj
            for obj in self.c.rendered(self.c.PROFILE)
            if obj and obj.get('kind') == 'DecoratorController'
        ]
        if (
            len(decorator) != 1
            or decorator[0]['spec']['resources'][0]['resource'] != 'namespaces'
        ):
            raise AssertionError(
                'Expected actual KFP Namespace DecoratorController'
            )
        self.c.apply(decorator)
        before = self.create_namespace(NAMESPACES[0])
        self.checks.append('baseline-real-namespace-reconciliation')
        self.repair_attachment(NAMESPACES[0], 'kfp-launcher')
        self.check_s3(before, NAMESPACES[0], 'before-upgrade')
        self.kubectl(
            'kubeflow',
            'set',
            'image',
            'statefulset/metacontroller',
            f'metacontroller={new}',
        )
        receipts['upgraded_pods'] = self.meta_ready(new)
        self.repair_attachment(NAMESPACES[0], 'metadata-grpc-configmap')
        require_unchanged_secret(before, self.wait_attachments(NAMESPACES[0]))
        self.checks.append(
            'upgrade-preserves-existing-secret-uid-and-credentials'
        )
        self.require_old_object(before)
        self.kubectl(
            'kubeflow', 'rollout', 'restart', 'statefulset/metacontroller'
        )
        receipts['restarted_pods'] = self.meta_ready(new)
        self.repair_attachment(NAMESPACES[0], 'artifact-repositories')
        require_unchanged_secret(before, self.wait_attachments(NAMESPACES[0]))
        self.checks.append('metacontroller-restart-preserves-credentials')
        self.kubectl(
            'kubeflow',
            'rollout',
            'restart',
            'deployment/kubeflow-pipelines-profile-controller',
        )
        self.c.wait_ready('kubeflow-pipelines-profile-controller')
        after = self.create_namespace(NAMESPACES[1])
        self.checks.append(
            'new-namespace-reconciles-after-upgrade-and-profile-restart'
        )
        self.check_s3(after, NAMESPACES[1], 'after-upgrade')
        self.check_isolation(after)
        require_unchanged_secret(before, self.wait_attachments(NAMESPACES[0]))
        self.require_old_object(before)
        for namespace in NAMESPACES:
            self.delete_namespace(namespace)
        if (
            self.get('kubeflow', 'pvc', 'seaweedfs-pvc')['metadata']['uid']
            != pvc_uid
        ):
            raise AssertionError(
                'Metacontroller comparison replaced storage PVC'
            )
        self.checks.append('same-storage-pvc')
        receipts['final_pods'] = json.loads(
            self.kubectl('kubeflow', 'get', 'pods', '-o', 'json')
        )
        result = {
            'result': 'passed',
            'source_sha': sha,
            'seaweed_version_lane': self.seaweed,
            'images': receipts,
            'pvc_uid': pvc_uid,
            'checks': self.checks,
            'limitations': [
                'KFP uses Namespace decoration, not Profile CRDs.',
                'Actual KFP decorator has no finalize hook; bounded namespace deletion checks for stuck finalizers, not arbitrary synthetic finalize-hook regressions.',
                'Default kind CNI does not enforce NetworkPolicy; production selectors are retained, but network isolation is not evaluated.',
            ],
        }
        (self.evidence / 'result.json').write_text(
            json.dumps(result, indent=2) + '\n'
        )

    def diagnostics(self):
        for name, command in {
            'pods.json': ('get', 'pods', '-A', '-o', 'json'),
            'namespaces.json': ('get', 'namespaces', '-o', 'json'),
            'events.txt': (
                'get',
                'events',
                '-A',
                '--sort-by=.metadata.creationTimestamp',
            ),
            'metacontroller.log': (
                'logs',
                'statefulset/metacontroller',
                '--all-containers=true',
            ),
            'profile.log': (
                'logs',
                'deployment/kubeflow-pipelines-profile-controller',
            ),
        }.items():
            try:
                output = self.kubectl('kubeflow', *command)
            except (OSError, subprocess.CalledProcessError) as error:
                output = getattr(error, 'stderr', '') or str(error)
            (self.evidence / name).write_text(output)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--pairs', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument(
        '--seaweed', choices=('baseline', 'candidate'), default='baseline'
    )
    args = parser.parse_args()
    args.evidence.mkdir(parents=True, exist_ok=True)
    exercise = Exercise(
        args.source,
        json.loads(args.pairs.read_text()),
        args.evidence,
        args.seaweed,
    )
    try:
        exercise.exercise()
    finally:
        exercise.diagnostics()
        exercise.c.delete_cluster()


if __name__ == '__main__':
    main()

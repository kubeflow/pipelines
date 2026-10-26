# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Exercise actual viewer reconciliation, storage and UI/SAR authorization in
Kind.

Trusted identity headers enter through localhost only. This tests
production KFP authorization, not ingress authentication or real Google
Cloud IAM.
"""
import json
import os
from pathlib import Path
import subprocess
import time
from typing import Callable, Optional, TypeVar
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request
from urllib.request import urlopen

NAMESPACE = 'tensorboard-acceptance'
CONTEXT = 'kind-kfp-tensorboard'
STATE = Path(os.environ['RUNNER_TEMP']) / 'tensorboard-deployed'
ROOT = Path(__file__).resolve().parents[2]
IMAGE = 'ghcr.io/kubeflow/kfp-tensorboard:' + (ROOT /
                                               'VERSION').read_text().strip()
CUSTOM_IMAGE = 'kind-registry:5000/tensorboard:custom-acceptance'
BASE = 'http://127.0.0.1:3000'
MANAGER = 'manager@fixture.invalid'
READER = 'reader@fixture.invalid'
DENIED = 'denied@fixture.invalid'
T = TypeVar('T')


def kube(*args: str, value: Optional[dict] = None) -> str:
    """Run a bounded kubectl command against only the fixture cluster."""
    result = subprocess.run(
        ['kubectl', '--context', CONTEXT, '--request-timeout=30s', *args],
        input=None if value is None else json.dumps(value),
        text=True,
        capture_output=True,
        timeout=330)
    if result.returncode:
        raise RuntimeError(f'kubectl {args[:3]} failed: {result.stderr}')
    return result.stdout


def apply(objects: list[dict]) -> None:
    """Apply fixture objects through a single Kubernetes List."""
    kube(
        'apply',
        '-f',
        '-',
        value={
            'apiVersion': 'v1',
            'kind': 'List',
            'items': objects
        })


def poll(description: str, check: Callable[[], T], timeout: int = 180) -> T:
    """Wait for a condition and retain its last error for diagnostics."""
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            last = check()
            if last:
                return last
        except (HTTPError, OSError, ValueError, KeyError,
                AssertionError) as error:
            last = str(error)
        time.sleep(2)
    raise AssertionError(f'{description} did not converge: {last}')


def request(path: str,
            method: str = 'GET',
            user: Optional[str] = MANAGER,
            expected: int = 200) -> bytes:
    """Call the UI as a fixture identity and assert the HTTP status."""
    headers = {'kubeflow-userid': user} if user else {}
    req = Request(BASE + '/' + path.lstrip('/'), headers=headers, method=method)
    try:
        with urlopen(req, timeout=90) as response:
            status, body = response.status, response.read()
    except HTTPError as error:
        status, body = error.code, error.read()
    assert status == expected, (method, path, status, expected, body[:500])
    return body


def role(name: str,
         verbs: list[str],
         subjects: list[dict],
         group: str = 'kubeflow.org',
         resources: Optional[list[str]] = None,
         cluster: bool = False) -> list[dict]:
    """Build a matching Role and binding with explicit fixture subjects."""
    metadata = {'name': 'tensorboard-acceptance-' + name}
    if not cluster:
        metadata['namespace'] = NAMESPACE
    kind = 'ClusterRole' if cluster else 'Role'
    return [{
        'apiVersion':
            'rbac.authorization.k8s.io/v1',
        'kind':
            kind,
        'metadata':
            metadata,
        'rules': [{
            'apiGroups': [group],
            'resources': resources or ['viewers'],
            'verbs': verbs
        }]
    }, {
        'apiVersion': 'rbac.authorization.k8s.io/v1',
        'kind': kind + 'Binding',
        'metadata': metadata,
        'subjects': subjects,
        'roleRef': {
            'apiGroup': 'rbac.authorization.k8s.io',
            'kind': kind,
            'name': metadata['name']
        }
    }]


def rbac_objects() -> list[dict]:
    """Build namespace, viewer grants and fixture credential/script objects."""
    objects = [{
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'metadata': {
            'name': NAMESPACE
        }
    }]
    for name, user, verbs in [('manager', MANAGER, ['get', 'create', 'delete']),
                              ('reader', READER, ['get'])]:
        objects += role(name, verbs, [{
            'kind': 'User',
            'name': user,
            'apiGroup': 'rbac.authorization.k8s.io'
        }])
    objects += role('ui', ['get', 'create', 'delete'], [{
        'kind': 'ServiceAccount',
        'name': 'ml-pipeline-ui',
        'namespace': 'kubeflow'
    }])
    objects += role(
        'sar', ['create'], [{
            'kind': 'ServiceAccount',
            'name': 'ml-pipeline',
            'namespace': 'kubeflow'
        }],
        group='authorization.k8s.io',
        resources=['subjectaccessreviews'],
        cluster=True)
    objects += [{
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'name': 'viewer-storage',
            'namespace': NAMESPACE
        },
        'stringData': {
            'credentials':
                '[default]\naws_access_key_id=fixture\naws_secret_access_key=fixture-secret\n'
        }
    }, {
        'apiVersion': 'v1',
        'kind': 'ConfigMap',
        'metadata': {
            'name': 'event-writer',
            'namespace': NAMESPACE
        },
        'data': {
            'events.py': Path('test/tensorboard/events.py').read_text()
        }
    }]
    return objects


def storage_emulators() -> list[dict]:
    """Build the isolated S3 and GCS services used by deployed viewer pods."""
    objects = []
    stores = [
        ('s3',
         'motoserver/moto:5.1.15@sha256:8860b18917e1ee4d067df19a406ae95dc26037e6965d5b499e1cffd33755b914',
         5000, []),
        ('gcs',
         'fsouza/fake-gcs-server:1.54.0@sha256:3730da0e31f7e5186a90ec4899dc2c336104e7599df400411392ef17e684c31f',
         4443, [
             '-scheme', 'http', '-port', '4443', '-external-url',
             'http://gcs:4443'
         ])
    ]
    for name, image, port, args in stores:
        objects += [{
            'apiVersion': 'apps/v1',
            'kind': 'Deployment',
            'metadata': {
                'name': name,
                'namespace': NAMESPACE
            },
            'spec': {
                'selector': {
                    'matchLabels': {
                        'app': name
                    }
                },
                'template': {
                    'metadata': {
                        'labels': {
                            'app': name
                        }
                    },
                    'spec': {
                        'containers': [{
                            'name': name,
                            'image': image,
                            'args': args,
                            'ports': [{
                                'containerPort': port
                            }]
                        }]
                    }
                }
            }
        }, {
            'apiVersion': 'v1',
            'kind': 'Service',
            'metadata': {
                'name': name,
                'namespace': NAMESPACE
            },
            'spec': {
                'selector': {
                    'app': name
                },
                'ports': [{
                    'port': port,
                    'targetPort': port
                }]
            }
        }]
    return objects


def enable_multiuser_authz() -> None:
    """Enable production viewer watches and fixture SAR checks."""
    # Use the production multi-user viewer grants and watch configuration.
    # Ingress/Istio are unnecessary for this localhost authorization fixture.
    viewer_base = Path(
        'manifests/kustomize/base/installs/multi-user/viewer-controller')
    for filename in ['cluster-role.yaml', 'cluster-role-binding.yaml']:
        resource = json.loads(
            kube('create', '--dry-run=client', '-f',
                 str(viewer_base / filename), '-o', 'json'))
        for subject in resource.get('subjects', []):
            if subject['kind'] == 'ServiceAccount':
                subject['namespace'] = 'kubeflow'
        apply([resource])
    kube('-n', 'kubeflow', 'patch', 'deployment/ml-pipeline-viewer-crd',
         '--type=strategic', '--patch-file',
         str(viewer_base / 'deployment-patch.yaml'))
    kube('-n', 'kubeflow', 'rollout', 'status',
         'deployment/ml-pipeline-viewer-crd', '--timeout=180s')
    # Use the real KFP AuthService and Kubernetes SAR, with trusted fixture users.
    kube('-n', 'kubeflow', 'set', 'env', 'deployment/ml-pipeline',
         'MULTIUSER=true', 'KUBEFLOW_USERID_HEADER=kubeflow-userid',
         'KUBEFLOW_USERID_PREFIX=')
    kube('-n', 'kubeflow', 'set', 'env', 'deployment/ml-pipeline-ui',
         'ENABLE_AUTHZ=true', 'KUBEFLOW_USERID_HEADER=kubeflow-userid',
         'KUBEFLOW_USERID_PREFIX=')
    for name in ['ml-pipeline', 'ml-pipeline-ui']:
        kube('-n', 'kubeflow', 'rollout', 'status', 'deployment/' + name,
             '--timeout=300s')


def setup() -> None:
    """Provision storage and identities before starting UI checks."""
    apply(rbac_objects() + storage_emulators())
    for name in ['s3', 'gcs']:
        kube('-n', NAMESPACE, 'rollout', 'status', 'deployment/' + name,
             '--timeout=180s')
    enable_multiuser_authz()


def pod_template() -> dict:
    """Build the non-root viewer pod settings and storage credentials."""
    return {
        'spec': {
            'securityContext': {
                'runAsNonRoot': True,
                'runAsUser': 1000,
                'runAsGroup': 1000
            },
            'containers': [{
                'name':
                    'viewer',
                'env': [{
                    'name': 'AWS_SHARED_CREDENTIALS_FILE',
                    'value': '/credentials/credentials'
                }, {
                    'name': 'AWS_DEFAULT_REGION',
                    'value': 'us-east-1'
                }, {
                    'name': 'AWS_EC2_METADATA_DISABLED',
                    'value': 'true'
                }, {
                    'name': 'S3_ENDPOINT',
                    'value': 'http://s3:5000'
                }, {
                    'name': 'STORAGE_EMULATOR_HOST',
                    'value': 'http://gcs:4443'
                }],
                'volumeMounts': [{
                    'name': 'credentials',
                    'mountPath': '/credentials',
                    'readOnly': True
                }]
            }],
            'volumes': [{
                'name': 'credentials',
                'secret': {
                    'secretName': 'viewer-storage'
                }
            }]
        }
    }


def upload(scheme: str, step: int) -> None:
    """Write events through the candidate runtime in a bounded Job."""
    template = pod_template()
    template['spec']['restartPolicy'] = 'Never'
    container = template['spec']['containers'][0]
    container.update(
        image=IMAGE,
        command=['python', '/fixture/events.py', scheme,
                 str(step)],
        imagePullPolicy='IfNotPresent')
    container['volumeMounts'].append({
        'name': 'script',
        'mountPath': '/fixture'
    })
    template['spec']['volumes'].append({
        'name': 'script',
        'configMap': {
            'name': 'event-writer'
        }
    })
    name = f'events-{scheme}-{step}'
    apply([{
        'apiVersion': 'batch/v1',
        'kind': 'Job',
        'metadata': {
            'name': name,
            'namespace': NAMESPACE
        },
        'spec': {
            'backoffLimit': 0,
            'activeDeadlineSeconds': 240,
            'template': template
        }
    }])
    try:
        kube('-n', NAMESPACE, 'wait', '--for=condition=complete', 'job/' + name,
             '--timeout=250s')
    finally:
        print(kube('-n', NAMESPACE, 'logs', 'job/' + name), flush=True)


def viewer_path(logdir: str, image: str = IMAGE) -> str:
    """Encode a viewer request including the same credential pod template."""
    return '/apps/tensorboard?' + urlencode({
        'namespace': NAMESPACE,
        'logdir': logdir,
        'image': image,
        'podtemplatespec': json.dumps(pod_template())
    })


def create(logdir: str, image: str) -> tuple[str, str, str, str]:
    """Create a viewer and verify the reconciled pod image and identity."""
    proxy = request(viewer_path(logdir, image), 'POST').decode()
    metadata = json.loads(request(viewer_path(logdir, image)))
    assert metadata['image'] == image, metadata
    assert metadata['proxyPath'] == proxy, metadata
    viewers = json.loads(kube('-n', NAMESPACE, 'get', 'viewers', '-o',
                              'json'))['items']
    viewer = next(
        v for v in viewers if v['spec']['tensorboardSpec']['logDir'] == logdir)
    name = viewer['metadata']['name']
    poll(
        'viewer Deployment creation',
        lambda: kube('-n', NAMESPACE, 'get', 'deployment/' + name +
                     '-deployment', '--ignore-not-found', '-o', 'name'))
    kube('-n', NAMESPACE, 'rollout', 'status',
         'deployment/' + name + '-deployment', '--timeout=180s')
    pods = json.loads(
        kube('-n', NAMESPACE, 'get', 'pods', '-l', 'viewer=' + name, '-o',
             'json'))['items']
    pod = next(p for p in pods if p['status'].get('phase') == 'Running')
    assert pod['spec']['containers'][0]['image'] == image, pod['spec']
    assert pod['spec']['securityContext']['runAsUser'] == 1000
    status = pod['status']['containerStatuses'][0]
    assert status['ready'] and status['imageID'], status
    if logdir.startswith('s3:'):
        kube(
            '-n', NAMESPACE, 'exec', pod['metadata']['name'], '--', 'python',
            '-c',
            "import boto3; assert boto3.Session().get_credentials().method == 'shared-credentials-file'"
        )
    return proxy, name, pod['metadata']['uid'], status['imageID'].split('@')[-1]


def viewer_process(name: str) -> dict:
    """Capture process identity so reload cannot pass by restarting a pod."""
    pods = json.loads(
        kube('-n', NAMESPACE, 'get', 'pods', '-l', 'viewer=' + name, '-o',
             'json'))['items']
    assert len(pods) == 1, pods
    pod = pods[0]
    assert not pod['metadata'].get('deletionTimestamp'), pod['metadata']
    status = pod['status']['containerStatuses'][0]
    assert status['ready'] and status['state'].get('running'), status
    assert status['containerID'], status
    return {
        'pod_uid': pod['metadata']['uid'],
        'container_id': status['containerID'],
        'restart_count': status['restartCount']
    }


def scalar(proxy: str, step: int) -> bool:
    """Find one expected scalar step through the real UI proxy."""
    raw = request(proxy + 'data/plugin/scalars/scalars?' + urlencode({
        'run': 'legacy',
        'tag': 'accuracy'
    }))
    values = json.loads(raw)
    return any(
        row[1] == step and abs(row[2] - step / 4) < 0.0001 for row in values)


def verify(proxy: str, step: int) -> None:
    """Require scalar, graph and PNG data from the deployed viewer."""
    poll(
        f'scalar step {step} through UI proxy',
        lambda: scalar(proxy, step),
        timeout=180)
    assert b'input' in request(proxy + 'data/plugin/graphs/graph?run=legacy')
    images = json.loads(
        request(proxy + 'data/plugin/images/images?run=legacy&tag=preview'))
    assert images
    image = request(proxy + 'data/plugin/images/individualImage?' +
                    images[0]['query'])
    assert image.startswith(b'\x89PNG'), image[:100]


def delete(logdir: str, name: str) -> None:
    """Delete a viewer and wait for its owned resources to vanish."""
    request(viewer_path(logdir), 'DELETE')
    for resource in [
            'viewer/' + name, 'deployment/' + name + '-deployment',
            'service/' + name + '-service'
    ]:
        kube('-n', NAMESPACE, 'wait', '--for=delete', resource,
             '--timeout=180s')


def exercise_scheme(scheme: str) -> dict:
    """Check one storage reader, authorization, reload and viewer lifecycle."""
    upload(scheme, 1)
    logdir = f'{scheme}://tensorboard-fixtures/logs'
    path = viewer_path(logdir)
    for method in ['POST', 'DELETE']:
        request(path, method, READER, expected=401)
    proxy, name, old_uid, image_id = create(logdir, IMAGE)
    verify(proxy, 1)
    reload_process = viewer_process(name)
    assert reload_process['pod_uid'] == old_uid
    request(proxy, user=READER)
    for user in [DENIED, None]:
        request(proxy, user=user, expected=403)
    subprocess.run(
        ['node', 'test/tensorboard/browser.cjs'],
        check=True,
        env={
            **os.environ, 'TENSORBOARD_URL': BASE + '/' + proxy,
            'TENSORBOARD_USER': MANAGER,
            'TENSORBOARD_ARTIFACT_PREFIX': str(STATE / (scheme + '-browser'))
        })
    upload(scheme, 2)
    verify(proxy, 2)
    current_process = viewer_process(name)
    assert current_process == reload_process, (reload_process, current_process)
    upload(scheme, 3)
    verify(proxy, 3)
    current_process = viewer_process(name)
    assert current_process == reload_process, (reload_process, current_process)
    # Restart using the normal default, then an explicitly selected image.
    delete(logdir, name)
    proxy, name, new_uid, restarted_id = create(logdir, IMAGE)
    assert new_uid != old_uid and restarted_id == image_id
    verify(proxy, 3)
    delete(logdir, name)
    proxy, name, custom_uid, custom_id = create(logdir, CUSTOM_IMAGE)
    assert custom_uid != new_uid and custom_id == image_id
    verify(proxy, 3)
    delete(logdir, name)
    return {
        'scheme': scheme,
        'image_id': image_id,
        'initial_pod_uid': old_uid,
        'reload_process': reload_process,
        'restarted_pod_uid': new_uid,
        'custom_pod_uid': custom_uid,
        'passed': True
    }


def run() -> None:
    """Own UI forwarding and publish a receipt even when a case fails."""
    setup()
    log = (STATE / 'ui-forward.log').open('w')
    forward = subprocess.Popen([
        'kubectl', '--context', CONTEXT, '-n', 'kubeflow', 'port-forward',
        '--address=127.0.0.1', 'service/ml-pipeline-ui', '3000:80'
    ],
                               stdout=log,
                               stderr=subprocess.STDOUT)
    report = {
        'source':
            subprocess.check_output(['git', 'rev-parse', 'HEAD'],
                                    text=True).strip(),
        'cases': [],
        'passed':
            False
    }
    try:
        poll(
            'UI forward',
            lambda: 'Forwarding from' in (STATE / 'ui-forward.log').read_text(),
            timeout=30)
        for scheme in ['s3', 'gs']:
            report['cases'].append(exercise_scheme(scheme))
        report['passed'] = True
    finally:
        (STATE / 'report.json').write_text(json.dumps(report, indent=2))
        forward.terminate()
        forward.wait(timeout=15)
        log.close()


if __name__ == '__main__':
    STATE.mkdir(parents=True, exist_ok=True)
    assert subprocess.check_output(['kubectl', 'config', 'current-context'],
                                   text=True).strip() == CONTEXT
    run()

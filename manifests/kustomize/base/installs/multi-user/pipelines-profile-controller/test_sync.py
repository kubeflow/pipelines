import base64
import threading

import pytest
import requests
import sync
from sync import server_factory

NAMESPACE = 'my-name'


def _base_attachments():
    return {
        'Secret.v1': {},
        'ConfigMap.v1': {},
        'Deployment.apps/v1': {},
        'Service.v1': {},
        'ServiceAccount.v1': {},
        'Role.rbac.authorization.k8s.io/v1': {},
        'RoleBinding.rbac.authorization.k8s.io/v1': {},
    }


def _ready_attachments(artifacts_proxy_enabled=False):
    attachments = _base_attachments()
    attachments['Secret.v1'] = {
        f'{NAMESPACE}/mlpipeline-minio-artifact': {
            'apiVersion': 'v1',
            'kind': 'Secret',
            'metadata': {
                'name': 'mlpipeline-minio-artifact',
                'namespace': NAMESPACE,
            },
            'data': {
                'accesskey': 'existing-access-key',
                'secretkey': 'existing-secret-key',
            },
        },
        f'{NAMESPACE}/default-editor.service-account-token': {
            'kind': 'Secret'
        },
    }
    attachments['ConfigMap.v1'] = {
        f'{NAMESPACE}/kfp-launcher': {
            'kind': 'ConfigMap'
        },
        f'{NAMESPACE}/metadata-grpc-configmap': {
            'kind': 'ConfigMap'
        },
        f'{NAMESPACE}/artifact-repositories': {
            'kind': 'ConfigMap'
        },
    }
    attachments['ServiceAccount.v1'] = {}
    attachments['Role.rbac.authorization.k8s.io/v1'] = {
        f'{NAMESPACE}/configmap-reader': {
            'kind': 'Role'
        },
        f'{NAMESPACE}/ml-pipeline-driver-pods-reader': {
            'kind': 'Role'
        },
        f'{NAMESPACE}/ml-pipeline-driver-pvc-editor': {
            'kind': 'Role'
        },
        f'{NAMESPACE}/artifact-secret-reader': {
            'kind': 'Role'
        },
    }
    attachments['RoleBinding.rbac.authorization.k8s.io/v1'] = {
        f'{NAMESPACE}/configmap-reader-binding': {
            'kind': 'RoleBinding'
        },
        f'{NAMESPACE}/ml-pipeline-driver-pods-reader-binding': {
            'kind': 'RoleBinding'
        },
        f'{NAMESPACE}/ml-pipeline-driver-pvc-editor-binding': {
            'kind': 'RoleBinding'
        },
        f'{NAMESPACE}/artifact-secret-reader-binding': {
            'kind': 'RoleBinding'
        },
    }
    if artifacts_proxy_enabled:
        attachments['Deployment.apps/v1'] = {
            f'{NAMESPACE}/ml-pipeline-ui-artifact': {
                'kind': 'Deployment'
            },
        }
        attachments['Service.v1'] = {
            f'{NAMESPACE}/ml-pipeline-ui-artifact': {
                'kind': 'Service'
            },
        }
    return attachments


def _observed(attachments, pipeline_enabled=True):
    labels = {
        'pipelines.kubeflow.org/enabled': 'true'
    } if pipeline_enabled else {}
    return {
        'object': {
            'metadata': {
                'name': NAMESPACE,
                'labels': labels,
            }
        },
        'attachments': attachments,
    }


def _count_kind(resources, kind_name):
    return sum(1 for resource in resources if resource.get('kind') == kind_name)


@pytest.fixture
def sync_server(request):
    settings = dict(request.param)
    settings['controller_port'] = 0
    server = server_factory(**settings)
    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()
    try:
        yield server
    finally:
        server.shutdown()
        server.server_close()


def _post(server, payload):
    url = f'http://{server.server_address[0]}:{server.server_address[1]}'
    response = requests.post(url, json=payload, timeout=5)
    response.raise_for_status()
    return response.json()


@pytest.mark.parametrize(
    'sync_server',
    [{
        'frontend_image': 'frontend-image',
        'frontend_tag': 'tag123',
        'disable_istio_sidecar': False,
        'artifacts_proxy_enabled': 'false',
        'artifact_retention_days': '7',
        'cluster_domain': '.svc.cluster.local',
        'object_store_host': 'seaweedfs',
    }],
    indirect=True,
)
def test_sync_pipeline_enabled_without_proxy(sync_server):
    observed = _observed(_ready_attachments(artifacts_proxy_enabled=False))
    results = _post(sync_server, observed)

    assert results['status'] == {'kubeflow-pipelines-ready': 'True'}
    assert _count_kind(results['attachments'], 'ServiceAccount') == 0
    assert _count_kind(results['attachments'], 'Role') == 4
    assert _count_kind(results['attachments'], 'RoleBinding') == 4
    assert _count_kind(results['attachments'], 'Deployment') == 0
    assert _count_kind(results['attachments'], 'Service') == 0


@pytest.mark.parametrize(
    'sync_server',
    [{
        'frontend_image': 'frontend-image',
        'frontend_tag': 'tag123',
        'disable_istio_sidecar': True,
        'artifacts_proxy_enabled': 'true',
        'artifact_retention_days': '7',
        'cluster_domain': '.svc.cluster.local',
        'object_store_host': 'seaweedfs',
    }],
    indirect=True,
)
def test_sync_pipeline_enabled_with_proxy(sync_server):
    observed = _observed(_ready_attachments(artifacts_proxy_enabled=True))
    results = _post(sync_server, observed)

    assert results['status'] == {'kubeflow-pipelines-ready': 'True'}
    deployments = [
        resource for resource in results['attachments']
        if resource.get('kind') == 'Deployment'
    ]
    assert len(deployments) == 1
    image = deployments[0]['spec']['template']['spec']['containers'][0]['image']
    assert image == 'frontend-image:tag123'


@pytest.mark.parametrize(
    'sync_server',
    [{
        'frontend_image': 'frontend-image',
        'frontend_tag': 'tag123',
        'disable_istio_sidecar': True,
        'artifacts_proxy_enabled': 'false',
        'artifact_retention_days': '7',
        'cluster_domain': '.svc.cluster.local',
        'object_store_host': 'seaweedfs',
    }],
    indirect=True,
)
def test_sync_pipeline_disabled_returns_empty(sync_server):
    results = _post(sync_server,
                    _observed(_base_attachments(), pipeline_enabled=False))
    assert results['status'] == {}
    assert results['attachments'] == []


def test_sync_creates_minio_secret_when_missing(monkeypatch):

    class DummyIAM:

        def __init__(self):
            self.put_policy_called = False

        def create_access_key(self, UserName):
            assert UserName == NAMESPACE
            return {
                'AccessKey': {
                    'AccessKeyId': 'AKIA_TEST',
                    'SecretAccessKey': 'SECRET_TEST',
                }
            }

        def put_user_policy(self, **kwargs):
            self.put_policy_called = True
            assert kwargs['UserName'] == NAMESPACE

    class DummyS3:

        def put_bucket_lifecycle_configuration(self, **kwargs):
            assert kwargs['Bucket'] == 'mlpipeline'
            return {}

    dummy_iam = DummyIAM()
    monkeypatch.setattr(sync, 'iam', dummy_iam)
    monkeypatch.setattr(sync, 's3', DummyS3())

    settings = {
        'frontend_image': 'frontend-image',
        'frontend_tag': 'tag123',
        'disable_istio_sidecar': True,
        'artifacts_proxy_enabled': 'false',
        'artifact_retention_days': '7',
        'cluster_domain': '.svc.cluster.local',
        'object_store_host': 'seaweedfs',
        'controller_port': 0,
    }
    server = server_factory(**settings)
    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()
    try:
        attachments = _ready_attachments(artifacts_proxy_enabled=False)
        attachments['Secret.v1'].pop(f'{NAMESPACE}/mlpipeline-minio-artifact')
        results = _post(server, _observed(attachments))
    finally:
        server.shutdown()
        server.server_close()

    minio_secrets = [
        resource for resource in results['attachments']
        if resource.get('kind') == 'Secret' and
        resource.get('metadata', {}).get('name') == 'mlpipeline-minio-artifact'
    ]
    assert len(minio_secrets) == 1
    assert base64.b64decode(
        minio_secrets[0]['data']['accesskey']).decode('utf-8') == 'AKIA_TEST'
    assert base64.b64decode(
        minio_secrets[0]['data']['secretkey']).decode('utf-8') == 'SECRET_TEST'
    assert dummy_iam.put_policy_called


def test_create_iam_client_uses_endpoint(monkeypatch):
    called = {}

    class DummySession:

        def create_client(self,
                          service_name,
                          region_name=None,
                          endpoint_url=None):
            called['service_name'] = service_name
            called['endpoint_url'] = endpoint_url
            return object()

    monkeypatch.setenv('AWS_ENDPOINT_URL', 'http://seaweedfs.kubeflow:8111')
    monkeypatch.setattr(sync, 'session', DummySession())

    sync.create_iam_client()

    assert called['service_name'] == 'iam'
    assert called['endpoint_url'] == 'http://seaweedfs.kubeflow:8111'

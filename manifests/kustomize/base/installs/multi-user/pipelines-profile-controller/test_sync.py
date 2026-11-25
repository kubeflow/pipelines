import base64
import json
import os
import threading
from unittest import mock
from urllib.error import HTTPError

import init_webhook_auth
import pytest
import requests
import sync
from sync import get_settings_from_env
from sync import server_factory

WEBHOOK_AUTH_TOKEN = 'test-webhook-token-with-at-least-32-characters'

PARENT = {
    "metadata": {
        "labels": {
            "pipelines.kubeflow.org/enabled": "true"
        },
        "name": "myName",
    }
}
EXISTING_SECRET = {
    "apiVersion": "v1",
    "kind": "Secret",
    "metadata": {
        "name": "mlpipeline-minio-artifact",
        "namespace": "myName"
    },
    "data": {
        "accesskey": "YWJj",
        "secretkey": "ZGVm"
    },
}
DATA_INCORRECT_ATTACHMENTS = {
    "object": PARENT,
    "attachments": {
        "Secret.v1": {
            "myName/mlpipeline-minio-artifact": EXISTING_SECRET
        },
        "ConfigMap.v1": {},
        "Deployment.apps/v1": {},
        "Service.v1": {},
    },
}
DATA_CORRECT_ATTACHMENTS = {
    "object": PARENT,
    "attachments": {
        "Secret.v1": {
            "myName/mlpipeline-minio-artifact": EXISTING_SECRET
        },
        "ConfigMap.v1": {
            name: {} for name in ("kfp-launcher", "artifact-repositories",
                                 "ml-pipeline-ui-configmap")
        },
        "Deployment.apps/v1": {
            "ml-pipeline-ui-artifact": {}
        },
        "Service.v1": {
            "ml-pipeline-ui-artifact": {}
        },
    },
}
ENV_KFP_VERSION_ONLY = {
    "KFP_VERSION": "x.y.z",
    "CONTROLLER_PORT": "0",
    "ARTIFACTS_PROXY_ENABLED": "true",
}
ENV_IMAGES_NO_TAGS = dict(ENV_KFP_VERSION_ONLY, FRONTEND_IMAGE="frontend-image")
ENV_IMAGES_WITH_TAGS = dict(ENV_IMAGES_NO_TAGS, FRONTEND_TAG="somehash")
ENV_ARTIFACT_PROXY_WITH_ALLOWED_ENDPOINTS = dict(
    ENV_KFP_VERSION_ONLY,
    ALLOWED_ARTIFACT_ENDPOINTS="https://objects.example.com:9443",
    HTTP_BASE_URL="https://artifacts.example.com:9443/pipelines/",
)


@pytest.fixture
def sync_server(request, start_sync_server):
    with mock.patch.dict(os.environ, request.param):
        server = start_sync_server(**get_settings_from_env())
        yield server, request.param


@pytest.mark.parametrize(
    "sync_server,data,expected_ready,expected_image",
    [
        (ENV_KFP_VERSION_ONLY, DATA_INCORRECT_ATTACHMENTS, "False",
         "ghcr.io/kubeflow/kfp-frontend:x.y.z"),
        (ENV_IMAGES_NO_TAGS, DATA_INCORRECT_ATTACHMENTS, "False",
         "frontend-image:x.y.z"),
        (ENV_IMAGES_WITH_TAGS, DATA_INCORRECT_ATTACHMENTS, "False",
         "frontend-image:somehash"),
        (ENV_IMAGES_WITH_TAGS, DATA_CORRECT_ATTACHMENTS, "True",
         "frontend-image:somehash"),
    ],
    indirect=["sync_server"],
)
def test_sync_server_with_pipeline_enabled(sync_server, data, expected_ready,
                                           expected_image):
    server, _ = sync_server
    result = post_sync(server, data["object"], data["attachments"])
    assert result["status"] == {"kubeflow-pipelines-ready": expected_ready}
    deployment = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    assert deployment["spec"]["template"]["spec"]["containers"][0][
        "image"] == expected_image
    assert EXISTING_SECRET in result["attachments"]


def test_sync_server_with_direct_passing_of_settings(start_sync_server):
    server = start_sync_server(
        frontend_image="direct-frontend",
        frontend_tag="direct-tag",
        controller_port=0,
        disable_istio_sidecar=True,
        artifacts_proxy_enabled="true",
        artifact_retention_days="30",
    )
    result = post_sync(server, DATA_CORRECT_ATTACHMENTS["object"],
                       DATA_CORRECT_ATTACHMENTS["attachments"])
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    deployment = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    template = deployment["spec"]["template"]
    assert template["spec"]["containers"][0][
        "image"] == "direct-frontend:direct-tag"
    assert template["metadata"]["annotations"][
        "sidecar.istio.io/inject"] == "false"


def test_sync_server_without_pipeline_enabled(start_sync_server):
    server = start_minimal_server(start_sync_server)
    assert post_sync(server, {}, {}) == {"status": {}, "attachments": []}


def test_sync_server_without_artifact_proxy(start_sync_server):
    server = start_minimal_server(start_sync_server)
    attachments = dict(DATA_CORRECT_ATTACHMENTS["attachments"], **{
        "Deployment.apps/v1": {},
        "Service.v1": {}
    })
    result = post_sync(server, PARENT, attachments)
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    assert not any(item["kind"] in ("Deployment", "Service")
                   for item in result["attachments"])


@pytest.mark.parametrize(
    "sync_server",
    [ENV_ARTIFACT_PROXY_WITH_ALLOWED_ENDPOINTS],
    indirect=True,
)
def test_artifact_proxy_receives_allowed_endpoints(sync_server):
    server, _ = sync_server
    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    existing_secret = {
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'name': 'mlpipeline-minio-artifact',
            'namespace': 'myName',
        },
    }
    response = requests.post(
        url + '/sync',
        json={
            'object': DATA_CORRECT_ATTACHMENTS['object'],
            'attachments': {
                'Secret.v1': {
                    'myName/mlpipeline-minio-artifact': existing_secret
                },
                'ConfigMap.v1': {},
                'Deployment.apps/v1': {},
                'Service.v1': {},
            },
        },
        headers={'Authorization': f'Bearer {WEBHOOK_AUTH_TOKEN}'})
    results = json.loads(response.text)
    artifact_deployment = next(
        child for child in results['attachments']
        if child.get('kind') == 'Deployment' and
        child.get('metadata', {}).get('name') == 'ml-pipeline-ui-artifact')
    container_env = artifact_deployment['spec']['template']['spec'][
        'containers'][0]['env']

    assert {
        'name': 'HTTP_BASE_URL',
        'value': 'https://artifacts.example.com:9443/pipelines/',
    } in container_env

    assert {
        'name': 'ALLOWED_ARTIFACT_ENDPOINTS',
        'value': 'https://objects.example.com:9443',
    } in container_env

    assert {
        'name': 'ARTIFACTS_RATE_LIMIT_DISABLED',
        'value': 'true',
    } in container_env


def test_create_iam_client_uses_endpoint(monkeypatch):
    called = {}

    class DummySession:

        def create_client(self,
                          service_name,
                          region_name=None,
                          endpoint_url=None):
            called["service_name"] = service_name
            called["endpoint_url"] = endpoint_url
            return object()

    monkeypatch.setenv("AWS_ENDPOINT_URL", "http://seaweedfs.kubeflow:8111")
    monkeypatch.setattr(sync, "session", DummySession())

    sync.create_iam_client()

    assert called["service_name"] == "iam"
    assert called["endpoint_url"] == "http://seaweedfs.kubeflow:8111"


def observed_attachments():
    return {}


def post_sync(server, parent, attachments):
    host, port = server.server_address
    response = requests.post(
        f"http://{host}:{port}/sync",
        json={
            "object": parent,
            "attachments": attachments
        },
        headers={'Authorization': f'Bearer {WEBHOOK_AUTH_TOKEN}'},
        timeout=5,
    )
    response.raise_for_status()
    return response.json()


@pytest.fixture
def start_sync_server():
    servers = []

    def start(**settings):
        with mock.patch.dict(os.environ,
                             {'WEBHOOK_AUTH_TOKEN': WEBHOOK_AUTH_TOKEN}):
            server = server_factory(url="127.0.0.1", **settings)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        servers.append((server, thread))
        return server

    yield start
    for server, thread in servers:
        server.shutdown()
        thread.join()
        server.server_close()


def start_minimal_server(start_sync_server):
    return start_sync_server(
        frontend_image="image",
        frontend_tag="tag",
        disable_istio_sidecar=False,
        artifacts_proxy_enabled="false",
        artifact_retention_days=-1,
        controller_port=0,
    )


def get_path(server, path, timeout=5):
    host, port = server.server_address
    return requests.get(f"http://{host}:{port}{path}", timeout=timeout)


def test_healthz_returns_ok(start_sync_server):
    response = get_path(start_minimal_server(start_sync_server), "/healthz")
    assert response.status_code == 200
    assert response.text == "ok"


def test_get_unknown_path_returns_not_found(start_sync_server):
    response = get_path(start_minimal_server(start_sync_server), "/unknown")
    assert response.status_code == 404


def test_healthz_is_not_logged(start_sync_server, capsys):
    server = start_minimal_server(start_sync_server)
    assert get_path(server, "/healthz").status_code == 200
    assert get_path(server, "/unknown").status_code == 404
    logs = capsys.readouterr().err
    assert "/healthz" not in logs
    assert "/unknown" in logs


def test_post_to_healthz_path_is_logged(start_sync_server, capsys):
    server = start_minimal_server(start_sync_server)
    host, port = server.server_address
    with mock.patch.object(
            server.RequestHandlerClass, "sync",
            lambda handler, parent, attachments: {
                "status": {},
                "attachments": []
            }):
        response = requests.post(
            f"http://{host}:{port}/healthz",
            json={
                "object": PARENT,
                "attachments": {}
            },
            headers={'Authorization': f'Bearer {WEBHOOK_AUTH_TOKEN}'},
            timeout=5)
        assert response.status_code == 404
    assert '"POST /healthz' in capsys.readouterr().err


def test_webhook_rejects_forged_requests_before_sync(start_sync_server):
    server = start_minimal_server(start_sync_server)
    host, port = server.server_address
    body = {'object': PARENT, 'attachments': observed_attachments()}
    with mock.patch.object(server.RequestHandlerClass, 'sync') as sync_call:
        for headers in ({}, {'Authorization': 'Bearer wrong-token'}):
            response = requests.post(
                f'http://{host}:{port}/sync',
                json=body,
                headers=headers,
                timeout=5)
            assert response.status_code == 403
        assert sync_call.call_count == 0


def test_webhook_rejects_missing_auth_configuration(monkeypatch):
    monkeypatch.delenv('WEBHOOK_AUTH_TOKEN', raising=False)
    with pytest.raises(ValueError, match='WEBHOOK_AUTH_TOKEN'):
        server_factory(
            frontend_image='image',
            frontend_tag='tag',
            disable_istio_sidecar=False,
            artifacts_proxy_enabled='false',
            artifact_retention_days=-1,
            controller_port=0)


def test_webhook_secret_initializer_preserves_existing_key():
    encoded = base64.b64encode(
        WEBHOOK_AUTH_TOKEN.encode('ascii')).decode('ascii')
    secret = {'metadata': {'resourceVersion': '5'}, 'data': {'token': encoded}}
    request = mock.Mock(return_value=secret)

    init_webhook_auth.initialize_webhook_secret(request)

    request.assert_called_once_with('GET', None)
    assert secret['data']['token'] == encoded


def test_webhook_secret_initializer_fills_stub_without_erasing_data():
    secret = {'metadata': {'resourceVersion': '5'}, 'data': {'other': 'abc'}}
    request = mock.Mock(side_effect=[secret, None])

    init_webhook_auth.initialize_webhook_secret(request)

    assert request.call_args_list[0] == mock.call('GET', None)
    assert request.call_args_list[1].args[0] == 'PUT'
    assert request.call_args_list[1].args[1]['data']['other'] == 'abc'
    assert len(base64.b64decode(secret['data']['token']).decode('ascii')) >= 32


def test_webhook_secret_initializer_retries_conflict():
    conflict = HTTPError('url', 409, 'conflict', {}, None)
    first = {'metadata': {'resourceVersion': '5'}}
    second = {'metadata': {'resourceVersion': '6'}}
    request = mock.Mock(side_effect=[first, conflict, second, None])
    wait = mock.Mock()

    init_webhook_auth.initialize_webhook_secret(request, wait=wait)

    assert request.call_args_list[2] == mock.call('GET', None)
    assert request.call_args_list[3].args[0] == 'PUT'
    wait.assert_called_once_with(1)


def test_healthz_responds_during_slow_sync(start_sync_server):
    server = start_minimal_server(start_sync_server)
    sync_started = threading.Event()
    release_sync = threading.Event()

    def slow_sync(handler, parent, attachments):
        sync_started.set()
        release_sync.wait(timeout=10)
        return {"status": {}, "attachments": []}

    with mock.patch.object(server.RequestHandlerClass, "sync", slow_sync):
        post = threading.Thread(
            target=post_sync, args=(server, PARENT, observed_attachments()))
        post.start()
        try:
            assert sync_started.wait(timeout=5)
            # A single-threaded server could not answer until the sync ends.
            assert get_path(server, "/healthz", timeout=2).status_code == 200
        finally:
            release_sync.set()
            post.join(timeout=10)


def test_import_supports_awscli_botocore_alias(monkeypatch):
    import builtins
    import runpy
    import types

    original_import = builtins.__import__
    alias_registered = False

    def import_with_bundled_botocore(name, *args, **kwargs):
        nonlocal alias_registered
        if name == 'botocore.session' and not alias_registered:
            raise ModuleNotFoundError(
                "No module named 'botocore'", name='botocore')
        if name == 'awscli':
            alias_registered = True
            return types.ModuleType('awscli')
        return original_import(name, *args, **kwargs)

    monkeypatch.setattr(builtins, '__import__', import_with_bundled_botocore)
    with mock.patch.object(sync.botocore.session, 'get_session') as get_session:
        loaded = runpy.run_path(sync.__file__)
    assert alias_registered
    assert loaded['session'] is get_session.return_value
    assert get_session.return_value.create_client.call_args_list[0].args == (
        's3',)


def test_import_does_not_hide_broken_botocore_dependency(monkeypatch):
    import builtins
    import runpy

    original_import = builtins.__import__

    def import_with_broken_dependency(name, *args, **kwargs):
        if name == 'botocore.session':
            raise ModuleNotFoundError(
                "Missing dependency", name='botocore.missing')
        if name == 'awscli':
            pytest.fail(
                'An internal botocore import failure must not trigger fallback')
        return original_import(name, *args, **kwargs)

    monkeypatch.setattr(builtins, '__import__', import_with_broken_dependency)
    with pytest.raises(ModuleNotFoundError, match='Missing dependency'):
        runpy.run_path(sync.__file__)

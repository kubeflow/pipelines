import json
import os
import threading
from unittest import mock

import pytest
import requests
import sync
from sync import get_settings_from_env
from sync import server_factory

KFP_VERSION = "x.y.z"
NAMESPACE = "myName"
PARENT = {
    "metadata": {
        "labels": {
            "pipelines.kubeflow.org/enabled": "true"
        },
        "name": NAMESPACE,
    }
}
EXISTING_SECRET = {
    "apiVersion": "v1",
    "kind": "Secret",
    "metadata": {
        "name": "mlpipeline-minio-artifact",
        "namespace": NAMESPACE,
    },
}
ENV_BASE = {"KFP_VERSION": KFP_VERSION, "CONTROLLER_PORT": "0"}
ENV_IMAGES = {
    **ENV_BASE,
    "FRONTEND_IMAGE": "frontend-image",
    "FRONTEND_TAG": "somehash",
    "ARTIFACTS_PROXY_ENABLED": "true",
}


def observed_attachments(proxy_enabled=False, ready=False):
    return {
        "Secret.v1": {
            f"{NAMESPACE}/mlpipeline-minio-artifact": EXISTING_SECRET,
        },
        "ConfigMap.v1": {
            "launcher": {},
            "repositories": {}
        } if ready else {},
        "Deployment.apps/v1": {
            "artifact": {}
        } if ready and proxy_enabled else {},
        "Service.v1": {
            "artifact": {}
        } if ready and proxy_enabled else {},
    }


@pytest.fixture
def start_sync_server():
    servers = []

    def start(**settings):
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


@pytest.fixture
def sync_server(request, start_sync_server):
    with mock.patch.dict(os.environ, request.param, clear=True):
        return start_sync_server(**get_settings_from_env())


def post_sync(server, parent, attachments):
    host, port = server.server_address
    response = requests.post(
        f"http://{host}:{port}",
        json={
            "object": parent,
            "attachments": attachments
        },
        timeout=5,
    )
    response.raise_for_status()
    return response.json()


def assert_artifact_resources(result, expected_image):
    resources = result["attachments"]
    identities = {(item["kind"], item["metadata"]["name"]) for item in resources
                 }
    expected = {
        ("Secret", "mlpipeline-minio-artifact"),
        ("ConfigMap", "kfp-launcher"),
        ("ConfigMap", "artifact-repositories"),
    }
    if expected_image:
        expected.update({
            ("Deployment", "ml-pipeline-ui-artifact"),
            ("Service", "ml-pipeline-ui-artifact"),
        })
        deployment = next(
            item for item in resources if item["kind"] == "Deployment")
        assert deployment["spec"]["template"]["spec"]["containers"][0][
            "image"] == expected_image
    # No Python visualization Deployment, Service, or Istio resources are desired.
    assert identities == expected
    assert len(resources) == len(expected)


@pytest.mark.parametrize(
    "sync_server, expected_image",
    [
        (ENV_BASE, None),
        ({
            **ENV_BASE, "ARTIFACTS_PROXY_ENABLED": "true"
        }, f"ghcr.io/kubeflow/kfp-frontend:{KFP_VERSION}"),
        ({
            **ENV_BASE, "ARTIFACTS_PROXY_ENABLED": "true",
            "FRONTEND_IMAGE": "custom"
        }, f"custom:{KFP_VERSION}"),
        (ENV_IMAGES, "frontend-image:somehash"),
        ({
            **ENV_IMAGES, "DISABLE_ISTIO_SIDECAR": "false"
        }, "frontend-image:somehash"),
    ],
    indirect=["sync_server"],
)
@pytest.mark.parametrize("ready", [False, True])
def test_sync_desires_only_artifact_resources(sync_server, expected_image,
                                              ready):
    result = post_sync(sync_server, PARENT,
                       observed_attachments(bool(expected_image), ready))
    assert result["status"] == {"kubeflow-pipelines-ready": str(ready)}
    assert_artifact_resources(result, expected_image)


def test_sync_server_with_direct_settings(start_sync_server):
    server = start_sync_server(
        frontend_image="direct-image",
        frontend_tag="direct-tag",
        disable_istio_sidecar=False,
        artifacts_proxy_enabled="true",
        artifact_retention_days=-1,
        controller_port=0,
    )
    result = post_sync(server, PARENT, observed_attachments(True, True))
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    assert_artifact_resources(result, "direct-image:direct-tag")


@pytest.mark.parametrize("sync_server", [ENV_BASE], indirect=True)
def test_sync_server_without_pipeline_enabled(sync_server):
    assert post_sync(sync_server, {}, {}) == {"status": {}, "attachments": []}


def test_allowed_gcs_universe_domains_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env(
        )["allowed_gcs_universe_domains"] == "googleapis.com"
    with mock.patch.dict(
            os.environ, {
                **ENV_BASE,
                "ALLOWED_GCS_UNIVERSE_DOMAINS":
                    "googleapis.com,gdc.example",
            },
            clear=True):
        assert get_settings_from_env(
        )["allowed_gcs_universe_domains"] == "googleapis.com,gdc.example"


@pytest.mark.parametrize(
    "sync_server", [{
        **ENV_IMAGES,
        "ALLOWED_ARTIFACT_ENDPOINTS":
            "https://objects.example.com:9443",
        "ALLOWED_GCS_UNIVERSE_DOMAINS":
            "googleapis.com,gdc.example",
    }],
    indirect=True)
def test_artifact_proxy_receives_allowed_endpoints(sync_server):
    result = post_sync(sync_server, PARENT, observed_attachments())
    deployment = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    environment = deployment["spec"]["template"]["spec"]["containers"][0]["env"]
    assert {
        "name": "ALLOWED_ARTIFACT_ENDPOINTS",
        "value": "https://objects.example.com:9443",
    } in environment
    assert {
        "name": "ALLOWED_GCS_UNIVERSE_DOMAINS",
        "value": "googleapis.com,gdc.example",
    } in environment
    assert not any("METADATA" in variable["name"] for variable in environment)
    repositories = next(item for item in result["attachments"]
                        if item["metadata"]["name"] == "artifact-repositories")
    archive = json.loads(repositories["data"]["default-namespaced"])
    assert archive["archiveLogs"] is True
    assert archive["s3"]["keyFormat"].startswith(
        f"private-artifacts/{NAMESPACE}/")


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

import os
from unittest import mock
from unittest.mock import MagicMock
import threading
import sync
from sync import (
    get_settings_from_env,
    server_factory,
    fill_json_template,
    load_desired_resources,
    compute_desired_status,
)
import json

import pytest
import requests

NAMESPACE = "myName"

# Existing secret to be reused by the controller when already present
EXISTING_SECRET = {
    "apiVersion": "v1",
    "kind": "Secret",
    "metadata": {
        "name": "mlpipeline-minio-artifact",
        "namespace": NAMESPACE,
    },
    "data": {
        "accesskey": "dGVzdA==",
        "secretkey": "dGVzdA==",
    },
}

# Attachments with correct counts (secret exists, no proxy)
DATA_CORRECT_ATTACHMENTS = {
    "object": {
        "metadata": {
            "labels": {"pipelines.kubeflow.org/enabled": "true"},
            "name": NAMESPACE,
        }
    },
    "attachments": {
        "Secret.v1": {
            f"{NAMESPACE}/mlpipeline-minio-artifact": EXISTING_SECRET,
        },
        "ConfigMap.v1": {
            f"{NAMESPACE}/kfp-launcher": {},
            f"{NAMESPACE}/metadata-grpc-configmap": {},
            f"{NAMESPACE}/artifact-repositories": {},
        },
        "Deployment.apps/v1": {},
        "Service.v1": {},
    },
}

# Attachments with incorrect counts (nothing exists yet)
DATA_INCORRECT_ATTACHMENTS = {
    "object": {
        "metadata": {
            "labels": {"pipelines.kubeflow.org/enabled": "true"},
            "name": NAMESPACE,
        }
    },
    "attachments": {
        "Secret.v1": {},
        "ConfigMap.v1": {},
        "Deployment.apps/v1": {},
        "Service.v1": {},
    },
}

DATA_MISSING_PIPELINE_ENABLED = {"object": {}, "attachments": {}}

# Default values when environments are not explicit
DEFAULT_FRONTEND_IMAGE = "ghcr.io/kubeflow/kfp-frontend"

# Variables used for environment variable sets
FRONTEND_IMAGE = "frontend-image"
FRONTEND_TAG = "somehash"

KFP_VERSION = "x.y.z"

# "Environments" used in tests
ENV_VARIABLES_BASE = {
    "CONTROLLER_PORT": "0",  # HTTPServer randomly assigns the port to a free port
}

ENV_KFP_VERSION_ONLY = dict(
    ENV_VARIABLES_BASE,
    **{
        "KFP_VERSION": KFP_VERSION,
    },
)

ENV_IMAGES_WITH_TAGS = dict(
    ENV_VARIABLES_BASE,
    **{
        "FRONTEND_IMAGE": FRONTEND_IMAGE,
        "FRONTEND_TAG": FRONTEND_TAG,
    },
)

ENV_IMAGES_WITH_TAGS_AND_ISTIO = dict(
    ENV_IMAGES_WITH_TAGS,
    **{
        "DISABLE_ISTIO_SIDECAR": "false",
        "ARTIFACTS_PROXY_ENABLED": "false",
        "ARTIFACT_RETENTION_DAYS": "-1",
    },
)

ENV_WITH_PROXY_ENABLED = dict(
    ENV_IMAGES_WITH_TAGS,
    **{
        "ARTIFACTS_PROXY_ENABLED": "true",
    },
)


def generate_image_name(imagename, tag):
    return f"{str(imagename)}:{str(tag)}"


@pytest.fixture(autouse=True)
def mock_iam_and_s3():
    """Mock IAM and S3 clients to prevent real AWS calls during tests."""
    mock_iam = MagicMock()
    mock_iam.create_access_key.return_value = {
        "AccessKey": {
            "AccessKeyId": "test-access-key",
            "SecretAccessKey": "test-secret-key",
        }
    }
    mock_s3 = MagicMock()

    with mock.patch.object(
        sync, "default_client_factory", return_value=(mock_s3, mock_iam)
    ):
        yield mock_iam, mock_s3


@pytest.fixture(
    scope="function",
)
def sync_server(request):
    """
    Starts the sync HTTP server for a given set of environment variables on a separate thread

    Yields:
    * the server (useful to interrogate for the server address)
    * environment variables (useful to interrogate for correct responses)
    """
    environ = request.param
    with mock.patch.dict(os.environ, environ):
        settings = get_settings_from_env()
        server = server_factory(**settings)
        server_thread = threading.Thread(target=server.serve_forever)
        server_thread.daemon = True
        server_thread.start()
        yield server, environ


@pytest.fixture(
    scope="function",
)
def sync_server_from_arguments(request):
    """
    Starts the sync HTTP server for a given set of parameters passed as
    arguments, with server on a separate thread.

    Yields:
    * the server (useful to interrogate for the server address)
    * settings dict (useful to interrogate for correct responses)
    """
    settings = {k.lower(): v for k, v in request.param.items()}
    server = server_factory(**settings)
    server_thread = threading.Thread(target=server.serve_forever)
    server_thread.daemon = True
    server_thread.start()
    yield server, settings


# ---------------------------------------------------------------------------
# Unit tests for helper functions
# ---------------------------------------------------------------------------


class TestFillJsonTemplate:
    """Tests for the fill_json_template function."""

    def test_fills_placeholders(self, tmp_path):
        """Template placeholders are replaced with provided values."""
        template_file = tmp_path / "template.json"
        template_file.write_text('[{"name": "${name}", "namespace": "${ns}"}]')
        result = fill_json_template(
            str(template_file), name="my-resource", ns="default"
        )
        assert result == [{"name": "my-resource", "namespace": "default"}]

    def test_empty_file_returns_empty_list(self, tmp_path):
        """An empty template file returns an empty list."""
        template_file = tmp_path / "empty.json"
        template_file.write_text("")
        result = fill_json_template(str(template_file))
        assert result == []


class TestComputeDesiredStatus:
    """Tests for the compute_desired_status function."""

    def test_all_counts_match(self):
        """Returns ready=True when all observed counts match expected."""
        attachments = {
            "ConfigMap.v1": {"ns/a": {}, "ns/b": {}},
            "Secret.v1": {"ns/s": {}},
        }
        desired = [
            {"apiVersion": "v1", "kind": "ConfigMap"},
            {"apiVersion": "v1", "kind": "ConfigMap"},
            {"apiVersion": "v1", "kind": "Secret"},
        ]
        status = compute_desired_status(attachments, desired)
        assert status == {"kubeflow-pipelines-ready": True}

    def test_counts_mismatch(self):
        """Returns ready=False when observed counts differ."""
        attachments = {
            "ConfigMap.v1": {},
            "Secret.v1": {},
        }
        desired = [
            {"apiVersion": "v1", "kind": "ConfigMap"},
            {"apiVersion": "v1", "kind": "Secret"},
        ]
        status = compute_desired_status(attachments, desired)
        assert status == {"kubeflow-pipelines-ready": False}


class TestLoadDesiredResources:
    """Tests for the load_desired_resources function."""

    def test_loads_base_resources(self):
        """Base templates produce kfp-launcher and metadata-grpc-configmap."""
        resources = load_desired_resources(
            namespace="test-ns",
            cluster_domain=".svc.cluster.local",
        )
        names = [r["metadata"]["name"] for r in resources]
        assert "kfp-launcher" in names
        assert "metadata-grpc-configmap" in names
        assert "artifact-repositories" in names

    def test_no_proxy_by_default(self):
        """Artifact proxy resources are not included when disabled."""
        resources = load_desired_resources(
            namespace="test-ns",
            cluster_domain=".svc.cluster.local",
            artifacts_proxy_enabled="false",
        )
        kinds = [r["kind"] for r in resources]
        assert "Deployment" not in kinds
        assert "Service" not in kinds

    def test_proxy_resources_when_enabled(self):
        """Artifact proxy resources are included when enabled."""
        resources = load_desired_resources(
            namespace="test-ns",
            cluster_domain=".svc.cluster.local",
            artifacts_proxy_enabled="true",
            disable_istio_sidecar=False,
            frontend_image="frontend",
            frontend_tag="v1",
        )
        kinds = [r["kind"] for r in resources]
        assert "Deployment" in kinds
        assert "Service" in kinds

    def test_artifact_repositories_configmap_content(self):
        """artifact-repositories ConfigMap includes correct S3 config."""
        resources = load_desired_resources(
            namespace="test-ns",
            cluster_domain=".svc.cluster.local",
            object_store_host="seaweedfs",
        )
        ar = next(
            r
            for r in resources
            if r["metadata"]["name"] == "artifact-repositories"
        )
        data = json.loads(ar["data"]["default-namespaced"])
        assert data["s3"]["bucket"] == "mlpipeline"
        assert "test-ns" in data["s3"]["keyFormat"]
        assert (
            data["s3"]["endpoint"]
            == "seaweedfs.kubeflow.svc.cluster.local:9000"
        )


# ---------------------------------------------------------------------------
# Integration tests: full HTTP server round-trip
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "sync_server, data, expected_ready",
    [
        (
            ENV_KFP_VERSION_ONLY,
            DATA_INCORRECT_ATTACHMENTS,
            False,
        ),
        (
            ENV_IMAGES_WITH_TAGS,
            DATA_CORRECT_ATTACHMENTS,
            True,
        ),
    ],
    indirect=["sync_server"],
)
def test_sync_server_with_pipeline_enabled(sync_server, data, expected_ready):
    """
    Nearly end-to-end test of how Controller serves .sync as a POST.

    Tests that the server produces the correct status based on the
    attachments and returns resource attachments loaded from templates.
    """
    server, environ = sync_server

    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    assert results["status"]["kubeflow-pipelines-ready"] is expected_ready

    # Verify ConfigMap resources are present in attachments
    resource_names = [
        r["metadata"]["name"]
        for r in results["attachments"]
        if r.get("kind") == "ConfigMap"
    ]
    assert "kfp-launcher" in resource_names
    assert "metadata-grpc-configmap" in resource_names
    assert "artifact-repositories" in resource_names


@pytest.mark.parametrize(
    "sync_server_from_arguments, data, expected_ready",
    [
        (
            ENV_IMAGES_WITH_TAGS_AND_ISTIO,
            DATA_CORRECT_ATTACHMENTS,
            True,
        ),
    ],
    indirect=["sync_server_from_arguments"],
)
def test_sync_server_with_direct_passing_of_settings(
    sync_server_from_arguments, data, expected_ready
):
    """
    Nearly end-to-end test of how Controller serves .sync as a POST,
    taking variables as arguments.
    """
    server, settings = sync_server_from_arguments

    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    assert results["status"]["kubeflow-pipelines-ready"] is expected_ready


@pytest.mark.parametrize(
    "sync_server, data, expected_status, expected_attachments",
    [
        (ENV_IMAGES_WITH_TAGS, DATA_MISSING_PIPELINE_ENABLED, {}, []),
    ],
    indirect=["sync_server"],
)
def test_sync_server_without_pipeline_enabled(
    sync_server, data, expected_status, expected_attachments
):
    """
    Tests case where metadata.labels.pipelines.kubeflow.org/enabled does not
    exist and thus server returns an empty reply.
    """
    server, environ = sync_server

    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    assert results["status"] == expected_status
    assert results["attachments"] == expected_attachments


@pytest.mark.parametrize(
    "sync_server, data",
    [
        (ENV_WITH_PROXY_ENABLED, DATA_CORRECT_ATTACHMENTS),
    ],
    indirect=["sync_server"],
)
def test_sync_server_with_proxy_enabled(sync_server, data):
    """
    Tests that artifact proxy Deployment and Service are returned when
    ARTIFACTS_PROXY_ENABLED is set to true.
    """
    server, environ = sync_server

    url = f"http://{server.server_address[0]}:{str(server.server_address[1])}"
    x = requests.post(url, data=json.dumps(data))
    results = json.loads(x.text)

    kinds = [r["kind"] for r in results["attachments"]]
    assert "Deployment" in kinds
    assert "Service" in kinds

    # Verify the frontend image is used in the proxy deployment
    deployment = next(
        r for r in results["attachments"] if r["kind"] == "Deployment"
    )
    container_image = deployment["spec"]["template"]["spec"]["containers"][0][
        "image"
    ]
    assert container_image == generate_image_name(FRONTEND_IMAGE, FRONTEND_TAG)


def test_create_iam_client_uses_endpoint(monkeypatch):
    called = {}

    class DummySession:
        def create_client(
            self, service_name, region_name=None, endpoint_url=None
        ):
            called["service_name"] = service_name
            called["endpoint_url"] = endpoint_url
            return object()

    monkeypatch.setenv("AWS_ENDPOINT_URL", "http://seaweedfs.kubeflow:8111")

    sync.create_iam_client(DummySession())

    assert called["service_name"] == "iam"
    assert called["endpoint_url"] == "http://seaweedfs.kubeflow:8111"

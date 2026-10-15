import base64
import copy
import itertools
import json
import os
from pathlib import Path
import runpy
import threading
from unittest import mock

import pytest
import requests
import sync
from sync import get_settings_from_env
from sync import server_factory
import yaml

KFP_VERSION = "x.y.z"
NAMESPACE = "myName"
PARENT = {
    "metadata": {
        "labels": {
            "pipelines.kubeflow.org/enabled": "true"
        },
        "name": NAMESPACE,
        "uid": "namespace-uid",
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


@pytest.mark.parametrize("bundled", [False, True])
def test_botocore_import_initializes_clients(bundled):
    original_import = __import__
    awscli_imported = False
    botocore = mock.Mock()
    session = botocore.session.get_session.return_value
    session.create_client.side_effect = [mock.sentinel.s3, mock.sentinel.iam]

    def import_package(name, *args, **kwargs):
        nonlocal awscli_imported
        if name == "awscli":
            assert bundled, "Standalone botocore must not require awscli"
            awscli_imported = True
            return mock.sentinel.awscli
        if name == "botocore.session":
            if bundled and not awscli_imported:
                raise ModuleNotFoundError(
                    "No module named 'botocore'", name="botocore")
            return botocore
        return original_import(name, *args, **kwargs)

    with mock.patch("builtins.__import__", side_effect=import_package):
        loaded = runpy.run_path(sync.__file__, run_name="profile_import_test")

    assert awscli_imported is bundled
    assert loaded["s3"] is mock.sentinel.s3
    assert loaded["iam"] is mock.sentinel.iam
    assert [call.args[0] for call in session.create_client.call_args_list
           ] == ["s3", "iam"]


@pytest.mark.parametrize("missing_module", ["urllib3", "botocore.session"])
def test_botocore_import_preserves_other_missing_modules(missing_module):
    original_import = __import__
    error = ModuleNotFoundError(
        f"No module named '{missing_module}'", name=missing_module)

    def import_package(name, *args, **kwargs):
        if name == "botocore.session":
            raise error
        assert name != "awscli", "An unrelated import error must not fall back"
        return original_import(name, *args, **kwargs)

    with mock.patch("builtins.__import__", side_effect=import_package):
        with pytest.raises(ModuleNotFoundError) as raised:
            runpy.run_path(sync.__file__, run_name="profile_import_test")
    assert raised.value is error


def observed_attachments(proxy_enabled=False, ready=False, ca=False):
    secrets = {
        f"{NAMESPACE}/mlpipeline-minio-artifact": EXISTING_SECRET,
    }
    if ready:
        secrets.update({
            f"{NAMESPACE}/ml-pipeline-driver-agent-executor-plugin.service-account-token":
                {},
            f"{NAMESPACE}/default-editor.service-account-token": {},
        })
    if ready and ca:
        secrets[f"{NAMESPACE}/{sync.DRIVER_AGENT_CA_SECRET_NAME}"] = {}
    return {
        "Secret.v1": secrets,
        "ConfigMap.v1": {
            "launcher": {},
            "repositories": {}
        } if ready else {},
        "Deployment.apps/v1": {
            f"{NAMESPACE}/ml-pipeline-ui-artifact": {}
        } if ready and proxy_enabled else {},
        "Service.v1": {
            f"{NAMESPACE}/ml-pipeline-ui-artifact": {}
        } if ready and proxy_enabled else {},
        "ServiceAccount.v1": {
            "driver-plugin": {}
        } if ready else {},
        "Role.rbac.authorization.k8s.io/v1": {},
        "RoleBinding.rbac.authorization.k8s.io/v1": {
            "driver-plugin": {}
        } if ready else {},
        "NetworkPolicy.networking.k8s.io/v1": {
            f"{NAMESPACE}/ml-pipeline-ui-artifact": {}
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


def post_sync(server, parent, attachments, converge_policy=True):
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
    result = response.json()
    # Most tests inspect the generated workload after the policy-first phase.
    if converge_policy and not any(item["kind"] == "Deployment"
                                   for item in result["attachments"]):
        policy = next((item for item in result["attachments"]
                       if item["kind"] == "NetworkPolicy"), None)
        if policy:
            observed = copy.deepcopy(attachments)
            observed["NetworkPolicy.networking.k8s.io/v1"] = {
                f"{parent['metadata']['name']}/ml-pipeline-ui-artifact": policy
            }
            return post_sync(server, parent, observed, converge_policy=False)
    return result


def assert_profile_resources(result, expected_image, ca_cert=None):
    resources = result["attachments"]
    identities = {(item["kind"], item["metadata"]["name"]) for item in resources
                 }
    expected = {
        ("Secret", "mlpipeline-minio-artifact"),
        ("Secret",
         "ml-pipeline-driver-agent-executor-plugin.service-account-token"),
        ("Secret", "default-editor.service-account-token"),
        ("ConfigMap", "kfp-launcher"),
        ("ConfigMap", "artifact-repositories"),
        ("ServiceAccount", "ml-pipeline-driver-agent-executor-plugin"),
        ("RoleBinding", "ml-pipeline-driver-agent-executor-plugin-binding"),
    }
    if expected_image:
        expected.update({
            ("Deployment", "ml-pipeline-ui-artifact"),
            ("Service", "ml-pipeline-ui-artifact"),
            ("NetworkPolicy", "ml-pipeline-ui-artifact"),
            ("Secret", sync.artifact_storage_record_name(NAMESPACE)),
        })
        deployment = next(
            item for item in resources if item["kind"] == "Deployment")
        assert deployment["spec"]["template"]["spec"]["containers"][0][
            "image"] == expected_image
    if ca_cert is not None:
        expected.add(("Secret", sync.DRIVER_AGENT_CA_SECRET_NAME))
        secret = next(
            item for item in resources
            if item["metadata"]["name"] == sync.DRIVER_AGENT_CA_SECRET_NAME)
        assert secret["metadata"]["namespace"] == NAMESPACE
        assert secret["data"] == {
            "ca.crt": base64.b64encode(ca_cert).decode("utf-8")
        }
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
def test_sync_desires_profile_resources(sync_server, expected_image, ready):
    result = post_sync(sync_server, PARENT,
                       observed_attachments(bool(expected_image), ready))
    assert result["status"] == {"kubeflow-pipelines-ready": str(ready)}
    assert_profile_resources(result, expected_image)


@pytest.mark.parametrize(
    "sync_server, expected_namespace", [
        (ENV_IMAGES, "kubeflow"),
        ({
            **ENV_IMAGES, "KFP_NAMESPACE": "custom-install"
        }, "custom-install"),
        ({
            **ENV_IMAGES, "DISABLE_ISTIO_SIDECAR": "true"
        }, "kubeflow"),
    ],
    indirect=["sync_server"])
def test_artifact_network_policy_binds_shared_ui_identity(
        sync_server, expected_namespace):
    result = post_sync(sync_server, PARENT, observed_attachments(True))
    policy = next(item for item in result["attachments"]
                  if item["kind"] == "NetworkPolicy")
    assert policy["metadata"] == {
        "name": "ml-pipeline-ui-artifact",
        "namespace": NAMESPACE,
    }
    assert policy["spec"] == {
        "podSelector": {
            "matchLabels": {
                "app": "ml-pipeline-ui-artifact"
            }
        },
        "policyTypes": ["Ingress"],
        "ingress": [{
            "from": [{
                "namespaceSelector": {
                    "matchLabels": {
                        "kubernetes.io/metadata.name": expected_namespace
                    }
                },
                "podSelector": {
                    "matchLabels": {
                        "app": "ml-pipeline-ui"
                    }
                },
            }],
            "ports": [{
                "protocol": "TCP",
                "port": 3000
            }],
        }],
    }
    deployment = next(item for item in result["attachments"]
                      if item["kind"] == "Deployment" and
                      item["metadata"]["name"] == "ml-pipeline-ui-artifact")
    environment = deployment["spec"]["template"]["spec"]["containers"][0]["env"]
    assert {
        "name": "ARTIFACTS_REQUIRE_PROXY_NAMESPACE",
        "value": "false"
    } in environment
    assert {
        "name": "FRONTEND_SERVER_NAMESPACE",
        "value": NAMESPACE
    } in environment
    # A single peer requires BOTH the install namespace and UI pod labels. Splitting
    # these into two peers would allow a tenant to impersonate the UI using a label.
    assert len(policy["spec"]["ingress"][0]["from"]) == 1


@pytest.mark.parametrize("sync_server", [ENV_IMAGES], indirect=True)
def test_artifact_proxy_waits_for_network_policy(sync_server):
    attachments = observed_attachments(True, True)
    attachments.pop("NetworkPolicy.networking.k8s.io/v1")
    result = post_sync(sync_server, PARENT, attachments)
    assert result["status"] == {"kubeflow-pipelines-ready": "False"}


def test_kfp_namespace_setting_override():
    with mock.patch.dict(
            os.environ, {
                **ENV_BASE, "KFP_NAMESPACE": "custom"
            }, clear=True):
        assert get_settings_from_env()["kfp_namespace"] == "custom"
        assert get_settings_from_env(
            kfp_namespace="other")["kfp_namespace"] == "other"


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
    assert_profile_resources(result, "direct-image:direct-tag")


def start_ca_server(start_sync_server, ca_cert_path):
    return start_sync_server(
        frontend_image="image",
        frontend_tag="tag",
        disable_istio_sidecar=False,
        artifacts_proxy_enabled="false",
        artifact_retention_days=-1,
        controller_port=0,
        driver_agent_ca_cert_path=str(ca_cert_path),
    )


def test_sync_copies_driver_agent_ca_into_profile(start_sync_server, tmp_path):
    ca_cert_path = tmp_path / "ca.crt"
    ca_cert_path.write_bytes(b"-----BEGIN CERTIFICATE-----\nfirst\n")
    server = start_ca_server(start_sync_server, ca_cert_path)

    result = post_sync(server, PARENT, observed_attachments(ready=True))
    # The copy is not observed yet, so the profile is not ready.
    assert result["status"] == {"kubeflow-pipelines-ready": "False"}
    assert_profile_resources(result, None, ca_cert_path.read_bytes())

    result = post_sync(server, PARENT,
                       observed_attachments(ready=True, ca=True))
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    assert_profile_resources(result, None, ca_cert_path.read_bytes())

    # Rotation: the next sync carries the current CA without a restart.
    ca_cert_path.write_bytes(b"-----BEGIN CERTIFICATE-----\nrotated\n")
    result = post_sync(server, PARENT,
                       observed_attachments(ready=True, ca=True))
    assert_profile_resources(result, None, ca_cert_path.read_bytes())


def test_ca_rotation_reconciles_without_changing_other_secrets(
        start_sync_server, tmp_path):
    controller = yaml.safe_load(
        Path(sync.__file__).with_name("decorator-controller.yaml").read_text())
    secret_rules = [
        rule for rule in controller["spec"]["attachments"]
        if rule["resource"] == "secrets"
    ]
    assert len(secret_rules) == 1
    assert secret_rules[0]["updateStrategy"]["method"] == "InPlace"

    ca_cert_path = tmp_path / "ca.crt"
    ca_cert_path.write_bytes(b"original CA")
    server = start_ca_server(start_sync_server, ca_cert_path)
    result = post_sync(server, PARENT, observed_attachments(ready=True))
    observed = observed_attachments(ready=True)
    for resource in result["attachments"]:
        if resource["kind"] == "Secret":
            name = resource["metadata"]["name"]
            observed["Secret.v1"][f"{NAMESPACE}/{name}"] = resource
    # Token controllers and operators may populate or customize these Secrets.
    for secret in observed["Secret.v1"].values():
        if secret["metadata"]["name"] != sync.DRIVER_AGENT_CA_SECRET_NAME:
            secret["data"] = {
                "token": "existing-token",
                "ca.crt": "existing-ca"
            }
            secret["metadata"]["annotations"] = {"custom": "preserve-me"}
    before = copy.deepcopy(observed)

    ca_cert_path.write_bytes(b"rotated CA")
    result = post_sync(server, PARENT, observed)
    assert_profile_resources(result, None, b"rotated CA")
    for secret in result["attachments"]:
        if secret["kind"] != "Secret":
            continue
        previous = before["Secret.v1"][
            f"{NAMESPACE}/{secret['metadata']['name']}"]
        if secret["metadata"]["name"] == sync.DRIVER_AGENT_CA_SECRET_NAME:
            assert secret["data"] != previous["data"]
        else:
            assert secret == previous
    assert observed == before


# Read-only metadata that Metacontroller v4.16.2 reverts after merging
# (objectMetaSystemFields in pkg/controller/common/manage_children.go).
METACONTROLLER_SYSTEM_FIELDS = (
    "selfLink",
    "uid",
    "resourceVersion",
    "generation",
    "creationTimestamp",
    "deletionTimestamp",
    "deletionGracePeriodSeconds",
    "managedFields",
)


def metacontroller_merge(destination, last_applied, desired):
    """Port of dynamicapply.merge from Metacontroller v4.16.2: apply the diff
    from last_applied to desired onto destination.

    Secrets hold no lists, so lists take the desired value.
    """
    if isinstance(destination, dict):
        last_applied = last_applied or {}
        desired = desired or {}
        for key in last_applied:
            if key not in desired:
                destination.pop(key, None)
        for key, value in desired.items():
            destination[key] = metacontroller_merge(
                destination.get(key), last_applied.get(key), value)
        return destination
    return desired


def metacontroller_apply_update(observed, desired):
    """Port of common.ApplyUpdate from Metacontroller v4.16.2, which decides
    whether an InPlace attachment differs from the observed object."""
    desired = copy.deepcopy(desired)
    desired.get("metadata", {}).get("annotations",
                                    {}).pop(sync.LAST_APPLIED_ANNOTATION, None)
    last_applied = json.loads(observed["metadata"].get("annotations", {}).get(
        sync.LAST_APPLIED_ANNOTATION) or "null")
    merged = metacontroller_merge(
        copy.deepcopy(observed), last_applied, desired)
    for field in METACONTROLLER_SYSTEM_FIELDS:
        if field in observed["metadata"]:
            merged["metadata"][field] = observed["metadata"][field]
        else:
            merged["metadata"].pop(field, None)
    merged["metadata"].setdefault(
        "annotations", {})[sync.LAST_APPLIED_ANNOTATION] = json.dumps(
            desired, sort_keys=True)
    return merged


def metacontroller_create(desired, resource_versions):
    """A Secret as the API server stores it after Metacontroller creates it:
    the last-applied annotation, the controller ownerReference, and the
    system metadata that observed attachments carry."""
    stored = copy.deepcopy(desired)
    stored["metadata"].setdefault(
        "annotations", {})[sync.LAST_APPLIED_ANNOTATION] = json.dumps(
            desired, sort_keys=True)
    stored["metadata"]["ownerReferences"] = [{
        "apiVersion": "v1",
        "kind": "Namespace",
        "name": NAMESPACE,
        "uid": "parent-uid",
        "controller": True,
        "blockOwnerDeletion": True,
    }]
    stored["metadata"].update({
        "uid": f"uid-{desired['metadata']['name']}",
        "creationTimestamp": "2026-10-14T00:00:00Z",
        "managedFields": [{
            "manager": "metacontroller"
        }],
        "resourceVersion": str(next(resource_versions)),
    })
    stored.setdefault("type", "Opaque")
    return stored


def test_preserved_secrets_converge_under_in_place_updates(
        start_sync_server, tmp_path):
    ca_cert_path = tmp_path / "ca.crt"
    ca_cert_path.write_bytes(b"original CA")
    server = start_ca_server(start_sync_server, ca_cert_path)
    resource_versions = itertools.count(1)
    observed = observed_attachments(ready=True)
    observed["Secret.v1"] = {
        f"{NAMESPACE}/mlpipeline-minio-artifact":
            metacontroller_create(
                {
                    **copy.deepcopy(EXISTING_SECRET), "data": {
                        "accesskey": "YWNjZXNz",
                        "secretkey": "c2VjcmV0"
                    }
                }, resource_versions),
    }

    def reconcile():
        """One Metacontroller sync: create missing Secrets, apply the others
        in place, and count how many the API server had to write."""
        result = post_sync(server, PARENT, observed)
        writes = 0
        for desired in result["attachments"]:
            if desired["kind"] != "Secret":
                continue
            key = f"{NAMESPACE}/{desired['metadata']['name']}"
            assert not set(desired["metadata"]) & (
                set(METACONTROLLER_SYSTEM_FIELDS) | {"ownerReferences"}), key
            assert sync.LAST_APPLIED_ANNOTATION not in desired["metadata"].get(
                "annotations", {}), key
            current = observed["Secret.v1"].get(key)
            if current is None:
                observed["Secret.v1"][key] = metacontroller_create(
                    desired, resource_versions)
                writes += 1
                continue
            updated = metacontroller_apply_update(current, desired)
            if updated != current:
                updated["metadata"]["resourceVersion"] = str(
                    next(resource_versions))
                observed["Secret.v1"][key] = updated
                writes += 1
        return writes

    # The first sync creates the CA and token Secrets and may adopt
    # server-defaulted fields of the pre-existing credentials once.
    assert reconcile() >= 3
    # The token controller then populates the service-account token Secrets.
    token_data = {
        "token": "ZXhpc3Rpbmctand0",
        "ca.crt": "Y2x1c3Rlci1jYQ==",
        "namespace": "bXlOYW1l",
    }
    for secret in observed["Secret.v1"].values():
        if secret.get("type") == "kubernetes.io/service-account-token":
            secret["data"] = dict(token_data)
            secret["metadata"]["annotations"][
                "kubernetes.io/service-account.uid"] = "sa-uid"
            secret["metadata"]["resourceVersion"] = str(next(resource_versions))
    assert reconcile() == 2
    for _ in range(8):
        assert reconcile() == 0, observed["Secret.v1"]

    # Rotation rewrites only the CA copy, and the loop settles again.
    ca_cert_path.write_bytes(b"rotated CA")
    assert reconcile() == 1
    for _ in range(8):
        assert reconcile() == 0, observed["Secret.v1"]

    secrets = observed["Secret.v1"]
    assert secrets[f"{NAMESPACE}/{sync.DRIVER_AGENT_CA_SECRET_NAME}"][
        "data"] == {
            "ca.crt": base64.b64encode(b"rotated CA").decode("utf-8")
        }
    assert secrets[f"{NAMESPACE}/mlpipeline-minio-artifact"]["data"] == {
        "accesskey": "YWNjZXNz",
        "secretkey": "c2VjcmV0"
    }
    for name in (
            "default-editor.service-account-token",
            "ml-pipeline-driver-agent-executor-plugin.service-account-token"):
        secret = secrets[f"{NAMESPACE}/{name}"]
        assert secret["data"] == token_data
        assert secret["metadata"]["annotations"][
            "kubernetes.io/service-account.uid"] == "sa-uid"
        assert secret["metadata"]["ownerReferences"][0]["uid"] == "parent-uid"


@pytest.mark.parametrize("state", ["missing", "empty"])
def test_sync_without_driver_agent_ca_is_unchanged(start_sync_server, tmp_path,
                                                   state):
    ca_cert_path = tmp_path / "ca.crt"
    if state == "empty":
        ca_cert_path.write_bytes(b"")
    server = start_ca_server(start_sync_server, ca_cert_path)

    result = post_sync(server, PARENT, observed_attachments(ready=True))
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    assert_profile_resources(result, None)


def test_driver_agent_ca_cert_path_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env()["driver_agent_ca_cert_path"] == ""
    with mock.patch.dict(
            os.environ, {
                **ENV_BASE, "DRIVER_AGENT_CA_CERT_PATH":
                    "/etc/kfp/driver-agent-ca/ca.crt"
            },
            clear=True):
        assert get_settings_from_env()["driver_agent_ca_cert_path"] == \
            "/etc/kfp/driver-agent-ca/ca.crt"
        assert get_settings_from_env(
            driver_agent_ca_cert_path="")["driver_agent_ca_cert_path"] == ""


@pytest.mark.parametrize("sync_server", [ENV_BASE], indirect=True)
def test_sync_server_without_pipeline_enabled(sync_server):
    assert post_sync(sync_server, {}, {}) == {"status": {}, "attachments": []}


def test_http_base_url_default_and_override():
    with mock.patch.dict(os.environ, ENV_BASE, clear=True):
        assert get_settings_from_env()["http_base_url"] == ""

    with mock.patch.dict(
            os.environ, {
                "KFP_VERSION": KFP_VERSION,
                "HTTP_BASE_URL": "https://artifacts.example.com/pipelines/",
            },
            clear=True):
        assert get_settings_from_env()["http_base_url"] == \
            "https://artifacts.example.com/pipelines/"
        assert get_settings_from_env(http_base_url="")["http_base_url"] == ""
        assert get_settings_from_env(
            http_base_url="https://other.example.com/prefix")["http_base_url"] == \
            "https://other.example.com/prefix"


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
        "HTTP_BASE_URL":
            "https://artifacts.example.com:9443/pipelines/",
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
        "name": "HTTP_BASE_URL",
        "value": "https://artifacts.example.com:9443/pipelines/",
    } in environment
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
        requests.post(
            f"http://{host}:{port}/healthz",
            json={
                "object": PARENT,
                "attachments": {}
            },
            timeout=5).raise_for_status()
    assert '"POST /healthz' in capsys.readouterr().err


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


@pytest.fixture
def storage_controller(start_sync_server, monkeypatch):
    with mock.patch.dict(os.environ, ENV_IMAGES, clear=True):
        server = start_sync_server(**get_settings_from_env())
    iam = mock.Mock()
    iam.list_access_keys.return_value = {
        "AccessKeyMetadata": [{
            "AccessKeyId": "tenant-access",
            "Status": "Active"
        }]
    }
    iam.get_user_policy.return_value = {
        "PolicyDocument": sync.managed_object_policy(NAMESPACE)
    }
    s3 = mock.Mock()
    s3.get_bucket_policy.return_value = {
        "Policy":
            json.dumps(sync.managed_bucket_list_policy(sync.S3_BUCKET_NAME))
    }
    monkeypatch.setattr(sync, "iam", iam)
    monkeypatch.setattr(sync, "s3", s3)
    attachments = observed_attachments(True)
    secret = copy.deepcopy(EXISTING_SECRET)
    secret["data"] = {
        "accesskey": base64.b64encode(b"tenant-access").decode(),
        "secretkey": base64.b64encode(b"tenant-secret").decode(),
    }
    attachments["Secret.v1"][f"{NAMESPACE}/mlpipeline-minio-artifact"] = secret
    return server, attachments, iam, s3


def storage_record(result):
    secret = next(item for item in result["attachments"] if item["metadata"]
                  ["name"] == sync.artifact_storage_record_name(NAMESPACE))
    assert secret["metadata"]["namespace"] == "kubeflow"
    return json.loads(base64.b64decode(secret["data"]["authorization.json"]))


def observe_receiver_resources(result, attachments, ready=True):
    observed = copy.deepcopy(attachments)
    for resource in result["attachments"]:
        kind = resource["kind"]
        if kind not in ["Deployment", "Service", "NetworkPolicy"]:
            continue
        resource = copy.deepcopy(resource)
        if kind == "Deployment":
            # The Kubernetes API omits EnvVar.value when it is an empty string.
            for container in resource["spec"]["template"]["spec"]["containers"]:
                for variable in container["env"]:
                    if variable.get("value") == "":
                        del variable["value"]
            resource["metadata"]["generation"] = 2
            if ready:
                resource["status"] = {
                    "observedGeneration": 2,
                    "replicas": 1,
                    "updatedReplicas": 1,
                    "availableReplicas": 1,
                    "readyReplicas": 1,
                }
        observed[f"{kind}.{resource['apiVersion']}"][
            f"{NAMESPACE}/ml-pipeline-ui-artifact"] = resource
    return observed


def ready_storage(storage_controller):
    server, attachments, iam, s3 = storage_controller
    result = post_sync(server, PARENT, attachments)
    observed = observe_receiver_resources(result, attachments)
    return post_sync(server, PARENT, observed), observed


def test_storage_authorization_waits_for_policy_and_current_receiver(
        storage_controller, capsys):
    server, attachments, iam, s3 = storage_controller
    first = post_sync(server, PARENT, attachments, converge_policy=False)
    assert storage_record(first)["state"] == "pending"
    assert first["resyncAfterSeconds"] == 10
    assert not any(
        resource["kind"] == "Deployment" for resource in first["attachments"])
    second = post_sync(
        server,
        PARENT,
        observe_receiver_resources(first, attachments),
        converge_policy=False)
    assert storage_record(second)["state"] == "pending"
    assert any(
        resource["kind"] == "Deployment" for resource in second["attachments"])
    observed = observe_receiver_resources(second, attachments)
    result = post_sync(server, PARENT, observed)
    record = storage_record(result)
    assert "resyncAfterSeconds" not in result
    assert record == {
        "version":
            1,
        "namespace":
            NAMESPACE,
        "namespaceUID":
            "namespace-uid",
        "state":
            "ready",
        "bucket":
            "mlpipeline",
        "credentialBinding":
            sync.artifact_credential_binding(NAMESPACE, "namespace-uid",
                                             "http://seaweedfs.kubeflow:8333",
                                             "tenant-access", "tenant-secret"),
    }
    receiver = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    environment = receiver["spec"]["template"]["spec"]["containers"][0]["env"]
    values = {item["name"]: item.get("value") for item in environment}
    assert values["KFP_ARTIFACT_NAMESPACE_UID"] == "namespace-uid"
    assert values["MINIO_HOST"] == "seaweedfs"
    assert values["MINIO_NAMESPACE"] == "kubeflow"
    assert values["MINIO_PORT"] == "8333"
    assert values["MINIO_SSL"] == "false"
    assert values["ARTIFACTS_REQUIRE_PROXY_NAMESPACE"] == "false"
    output = capsys.readouterr().out
    assert "tenant-secret" not in output
    assert record["credentialBinding"] not in output
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


def test_storage_authorization_accepts_api_omitted_empty_environment_values(
        storage_controller):
    server, attachments, _, _ = storage_controller
    result = post_sync(server, PARENT, attachments)
    original = copy.deepcopy(result)
    observed = observe_receiver_resources(result, attachments)
    assert result == original
    deployment = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]
    environment = deployment["spec"]["template"]["spec"]["containers"][0]["env"]
    for name in ["HTTP_BASE_URL", "ALLOWED_ARTIFACT_ENDPOINTS"]:
        assert next(variable for variable in environment
                    if variable["name"] == name) == {
                        "name": name
                    }
    original_observation = copy.deepcopy(observed)
    assert sync.receiver_rollout_ready(result["attachments"], observed,
                                       NAMESPACE)
    assert observed == original_observation
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "ready"


@pytest.mark.parametrize("mutation", [
    "old-generation", "old-replicas", "unready", "wrong-image", "wrong-policy",
    "wrong-service", "missing-env-value", "wrong-env-value", "empty-env-value",
    "empty-env-value-from"
])
def test_storage_authorization_does_not_trust_incomplete_rollout(
        storage_controller, mutation):
    server, _, _, _ = storage_controller
    result, observed = ready_storage(storage_controller)
    assert storage_record(result)["state"] == "ready"
    deployment = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]
    if mutation == "old-generation":
        deployment["status"]["observedGeneration"] = 1
    elif mutation == "old-replicas":
        deployment["status"]["replicas"] = 2
    elif mutation == "unready":
        deployment["status"]["availableReplicas"] = 0
    elif mutation == "wrong-image":
        deployment["spec"]["template"]["spec"]["containers"][0][
            "image"] = "old-image"
    elif mutation in ["missing-env-value", "wrong-env-value"]:
        environment = deployment["spec"]["template"]["spec"]["containers"][0][
            "env"]
        variable = next(
            item for item in environment if item["name"] == "MINIO_HOST")
        if mutation == "missing-env-value":
            del variable["value"]
        else:
            variable["value"] = "other-storage"
    elif mutation in ["empty-env-value", "empty-env-value-from"]:
        environment = deployment["spec"]["template"]["spec"]["containers"][0][
            "env"]
        variable = next(
            item for item in environment if item["name"] == "HTTP_BASE_URL")
        if mutation == "empty-env-value":
            variable["value"] = "http://untrusted-storage"
        else:
            variable["valueFrom"] = {
                "secretKeyRef": {
                    "name": "untrusted-config",
                    "key": "url"
                }
            }
    elif mutation == "wrong-policy":
        observed["NetworkPolicy.networking.k8s.io/v1"][
            f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["ingress"].append(
                {})
    else:
        observed["Service.v1"][f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"][
            "type"] = "ExternalName"
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "pending"


@pytest.mark.parametrize(
    "cause", ["foreign-key", "inactive-key", "foreign-policy", "broad-policy"])
def test_storage_authorization_leaves_unmanaged_credentials_unsupported(
        storage_controller, cause):
    server, attachments, iam, s3 = storage_controller
    if cause == "foreign-key":
        iam.list_access_keys.return_value["AccessKeyMetadata"][0][
            "AccessKeyId"] = "other-key"
    elif cause == "inactive-key":
        iam.list_access_keys.return_value["AccessKeyMetadata"][0][
            "Status"] = "Inactive"
    elif cause == "foreign-policy":
        iam.get_user_policy.return_value[
            "PolicyDocument"] = sync.managed_object_policy("other-namespace")
    else:
        iam.get_user_policy.return_value["PolicyDocument"]["Statement"][0][
            "Resource"].append("*")
    assert storage_record(post_sync(server, PARENT,
                                    attachments))["state"] == "unsupported"
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


@pytest.mark.parametrize(
    "operation", ["list_access_keys", "get_user_policy", "get_bucket_policy"])
def test_storage_authorization_api_failure_is_pending(storage_controller,
                                                      operation):
    server, attachments, iam, s3 = storage_controller
    client = s3 if operation == "get_bucket_policy" else iam
    getattr(client, operation).side_effect = RuntimeError(
        "do not log credential-bearing details")
    assert storage_record(post_sync(server, PARENT,
                                    attachments))["state"] == "pending"
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


def test_storage_authorization_keeps_admin_grants_and_conflicting_policy_pending(
        storage_controller):
    server, attachments, iam, s3 = storage_controller
    custom = {
        "Sid": "HistoricalRoots",
        "Effect": "Allow",
        "Action": ["s3:GetObject"],
        "Resource": ["arn:aws:s3:::mlpipeline/historical/team-a/*"]
    }
    iam.get_user_policy.return_value["PolicyDocument"]["Statement"].append(
        custom)
    bucket_policy = sync.managed_bucket_list_policy(sync.S3_BUCKET_NAME)
    bucket_policy["Statement"].append(custom)
    s3.get_bucket_policy.return_value["Policy"] = json.dumps(bucket_policy)
    result, observed = ready_storage(storage_controller)
    assert storage_record(result)["state"] == "ready"
    bucket_policy["Statement"][0]["Resource"] = "custom-bucket"
    s3.get_bucket_policy.return_value["Policy"] = json.dumps(bucket_policy)
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "pending"
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


@pytest.mark.parametrize("failure_operation", ["get", "put"])
@pytest.mark.parametrize("persistent", [False, True])
def test_bucket_policy_initialization_retries_with_persisted_credentials(
        storage_controller, failure_operation, persistent):
    server, attachments, iam, s3 = storage_controller
    attachments["Secret.v1"] = {}
    iam.create_access_key.return_value = {
        "AccessKey": {
            "AccessKeyId": "tenant-access",
            "SecretAccessKey": "tenant-secret",
        }
    }
    unavailable = True
    stored_policy = None

    def read_policy(**kwargs):
        if unavailable and failure_operation == "get":
            raise RuntimeError("temporary bucket read failure")
        if stored_policy is None:
            raise iam_error("NoSuchBucketPolicy")
        return {"Policy": json.dumps(stored_policy)}

    def write_policy(**kwargs):
        nonlocal stored_policy
        if unavailable and failure_operation == "put":
            raise RuntimeError("temporary bucket write failure")
        stored_policy = json.loads(kwargs["Policy"])

    s3.get_bucket_policy.side_effect = read_policy
    s3.put_bucket_policy.side_effect = write_policy
    first = post_sync(server, PARENT, attachments, converge_policy=False)
    assert storage_record(first)["state"] == "pending"
    assert first["resyncAfterSeconds"] == 10
    credential = next(
        resource for resource in first["attachments"]
        if resource["metadata"]["name"] == "mlpipeline-minio-artifact")
    attachments["Secret.v1"][
        f"{NAMESPACE}/mlpipeline-minio-artifact"] = credential
    unavailable = persistent
    retry = post_sync(server, PARENT, attachments)
    observed = observe_receiver_resources(retry, attachments)
    result = post_sync(server, PARENT, observed)
    actual_credential = next(
        resource for resource in result["attachments"]
        if resource["metadata"]["name"] == "mlpipeline-minio-artifact")
    assert actual_credential["data"] == credential["data"]
    iam.create_access_key.assert_called_once_with(UserName=NAMESPACE)
    iam.put_user_policy.assert_not_called()
    if persistent:
        assert storage_record(result)["state"] == "pending"
        assert result["resyncAfterSeconds"] == 10
        assert stored_policy is None
    else:
        assert storage_record(result)["state"] == "ready"
        assert sync.contains_managed_bucket_policy(stored_policy)


def test_existing_namespace_repairs_missing_bucket_statements_preserving_admin(
        storage_controller):
    server, _, iam, s3 = storage_controller
    _, observed = ready_storage(storage_controller)
    custom = {
        "Sid": "CustomGrant",
        "Effect": "Allow",
        "Principal": "*",
        "Action": "s3:GetObject",
        "Resource": "arn:aws:s3:::mlpipeline/shared/*",
    }
    stored = {"Version": "2012-10-17", "Statement": [custom]}
    s3.get_bucket_policy.side_effect = lambda **kwargs: {
        "Policy": json.dumps(stored)
    }

    def write_policy(**kwargs):
        nonlocal stored
        stored = json.loads(kwargs["Policy"])

    s3.put_bucket_policy.side_effect = write_policy
    result = post_sync(server, PARENT, observed)
    assert storage_record(result)["state"] == "ready"
    assert custom in stored["Statement"]
    assert sync.contains_managed_bucket_policy(stored)
    s3.put_bucket_policy.assert_called_once()
    post_sync(server, PARENT, observed)
    s3.put_bucket_policy.assert_called_once()
    iam.create_access_key.assert_not_called()
    iam.put_user_policy.assert_not_called()


def test_storage_credential_rotation_and_namespace_recreation_require_rollout(
        storage_controller):
    server, _, _, _ = storage_controller
    result, observed = ready_storage(storage_controller)
    previous = storage_record(result)
    observed["Secret.v1"][f"{NAMESPACE}/mlpipeline-minio-artifact"]["data"][
        "secretkey"] = base64.b64encode(b"rotated-secret").decode()
    rotated = post_sync(server, PARENT, observed)
    record = storage_record(rotated)
    assert record["state"] == "pending"
    assert record["credentialBinding"] != previous["credentialBinding"]
    observed = observe_receiver_resources(rotated, observed)
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "ready"
    recreated_parent = copy.deepcopy(PARENT)
    recreated_parent["metadata"]["uid"] = "recreated-namespace-uid"
    recreated = storage_record(post_sync(server, recreated_parent, observed))
    assert recreated["state"] == "pending"
    assert recreated["credentialBinding"] != record["credentialBinding"]


def test_storage_registry_updates_in_place_and_is_removed_when_proxy_disabled(
        storage_controller, start_sync_server):
    server, _, _, _ = storage_controller
    result, observed = ready_storage(storage_controller)
    registry = next(item for item in result["attachments"] if item["metadata"]
                    ["name"] == sync.artifact_storage_record_name(NAMESPACE))
    observed["Secret.v1"][
        f"kubeflow/{sync.artifact_storage_record_name(NAMESPACE)}"] = registry
    observed["Deployment.apps/v1"][f"{NAMESPACE}/ml-pipeline-ui-artifact"][
        "status"]["readyReplicas"] = 0
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "pending"
    disabled = start_minimal_server(start_sync_server)
    result = post_sync(disabled, PARENT, observed)
    assert not any(
        item["metadata"]["name"] == sync.artifact_storage_record_name(NAMESPACE)
        for item in result["attachments"])


def test_new_profile_bucket_policy_preserves_admin_statements(
        storage_controller):
    server, _, _, s3 = storage_controller
    handler = object.__new__(server.RequestHandlerClass)
    custom = {
        "Sid": "CustomGrant",
        "Effect": "Allow",
        "Principal": "*",
        "Action": "s3:GetObject",
        "Resource": "arn:aws:s3:::mlpipeline/shared/*"
    }
    s3.get_bucket_policy.return_value["Policy"] = json.dumps({
        "Version": "2012-10-17",
        "Statement": [custom]
    })
    handler.ensure_bucket_list_policy("mlpipeline")
    written = json.loads(s3.put_bucket_policy.call_args.kwargs["Policy"])
    assert custom in written["Statement"]
    assert sync.contains_managed_bucket_policy(written)
    conflicting = sync.managed_bucket_list_policy("mlpipeline")
    conflicting["Statement"][0]["Resource"] = "custom-bucket"
    s3.get_bucket_policy.return_value["Policy"] = json.dumps(conflicting)
    s3.put_bucket_policy.reset_mock()
    handler.ensure_bucket_list_policy("mlpipeline")
    s3.put_bucket_policy.assert_not_called()


def test_only_multi_user_overlay_enables_artifact_proxy():
    multi_user = Path(sync.__file__).resolve().parent.parent
    patch = yaml.safe_load(
        (multi_user / "pipeline-install-config-patch.yaml").read_text())
    generic = yaml.safe_load(
        (multi_user.parent /
         "generic/pipeline-install-config.yaml").read_text())
    kustomization = yaml.safe_load(
        (multi_user / "kustomization.yaml").read_text())
    assert patch["data"]["ARTIFACTS_PROXY_ENABLED"] == "true"
    assert generic["data"]["ARTIFACTS_PROXY_ENABLED"] == "false"
    assert {
        "path": "pipeline-install-config-patch.yaml"
    } in kustomization["patches"]


def iam_error(code):
    error = RuntimeError(code)
    error.response = {"Error": {"Code": code}}
    return error


def test_missing_credential_secret_preserves_existing_iam_policy(
        storage_controller):
    server, _, iam, _ = storage_controller
    custom = {
        "Version":
            "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Action": ["s3:GetObject"],
            "Resource": ["arn:aws:s3:::mlpipeline/legacy-custom/*"]
        }]
    }
    iam.get_user_policy.return_value = {"PolicyDocument": custom}
    handler = object.__new__(server.RequestHandlerClass)
    handler.ensure_namespace_object_policy(NAMESPACE)
    iam.create_user.assert_not_called()
    iam.put_user_policy.assert_not_called()
    assert iam.get_user_policy.return_value["PolicyDocument"] == custom


@pytest.mark.parametrize("already_exists", [False, True])
def test_new_namespace_policy_is_established_before_access_key(
        storage_controller, already_exists):
    server, _, iam, _ = storage_controller
    iam.get_user_policy.side_effect = [
        iam_error("NoSuchEntity"),
        iam_error("NoSuchEntity")
    ]
    if already_exists:
        iam.create_user.side_effect = iam_error("EntityAlreadyExists")
    handler = object.__new__(server.RequestHandlerClass)
    handler.ensure_namespace_object_policy(NAMESPACE)
    iam.create_user.assert_called_once_with(UserName=NAMESPACE)
    iam.put_user_policy.assert_called_once_with(
        UserName=NAMESPACE,
        PolicyName=f"KubeflowProject{NAMESPACE}",
        PolicyDocument=json.dumps(sync.managed_object_policy(NAMESPACE)))
    iam.create_access_key.assert_not_called()


@pytest.mark.parametrize("failure", ["inspection", "user", "policy"])
def test_namespace_iam_failures_do_not_mint_orphan_keys(storage_controller,
                                                        failure):
    server, attachments, iam, _ = storage_controller
    attachments["Secret.v1"].clear()
    if failure == "inspection":
        iam.get_user_policy.side_effect = RuntimeError("IAM unavailable")
    else:
        iam.get_user_policy.side_effect = [
            iam_error("NoSuchEntity"),
            iam_error("NoSuchEntity")
        ]
        if failure == "user":
            iam.create_user.side_effect = RuntimeError("IAM unavailable")
        else:
            iam.put_user_policy.side_effect = RuntimeError("IAM unavailable")
    handler = object.__new__(server.RequestHandlerClass)
    with pytest.raises(RuntimeError, match="IAM unavailable"):
        handler.sync(PARENT, attachments)
    iam.create_access_key.assert_not_called()

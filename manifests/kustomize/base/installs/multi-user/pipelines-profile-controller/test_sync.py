import base64
import copy
import json
import os
from pathlib import Path
import threading
from unittest import mock

import pytest
import requests
import sync
from sync import get_settings_from_env
from sync import server_factory
import yaml

KFP_VERSION = "x.y.z"
ENV_KFP_VERSION_ONLY = {"KFP_VERSION": KFP_VERSION, "CONTROLLER_PORT": "0"}
ENV_ARTIFACT_PROXY_WITH_ALLOWED_ENDPOINTS = {
    **ENV_KFP_VERSION_ONLY,
    "ALLOWED_ARTIFACT_ENDPOINTS":
        "https://objects.example.com:9443",
    "ARTIFACTS_PROXY_ENABLED":
        "true",
    "HTTP_BASE_URL":
        "https://artifacts.example.com:9443/pipelines/",
}


@pytest.fixture
def sync_server(request, start_sync_server):
    with mock.patch.dict(os.environ, request.param, clear=True):
        server = start_sync_server(**get_settings_from_env())
    return server, request.param


@pytest.mark.parametrize(
    "sync_server, expected_image", [
        (ENV_KFP_VERSION_ONLY, None),
        ({
            **ENV_KFP_VERSION_ONLY, "ARTIFACTS_PROXY_ENABLED": "true"
        }, "ghcr.io/kubeflow/kfp-frontend:x.y.z"),
        ({
            **ENV_KFP_VERSION_ONLY, "ARTIFACTS_PROXY_ENABLED": "true",
            "FRONTEND_IMAGE": "custom"
        }, "custom:x.y.z"),
        ({
            **ENV_KFP_VERSION_ONLY, "ARTIFACTS_PROXY_ENABLED": "true",
            "FRONTEND_IMAGE": "custom",
            "FRONTEND_TAG": "tag"
        }, "custom:tag"),
    ],
    indirect=["sync_server"])
def test_sync_server_with_pipeline_enabled(sync_server, expected_image):
    server, _ = sync_server
    result = post_sync(server, PARENT, observed_attachments())
    assert result["status"] == {"kubeflow-pipelines-ready": "False"}
    resources = result["attachments"]
    assert {
        item["metadata"]["name"]
        for item in resources
        if item["kind"] == "ConfigMap"
    } == {"kfp-launcher", "metadata-grpc-configmap", "artifact-repositories"}
    deployments = [item for item in resources if item["kind"] == "Deployment"]
    if expected_image:
        assert len(deployments) == 1
        assert deployments[0]["spec"]["template"]["spec"]["containers"][0][
            "image"] == expected_image
    else:
        assert deployments == []


def test_sync_server_with_direct_passing_of_settings(start_sync_server):
    server = start_sync_server(
        frontend_image="custom",
        frontend_tag="tag",
        disable_istio_sidecar=True,
        artifacts_proxy_enabled="true",
        artifact_retention_days=-1,
        controller_port=0)
    result = post_sync(server, PARENT, observed_attachments())
    deployment = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    assert deployment["spec"]["template"]["spec"]["containers"][0][
        "image"] == "custom:tag"
    assert deployment["spec"]["template"]["metadata"]["annotations"][
        "sidecar.istio.io/inject"] == "false"


def test_sync_server_without_pipeline_enabled(start_sync_server):
    server = start_minimal_server(start_sync_server)
    assert post_sync(server, {}, {}) == {"status": {}, "attachments": []}


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
    results = post_sync(
        server, PARENT, {
            'Secret.v1': {
                'myName/mlpipeline-minio-artifact': existing_secret
            },
            'ConfigMap.v1': {},
            'Deployment.apps/v1': {},
            'Service.v1': {},
        })
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


@pytest.mark.parametrize("install_namespace", ["kubeflow", "custom-install"])
@pytest.mark.parametrize("sidecar_disabled", [False, True])
def test_artifact_network_policy_binds_shared_ui_identity(
        start_sync_server, install_namespace, sidecar_disabled):
    server = start_sync_server(
        frontend_image="image",
        frontend_tag="tag",
        disable_istio_sidecar=sidecar_disabled,
        artifacts_proxy_enabled="true",
        artifact_retention_days=-1,
        controller_port=0,
        kfp_namespace=install_namespace)
    secret = {
        "apiVersion": "v1",
        "kind": "Secret",
        "metadata": {
            "name": "mlpipeline-minio-artifact",
            "namespace": "myName"
        }
    }
    attachments = {
        "Secret.v1": {
            "myName/mlpipeline-minio-artifact": secret
        },
        "ConfigMap.v1": {
            "launcher": {},
            "repository": {},
            "metadata": {}
        },
        "Deployment.apps/v1": {
            "artifact": {}
        },
        "Service.v1": {
            "artifact": {}
        },
    }
    result = post_sync(server, PARENT, attachments, converge_policy=False)
    assert result["status"] == {"kubeflow-pipelines-ready": "False"}
    policy = next(item for item in result["attachments"]
                  if item["kind"] == "NetworkPolicy")
    assert policy["metadata"] == {
        "name": "ml-pipeline-ui-artifact",
        "namespace": "myName"
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
                        "kubernetes.io/metadata.name": install_namespace
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
    attachments["NetworkPolicy.networking.k8s.io/v1"] = {
        "myName/ml-pipeline-ui-artifact": policy
    }
    result = post_sync(server, PARENT, attachments)
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
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
        "value": "myName"
    } in environment


def test_kfp_namespace_setting_override():
    with mock.patch.dict(os.environ, {"KFP_VERSION": KFP_VERSION}, clear=True):
        assert get_settings_from_env()["kfp_namespace"] == "kubeflow"
    with mock.patch.dict(
            os.environ, {
                "KFP_VERSION": KFP_VERSION,
                "KFP_NAMESPACE": "custom"
            },
            clear=True):
        assert get_settings_from_env()["kfp_namespace"] == "custom"
        assert get_settings_from_env(
            kfp_namespace="other")["kfp_namespace"] == "other"


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
ENV_IMAGES = {
    "KFP_VERSION": KFP_VERSION,
    "CONTROLLER_PORT": "0",
    "FRONTEND_IMAGE": "frontend-image",
    "FRONTEND_TAG": "somehash",
    "ARTIFACTS_PROXY_ENABLED": "true"
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


def observed_attachments(proxy_enabled=False, ready=False):
    return {
        "Secret.v1": {
            f"{NAMESPACE}/mlpipeline-minio-artifact":
                copy.deepcopy(EXISTING_SECRET)
        },
        "ConfigMap.v1": {},
        "Deployment.apps/v1": {},
        "Service.v1": {},
        "NetworkPolicy.networking.k8s.io/v1": {},
    }


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


@pytest.mark.parametrize("mutation", [
    "old-generation", "old-replicas", "unready", "wrong-image", "wrong-policy",
    "wrong-service"
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


def test_storage_authorization_keeps_admin_grants_and_missing_list_policy_pending(
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
    bucket_policy["Statement"] = [custom]
    s3.get_bucket_policy.return_value["Policy"] = json.dumps(bucket_policy)
    assert storage_record(post_sync(server, PARENT,
                                    observed))["state"] == "pending"
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


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

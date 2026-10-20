import base64
import copy
import json
import os
from pathlib import Path
import threading
from unittest import mock

from botocore.exceptions import ClientError
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
            server.RequestHandlerClass,
            "sync",
            lambda handler, parent, attachments, related=None: {
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

    def slow_sync(handler, parent, attachments, related=None):
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


def test_storage_authorization_records_are_updated_in_place():
    controller = yaml.safe_load(
        Path(sync.__file__).with_name("decorator-controller.yaml").read_text())
    secret_rules = [
        rule for rule in controller["spec"]["attachments"]
        if rule["resource"] == "secrets"
    ]
    assert len(secret_rules) == 1
    assert secret_rules[0]["updateStrategy"]["method"] == "InPlace"


def test_existing_credentials_preserve_data_without_metadata_churn(
        storage_controller):
    server, observed, _, _ = storage_controller
    key = f"{NAMESPACE}/mlpipeline-minio-artifact"
    credential = observed["Secret.v1"][key]
    credential["type"] = "Opaque"
    credential["metadata"].update({
        "labels": {
            "operator": "managed"
        },
        "annotations": {
            "custom": "preserve-me"
        },
        "uid": "credential-uid",
        "ownerReferences": [{
            "kind": "Namespace",
            "uid": "namespace-uid"
        }],
        "creationTimestamp": "2026-10-15T00:00:00Z",
        "managedFields": [{
            "manager": "metacontroller"
        }],
    })
    expected = copy.deepcopy(credential)
    expected["metadata"] = {
        key: expected["metadata"][key]
        for key in ("name", "namespace", "labels", "annotations")
    }
    for version in range(1, 9):
        credential["metadata"]["resourceVersion"] = str(version)
        credential["metadata"]["annotations"][
            "metacontroller.k8s.io/last-applied-configuration"] = json.dumps(
                expected)
        before = copy.deepcopy(observed)
        result = post_sync(server, PARENT, observed)
        desired = next(
            resource for resource in result["attachments"]
            if resource["kind"] == "Secret" and
            resource["metadata"]["name"] == "mlpipeline-minio-artifact")
        assert desired == expected
        assert observed == before


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
def test_storage_authorization_api_failure_is_pending_and_recovers(
        storage_controller, operation):
    server, _, iam, s3 = storage_controller
    ready, observed = ready_storage(storage_controller)
    ready_record = storage_record(ready)
    assert ready_record['state'] == 'ready'
    credential_name = f'{NAMESPACE}/mlpipeline-minio-artifact'
    original_credentials = copy.deepcopy(observed['Secret.v1'][credential_name])
    for resource in ready['attachments']:
        if resource['metadata']['name'] == sync.artifact_storage_record_name(
                NAMESPACE):
            observed['Secret.v1'][
                f"kubeflow/{resource['metadata']['name']}"] = resource
    client = s3 if operation == 'get_bucket_policy' else iam
    read = getattr(client, operation)
    read.side_effect = RuntimeError('do not log credential-bearing details')
    pending = post_sync(server, PARENT, observed)
    assert storage_record(pending)['state'] == 'pending'
    assert pending['resyncAfterSeconds'] == 10
    for resource in pending['attachments']:
        if resource['metadata']['name'] == sync.artifact_storage_record_name(
                NAMESPACE):
            observed['Secret.v1'][
                f"kubeflow/{resource['metadata']['name']}"] = resource
    read.side_effect = None
    recovered = post_sync(server, PARENT, observed)
    assert storage_record(recovered) == ready_record
    assert 'resyncAfterSeconds' not in recovered
    for result in [pending, recovered]:
        credential = next(
            resource for resource in result['attachments']
            if resource['metadata']['name'] == 'mlpipeline-minio-artifact')
        assert credential['data'] == original_credentials['data']
    assert observed['Secret.v1'][credential_name] == original_credentials
    iam.create_access_key.assert_not_called()
    iam.put_user_policy.assert_not_called()
    s3.put_bucket_policy.assert_not_called()


@pytest.mark.parametrize("admin_resource",
                         ["arn:aws:s3:::mlpipeline/historical/team-a/*", "*"])
def test_storage_authorization_keeps_admin_grants_and_conflicting_policy_pending(
        storage_controller, admin_resource):
    server, attachments, iam, s3 = storage_controller
    custom = {
        "Sid": "HistoricalRoots",
        "Effect": "Allow",
        "Action": ["s3:GetObject"],
        "Resource": [admin_resource]
    }
    iam.get_user_policy.return_value["PolicyDocument"]["Statement"].append(
        custom)
    bucket_policy = sync.managed_bucket_list_policy(sync.S3_BUCKET_NAME)
    bucket_policy["Statement"].append(custom)
    s3.get_bucket_policy.return_value["Policy"] = json.dumps(bucket_policy)
    original_object_policy = copy.deepcopy(
        iam.get_user_policy.return_value["PolicyDocument"])
    result, observed = ready_storage(storage_controller)
    assert storage_record(result)["state"] == "ready"
    assert iam.get_user_policy.return_value[
        "PolicyDocument"] == original_object_policy
    assert json.loads(
        s3.get_bucket_policy.return_value["Policy"]) == bucket_policy
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


def external_approval():
    return {
        "namespaceUID": "namespace-uid",
        "storage": {
            "source": "s3",
            "bucket": "historical-models",
            "endpoint": "https://objects.example.test",
            "region": "us-east-1"
        },
        "identity": {
            "kind": "shared-credentials-file",
            "principal": "broker/team-a",
            "credentialFile": "/var/run/broker/credentials",
            "profile": "tenant"
        },
        "podTemplate": {
            "metadata": {
                "annotations": {
                    "operator.example/broker": "approved"
                }
            },
            "spec": {
                "serviceAccountName":
                    "artifact-storage",
                "volumes": [{
                    "name": "credentials",
                    "emptyDir": {}
                }],
                "containers": [
                    {
                        "name":
                            "ml-pipeline-ui-artifact",
                        "image":
                            "old-image",
                        "env": [],
                        "volumeMounts": [{
                            "name": "credentials",
                            "mountPath": "/var/run/broker",
                            "readOnly": True
                        }]
                    },
                    {
                        "name":
                            "credential-broker",
                        "image":
                            "broker@sha256:" + "a" * 64,
                        "env": [],
                        "volumeMounts": [{
                            "name": "credentials",
                            "mountPath": "/var/run/broker"
                        }]
                    },
                ]
            }
        },
    }


def external_related(approval):
    return {
        "ConfigMap.v1": {
            "kubeflow/kfp-artifact-storage-config": {
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "kfp-artifact-storage-config",
                    "namespace": "kubeflow"
                },
                "data": {
                    "approvals.json": json.dumps({NAMESPACE: approval})
                },
            }
        }
    }


def post_external_sync(server, attachments, related):
    host, port = server.server_address
    response = requests.post(
        f"http://{host}:{port}/sync",
        json={
            "object": PARENT,
            "attachments": attachments,
            "related": related
        },
        timeout=5)
    response.raise_for_status()
    return response.json()


def external_ready(storage_controller):
    server, observed, _, _ = storage_controller
    related = external_related(external_approval())
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    return result, observed, related


def test_external_storage_customization_watches_admin_config_and_receiver(
        storage_controller):
    server, _, _, _ = storage_controller
    host, port = server.server_address
    response = requests.post(
        f"http://{host}:{port}/customize", json={"parent": PARENT}, timeout=5)
    response.raise_for_status()
    resources = response.json()["relatedResources"]
    assert resources[0] == {
        "apiVersion": "v1",
        "resource": "configmaps",
        "namespace": "kubeflow",
        "names": ["kfp-artifact-storage-config"]
    }
    assert resources[1] == {
        "apiVersion": "v1",
        "resource": "configmaps",
        "namespace": "kubeflow",
        "labelSelector": {
            "matchLabels": {
                sync.POLICY_SOURCE_LABEL: "true"
            }
        }
    }
    assert {resource["resource"] for resource in resources[2:]
           } == {"deployments", "services", "networkpolicies"}
    assert all(resource["namespace"] == NAMESPACE for resource in resources[2:])


def test_external_storage_approval_preserves_broker_and_never_manages_credentials(
        storage_controller):
    server, _, iam, s3 = storage_controller
    result, observed, related = external_ready(storage_controller)
    record = storage_record(result)
    assert record["version"] == 2
    assert record["state"] == "ready"
    assert "resyncAfterSeconds" not in result
    approval = external_approval()
    assert record["storage"] == approval["storage"]
    assert record["identity"] == approval["identity"]
    assert record["credentialBinding"] == sync.external_credential_binding(
        NAMESPACE, "namespace-uid", approval["storage"], approval["identity"],
        record["configurationRevision"])
    receiver = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    spec = receiver["spec"]["template"]["spec"]
    assert spec["containers"][1] == approval["podTemplate"]["spec"][
        "containers"][1]
    assert spec["volumes"] == approval["podTemplate"]["spec"]["volumes"]
    assert spec["serviceAccountName"] == "artifact-storage"
    assert spec["containers"][0]["image"] == "frontend-image:somehash"
    env = {item["name"]: item for item in spec["containers"][0]["env"]}
    assert not any(name.startswith("MINIO_") for name in env)
    assert json.loads(env[sync.EXTERNAL_STORAGE_ENV]["value"]) == {
        **{
            key: approval[key]
            for key in ["namespaceUID", "storage", "identity"]
        },
        "configurationRevision": record["configurationRevision"],
    }
    # Rotating credentials outside the approved pod template do not change its binding.
    observed["Secret.v1"][f"{NAMESPACE}/mlpipeline-minio-artifact"]["data"][
        "secretkey"] = base64.b64encode(b"rotated-unused-key").decode()
    again = post_external_sync(server, observed, related)
    assert storage_record(again) == record
    iam.assert_not_called()
    assert iam.mock_calls == []
    assert s3.mock_calls == []


@pytest.mark.parametrize("mutation", [
    "env", "envFrom", "sidecar", "volume", "serviceAccount", "notReady",
    "staleGeneration"
])
def test_external_storage_rejects_unapproved_or_unready_receiver(
        storage_controller, mutation):
    server, _, _, _ = storage_controller
    _, observed, related = external_ready(storage_controller)
    receiver = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]
    spec = receiver["spec"]["template"]["spec"]
    if mutation == "env":
        spec["containers"][0]["env"].append({
            "name": "AWS_ACCESS_KEY_ID",
            "value": "unapproved"
        })
    elif mutation == "envFrom":
        spec["containers"][0]["envFrom"] = [{
            "secretRef": {
                "name": "unapproved"
            }
        }]
    elif mutation == "sidecar":
        spec["containers"].append({"name": "extra-broker", "image": "other"})
    elif mutation == "volume":
        spec["volumes"].append({"name": "extra-credentials", "emptyDir": {}})
    elif mutation == "serviceAccount":
        spec["serviceAccountName"] = "other-identity"
    elif mutation == "notReady":
        receiver["status"]["readyReplicas"] = 0
    else:
        receiver["metadata"]["generation"] += 1
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    assert result["resyncAfterSeconds"] == 10


@pytest.mark.parametrize(
    "invalid",
    ["namespace", "endpoint", "principal", "credentialFile", "template"])
def test_external_invalid_approval_fails_closed_without_minting_credentials(
        storage_controller, invalid):
    server, observed, iam, s3 = storage_controller
    observed["Secret.v1"].clear()
    approval = external_approval()
    if invalid == "namespace":
        approval["namespaceUID"] = "old-namespace"
    elif invalid == "endpoint":
        approval["storage"][
            "endpoint"] = "https://user:password@objects.example.test"
    elif invalid == "principal":
        approval["identity"]["principal"] = ""
    elif invalid == "credentialFile":
        approval["identity"]["credentialFile"] = "../credentials"
    else:
        approval["podTemplate"]["spec"]["containers"] = []
    result = post_external_sync(server, observed, external_related(approval))
    assert storage_record(result) == {
        "version": 2,
        "namespace": NAMESPACE,
        "namespaceUID": "namespace-uid",
        "state": "pending"
    }
    assert iam.mock_calls == s3.mock_calls == []


def test_external_approval_revocation_preserves_receiver_and_pending_tombstone(
        storage_controller):
    server, _, iam, s3 = storage_controller
    result, observed, _ = external_ready(storage_controller)
    record_secret = next(item for item in result["attachments"]
                         if item["metadata"]["name"] ==
                         sync.artifact_storage_record_name(NAMESPACE))
    observed["Secret.v1"][
        f"kubeflow/{sync.artifact_storage_record_name(NAMESPACE)}"] = record_secret
    revoked = post_external_sync(server, observed, {})
    assert storage_record(revoked)["state"] == "pending"
    assert storage_record(revoked)["version"] == 2
    desired = next(
        item for item in revoked["attachments"] if item["kind"] == "Deployment")
    assert desired["spec"] == observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]
    assert iam.mock_calls == s3.mock_calls == []


@pytest.mark.parametrize("kind,api_version",
                         [("Deployment", "apps/v1"), ("Service", "v1"),
                          ("NetworkPolicy", "networking.k8s.io/v1")])
@pytest.mark.parametrize("namespace_owner", [False, True])
def test_external_approval_does_not_adopt_foreign_receiver(
        storage_controller, kind, api_version, namespace_owner):
    server, observed, iam, s3 = storage_controller
    related = external_related(external_approval())
    owner = {
        "apiVersion": "example/v1",
        "kind": "Broker",
        "name": "broker",
        "uid": "foreign-uid",
    }
    if namespace_owner:
        owner = {
            "apiVersion": "v1",
            "kind": "Namespace",
            "name": NAMESPACE,
            "uid": PARENT["metadata"]["uid"],
            "controller": True,
        }
    related[f"{kind}.{api_version}"] = {
        f"{NAMESPACE}/ml-pipeline-ui-artifact": {
            "apiVersion": api_version,
            "kind": kind,
            "metadata": {
                "name": "ml-pipeline-ui-artifact",
                "namespace": NAMESPACE,
                "ownerReferences": [owner],
            },
            "spec": {},
        }
    }
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    # Even a matching Namespace owner cannot authorize taking over a resource
    # absent from this decorator's attachments.
    assert not any(item["kind"] == kind and
                   item["metadata"]["name"] == "ml-pipeline-ui-artifact"
                   for item in result["attachments"])
    assert iam.mock_calls == s3.mock_calls == []


def test_external_approval_change_invalidates_readiness_and_binding(
        storage_controller):
    server, _, _, _ = storage_controller
    before, observed, _ = external_ready(storage_controller)
    approval = external_approval()
    approval["identity"]["principal"] = "broker/new-identity"
    result = post_external_sync(server, observed, external_related(approval))
    assert storage_record(result)["state"] == "pending"
    assert storage_record(result)["credentialBinding"] != storage_record(
        before)["credentialBinding"]


def test_external_storage_requires_network_policy_before_initial_receiver(
        storage_controller):
    server, observed, _, _ = storage_controller
    result = post_external_sync(server, observed,
                                external_related(external_approval()))
    assert not any(
        item["kind"] == "Deployment" for item in result["attachments"])
    assert any(
        item["kind"] == "NetworkPolicy" for item in result["attachments"])
    assert storage_record(result)["state"] == "pending"


@pytest.mark.parametrize(
    "mutation",
    ["invalidJSON", "malformedEndpoint", "hostNetwork", "duplicateContainer"])
def test_external_malformed_configuration_remains_pending(
        storage_controller, mutation):
    server, observed, iam, s3 = storage_controller
    approval = external_approval()
    if mutation == "malformedEndpoint":
        approval["storage"]["endpoint"] = "https://[malformed"
    elif mutation == "hostNetwork":
        approval["podTemplate"]["spec"]["hostNetwork"] = True
    elif mutation == "duplicateContainer":
        approval["podTemplate"]["spec"]["containers"].append(
            copy.deepcopy(approval["podTemplate"]["spec"]["containers"][1]))
    related = external_related(approval)
    if mutation == "invalidJSON":
        related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
            "approvals.json"] = "{"
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    assert iam.mock_calls == s3.mock_calls == []


def test_external_receiver_with_additional_owner_is_preserved(
        storage_controller):
    server, _, _, _ = storage_controller
    _, observed, related = external_ready(storage_controller)
    receiver = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]
    receiver["metadata"]["ownerReferences"] = [{
        "kind": "Namespace",
        "uid": "namespace-uid"
    }, {
        "kind": "Broker",
        "uid": "other-owner"
    }]
    receiver["spec"]["template"]["spec"]["containers"][0][
        "image"] = "foreign-image"
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    returned = next(
        item for item in result["attachments"] if item["kind"] == "Deployment")
    assert returned["spec"] == receiver["spec"]
    assert returned["metadata"]["ownerReferences"] == receiver["metadata"][
        "ownerReferences"]


@pytest.mark.parametrize("field",
                         ["serviceAccountName", "brokerImage", "volumeSource"])
def test_external_receiver_configuration_revision_invalidates_binding(
        storage_controller, field):
    server, _, _, _ = storage_controller
    before, observed, _ = external_ready(storage_controller)
    approval = external_approval()
    spec = approval["podTemplate"]["spec"]
    if field == "serviceAccountName":
        spec["serviceAccountName"] = "changed-account"
    elif field == "brokerImage":
        spec["containers"][1]["image"] = "new-broker@sha256:" + "b" * 64
    else:
        spec["volumes"][0] = {
            "name": "credentials",
            "secret": {
                "secretName": "another-source"
            }
        }
    result = post_external_sync(server, observed, external_related(approval))
    assert storage_record(result)["state"] == "pending"
    assert storage_record(result)["configurationRevision"] != storage_record(
        before)["configurationRevision"]
    assert storage_record(result)["credentialBinding"] != storage_record(
        before)["credentialBinding"]


@pytest.mark.parametrize(
    "bad_approval",
    [None, {
        "namespaceUID": "namespace-uid",
        "podTemplate": None
    }])
def test_external_null_or_invalid_approval_does_not_select_managed_storage(
        storage_controller, bad_approval):
    server, observed, iam, s3 = storage_controller
    result = post_external_sync(server, observed,
                                external_related(bad_approval))
    assert storage_record(result)["version"] == 2
    assert storage_record(result)["state"] == "pending"
    assert iam.mock_calls == s3.mock_calls == []


@pytest.mark.parametrize("section,field,value", [
    ("storage", "endpoint", "https://objects.example.test:443"),
    ("storage", "endpoint", "https://OBJECTS.example.test"),
    ("storage", "endpoint", "http://127.1"),
    ("storage", "endpoint", "https://objects.example.test/"),
    ("storage", "region", "with spaces"),
    ("storage", "region", "a" * 65),
    ("identity", "principal", "a" * 513),
    ("identity", "principal", "with spaces"),
    ("identity", "profile", "a" * 129),
    ("identity", "profile", "with/slash"),
    ("identity", "credentialFile", "/var/run/../credentials"),
    ("identity", "credentialFile", "/var//credentials"),
    ("identity", "credentialFile", "//var/credentials"),
    ("identity", "credentialFile", "/var/credentials/"),
])
def test_external_approval_uses_receiver_configuration_validation(
        section, field, value):
    approval = external_approval()
    approval[section][field] = value
    assert not sync.external_approval_valid(approval, "namespace-uid")


def test_external_binding_matches_frontend_protocol_vector():
    assert sync.external_credential_binding(
        "team-a", "uid-a", {
            "source": "s3",
            "bucket": "external-artifacts",
            "endpoint": "https://store.example:9443",
            "region": "us-east-1"
        }, {
            "kind": "shared-credentials-file",
            "principal": "broker:team-a",
            "credentialFile": "/var/run/broker/credentials",
            "profile": "artifacts"
        }, "a" * 64
    ) == "d1b1915844609b2d71e35a33f6d8e9917f34237a0a9f61135cdbc8679b0fc9a2"


@pytest.mark.parametrize(
    "mutation",
    ["command", "args", "envFrom", "env", "projectedSources", "projectedItems"])
def test_external_receiver_rejects_credential_wiring_reordering(
        storage_controller, mutation):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    spec = approval["podTemplate"]["spec"]
    broker = spec["containers"][1]
    broker["command"] = ["python", "-m", "broker"]
    broker["args"] = ["--role", "read-only", "--fallback-role", "admin"]
    broker["envFrom"] = [{
        "configMapRef": {
            "name": "read-only"
        }
    }, {
        "configMapRef": {
            "name": "admin"
        }
    }]
    broker["env"] = [{
        "name": "ROLE",
        "value": "read-only"
    }, {
        "name": "EFFECTIVE_ROLE",
        "value": "$(ROLE)"
    }]
    spec["volumes"] = [{
        "name": "credentials",
        "projected": {
            "sources": [
                {
                    "secret": {
                        "name":
                            "read-only",
                        "items": [{
                            "key": "role",
                            "path": "credentials"
                        }, {
                            "key": "fallback",
                            "path": "credentials"
                        }]
                    }
                },
                {
                    "secret": {
                        "name": "admin",
                        "items": [{
                            "key": "role",
                            "path": "credentials"
                        }]
                    }
                },
            ]
        }
    }]
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    broker = actual["containers"][1]
    if mutation == "command":
        broker["command"] = ["broker", "-m", "python"]
    elif mutation == "args":
        broker["args"] = ["--role", "admin", "--fallback-role", "read-only"]
    elif mutation in ["env", "envFrom"]:
        broker[mutation].reverse()
    elif mutation == "projectedSources":
        actual["volumes"][0]["projected"]["sources"].reverse()
    else:
        actual["volumes"][0]["projected"]["sources"][0]["secret"][
            "items"].reverse()
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"


def test_external_receiver_accepts_defaulted_init_environment_and_volume(
        storage_controller):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    spec = approval["podTemplate"]["spec"]
    spec["initContainers"] = [{
        "name": "initialize-credentials",
        "image": "broker",
        "env": [{
            "name": "OPTIONAL_SCOPE",
            "value": ""
        }]
    }]
    spec["volumes"] = [{
        "name": "credentials",
        "projected": {
            "sources": [
                {
                    "serviceAccountToken": {
                        "audience": "artifact-broker",
                        "path": "token"
                    }
                },
                {
                    "downwardAPI": {
                        "items": [{
                            "path": "namespace",
                            "fieldRef": {
                                "fieldPath": "metadata.namespace"
                            }
                        }]
                    }
                },
            ]
        }
    }]
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    actual["initContainers"][0]["env"][0].pop("value")
    actual["volumes"][0]["projected"]["defaultMode"] = 420
    actual["volumes"][0]["projected"]["sources"][0]["serviceAccountToken"][
        "expirationSeconds"] = 3600
    actual["volumes"][0]["projected"]["sources"][1]["downwardAPI"]["items"][0][
        "fieldRef"]["apiVersion"] = "v1"
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "ready"


def test_external_receiver_rejects_new_environment_prefix(storage_controller):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    approval["podTemplate"]["spec"]["containers"][1]["envFrom"] = [{
        "configMapRef": {
            "name": "broker-settings"
        }
    }]
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    actual["containers"][1]["envFrom"][0]["prefix"] = "IGNORED_"
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"


@pytest.mark.parametrize(
    "mutation",
    ["postStartAdded", "postStartChanged", "restartPolicy", "securityContext"])
def test_external_receiver_rejects_unapproved_execution_hooks(
        storage_controller, mutation):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    spec = approval["podTemplate"]["spec"]
    spec["initContainers"] = [{
        "name": "credential-initializer",
        "image": "broker",
        "env": []
    }]
    if mutation == "postStartChanged":
        spec["containers"][1]["lifecycle"] = {
            "postStart": {
                "exec": {
                    "command": [
                        "broker", "--role", "read-only", "--fallback-role",
                        "admin"
                    ]
                }
            }
        }
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    if mutation == "postStartAdded":
        actual["containers"][1]["lifecycle"] = {
            "postStart": {
                "exec": {
                    "command": ["broker", "--role", "admin"]
                }
            }
        }
    elif mutation == "postStartChanged":
        actual["containers"][1]["lifecycle"]["postStart"]["exec"]["command"] = [
            "broker", "--role", "admin", "--fallback-role", "read-only"
        ]
    elif mutation == "restartPolicy":
        actual["initContainers"][0]["restartPolicy"] = "Always"
    else:
        actual["containers"][1]["securityContext"] = {"privileged": True}
    snapshot = copy.deepcopy(observed)
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    assert observed == snapshot


@pytest.mark.parametrize("omitted", [
    "env", "initEnv", "envFrom", "command", "args", "volumeMounts",
    "volumeDevices", "volumes", "initContainers", "prefix", "subPath",
    "readOnly"
])
def test_external_receiver_accepts_api_omitted_optional_credential_fields(
        storage_controller, omitted):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    spec = approval["podTemplate"]["spec"]
    broker = spec["containers"][1]
    if omitted == "initEnv":
        spec["initContainers"] = [{
            "name": "initialize",
            "image": "broker",
            "env": []
        }]
    elif omitted == "initContainers":
        spec["initContainers"] = []
    elif omitted == "volumes":
        spec["volumes"] = []
        for container in spec["containers"]:
            container["volumeMounts"] = []
    elif omitted == "prefix":
        broker["envFrom"] = [{"prefix": "", "configMapRef": {"name": "broker"}}]
    elif omitted in ["subPath", "readOnly"]:
        broker["volumeMounts"][0][
            omitted] = False if omitted == "readOnly" else ""
    else:
        broker[omitted] = []
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    if omitted == "initEnv":
        actual["initContainers"][0].pop("env")
    elif omitted in ["volumes", "initContainers"]:
        actual.pop(omitted)
    elif omitted == "prefix":
        actual["containers"][1]["envFrom"][0].pop("prefix")
    elif omitted in ["subPath", "readOnly"]:
        actual["containers"][1]["volumeMounts"][0].pop(omitted)
    else:
        actual["containers"][1].pop(omitted)
    before = copy.deepcopy(observed)
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "ready"
    assert observed == before


def test_external_receiver_does_not_normalize_away_required_containers(
        storage_controller):
    server, _, _, _ = storage_controller
    _, observed, related = external_ready(storage_controller)
    actual = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]["template"]["spec"]
    actual.pop("containers")
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"


def adoption_policy():
    return {
        "name": "external-storage",
        "namespaceSelector": {
            "matchLabels": {
                "pipelines.kubeflow.org/enabled": "true"
            }
        },
        "bindingsSource": {
            "namespace": "kubeflow",
            "configMap": "storage-bindings",
            "key": "bindings.json"
        },
        "allowedStorage": external_approval()["storage"],
        "receiverTemplateRef": {
            "namespace": "kubeflow",
            "configMap": "storage-template",
            "key": "template.json"
        },
    }


def adoption_related(policy=None):
    approval = external_approval()
    identity = {
        key: value
        for key, value in approval["identity"].items()
        if key != "principal"
    }
    contents = [
        ("kfp-artifact-storage-config", "adoption-policies.json", {
            "version": 1,
            "policies": [policy or adoption_policy()]
        }),
        ("storage-bindings", "bindings.json", {
            NAMESPACE: {
                "namespaceUID": "namespace-uid",
                "principal": approval["identity"]["principal"]
            }
        }),
        ("storage-template", "template.json", {
            "identity": identity,
            "podTemplate": approval["podTemplate"]
        }),
    ]
    return {
        "ConfigMap.v1": {
            f"kubeflow/{name}": {
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": name,
                    "namespace": "kubeflow",
                    "labels": {
                        sync.POLICY_SOURCE_LABEL: "true"
                    }
                },
                "data": {
                    key: json.dumps(value)
                }
            } for name, key, value in contents
        }
    }


def resolve_adoption(related, parent=None):
    return sync.external_storage_approval(
        parent or PARENT,
        related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"],
        related, "kubeflow")


def test_adoption_policy_reuses_external_authorization_without_minting_credentials(
        storage_controller):
    server, observed, iam, s3 = storage_controller
    related = adoption_related()
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    record = storage_record(result)
    assert record["state"] == "ready"
    assert record["adoptionPolicy"] == {"name": "external-storage"}
    assert record["identity"] == external_approval()["identity"]
    assert record["storage"] == external_approval()["storage"]
    assert iam.mock_calls == []
    assert s3.mock_calls == []
    # Another namespace's credential rotation/record update does not restart this receiver.
    source = related["ConfigMap.v1"]["kubeflow/storage-bindings"]
    bindings = json.loads(source["data"]["bindings.json"])
    bindings["other-team"] = {
        "namespaceUID": "other-uid",
        "principal": "broker/other-team"
    }
    source["data"]["bindings.json"] = json.dumps(bindings)
    source["metadata"]["resourceVersion"] = "new-version"
    again = post_external_sync(server, observed, related)
    assert storage_record(again) == record
    assert again["attachments"] == result["attachments"]


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


@pytest.mark.parametrize("initial_state", ["stock", "external"])
def test_external_pending_recovery_keeps_receiver_ownership(
        storage_controller, initial_state):
    server, observed, iam, s3 = storage_controller
    related = adoption_related()
    if initial_state == "stock":
        initial = post_sync(server, PARENT, observed)
    else:
        for _ in range(3):
            initial = post_external_sync(server, observed, related)
            observed = observe_receiver_resources(initial, observed)
        assert storage_record(initial)["state"] == "ready"

    owner = {
        "apiVersion": "v1",
        "kind": "Namespace",
        "name": NAMESPACE,
        "uid": PARENT["metadata"]["uid"],
        "controller": True,
        "blockOwnerDeletion": True,
    }
    receiver_kinds = {"Deployment", "Service", "NetworkPolicy"}
    for resource in initial["attachments"]:
        if resource["kind"] not in receiver_kinds:
            continue
        stored = copy.deepcopy(resource)
        stored["metadata"].setdefault(
            "annotations",
            {})[sync.LAST_APPLIED_ANNOTATION] = json.dumps(resource)
        # DynamicApply claims ownership after saving the initial desired state.
        stored["metadata"]["ownerReferences"] = [copy.deepcopy(owner)]
        observed = observe_receiver_resources({"attachments": [stored]},
                                              observed)
    iam.reset_mock()
    s3.reset_mock()

    def apply_receiver_update(result):
        nonlocal observed
        for resource in result["attachments"]:
            if resource["kind"] == "Secret" and resource["metadata"][
                    "name"] == sync.artifact_storage_record_name(NAMESPACE):
                observed["Secret.v1"][
                    f"kubeflow/{resource['metadata']['name']}"] = copy.deepcopy(
                        resource)
            if resource["kind"] not in receiver_kinds:
                continue
            group = f"{resource['kind']}.{resource['apiVersion']}"
            key = f"{NAMESPACE}/ml-pipeline-ui-artifact"
            updated = metacontroller_apply_update(observed[group][key],
                                                  resource)
            # getChildren omits an attachment as soon as its Namespace owner is lost.
            assert updated["metadata"].get("ownerReferences") == [owner]
            observed = observe_receiver_resources({"attachments": [updated]},
                                                  observed)

    unavailable = copy.deepcopy(related)
    del unavailable["ConfigMap.v1"]["kubeflow/storage-bindings"]
    pending = post_external_sync(server, observed, unavailable)
    assert storage_record(pending)["state"] == "pending"
    apply_receiver_update(pending)

    # Applying the restored desired state must retain ownership, not merely
    # publish one ready record before the next reconciliation loses the objects.
    restored = post_external_sync(server, observed, related)
    apply_receiver_update(restored)
    for _ in range(2):
        ready = post_external_sync(server, observed, related)
        assert storage_record(ready)["state"] == "ready"
        apply_receiver_update(ready)
    assert iam.mock_calls == []
    assert s3.mock_calls == []


@pytest.mark.parametrize("invalidation", ["namespace-uid", "missing-source"])
def test_external_restoration_waits_for_annotation_generation_acknowledgement(
        storage_controller, invalidation):
    server, observed, iam, s3 = storage_controller
    related = adoption_related()
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"

    key = f"{NAMESPACE}/ml-pipeline-ui-artifact"
    deployment = observed["Deployment.apps/v1"][key]
    deployment["metadata"]["ownerReferences"] = [{
        "apiVersion": "v1",
        "kind": "Namespace",
        "name": NAMESPACE,
        "uid": PARENT["metadata"]["uid"],
        "controller": True,
        "blockOwnerDeletion": True,
    }]
    desired = next(
        resource for resource in post_external_sync(server, observed, related)
        ["attachments"] if resource["kind"] == "Deployment")
    deployment["metadata"]["annotations"] = {
        sync.LAST_APPLIED_ANNOTATION: json.dumps(desired, sort_keys=True)
    }
    # API-defaulted replicas and omitted empty EnvVar values are absent from
    # last-applied until pending preservation observes the stored Deployment.
    deployment["spec"]["replicas"] = 1
    original_template = copy.deepcopy(deployment["spec"]["template"])

    def apply_deployment_update(result):
        current = observed["Deployment.apps/v1"][key]
        desired = next(resource for resource in result["attachments"]
                       if resource["kind"] == "Deployment")
        updated = metacontroller_apply_update(current, desired)
        updated["spec"].setdefault("replicas", 1)
        for container in updated["spec"]["template"]["spec"]["containers"]:
            for variable in container.get("env", []):
                if variable.get("value") == "":
                    variable.pop("value")
        # Kubernetes v1.36.1 deploymentStrategy.PrepareForUpdate increments
        # generation for spec OR annotation changes, including last-applied.
        # The Deployment controller acknowledges that generation asynchronously.
        changed = (
            updated["spec"] != current["spec"] or
            updated["metadata"].get("annotations")
            != current["metadata"].get("annotations"))
        assert changed
        updated["metadata"][
            "generation"] = current["metadata"]["generation"] + 1
        updated["status"] = copy.deepcopy(current["status"])
        assert updated["spec"]["template"] == original_template
        assert updated["metadata"]["ownerReferences"] == current["metadata"][
            "ownerReferences"]
        observed["Deployment.apps/v1"][key] = updated
        return updated

    unavailable = copy.deepcopy(related)
    source = unavailable["ConfigMap.v1"]["kubeflow/storage-bindings"]
    if invalidation == "namespace-uid":
        bindings = json.loads(source["data"]["bindings.json"])
        bindings[NAMESPACE]["namespaceUID"] = "different-namespace-uid"
        source["data"]["bindings.json"] = json.dumps(bindings)
    else:
        del unavailable["ConfigMap.v1"]["kubeflow/storage-bindings"]
    pending = post_external_sync(server, observed, unavailable)
    assert storage_record(pending)["state"] == "pending"
    deployment = apply_deployment_update(pending)
    deployment["status"]["observedGeneration"] = deployment["metadata"][
        "generation"]

    # Restoration can observe a ready receiver before applying its own
    # annotation-only update. A single ready observation is not a settling fence.
    restored = post_external_sync(server, observed, related)
    assert storage_record(restored)["state"] == "ready"
    deployment = apply_deployment_update(restored)
    pending = post_external_sync(server, observed, related)
    assert storage_record(pending)["state"] == "pending"
    assert pending["resyncAfterSeconds"] == 10

    deployment["status"]["observedGeneration"] = deployment["metadata"][
        "generation"]
    for _ in range(2):
        ready = post_external_sync(server, observed, related)
        assert storage_record(ready)["state"] == "ready"
        desired = next(resource for resource in ready["attachments"]
                       if resource["kind"] == "Deployment")
        assert json.loads(deployment["metadata"]["annotations"][
            sync.LAST_APPLIED_ANNOTATION]) == desired
    assert iam.mock_calls == []
    assert s3.mock_calls == []


@pytest.mark.parametrize("existing,new,created,eligible", [
    (True, True, None, True),
    (False, False, None, False),
    (True, False, "2026-10-19T23:59:59Z", True),
    (True, False, "2026-10-20T00:00:00Z", True),
    (True, False, "2026-10-20T00:00:01Z", False),
    (False, True, "2026-10-20T00:00:00Z", False),
    (False, True, "2026-10-20T00:00:01Z", True),
])
def test_adoption_policy_enrollment_uses_explicit_cutoff(
        existing, new, created, eligible):
    policy = adoption_policy()
    policy.update(adoptExistingNamespaces=existing, enrollNewNamespaces=new)
    if existing != new:
        policy["enrollmentCutoff"] = "2026-10-20T00:00:00Z"
    parent = copy.deepcopy(PARENT)
    if created:
        parent["metadata"]["creationTimestamp"] = created
    if eligible:
        approval, present, _ = resolve_adoption(
            adoption_related(policy), parent)
        assert present and approval == external_approval()
    else:
        with pytest.raises(ValueError, match="excludes"):
            resolve_adoption(adoption_related(policy), parent)


@pytest.mark.parametrize("change", [
    {
        "unexpectedSecuritySetting": True
    },
    {
        "adoptExistingNamespaces": "false"
    },
    {
        "enrollNewNamespaces": 0
    },
    {
        "adoptExistingNamespaces": False
    },
    {
        "namespaceSelector": {
            "matchExpressions": []
        }
    },
    {
        "namespaceSelector": {
            "matchLabels": {},
            "matchExpressions": []
        }
    },
    {
        "namespaceSelector": {
            "matchLabels": {
                "team": True
            }
        }
    },
    {
        "bindingsSource": {
            "namespace": NAMESPACE,
            "configMap": "storage-bindings",
            "key": "bindings.json"
        }
    },
    {
        "receiverTemplateRef": {
            "namespace": "kubeflow",
            "configMap": "storage-template",
            "key": "template.json",
            "allowTenantOverrides": True
        }
    },
    {
        "allowedStorage": {
            **external_approval()["storage"], "allowAnyBucket": True
        }
    },
    {
        "enrollmentCutoff": "2026-10-20T00:00:00"
    },
    {
        "enrollmentCutoff": "2026-10-20T00:00:00+01:00"
    },
    {
        "enrollmentCutoff": "2026-02-30T00:00:00Z"
    },
])
def test_adoption_policy_rejects_unknown_or_malformed_settings(change):
    policy = adoption_policy()
    policy.update(change)
    with pytest.raises(ValueError):
        resolve_adoption(adoption_related(policy))


@pytest.mark.parametrize("field", ["version", "policies", "additional"])
def test_adoption_policy_rejects_unsupported_document(field):
    related = adoption_related()
    document = {"version": 1, "policies": [adoption_policy()]}
    document[field] = True
    related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
        "adoption-policies.json"] = json.dumps(document)
    with pytest.raises(ValueError):
        resolve_adoption(related)


@pytest.mark.parametrize("explicit,duplicate_name,disabled", [
    (True, False, False),
    (False, True, False),
    (False, False, False),
    (False, False, True),
])
def test_adoption_policy_rejects_ambiguous_approvals(explicit, duplicate_name,
                                                     disabled):
    related = adoption_related()
    data = related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"][
        "data"]
    if explicit:
        data["approvals.json"] = json.dumps({NAMESPACE: external_approval()})
    else:
        policy = adoption_policy()
        if not duplicate_name:
            policy["name"] = "other-storage"
        if disabled:
            policy.update(
                adoptExistingNamespaces=False, enrollNewNamespaces=False)
        data["adoption-policies.json"] = json.dumps({
            "version": 1,
            "policies": [adoption_policy(), policy]
        })
    with pytest.raises(ValueError):
        resolve_adoption(related)


@pytest.mark.parametrize("source_name,key",
                         [("storage-bindings", "bindings.json"),
                          ("storage-template", "template.json")])
@pytest.mark.parametrize("change", [
    "missing", "unlabeled", "wrong-name", "tenant-namespace", "missing-key",
    "malformed-json", "duplicate-json"
])
def test_adoption_policy_sources_require_exact_protected_configmap(
        source_name, key, change):
    related = adoption_related()
    source = related["ConfigMap.v1"][f"kubeflow/{source_name}"]
    if change == "missing":
        del related["ConfigMap.v1"][f"kubeflow/{source_name}"]
    elif change == "unlabeled":
        source["metadata"]["labels"] = {}
    elif change == "wrong-name":
        source["metadata"]["name"] = "other-config"
    elif change == "tenant-namespace":
        source["metadata"]["namespace"] = NAMESPACE
    elif change == "missing-key":
        source["data"] = {}
    elif change == "malformed-json":
        source["data"][key] = "{"
    else:
        source["data"][key] = '{"duplicate":1,"duplicate":2}'
    with pytest.raises((ValueError, TypeError)):
        resolve_adoption(related)


@pytest.mark.parametrize("change", [
    "missing-binding", "stale-uid", "unknown-binding-field",
    "template-principal", "unknown-template-field"
])
def test_adoption_policy_identity_only_comes_from_current_protected_binding(
        change):
    related = adoption_related()
    bindings_data = related["ConfigMap.v1"]["kubeflow/storage-bindings"]["data"]
    template_data = related["ConfigMap.v1"]["kubeflow/storage-template"]["data"]
    bindings = json.loads(bindings_data["bindings.json"])
    template = json.loads(template_data["template.json"])
    if change == "missing-binding":
        bindings = {"another-namespace": bindings[NAMESPACE]}
    elif change == "stale-uid":
        bindings[NAMESPACE]["namespaceUID"] = "deleted-namespace-uid"
    elif change == "unknown-binding-field":
        bindings[NAMESPACE]["storage"] = external_approval()["storage"]
    elif change == "template-principal":
        template["identity"]["principal"] = "broker/other-team"
    else:
        template["allowTenantSubstitution"] = True
    bindings_data["bindings.json"] = json.dumps(bindings)
    template_data["template.json"] = json.dumps(template)
    # Even a valid-looking tenant-local source cannot replace the central source.
    forged = copy.deepcopy(
        adoption_related()["ConfigMap.v1"]["kubeflow/storage-bindings"])
    forged["metadata"]["namespace"] = NAMESPACE
    related["ConfigMap.v1"][f"{NAMESPACE}/storage-bindings"] = forged
    with pytest.raises(ValueError):
        resolve_adoption(related)


@pytest.mark.parametrize("change", [
    "missing-source", "invalid-template", "invalid-policy", "disabled",
    "selector-removed", "policy-removed"
])
def test_adoption_policy_loss_keeps_pending_tombstone_and_recovers(
        storage_controller, change):
    server, observed, iam, s3 = storage_controller
    related = adoption_related()
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"
    original = copy.deepcopy(related)
    config = related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"][
        "data"]
    policy = adoption_policy()
    if change == "missing-source":
        del related["ConfigMap.v1"]["kubeflow/storage-bindings"]
    elif change == "invalid-template":
        related["ConfigMap.v1"]["kubeflow/storage-template"]["data"][
            "template.json"] = "null"
    elif change == "invalid-policy":
        config[
            "adoption-policies.json"] = '{"version":1,"version":2,"policies":[]}'
    elif change == "disabled":
        policy.update(adoptExistingNamespaces=False, enrollNewNamespaces=False)
        config["adoption-policies.json"] = json.dumps({
            "version": 1,
            "policies": [policy]
        })
    elif change == "selector-removed":
        policy["namespaceSelector"]["matchLabels"]["team"] = "other"
        config["adoption-policies.json"] = json.dumps({
            "version": 1,
            "policies": [policy]
        })
    else:
        del config["adoption-policies.json"]
    pending = post_external_sync(server, observed, related)
    record = storage_record(pending)
    assert record["version"] == 2 and record["state"] == "pending"
    assert next(
        item for item in pending["attachments"] if item["kind"] == "Deployment"
    )["spec"] == observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"]
    # The sources may be recreated; exact protected names and current bindings determine trust.
    for source in original["ConfigMap.v1"].values():
        source["metadata"]["uid"] = "replacement-configmap-uid"
    recovered = post_external_sync(server, observed, original)
    assert storage_record(recovered)["state"] == "ready"
    assert iam.mock_calls == []
    assert s3.mock_calls == []


@pytest.mark.parametrize("disabled", [False, True])
def test_matching_unenrolled_policy_never_mints_stock_credentials(
        storage_controller, disabled):
    server, observed, iam, s3 = storage_controller
    policy = adoption_policy()
    if disabled:
        policy.update(adoptExistingNamespaces=False, enrollNewNamespaces=False)
    related = adoption_related(policy)
    del related["ConfigMap.v1"]["kubeflow/storage-bindings"]
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["version"] == 2
    assert storage_record(result)["state"] == "pending"
    assert iam.mock_calls == []
    assert s3.mock_calls == []


def test_nonmatching_policy_leaves_unadopted_namespace_on_existing_path():
    policy = adoption_policy()
    policy["namespaceSelector"]["matchLabels"]["team"] = "other"
    related = adoption_related(policy)
    del related["ConfigMap.v1"]["kubeflow/storage-bindings"]
    assert resolve_adoption(related) == (None, False, None)


def test_policy_cutoff_update_recomputes_enrollment_without_hidden_state():
    policy = adoption_policy()
    policy.update(
        adoptExistingNamespaces=True,
        enrollNewNamespaces=False,
        enrollmentCutoff="2026-10-20T00:00:00Z")
    parent = copy.deepcopy(PARENT)
    parent["metadata"]["creationTimestamp"] = "2026-10-20T00:00:00Z"
    assert resolve_adoption(adoption_related(policy), parent)[1]
    policy["enrollmentCutoff"] = "2026-10-19T00:00:00Z"
    with pytest.raises(ValueError, match="excludes"):
        resolve_adoption(adoption_related(policy), parent)


@pytest.mark.parametrize("malformed", [False, True])
def test_adoption_policy_diagnostics_do_not_log_configuration(
        storage_controller, capsys, malformed):
    server, observed, _, _ = storage_controller
    related = adoption_related()
    private_marker = "administrator-configuration-must-not-be-logged"
    if malformed:
        related["ConfigMap.v1"]["kubeflow/storage-bindings"]["data"][
            "bindings.json"] = '{"' + private_marker + '":INVALID}'
    else:
        del related["ConfigMap.v1"]["kubeflow/storage-bindings"]
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    output = capsys.readouterr().out
    assert private_marker not in output
    assert ("verify the installation administrator" if malformed else
            "source is missing or lacks its discovery label") in output


@pytest.mark.parametrize("mutation,stage,path", [
    ("env-value", "deployment-template",
     "Deployment.spec.template.spec.containers[1].env[0].value"),
    ("container-order", "credential-wiring",
     "Deployment.spec.template.spec.containers[0].name"),
    ("volume-order", "credential-wiring",
     "Deployment.spec.template.spec.volumes[0].name"),
    ("init-order", "credential-wiring",
     "Deployment.spec.template.spec.initContainers[0].name"),
    ("pod-identity", "pod-identity", "Deployment.spec.template.spec.hostPID"),
    ("service-type", "service-type", "Service.spec.type"),
    ("args-order", "credential-wiring",
     "Deployment.spec.template.spec.containers[1].args[1]"),
    ("rollout", "deployment-rollout", "Deployment.status.observedGeneration"),
    ("service", "service-spec", "Service.spec.ports[0].targetPort"),
    ("network-policy", "network-policy", "NetworkPolicy.spec.ingress"),
    ("deleting", "resource-deleting", "Deployment.metadata.deletionTimestamp"),
])
def test_external_readiness_diagnostics_report_stage_and_paths_without_values(
        storage_controller, capsys, mutation, stage, path):
    server, observed, _, _ = storage_controller
    approval = external_approval()
    broker = approval["podTemplate"]["spec"]["containers"][1]
    broker["env"] = [{"name": "BROKER_ROLE", "value": "private-approved-role"}]
    broker["args"] = [
        "--role", "private-approved-role", "--fallback", "private-other-role"
    ]
    if mutation == "volume-order":
        approval["podTemplate"]["spec"]["volumes"].append({
            "name": "other-credentials",
            "emptyDir": {}
        })
    if mutation == "init-order":
        approval["podTemplate"]["spec"]["initContainers"] = [{
            "name": "first",
            "image": "broker",
            "env": []
        }, {
            "name": "second",
            "image": "broker",
            "env": []
        }]
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    assert storage_record(result)["state"] == "ready"
    capsys.readouterr()
    deployment = observed["Deployment.apps/v1"][
        f"{NAMESPACE}/ml-pipeline-ui-artifact"]
    spec = deployment["spec"]["template"]["spec"]
    if mutation == "env-value":
        spec["containers"][1]["env"][0]["value"] = "private-unapproved-role"
    elif mutation == "container-order":
        spec["containers"].reverse()
    elif mutation == "volume-order":
        spec["volumes"].reverse()
    elif mutation == "init-order":
        spec["initContainers"].reverse()
    elif mutation == "pod-identity":
        spec["hostPID"] = True
    elif mutation == "service-type":
        observed["Service.v1"][f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"][
            "type"] = "ExternalName"
    elif mutation == "args-order":
        spec["containers"][1]["args"] = [
            "--role", "private-other-role", "--fallback",
            "private-approved-role"
        ]
    elif mutation == "rollout":
        deployment["status"]["observedGeneration"] = 0
    elif mutation == "service":
        observed["Service.v1"][f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"][
            "ports"][0]["targetPort"] = 9999
    elif mutation == "network-policy":
        observed["NetworkPolicy.networking.k8s.io/v1"][
            f"{NAMESPACE}/ml-pipeline-ui-artifact"]["spec"].pop("ingress")
    else:
        deployment["metadata"]["deletionTimestamp"] = "2026-10-20T00:00:00Z"
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["state"] == "pending"
    output = capsys.readouterr().out
    prefix = f"External artifact storage readiness for {NAMESPACE} is pending: "
    line = next(line for line in output.splitlines() if line.startswith(prefix))
    detail = json.loads(line[len(prefix):])
    assert detail["stage"] == stage
    assert path in detail["paths"]
    assert all(len(item) <= 192 for item in detail["paths"])
    assert len(detail["paths"]) <= 5
    assert "private-" not in output
    assert "9999" not in output


def test_readiness_paths_redact_arbitrary_keys_and_bound_differences():
    actual = {
        "metadata": {
            "annotations": {
                "private-secret-key": "private-secret-value"
            }
        }
    }
    desired = {
        "metadata": {
            "annotations": {
                "private-secret-key": "private-other-value"
            }
        }
    }
    paths = sync.readiness_mismatch_paths(actual, desired,
                                          "Deployment.spec.template")
    assert paths == ["Deployment.spec.template.metadata.annotations[key]"]
    assert "private-" not in json.dumps(paths)
    paths = sync.readiness_mismatch_paths([], list(
        range(1000)), "Deployment.spec.template.spec.containers")
    assert len(paths) == 5
    assert all(len(path) <= 192 for path in paths)
    assert sync.readiness_mismatch_paths(
        {"private-secret-key": "private-secret-value"}, {},
        "Deployment.spec") == ["Deployment.spec[key]"]


def observe_all_resources(result, attachments):
    observed = observe_receiver_resources(result, attachments)
    for resource in result["attachments"]:
        if resource["kind"] not in ["Deployment", "Service", "NetworkPolicy"]:
            key = f"{resource['kind']}.{resource['apiVersion']}"
            name = f"{resource['metadata']['namespace']}/{resource['metadata']['name']}"
            observed.setdefault(key, {})[name] = copy.deepcopy(resource)
    return observed


@pytest.mark.parametrize("field", ["approvals.json", "adoption-policies.json"])
@pytest.mark.parametrize("recovery", ["corrected", "deleted"])
@pytest.mark.parametrize("initial", ["unreconciled", "managed"])
def test_unadopted_config_error_recovers_without_deleting_record(
        storage_controller, field, recovery, initial):
    server, observed, _, _ = storage_controller
    if initial == "managed":
        result, observed = ready_storage(storage_controller)
        observed = observe_all_resources(result, observed)
    related = adoption_related()
    policy = adoption_policy()
    policy["namespaceSelector"]["matchLabels"]["team"] = "other"
    config = related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]
    config["data"]["adoption-policies.json"] = json.dumps({
        "version": 1,
        "policies": [policy]
    })
    corrected = copy.deepcopy(related)
    config["data"][field] = "{malformed"
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        assert storage_record(result)["state"] == "pending"
        assert result["status"] == {"kubeflow-pipelines-ready": "False"}
        observed = observe_all_resources(result, observed)
    if recovery == "deleted":
        corrected = {}
    result = post_external_sync(server, observed, corrected)
    assert storage_record(result)["version"] == 1
    for _ in range(3):
        observed = observe_all_resources(result, observed)
        result = post_external_sync(server, observed, corrected)
    assert storage_record(result)["state"] == "ready"


@pytest.mark.parametrize("change",
                         ["payload", "source", "cutoff", "duplicate-name"])
def test_nonmatching_invalid_policy_does_not_disable_managed_namespace(
        storage_controller, change):
    server, _, _, _ = storage_controller
    ready, observed = ready_storage(storage_controller)
    observed = observe_all_resources(ready, observed)
    policy = adoption_policy()
    policy["namespaceSelector"]["matchLabels"]["team"] = "other"
    if change == "payload":
        policy["allowedStorage"] = None
    elif change == "source":
        policy["bindingsSource"] = {"namespace": NAMESPACE}
    elif change == "cutoff":
        policy["enrollmentCutoff"] = "invalid"
    policies = [policy, copy.deepcopy(policy)
               ] if change == "duplicate-name" else [policy]
    related = adoption_related()
    related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
        "adoption-policies.json"] = json.dumps({
            "version": 1,
            "policies": policies
        })
    result = post_external_sync(server, observed, related)
    assert storage_record(result) == storage_record(ready)
    assert "resyncAfterSeconds" not in result


@pytest.mark.parametrize("invalid_approval", [None, {"storage": None}, []])
def test_invalid_other_namespace_approval_preserves_managed_readiness(
        storage_controller, invalid_approval):
    server, _, _, _ = storage_controller
    ready, observed = ready_storage(storage_controller)
    observed = observe_all_resources(ready, observed)
    related = external_related(None)
    related["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
        "approvals.json"] = json.dumps({"another-namespace": invalid_approval})
    before = copy.deepcopy(observed)
    for _ in range(2):
        result = post_external_sync(server, observed, related)
        assert storage_record(result) == storage_record(ready)
        assert storage_record(result)["state"] == "ready"
        assert "resyncAfterSeconds" not in result
    assert observed == before


@pytest.mark.parametrize("prior", [
    "approved-before-rollout", "approved-ready", "legacy-pending",
    "malformed-record", "unknown-version-marker"
])
def test_external_revocation_without_receiver_never_returns_to_managed_storage(
        storage_controller, prior):
    server, observed, iam, s3 = storage_controller
    result = post_external_sync(server, observed,
                                external_related(external_approval()))
    if prior == "approved-ready":
        result, observed, _ = external_ready(storage_controller)
    if prior == "legacy-pending":
        record = storage_record(result)
        record = {
            key: record[key]
            for key in ["version", "namespace", "namespaceUID", "state"]
        }
        record["state"] = "pending"
        secret = next(item for item in result["attachments"] if item["metadata"]
                      ["name"] == sync.artifact_storage_record_name(NAMESPACE))
        secret["data"]["authorization.json"] = base64.b64encode(
            json.dumps(record).encode()).decode()
    if prior == "unknown-version-marker":
        secret = next(item for item in result["attachments"] if item["metadata"]
                      ["name"] == sync.artifact_storage_record_name(NAMESPACE))
        record = {
            "version": 3,
            "namespace": NAMESPACE,
            "namespaceUID": "namespace-uid",
            "state": "pending"
        }
        secret["data"]["authorization.json"] = base64.b64encode(
            json.dumps(record).encode()).decode()
        secret["metadata"]["annotations"] = {
            sync.EXTERNAL_STORAGE_PROVISIONAL: "true"
        }
    if prior == "malformed-record":
        secret = next(item for item in result["attachments"] if item["metadata"]
                      ["name"] == sync.artifact_storage_record_name(NAMESPACE))
        secret["data"]["authorization.json"] = base64.b64encode(
            b"{malformed").decode()
    observed = observe_all_resources(result, observed)
    observed["Deployment.apps/v1"] = {}
    malformed = external_related(None)
    malformed["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
        "approvals.json"] = "{malformed"
    for related in [{}, malformed, {}, {}]:
        result = post_external_sync(server, observed, related)
        assert storage_record(result)["version"] == 2
        assert storage_record(result)["state"] == "pending"
        assert result["status"] == {"kubeflow-pipelines-ready": "False"}
        observed = observe_all_resources(result, observed)
    assert iam.mock_calls == s3.mock_calls == []


def test_provisional_record_becomes_sticky_as_soon_as_approval_validates(
        storage_controller):
    server, observed, iam, s3 = storage_controller
    malformed = external_related(None)
    malformed["ConfigMap.v1"]["kubeflow/kfp-artifact-storage-config"]["data"][
        "approvals.json"] = "{malformed"
    result = post_external_sync(server, observed, malformed)
    observed = observe_all_resources(result, observed)
    result = post_external_sync(server, observed,
                                external_related(external_approval()))
    assert storage_record(result)["state"] == "pending"
    observed = observe_all_resources(result, observed)
    observed["Deployment.apps/v1"] = {}
    result = post_external_sync(server, observed, {})
    assert storage_record(result)["version"] == 2
    assert iam.mock_calls == s3.mock_calls == []


def test_external_profile_status_requires_observed_receiver(storage_controller):
    server, observed, _, _ = storage_controller
    related = external_related(external_approval())
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_all_resources(result, observed)
    assert result["status"] == {"kubeflow-pipelines-ready": "True"}
    observed["Deployment.apps/v1"] = {}
    result = post_external_sync(server, observed, related)
    assert result["status"] == {"kubeflow-pipelines-ready": "False"}


@pytest.mark.parametrize("source", ["attached", "related"])
def test_external_receiver_overrides_provisional_record_marker(
        storage_controller, source):
    server, _, iam, s3 = storage_controller
    result, observed, _ = external_ready(storage_controller)
    secret = next(item for item in result["attachments"] if item["metadata"]
                  ["name"] == sync.artifact_storage_record_name(NAMESPACE))
    record = {
        "version": 2,
        "namespace": NAMESPACE,
        "namespaceUID": "namespace-uid",
        "state": "pending"
    }
    secret["data"]["authorization.json"] = base64.b64encode(
        json.dumps(record).encode()).decode()
    secret["metadata"]["annotations"] = {
        sync.EXTERNAL_STORAGE_PROVISIONAL: "true"
    }
    observed = observe_all_resources(result, observed)
    related = {}
    if source == "related":
        related["Deployment.apps/v1"] = observed.pop("Deployment.apps/v1")
    result = post_external_sync(server, observed, related)
    assert storage_record(result)["version"] == 2
    returned = next(item for item in result["attachments"] if item["metadata"]
                    ["name"] == sync.artifact_storage_record_name(NAMESPACE))
    assert sync.EXTERNAL_STORAGE_PROVISIONAL not in returned["metadata"].get(
        "annotations", {})
    assert iam.mock_calls == s3.mock_calls == []


def test_managed_recovery_preserves_all_attachment_ownership(
        storage_controller):
    server, observed, _, _ = storage_controller
    owners = [{
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'name': NAMESPACE,
        'uid': PARENT['metadata']['uid'],
        'controller': True,
    }]

    def apply(result):
        # Exercise Metacontroller creation bookkeeping and actual three-way
        # updates, rather than replacing observed objects with desired ones.
        normalized = observe_receiver_resources(result, observed)
        for desired in result['attachments']:
            group = f"{desired['kind']}.{desired['apiVersion']}"
            key = f"{desired['metadata']['namespace']}/{desired['metadata']['name']}"
            previous = observed.get(group, {}).get(key, {})
            if sync.LAST_APPLIED_ANNOTATION in previous.get('metadata', {}).get(
                    'annotations', {}):
                stored = metacontroller_apply_update(previous, desired)
            else:
                stored = copy.deepcopy(desired)
                stored['metadata'].setdefault(
                    'annotations',
                    {})[sync.LAST_APPLIED_ANNOTATION] = json.dumps(desired)
                stored['metadata']['ownerReferences'] = copy.deepcopy(owners)
                stored['metadata']['uid'] = 'uid-' + desired['metadata']['name']
            if desired['kind'] == 'Deployment':
                stored['metadata']['generation'] = 2
                stored['status'] = normalized[group][key]['status']
            observed.setdefault(group, {})[key] = stored

    for _ in range(3):
        result = post_external_sync(server, observed, {})
        apply(result)
    assert storage_record(result)['state'] == 'ready'
    identities = {
        (group, key): resource['metadata']['uid']
        for group, resources in observed.items()
        for key, resource in resources.items()
    }
    malformed = external_related(None)
    malformed['ConfigMap.v1']['kubeflow/kfp-artifact-storage-config']['data'][
        'approvals.json'] = '{invalid'
    for _ in range(2):
        pending = post_external_sync(server, observed, malformed)
        assert storage_record(pending)['state'] == 'pending'
        apply(pending)
        for _ in range(3):
            restored = post_external_sync(server, observed, {})
            apply(restored)
            for (group, key), uid in identities.items():
                resource = observed[group][key]
                assert resource['metadata']['uid'] == uid
                assert resource['metadata']['ownerReferences'] == owners, (
                    group, key)
        assert storage_record(restored)['version'] == 1
        assert storage_record(restored)['state'] == 'ready'


@pytest.mark.parametrize('change', [
    'orphan',
    'foreign-uid',
    'foreign-name',
    'foreign-kind',
    'foreign-api',
    'extra-owner',
    'malformed-owner',
    'resource-name',
    'resource-namespace',
    'resource-kind',
    'resource-api',
    'missing-parent-uid',
])
def test_managed_ownership_preservation_never_adds_unverified_owners(change):
    desired = {
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'namespace': NAMESPACE,
            'name': 'owned-resource'
        },
    }
    owner = {
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'name': NAMESPACE,
        'uid': PARENT['metadata']['uid'],
        'controller': True
    }
    actual = copy.deepcopy(desired)
    actual['metadata']['ownerReferences'] = [owner]
    parent = copy.deepcopy(PARENT)
    if change == 'orphan':
        actual['metadata'].pop('ownerReferences')
    elif change.startswith('foreign-'):
        field = {
            'uid': 'uid',
            'name': 'name',
            'kind': 'kind',
            'api': 'apiVersion'
        }[change.removeprefix('foreign-')]
        owner[field] = 'foreign'
    elif change == 'extra-owner':
        actual['metadata']['ownerReferences'].append({
            'kind': 'Other',
            'uid': 'other'
        })
    elif change == 'malformed-owner':
        actual['metadata']['ownerReferences'] = [None]
    elif change in ['resource-name', 'resource-namespace']:
        actual['metadata'][change.removeprefix('resource-')] = 'other'
    elif change == 'resource-kind':
        actual['kind'] = 'ConfigMap'
    elif change == 'resource-api':
        actual['apiVersion'] = 'other/v1'
    else:
        parent['metadata'].pop('uid')
    before = copy.deepcopy(actual)
    attachments = {'Secret.v1': {f'{NAMESPACE}/owned-resource': actual}}
    returned = sync.preserve_namespace_ownership(parent, attachments, [desired])
    assert 'ownerReferences' not in returned[0]['metadata']
    assert actual == before


def test_managed_ownership_preservation_copies_only_verified_attachment_owner():
    desired = {
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'namespace': NAMESPACE,
            'name': 'owned-resource'
        }
    }
    actual = copy.deepcopy(desired)
    owners = [{
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'name': NAMESPACE,
        'uid': PARENT['metadata']['uid'],
        'controller': True
    }]
    actual['metadata']['ownerReferences'] = copy.deepcopy(owners)
    attachments = {'Secret.v1': {f'{NAMESPACE}/owned-resource': actual}}
    returned = sync.preserve_namespace_ownership(PARENT, attachments, [desired])
    assert returned[0]['metadata']['ownerReferences'] == owners
    returned[0]['metadata']['ownerReferences'][0]['uid'] = 'changed-copy'
    assert actual['metadata']['ownerReferences'] == owners


def test_managed_ownership_preservation_excludes_central_record_in_enabled_installation_namespace(
):
    parent = copy.deepcopy(PARENT)
    parent['metadata']['name'] = 'kubeflow'
    record = {
        'apiVersion': 'v1',
        'kind': 'Secret',
        'metadata': {
            'namespace': 'kubeflow',
            'name': sync.artifact_storage_record_name('kubeflow'),
        },
    }
    observed = copy.deepcopy(record)
    observed['metadata']['ownerReferences'] = [{
        'apiVersion': 'v1',
        'kind': 'Namespace',
        'name': 'kubeflow',
        'uid': parent['metadata']['uid'],
        'controller': True,
    }]
    attachments = {
        'Secret.v1': {
            'kubeflow/' + record['metadata']['name']: observed
        }
    }
    returned = sync.preserve_namespace_ownership(parent, attachments, [record])
    assert 'ownerReferences' not in returned[0]['metadata']


@pytest.mark.parametrize('error_code',
                         ['NotImplemented', 'MethodNotAllowed', 'AccessDenied'])
@pytest.mark.parametrize('previously_ready', [False, True])
def test_existing_credentials_without_verifiable_iam_remain_pending(
        storage_controller, capsys, error_code, previously_ready):
    server, observed, iam, s3 = storage_controller
    if previously_ready:
        ready, observed = ready_storage(storage_controller)
        assert storage_record(ready)['state'] == 'ready'
        observed = observe_all_resources(ready, observed)
    key = f'{NAMESPACE}/mlpipeline-minio-artifact'
    original = copy.deepcopy(observed['Secret.v1'][key])
    iam.reset_mock()
    s3.reset_mock()
    # S3 data-plane credentials can work even when the store has no compatible
    # IAM endpoint. Neither that fact nor an existing Secret proves isolation.
    iam.list_access_keys.side_effect = ClientError(
        {
            'Error': {
                'Code': error_code,
                'Message': 'credential-bearing error details'
            },
        }, 'ListAccessKeys')
    for _ in range(3):
        result = post_sync(server, PARENT, observed)
        record = storage_record(result)
        assert record['version'] == 1
        assert record['state'] == 'pending'
        assert result['resyncAfterSeconds'] == 10
        retained = next(
            resource for resource in result['attachments']
            if resource['kind'] == 'Secret' and
            resource['metadata']['name'] == 'mlpipeline-minio-artifact')
        assert retained['data'] == original['data']
        assert retained.get('type') == original.get('type')
        observed = observe_all_resources(result, observed)
    assert iam.list_access_keys.call_count >= 3
    iam.create_user.assert_not_called()
    iam.create_access_key.assert_not_called()
    iam.put_user_policy.assert_not_called()
    assert s3.mock_calls == []
    assert 'credential-bearing error details' not in capsys.readouterr().out


@pytest.mark.parametrize('bucket', ['team.artifacts.example', 'models.2026'])
def test_external_storage_preserves_approved_dotted_bucket(
        storage_controller, bucket):
    server, observed, iam, s3 = storage_controller
    approval = external_approval()
    approval['storage']['bucket'] = bucket
    assert sync.external_approval_valid(approval, PARENT['metadata']['uid'])
    related = external_related(approval)
    for _ in range(3):
        result = post_external_sync(server, observed, related)
        observed = observe_receiver_resources(result, observed)
    record = storage_record(result)
    assert record['state'] == 'ready'
    assert record['storage']['bucket'] == bucket
    receiver = next(resource for resource in result['attachments']
                    if resource['kind'] == 'Deployment')
    environment = receiver['spec']['template']['spec']['containers'][0]['env']
    binding = json.loads(
        next(item['value']
             for item in environment
             if item['name'] == sync.EXTERNAL_STORAGE_ENV))
    assert binding['storage']['bucket'] == bucket
    assert iam.mock_calls == s3.mock_calls == []

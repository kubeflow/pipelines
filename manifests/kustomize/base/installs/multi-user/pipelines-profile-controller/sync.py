# Copyright 2020-2021 The Kubeflow Authors
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

import base64
import copy
import hashlib
from http.server import BaseHTTPRequestHandler
from http.server import ThreadingHTTPServer
import json
import os
from pathlib import Path
from urllib.parse import unquote

try:
    import botocore.session
except ModuleNotFoundError as error:
    if error.name != 'botocore':
        raise
    # AWS CLI 1.46+ bundles botocore and registers its import alias on startup.
    import awscli  # noqa: F401
    import botocore.session

S3_BUCKET_NAME = 'mlpipeline'
HEALTHZ_PATH = '/healthz'
# Argo mounts this Secret from the workflow's namespace into every agent Pod,
# so TLS installations need a copy of the kubeflow-namespace CA per profile.
DRIVER_AGENT_CA_SECRET_NAME = 'argo-workflows-agent-ca-certificates'
# Metacontroller records each returned attachment in this annotation and
# three-way merges against it on the next sync.
LAST_APPLIED_ANNOTATION = 'metacontroller.k8s.io/last-applied-configuration'
# Metadata worth echoing back for a Secret this controller does not rewrite.
# Server- and Metacontroller-owned fields such as resourceVersion, uid, and
# ownerReferences change on every write, so returning them would alter the
# last-applied annotation each cycle and reconcile InPlace Secrets forever.
PRESERVED_SECRET_METADATA = ('name', 'namespace', 'labels', 'annotations')

session = botocore.session.get_session()
# S3 client for lifecycle policy and bucket policy management
s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL",
                                 "http://seaweedfs.kubeflow:8333")
s3 = session.create_client(
    's3', region_name='foobar', endpoint_url=s3_endpoint_url)


def preserved_secret(observed):
    """Desired state for an observed Secret that only external controllers
    populate: its data and intentional metadata, without volatile fields."""
    secret = copy.deepcopy(observed)
    metadata = secret.get('metadata', {})
    secret['metadata'] = {
        key: metadata[key]
        for key in PRESERVED_SECRET_METADATA
        if key in metadata
    }
    annotations = secret['metadata'].get('annotations')
    if annotations is not None:
        annotations.pop(LAST_APPLIED_ANNOTATION, None)
        if not annotations:
            del secret['metadata']['annotations']
    return secret


def _normalize_domain(domain):
    return domain if domain.startswith('.') else '.' + domain


def artifact_storage_record_name(namespace):
    return f"kfp-artifact-storage-{namespace}"


def artifact_credential_binding(namespace, namespace_uid, endpoint, access_key,
                                secret_key):
    values = [namespace, namespace_uid, endpoint, access_key, secret_key]
    return hashlib.sha256("\0".join(values).encode("utf-8")).hexdigest()


def managed_object_policy(namespace):
    return {
        "Version":
            "2012-10-17",
        "Statement": [{
            "Effect":
                "Allow",
            "Action": ["s3:Put*", "s3:Get*", "s3:List*", "s3:DeleteObject"],
            "Resource": [
                f"arn:aws:s3:::{S3_BUCKET_NAME}/artifacts/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/private-artifacts/{namespace}/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/private/{namespace}/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/shared/*",
            ],
        }],
    }


def policy_document(value):
    return json.loads(unquote(value)) if isinstance(value, str) else value


def has_managed_object_policy(document, namespace):
    """Recognize the upstream grant without changing storage-admin
    additions."""
    expected = managed_object_policy(namespace)["Statement"][0]
    for statement in document.get("Statement", []):
        if (statement.get("Effect") == expected["Effect"] and
                not statement.get("Condition") and
                set(statement.get("Action", [])) == set(expected["Action"]) and
                set(statement.get("Resource", [])) == set(
                    expected["Resource"])):
            return True
    return False


def contains_managed_bucket_policy(document):
    statements = document.get("Statement", [])
    return all(
        statement in statements for statement in managed_bucket_list_policy(
            S3_BUCKET_NAME)["Statement"])


def receiver_rollout_ready(desired_resources, attachments, namespace):
    """Only observed current-template replicas activate the new request
    path."""
    desired = {
        resource["kind"]: resource
        for resource in desired_resources
        if resource["metadata"]["name"] == "ml-pipeline-ui-artifact"
    }
    for kind, version in [("Deployment", "apps/v1"), ("Service", "v1"),
                          ("NetworkPolicy", "networking.k8s.io/v1")]:
        actual = attachments.get(f"{kind}.{version}",
                                 {}).get(f"{namespace}/ml-pipeline-ui-artifact",
                                         {})
        if actual.get("metadata", {}).get("deletionTimestamp"):
            return False
        actual_spec = actual.get("spec", {})
        expected_spec = desired[kind]["spec"]
        if kind == "Deployment":
            actual_template = copy.deepcopy(actual_spec.get("template", {}))
            # Kubernetes omits empty literal EnvVar.value fields on readback.
            # Preserve nonempty values and valueFrom so changed settings fail.
            for container in actual_template.get("spec",
                                                 {}).get("containers", []):
                for variable in container.get("env", []):
                    if "valueFrom" not in variable:
                        variable.setdefault("value", "")
            # API defaulting adds fields, so compare only fields we generate.
            if not contains_desired(actual_template, expected_spec["template"]):
                return False
            generation = actual.get("metadata", {}).get("generation", 0)
            status = actual.get("status", {})
            replicas = actual_spec.get("replicas", 1)
            if (generation < 1 or replicas < 1 or
                    status.get("observedGeneration", 0) < generation or any(
                        status.get(field, 0) != replicas for field in [
                            "replicas", "updatedReplicas", "availableReplicas",
                            "readyReplicas"
                        ]) or status.get("unavailableReplicas", 0) != 0):
                return False
        elif kind == "NetworkPolicy" and actual_spec != expected_spec:
            return False
        elif kind == "Service" and (actual_spec.get(
                "type", "ClusterIP") != "ClusterIP" or
                                    actual_spec.get("externalName")):
            return False
        elif not contains_desired(actual_spec, expected_spec):
            return False
    return True


def contains_desired(actual, desired):
    if isinstance(desired, dict):
        return isinstance(actual, dict) and all(
            key in actual and contains_desired(actual[key], value)
            for key, value in desired.items())
    if isinstance(desired, list):
        return isinstance(actual, list) and len(actual) >= len(desired) and all(
            any(contains_desired(item, expected)
                for item in actual)
            for expected in desired)
    return actual == desired


def artifact_server_environment(namespace,
                                cluster_domain,
                                allowed_artifact_endpoints,
                                allowed_gcs_universe_domains="googleapis.com",
                                http_base_url="",
                                object_store_host="seaweedfs",
                                kfp_namespace="kubeflow",
                                namespace_uid=""):
    return [
        {
            "name": "KFP_ARTIFACT_NAMESPACE_UID",
            "value": namespace_uid
        },
        {
            "name": "MINIO_HOST",
            "value": object_store_host
        },
        {
            "name": "MINIO_NAMESPACE",
            "value": kfp_namespace
        },
        {
            "name": "MINIO_PORT",
            "value": "8333"
        },
        {
            "name": "MINIO_SSL",
            "value": "false"
        },
        {
            "name": "ARTIFACTS_REQUIRE_PROXY_NAMESPACE",
            "value": "false",
        },
        {
            "name": "MINIO_ACCESS_KEY",
            "valueFrom": {
                "secretKeyRef": {
                    "key": "accesskey",
                    "name": "mlpipeline-minio-artifact"
                }
            }
        },
        {
            "name": "MINIO_SECRET_KEY",
            "valueFrom": {
                "secretKeyRef": {
                    "key": "secretkey",
                    "name": "mlpipeline-minio-artifact"
                }
            }
        },
        {
            "name": "ML_PIPELINE_SERVICE_HOST",
            "value": f"ml-pipeline.kubeflow{_normalize_domain(cluster_domain)}"
        },
        {
            "name": "ML_PIPELINE_SERVICE_PORT",
            "value": "8888"
        },
        {
            "name": "FRONTEND_SERVER_NAMESPACE",
            "value": namespace,
        },
        {
            "name": "CLUSTER_DOMAIN",
            "value": cluster_domain,
        },
        {
            "name": "HTTP_BASE_URL",
            "value": http_base_url,
        },
        {
            "name": "ALLOWED_ARTIFACT_ENDPOINTS",
            "value": allowed_artifact_endpoints,
        },
        {
            "name": "ALLOWED_GCS_UNIVERSE_DOMAINS",
            "value": allowed_gcs_universe_domains,
        },
    ]


def managed_bucket_list_policy(bucket_name):
    bucket_arn = f"arn:aws:s3:::{bucket_name}"
    desired = {
        "Version":
            "2012-10-17",
        "Statement": [
            {
                "Sid": "AllowOwnDeepList",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringLike": {
                        "s3:prefix": [
                            "private-artifacts/${aws:username}/*",
                            "private/${aws:username}/*",
                            "artifacts/*",
                            "shared/*",
                        ]
                    }
                },
            },
            {
                "Sid": "AllowHeadBucket",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringEquals": {
                        "s3:RequestMethod": "HEAD"
                    }
                },
            },
            {
                "Sid": "AllowTopLevelDelimitedBrowse",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "Null": {
                        "s3:prefix": "true"
                    },
                    "StringEquals": {
                        "s3:delimiter": "/"
                    },
                },
            },
            {
                "Sid": "DenyOtherDeepList",
                "Effect": "Deny",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringLike": {
                        "s3:prefix": ["private-artifacts/*", "private/*"]
                    },
                    "StringNotLike": {
                        "s3:prefix": [
                            "private-artifacts/${aws:username}/*",
                            "private/${aws:username}/*",
                        ]
                    },
                },
            },
        ],
    }
    return desired


def create_iam_client():
    # To interact with SeaweedFS user management. Region does not matter.
    endpoint_url = os.environ.get("AWS_ENDPOINT_URL")
    if endpoint_url:
        return session.create_client(
            'iam', region_name='foobar', endpoint_url=endpoint_url)
    return session.create_client('iam', region_name='foobar')


iam = create_iam_client()


def main():
    settings = get_settings_from_env()
    server = server_factory(**settings)
    server.serve_forever()


def get_settings_from_env(controller_port=None,
                          frontend_image=None,
                          frontend_tag=None,
                          disable_istio_sidecar=None,
                          artifacts_proxy_enabled=None,
                          allowed_artifact_endpoints=None,
                          allowed_gcs_universe_domains=None,
                          artifact_retention_days=None,
                          cluster_domain=None,
                          object_store_host=None,
                          http_base_url=None,
                          driver_agent_ca_cert_path=None,
                          kfp_namespace=None):
    """Returns a dict of settings from environment variables relevant to the
    controller.

    Environment settings can be overridden by passing them here as arguments.

    Settings are pulled from the all-caps version of the setting name.  The
    following defaults are used if those environment variables are not set
    to enable backwards compatibility with previous versions of this script:
        frontend_image: ghcr.io/kubeflow/kfp-frontend
        frontend_tag: value of KFP_VERSION environment variable
        disable_istio_sidecar: Required (no default)
    """
    settings = dict()
    settings["controller_port"] = \
        controller_port or \
        os.environ.get("CONTROLLER_PORT", "8080")

    settings["frontend_image"] = \
        frontend_image or \
        os.environ.get("FRONTEND_IMAGE", "ghcr.io/kubeflow/kfp-frontend")

    settings["artifacts_proxy_enabled"] = \
        artifacts_proxy_enabled or \
        os.environ.get("ARTIFACTS_PROXY_ENABLED", "false")

    settings["http_base_url"] = \
        http_base_url if http_base_url is not None \
            else os.environ.get("HTTP_BASE_URL", "")

    settings["allowed_artifact_endpoints"] = \
        allowed_artifact_endpoints if allowed_artifact_endpoints is not None \
            else os.environ.get("ALLOWED_ARTIFACT_ENDPOINTS", "")
    settings["allowed_gcs_universe_domains"] = \
        allowed_gcs_universe_domains if allowed_gcs_universe_domains is not None \
            else os.environ.get("ALLOWED_GCS_UNIVERSE_DOMAINS", "googleapis.com")

    settings["artifact_retention_days"] = \
        artifact_retention_days or \
        os.environ.get("ARTIFACT_RETENTION_DAYS", -1)

    settings["cluster_domain"] = \
        cluster_domain or \
        os.environ.get("CLUSTER_DOMAIN", ".svc.cluster.local")

    settings["kfp_namespace"] = \
        kfp_namespace or os.environ.get("KFP_NAMESPACE", "kubeflow")

    settings["object_store_host"] = \
        object_store_host or \
        os.environ.get("OBJECT_STORE_HOST", "seaweedfs")

    settings["driver_agent_ca_cert_path"] = \
        driver_agent_ca_cert_path if driver_agent_ca_cert_path is not None \
            else os.environ.get("DRIVER_AGENT_CA_CERT_PATH", "")

    # Look for specific tags for each image first, falling back to
    # previously used KFP_VERSION environment variable for backwards
    # compatibility
    settings["frontend_tag"] = \
        frontend_tag or \
        os.environ.get("FRONTEND_TAG") or \
        os.environ["KFP_VERSION"]

    settings["disable_istio_sidecar"] = \
        disable_istio_sidecar if disable_istio_sidecar is not None \
            else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"

    return settings


def server_factory(frontend_image,
                   frontend_tag,
                   disable_istio_sidecar,
                   artifacts_proxy_enabled,
                   artifact_retention_days,
                   allowed_artifact_endpoints="",
                   cluster_domain=".svc.cluster.local",
                   object_store_host="seaweedfs",
                   allowed_gcs_universe_domains="googleapis.com",
                   url="",
                   controller_port=8080,
                   http_base_url="",
                   driver_agent_ca_cert_path="",
                   kfp_namespace="kubeflow"):
    """Returns a ThreadingHTTPServer populated with Handler with customized
    settings."""

    class Controller(BaseHTTPRequestHandler):

        def ensure_namespace_object_policy(self, namespace):
            policy_name = f"KubeflowProject{namespace}"
            try:
                iam.get_user_policy(UserName=namespace, PolicyName=policy_name)
                return
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error", {}).get("Code") != "NoSuchEntity":
                    raise
            # Establish the user and policy before minting credentials: a failed
            # IAM request can then retry without creating orphan access keys.
            try:
                iam.create_user(UserName=namespace)
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error",
                                   {}).get("Code") != "EntityAlreadyExists":
                    raise
            # Another provisioner or storage administrator may have supplied a
            # policy while the user was created. Never replace that policy.
            try:
                iam.get_user_policy(UserName=namespace, PolicyName=policy_name)
                return
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error", {}).get("Code") != "NoSuchEntity":
                    raise
            iam.put_user_policy(
                UserName=namespace,
                PolicyName=policy_name,
                PolicyDocument=json.dumps(managed_object_policy(namespace)))

        def artifact_storage_record(self, parent, attachments, resources):
            namespace = parent["metadata"]["name"]
            namespace_uid = parent["metadata"].get("uid", "")
            record = {
                "version": 1,
                "namespace": namespace,
                "namespaceUID": namespace_uid,
                "state": "unsupported",
                "bucket": S3_BUCKET_NAME,
            }
            secret = next(
                (resource for resource in resources
                 if resource["kind"] == "Secret" and
                 resource["metadata"]["name"] == "mlpipeline-minio-artifact"),
                {})
            try:
                data = secret.get("data", {})
                access_key = base64.b64decode(
                    data["accesskey"], validate=True).decode("utf-8")
                secret_key = base64.b64decode(
                    data["secretkey"], validate=True).decode("utf-8")
                if not access_key or not secret_key or not namespace_uid:
                    raise ValueError("Incomplete namespace storage credentials")
            except (KeyError, ValueError, UnicodeError):
                return self.artifact_storage_record_secret(namespace, record)

            record["credentialBinding"] = artifact_credential_binding(
                namespace, namespace_uid,
                f"http://{object_store_host}.{kfp_namespace}:8333", access_key,
                secret_key)
            # Environment Secret references refresh only when the Pod restarts.
            # A separate digest triggers rollout without exposing the request binding.
            receiver = next(resource for resource in resources
                            if resource["kind"] == "Deployment")
            receiver["spec"]["template"]["metadata"]["annotations"][
                "kubeflow-pipelines/artifact-storage-revision"] = hashlib.sha256(
                    ("receiver-rollout\0" + record["credentialBinding"]
                    ).encode("utf-8")).hexdigest()[:16]
            try:
                # IAM is the admin-controlled authority; a tenant Secret's name,
                # labels and readiness never establish its storage identity.
                keys = iam.list_access_keys(UserName=namespace)
                markers = set()
                while not any(
                        key.get("AccessKeyId") == access_key and
                        key.get("Status") == "Active"
                        for key in keys.get("AccessKeyMetadata", [])):
                    if not keys.get("IsTruncated"):
                        return self.artifact_storage_record_secret(
                            namespace, record)
                    marker = keys.get("Marker")
                    if not marker or marker in markers:
                        raise ValueError("Incomplete IAM access-key pagination")
                    markers.add(marker)
                    keys = iam.list_access_keys(
                        UserName=namespace, Marker=marker)
                policy = policy_document(
                    iam.get_user_policy(
                        UserName=namespace,
                        PolicyName=f"KubeflowProject{namespace}")
                    ["PolicyDocument"])
                if not has_managed_object_policy(policy, namespace):
                    return self.artifact_storage_record_secret(
                        namespace, record)
                # Preserve administrator changes. A missing/conflicting managed
                # list boundary is a rollout blocker, never an implicit rewrite.
                bucket_policy = policy_document(
                    s3.get_bucket_policy(Bucket=S3_BUCKET_NAME)["Policy"])
                record["state"] = "pending"
                if contains_managed_bucket_policy(
                        bucket_policy) and receiver_rollout_ready(
                            resources, attachments, namespace):
                    record["state"] = "ready"
            except Exception as exception:
                error_code = getattr(exception, "response",
                                     {}).get("Error", {}).get("Code")
                record[
                    "state"] = "unsupported" if error_code == "NoSuchEntity" else "pending"
                # Exceptions can carry request credentials; log only the class.
                print(
                    f"Artifact storage authorization for {namespace} is {record['state']}; "
                    f"verify namespace IAM and the bucket list policy ({type(exception).__name__})."
                )
            return self.artifact_storage_record_secret(namespace, record)

        def artifact_storage_record_secret(self, namespace, record):
            return {
                "apiVersion": "v1",
                "kind": "Secret",
                "metadata": {
                    "name": artifact_storage_record_name(namespace),
                    "namespace": kfp_namespace,
                },
                "type": "Opaque",
                "data": {
                    "authorization.json":
                        base64.b64encode(
                            json.dumps(record, sort_keys=True).encode("utf-8")
                        ).decode("ascii")
                },
            }

        def upsert_lifecycle_policy(self, bucket_name, artifact_retention_days):
            """Configures or deletes the lifecycle policy based on the
            artifact_retention_days string."""
            try:
                retention_days = int(artifact_retention_days)
            except ValueError:
                print(
                    f"ERROR: ARTIFACT_RETENTION_DAYS value '{artifact_retention_days}' is not a valid integer. Aborting policy update."
                )
                return

            # To disable lifecycle policy we need to delete it
            if retention_days <= 0:
                print(
                    f"ARTIFACT_RETENTION_DAYS is non-positive ({retention_days} days). Attempting to delete lifecycle policy."
                )
                try:
                    response = s3.get_bucket_lifecycle_configuration(
                        Bucket=bucket_name)
                    # Check if there are any enabled rules
                    has_enabled_rules = any(
                        rule.get('Status') == 'Enabled'
                        for rule in response.get('Rules', []))

                    if has_enabled_rules:
                        s3.delete_bucket_lifecycle(Bucket=bucket_name)
                        print("Successfully deleted lifecycle policy.")
                    else:
                        print("No enabled lifecycle rules found to delete.")
                except Exception:
                    print(f"Warning: No lifecycle policy exists")
                return

            # Create/update lifecycle policy
            life_cycle_policy = {
                "Rules": [{
                    "Status": "Enabled",
                    "Filter": {
                        "Prefix": "private-artifacts"
                    },
                    "Expiration": {
                        "Days": retention_days
                    },
                    "ID": "private-artifacts",
                },]
            }
            print('upsert_lifecycle_policy:', life_cycle_policy)

            try:
                api_response = s3.put_bucket_lifecycle_configuration(
                    Bucket=bucket_name,
                    LifecycleConfiguration=life_cycle_policy)
                print('Lifecycle policy configured successfully:', api_response)
            except Exception as exception:
                if hasattr(exception,
                           'response') and 'Error' in exception.response:
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception.response['Error']['Code']} - {exception}"
                    )
                else:
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception}"
                    )

        def executor_plugin_resources(self, namespace):
            print('Creating executor-plugin resources for namespace:',
                  namespace)
            # Argo Workflow Executor Plugin Necessary Resources
            agent_service_account_name = 'ml-pipeline-driver-agent-executor-plugin'
            agent_secret_name = f'{agent_service_account_name}.service-account-token'
            return [
                {
                    'apiVersion': 'v1',
                    'kind': 'ServiceAccount',
                    'metadata': {
                        'name': agent_service_account_name,
                        'namespace': namespace,
                        'labels': {
                            'application-crd-id': 'kubeflow-pipelines'
                        }
                    },
                    'secrets': [{
                        'name': agent_secret_name
                    }]
                },
                {
                    'apiVersion': 'rbac.authorization.k8s.io/v1',
                    'kind': 'RoleBinding',
                    'metadata': {
                        'name':
                            'ml-pipeline-driver-agent-executor-plugin-binding',
                        'namespace':
                            namespace,
                    },
                    'subjects': [{
                        'kind': 'ServiceAccount',
                        'name': agent_service_account_name,
                        'namespace': namespace,
                    }],
                    'roleRef': {
                        'kind':
                            'ClusterRole',
                        'name':
                            'ml-pipeline-driver-agent-executor-plugin-cluster-role',
                        'apiGroup':
                            'rbac.authorization.k8s.io',
                    }
                },
                {
                    'apiVersion': 'v1',
                    'kind': 'Secret',
                    'metadata': {
                        'name': agent_secret_name,
                        'namespace': namespace,
                        'annotations': {
                            'kubernetes.io/service-account.name':
                                agent_service_account_name,
                        },
                    },
                    'type': 'kubernetes.io/service-account-token',
                },
            ]

        def driver_agent_ca_secret(self, namespace):
            """Copy of the driver-agent CA for this namespace, or None when the
            controller is not configured with one (non-TLS installations)."""
            if not driver_agent_ca_cert_path:
                return None
            # Read on every sync rather than once at startup: the file comes
            # from an optional Secret volume, so it may appear after this
            # controller starts (cert-manager issuance) and changes on rotation.
            try:
                ca_cert = Path(driver_agent_ca_cert_path).read_bytes()
            except FileNotFoundError:
                return None
            if not ca_cert:
                return None
            return {
                "apiVersion": "v1",
                "kind": "Secret",
                "metadata": {
                    "name": DRIVER_AGENT_CA_SECRET_NAME,
                    "namespace": namespace,
                },
                "data": {
                    "ca.crt": base64.b64encode(ca_cert).decode("utf-8"),
                },
            }

        def ensure_bucket_list_policy(self, bucket_name):
            """Bucket-wide ListBucket isolation, keyed on ${aws:username}.

            Top-level folder browsing is allowed for everyone; listing
            deeper inside another namespace's private prefix is denied.
            HeadBucket (KServe storage-initializer) is allowed. Set as a
            bucket policy because the embedded user-policy engine is
            Allow-only and cannot express Deny; SeaweedFS evaluates the
            bucket policy before it.
            """
            desired = managed_bucket_list_policy(bucket_name)
            try:
                current = policy_document(
                    s3.get_bucket_policy(Bucket=bucket_name)["Policy"])
            except Exception as exception:
                code = getattr(exception, "response", {}).get("Error",
                                                              {}).get("Code")
                if code != "NoSuchBucketPolicy":
                    print(
                        f"Cannot inspect bucket list policy for {bucket_name}; "
                        "leaving existing policy unchanged.")
                    return
                current = {"Version": "2012-10-17", "Statement": []}
            statements = current.get("Statement", [])
            for required in desired["Statement"]:
                same_sid = [
                    statement for statement in statements
                    if statement.get("Sid") == required["Sid"]
                ]
                if same_sid and same_sid != [required]:
                    print(
                        f"Bucket list policy {bucket_name} has a conflicting managed "
                        "statement; preserve it and review the storage configuration."
                    )
                    return
            missing = [
                required for required in desired["Statement"]
                if required not in statements
            ]
            if not missing:
                return
            # Existing administrator statements remain intact.
            current["Statement"] = statements + missing
            try:
                s3.put_bucket_policy(
                    Bucket=bucket_name, Policy=json.dumps(current))
                print(f"Added managed ListBucket statements on {bucket_name}")
            except Exception:
                print(f"Cannot update bucket list policy for {bucket_name}; "
                      "storage authorization remains pending.")

        def sync(self, parent, attachments):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = parent.get("metadata", {}).get(
                "labels", {}).get("pipelines.kubeflow.org/enabled")

            if pipeline_enabled != "true":
                return {"status": {}, "attachments": []}

            driver_agent_ca_secret = self.driver_agent_ca_secret(namespace)

            # Compute status based on observed state.
            desired_status = {
                "kubeflow-pipelines-ready":
                    len([
                        key for key in attachments["Secret.v1"]
                        if key.startswith(f"{namespace}/")
                    ]) == (4 if driver_agent_ca_secret else 3) and
                    len(attachments["ConfigMap.v1"]) == 2 and
                    len(attachments["Deployment.apps/v1"]) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    len(attachments["Service.v1"]) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    len(attachments["ServiceAccount.v1"]) == 1 and
                    len(attachments["Role.rbac.authorization.k8s.io/v1"]) == 0
                    and
                    len(attachments["RoleBinding.rbac.authorization.k8s.io/v1"])
                    == 1 and len(
                        attachments.get("NetworkPolicy.networking.k8s.io/v1",
                                        {})) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    "True" or "False"
            }

            # Generate the desired attachment object(s).
            desired_resources = [
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "kfp-launcher",
                        "namespace": namespace,
                    },
                    "data": {
                        "defaultPipelineRoot":
                            f"minio://{S3_BUCKET_NAME}/private-artifacts/{namespace}/v2/artifacts",
                        "clusterDomain":
                            cluster_domain,
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "artifact-repositories",
                        "namespace": namespace,
                        "annotations": {
                            "workflows.argoproj.io/default-artifact-repository":
                                "default-namespaced"
                        }
                    },
                    "data": {
                        "default-namespaced":
                            json.dumps({
                                "archiveLogs": True,
                                "s3": {
                                    "endpoint":
                                        f"{object_store_host}.kubeflow{_normalize_domain(cluster_domain)}:9000",
                                    "bucket":
                                        S3_BUCKET_NAME,
                                    "keyFormat":
                                        f"private-artifacts/{namespace}/{{{{workflow.name}}}}/{{{{workflow.creationTimestamp.Y}}}}/{{{{workflow.creationTimestamp.m}}}}/{{{{workflow.creationTimestamp.d}}}}/{{{{pod.name}}}}",
                                    "insecure":
                                        True,
                                    "accessKeySecret": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "accesskey",
                                    },
                                    "secretKeySecret": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "secretkey",
                                    }
                                }
                            })
                    }
                },
            ]

            executor_plugin_resources = self.executor_plugin_resources(
                namespace)
            desired_resources.extend(executor_plugin_resources)
            if driver_agent_ca_secret:
                desired_resources.append(driver_agent_ca_secret)

            if artifacts_proxy_enabled.lower() == "true":
                desired_resources.extend([
                    {
                        "apiVersion": "networking.k8s.io/v1",
                        "kind": "NetworkPolicy",
                        "metadata": {
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                        },
                        "spec": {
                            "podSelector": {
                                "matchLabels": {
                                    "app": "ml-pipeline-ui-artifact"
                                }
                            },
                            "policyTypes": ["Ingress"],
                            # Both selectors must match: a tenant must not gain access by
                            # labeling its own pod as the shared UI. Other NetworkPolicies
                            # are additive and must not grant cross-tenant proxy access.
                            "ingress": [{
                                "from": [{
                                    "namespaceSelector": {
                                        "matchLabels": {
                                            "kubernetes.io/metadata.name":
                                                kfp_namespace
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
                        },
                    },
                    {
                        "apiVersion": "apps/v1",
                        "kind": "Deployment",
                        "metadata": {
                            "labels": {
                                "app": "ml-pipeline-ui-artifact"
                            },
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                        },
                        "spec": {
                            "selector": {
                                "matchLabels": {
                                    "app": "ml-pipeline-ui-artifact"
                                }
                            },
                            "template": {
                                "metadata": {
                                    "labels": {
                                        "app": "ml-pipeline-ui-artifact"
                                    },
                                    "annotations": {
                                        **({
                                            "sidecar.istio.io/inject": "false"
                                        } if disable_istio_sidecar else {}),
                                        "kubeflow-pipelines/image-spec-hash":
                                            hashlib.sha256(
                                                f"{frontend_image}:{frontend_tag}"
                                                .encode()).hexdigest()[:16],
                                    },
                                },
                                "spec": {
                                    "containers": [{
                                        "name":
                                            "ml-pipeline-ui-artifact",
                                        "image":
                                            f"{frontend_image}:{frontend_tag}",
                                        "imagePullPolicy":
                                            "IfNotPresent",
                                        "ports": [{
                                            "containerPort": 3000
                                        }],
                                        "readinessProbe": {
                                            "tcpSocket": {
                                                "port": 3000
                                            },
                                            "periodSeconds": 2,
                                            "timeoutSeconds": 1,
                                        },
                                        "env":
                                            artifact_server_environment(
                                                namespace,
                                                cluster_domain,
                                                allowed_artifact_endpoints,
                                                allowed_gcs_universe_domains,
                                                http_base_url,
                                                object_store_host,
                                                kfp_namespace,
                                                parent.get("metadata",
                                                           {}).get("uid", ""),
                                            ),
                                        "resources": {
                                            "requests": {
                                                "cpu": "10m",
                                                "memory": "70Mi"
                                            },
                                            "limits": {
                                                "cpu": "100m",
                                                "memory": "500Mi"
                                            },
                                        }
                                    }],
                                    "serviceAccountName": "default-editor"
                                }
                            }
                        }
                    },
                    {
                        "apiVersion": "v1",
                        "kind": "Service",
                        "metadata": {
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                            "labels": {
                                "app": "ml-pipeline-ui-artifact"
                            }
                        },
                        "spec": {
                            "ports": [{
                                "name":
                                    "http",  # name is required to let istio understand request protocol
                                "port": 80,
                                "protocol": "TCP",
                                "targetPort": 3000
                            }],
                            "selector": {
                                "app": "ml-pipeline-ui-artifact"
                            }
                        }
                    },
                ])
            desired_resources.extend([
                {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": "default-editor.service-account-token",
                        "namespace": namespace,
                        "annotations": {
                            "kubernetes.io/service-account.name":
                                "default-editor"
                        }
                    },
                    "type": "kubernetes.io/service-account-token"
                },
            ])

            print('Received request:\n', json.dumps(parent, sort_keys=True))
            print('Desired resources except secrets:\n',
                  json.dumps(desired_resources, sort_keys=True))

            # Moved after the print argument because this is sensitive data.

            # Check if secret is already there when the controller made the request. If yes, then
            # use it. Else create a new credentials on seaweedfs for the namespace.
            if s3_secret := attachments["Secret.v1"].get(
                    f"{namespace}/mlpipeline-minio-artifact"):
                desired_resources.append(s3_secret)
                print('Using existing secret')
            else:
                self.ensure_namespace_object_policy(namespace)
                print('Creating new access key.')
                s3_access_key = iam.create_access_key(UserName=namespace)

                self.ensure_bucket_list_policy(S3_BUCKET_NAME)

                self.upsert_lifecycle_policy(S3_BUCKET_NAME,
                                             artifact_retention_days)

                desired_resources.insert(
                    0, {
                        "apiVersion": "v1",
                        "kind": "Secret",
                        "metadata": {
                            "name": "mlpipeline-minio-artifact",
                            "namespace": namespace,
                        },
                        "data": {
                            "accesskey":
                                base64.b64encode(s3_access_key["AccessKey"]
                                                 ["AccessKeyId"].encode('utf-8')
                                                ).decode("utf-8"),
                            "secretkey":
                                base64.b64encode(
                                    s3_access_key["AccessKey"]
                                    ["SecretAccessKey"].encode('utf-8')
                                ).decode("utf-8"),
                        },
                    })

            # Only the CA rotates in place; retain observed credentials and
            # service-account tokens, including controller-populated fields,
            # but not the volatile metadata that would never converge.
            for index, resource in enumerate(desired_resources):
                name = resource["metadata"]["name"]
                if (resource["kind"] == "Secret" and
                        name != DRIVER_AGENT_CA_SECRET_NAME):
                    existing = attachments["Secret.v1"].get(
                        f"{namespace}/{name}")
                    if existing:
                        desired_resources[index] = preserved_secret(existing)

            if artifacts_proxy_enabled.lower() == "true":
                storage_record_secret = self.artifact_storage_record(
                    parent, attachments, desired_resources)
                desired_resources.append(storage_record_secret)
                policy = next(resource for resource in desired_resources
                              if resource["kind"] == "NetworkPolicy")
                observed_policy = attachments.get(
                    "NetworkPolicy.networking.k8s.io/v1",
                    {}).get(f"{namespace}/ml-pipeline-ui-artifact", {})
                existing_receiver = attachments.get(
                    "Deployment.apps/v1",
                    {}).get(f"{namespace}/ml-pipeline-ui-artifact")
                # Establish ingress isolation before a new receiver can listen.
                # Retain existing receivers while their policy converges.
                if existing_receiver is None and observed_policy.get(
                        "spec") != policy["spec"]:
                    desired_resources = [
                        resource for resource in desired_resources
                        if resource["kind"] != "Deployment"
                    ]
                result = {
                    "status": desired_status,
                    "attachments": desired_resources
                }
                storage_state = json.loads(
                    base64.b64decode(storage_record_secret["data"]
                                     ["authorization.json"]))["state"]
                if storage_state == "pending":
                    result["resyncAfterSeconds"] = 10
                return result
            return {"status": desired_status, "attachments": desired_resources}

        def do_GET(self):
            # Liveness/readiness endpoint; must not call external services.
            if self.path.split("?", 1)[0] == HEALTHZ_PATH:
                self.send_response(200)
                self.send_header("Content-type", "text/plain")
                self.end_headers()
                self.wfile.write(b"ok")
            else:
                self.send_error(404)

        def log_request(self, code="-", size="-"):
            # Skip probe requests so they don't flood the controller logs.
            # Malformed requests are logged before `command`/`path` are set.
            is_probe = (
                getattr(self, "command", None) == "GET" and
                getattr(self, "path", "").split("?", 1)[0] == HEALTHZ_PATH)
            if not is_probe:
                super().log_request(code, size)

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length"))))
            desired = self.sync(observed["object"], observed["attachments"])

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), 'utf-8'))

    return ThreadingHTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()

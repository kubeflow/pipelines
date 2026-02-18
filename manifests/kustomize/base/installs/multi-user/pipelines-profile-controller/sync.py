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

from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import os
import base64
import hashlib
from string import Template

# From awscli installed in alpine/k8s image
import botocore.session

S3_BUCKET_NAME = "mlpipeline"

_TEMPLATE_DIR = os.path.dirname(os.path.abspath(__file__))

session = botocore.session.get_session()
# S3 client for lifecycle policy management
s3_endpoint_url = os.environ.get(
    "S3_ENDPOINT_URL", "http://seaweedfs.kubeflow:8333"
)
s3 = session.create_client(
    "s3", region_name="foobar", endpoint_url=s3_endpoint_url
)


def _normalize_domain(domain):
    return domain if domain.startswith(".") else "." + domain


def create_iam_client():
    # To interact with SeaweedFS user management. Region does not matter.
    endpoint_url = os.environ.get("AWS_ENDPOINT_URL")
    if endpoint_url:
        return session.create_client(
            "iam", region_name="foobar", endpoint_url=endpoint_url
        )
    return session.create_client("iam", region_name="foobar")


iam = create_iam_client()


def main():
    settings = get_settings_from_env()
    server = server_factory(**settings)
    server.serve_forever()


def get_settings_from_env(
    controller_port=None,
    frontend_image=None,
    frontend_tag=None,
    disable_istio_sidecar=None,
    artifacts_proxy_enabled=None,
    artifact_retention_days=None,
    cluster_domain=None,
    object_store_host=None,
):
    """
    Returns a dict of settings from environment variables relevant to the
    controller

    Environment settings can be overridden by passing them here as arguments.

    Settings are pulled from the all-caps version of the setting name.  The
    following defaults are used if those environment variables are not set
    to enable backwards compatibility with previous versions of this script:
        frontend_image: ghcr.io/kubeflow/kfp-frontend
        frontend_tag: value of KFP_VERSION environment variable
        disable_istio_sidecar: Required (no default)
    """
    settings = dict()
    settings["controller_port"] = controller_port or os.environ.get(
        "CONTROLLER_PORT", "8080"
    )

    settings["frontend_image"] = frontend_image or os.environ.get(
        "FRONTEND_IMAGE", "ghcr.io/kubeflow/kfp-frontend"
    )

    settings["artifacts_proxy_enabled"] = (
        artifacts_proxy_enabled
        or os.environ.get("ARTIFACTS_PROXY_ENABLED", "false")
    )

    settings["artifact_retention_days"] = (
        artifact_retention_days or os.environ.get("ARTIFACT_RETENTION_DAYS", -1)
    )

    settings["cluster_domain"] = cluster_domain or os.environ.get(
        "CLUSTER_DOMAIN", ".svc.cluster.local"
    )

    settings["object_store_host"] = object_store_host or os.environ.get(
        "OBJECT_STORE_HOST", "seaweedfs"
    )

    # Look for specific tags for each image first, falling back to
    # previously used KFP_VERSION environment variable for backwards
    # compatibility
    settings["frontend_tag"] = (
        frontend_tag
        or os.environ.get("FRONTEND_TAG")
        or os.environ["KFP_VERSION"]
    )

    settings["disable_istio_sidecar"] = (
        disable_istio_sidecar
        if disable_istio_sidecar is not None
        else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"
    )

    return settings


def _template_path(filename):
    """Return the absolute path to a template file co-located with this script."""
    return os.path.join(_TEMPLATE_DIR, filename)


def fill_json_template(path, **kwargs):
    """
    Fill a JSON template with values provided as keyword arguments.

    :param path: Path to the JSON template file.
    :param kwargs: Keyword arguments representing placeholder values.
    :return: A JSON object with placeholders filled.
    """
    with open(path, "r") as file:
        loaded_template = file.read()

    # Treat empty and whitespace-only templates as "no template".
    # This avoids json.JSONDecodeError when the file contains only
    # whitespace or newlines, which is easy to introduce when editing.
    if not loaded_template.strip():
        return []

    template = Template(loaded_template)
    try:
        rendered_template = template.substitute(**kwargs)
    except KeyError as error:
        missing_key = error.args[0]
        raise ValueError(
            f"Template {path} missing required key: {missing_key}"
        ) from error

    json_object = json.loads(rendered_template)

    return json_object


def load_desired_resources(**kwargs):
    """Load desired resources from JSON template files.

    All resources are loaded from template files:
    - desired_resources.json: Base resources and artifact proxy resources
    - additional_resources.json: Additional resources
    """
    namespace = kwargs.get("namespace")
    cluster_domain = kwargs.get("cluster_domain", ".svc.cluster.local")
    object_store_host = kwargs.get("object_store_host", "seaweedfs")
    artifacts_proxy_enabled = kwargs.get("artifacts_proxy_enabled", "false")
    disable_istio_sidecar = kwargs.get("disable_istio_sidecar", False)
    frontend_image = kwargs.get("frontend_image", "")
    frontend_tag = kwargs.get("frontend_tag", "")

    istio_sidecar_annotation = (
        '"sidecar.istio.io/inject": "false",' if disable_istio_sidecar else ""
    )
    image_spec_hash = hashlib.sha256(
        f"{frontend_image}:{frontend_tag}".encode()
    ).hexdigest()[:16]
    normalized_cluster_domain = _normalize_domain(cluster_domain)
    ml_pipeline_service_host = (
        f"ml-pipeline.kubeflow{normalized_cluster_domain}"
    )

    # Load consolidated resources from template
    resources_template = fill_json_template(
        path=_template_path("desired_resources.json"),
        namespace=namespace,
        cluster_domain=cluster_domain,
        object_store_host=object_store_host,
        normalized_cluster_domain=normalized_cluster_domain,
        istio_sidecar_annotation=istio_sidecar_annotation,
        image_spec_hash=image_spec_hash,
        frontend_image=frontend_image,
        frontend_tag=frontend_tag,
        ml_pipeline_service_host=ml_pipeline_service_host,
    )

    if isinstance(resources_template, dict):
        resources = list(resources_template.get("base", []))
        artifact_proxy_resources = resources_template.get("artifact_proxy", [])
    else:
        resources = list(resources_template)
        artifact_proxy_resources = []

    resources += fill_json_template(
        path=_template_path("additional_resources.json"),
        namespace=namespace,
        cluster_domain=cluster_domain,
        object_store_host=object_store_host,
        normalized_cluster_domain=normalized_cluster_domain,
        istio_sidecar_annotation=istio_sidecar_annotation,
        image_spec_hash=image_spec_hash,
        frontend_image=frontend_image,
        frontend_tag=frontend_tag,
        ml_pipeline_service_host=ml_pipeline_service_host,
    )

    # Conditionally add artifact proxy resources from template
    if str(artifacts_proxy_enabled).lower() == "true":
        resources += list(artifact_proxy_resources)

    for resource in resources:
        if (
            resource.get("kind") == "ConfigMap"
            and resource.get("metadata", {}).get("name")
            == "artifact-repositories"
        ):
            data = resource.get("data", {})
            repository_config = data.get("default-namespaced")
            if isinstance(repository_config, dict):
                data["default-namespaced"] = json.dumps(repository_config)
            break

    return resources


def compute_desired_status(attachments, desired_resources):
    """Calculate the desired status of Kubernetes resources by comparing
    expected and observed counts."""
    expected_counts = {}

    for resource in desired_resources:
        kind = resource["kind"]
        api_version = resource["apiVersion"]
        key = f"{kind}.{api_version}"
        if key not in expected_counts:
            expected_counts[key] = 0
        expected_counts[key] += 1

    desired_status = {}
    for key, expected_count in expected_counts.items():
        observed_count = len(attachments.get(key, []))
        desired_status[key] = observed_count == expected_count

    return {"kubeflow-pipelines-ready": all(desired_status.values())}


def server_factory(
    frontend_image,
    frontend_tag,
    disable_istio_sidecar,
    artifacts_proxy_enabled,
    artifact_retention_days,
    cluster_domain=".svc.cluster.local",
    object_store_host="seaweedfs",
    url="",
    controller_port=8080,
):
    """
    Returns an HTTPServer populated with Handler with customized settings
    """

    class Controller(BaseHTTPRequestHandler):
        def upsert_lifecycle_policy(self, bucket_name, artifact_retention_days):
            """Configures or deletes the lifecycle policy based on the artifact_retention_days string."""
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
                        Bucket=bucket_name
                    )
                    # Check if there are any enabled rules
                    has_enabled_rules = any(
                        rule.get("Status") == "Enabled"
                        for rule in response.get("Rules", [])
                    )

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
                "Rules": [
                    {
                        "Status": "Enabled",
                        "Filter": {"Prefix": "private-artifacts"},
                        "Expiration": {"Days": retention_days},
                        "ID": "private-artifacts",
                    },
                ]
            }
            print("upsert_lifecycle_policy:", life_cycle_policy)

            try:
                api_response = s3.put_bucket_lifecycle_configuration(
                    Bucket=bucket_name, LifecycleConfiguration=life_cycle_policy
                )
                print("Lifecycle policy configured successfully:", api_response)
            except Exception as exception:
                if (
                    hasattr(exception, "response")
                    and "Error" in exception.response
                ):
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception.response['Error']['Code']} - {exception}"
                    )
                else:
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception}"
                    )

        def sync(self, parent, attachments):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = (
                parent.get("metadata", {})
                .get("labels", {})
                .get("pipelines.kubeflow.org/enabled")
            )

            if pipeline_enabled != "true":
                return {"status": {}, "attachments": []}

            desired_resources = load_desired_resources(
                namespace=namespace,
                cluster_domain=cluster_domain,
                object_store_host=object_store_host,
                artifacts_proxy_enabled=artifacts_proxy_enabled,
                disable_istio_sidecar=disable_istio_sidecar,
                frontend_image=frontend_image,
                frontend_tag=frontend_tag,
            )

            # Exclude secrets from the print statement
            non_secret_resources = []
            invalid_resources = []
            for r in desired_resources:
                if not isinstance(r, dict):
                    invalid_resources.append(r)
                    continue
                kind = r.get("kind")
                if kind is None:
                    invalid_resources.append(r)
                    continue
                if kind != "Secret":
                    non_secret_resources.append(r)
            if invalid_resources:
                # Log malformed resources instead of failing the controller.
                print(
                    "Warning: Skipping malformed desired_resources items "
                    "without a valid 'kind' when logging:",
                    json.dumps(invalid_resources, sort_keys=True),
                )
            print("Received request:\n", json.dumps(parent, sort_keys=True))
            print(
                "Desired resources except secrets:\n",
                json.dumps(non_secret_resources, sort_keys=True),
            )

            # Check if secret already exists in attachments. If yes,
            # reuse it. Otherwise create new IAM credentials on
            # seaweedfs for the namespace.
            if s3_secret := attachments["Secret.v1"].get(
                f"{namespace}/mlpipeline-minio-artifact"
            ):
                desired_resources.append(s3_secret)
                print("Using existing secret")
            else:
                print("Creating new access key.")
                s3_access_key = iam.create_access_key(UserName=namespace)
                # Use the AWS IAM API of seaweedfs to manage access
                # policies to bucket. This policy ensures that a user
                # can only access artifacts from his own profile.
                iam.put_user_policy(
                    UserName=namespace,
                    PolicyName=f"KubeflowProject{namespace}",
                    PolicyDocument=json.dumps(
                        {
                            "Version": "2012-10-17",
                            "Statement": [
                                {
                                    "Effect": "Allow",
                                    "Action": [
                                        "s3:Put*",
                                        "s3:Get*",
                                        "s3:List*",
                                    ],
                                    "Resource": [
                                        f"arn:aws:s3:::{S3_BUCKET_NAME}/artifacts/*",
                                        f"arn:aws:s3:::{S3_BUCKET_NAME}/private-artifacts/{namespace}/*",
                                        f"arn:aws:s3:::{S3_BUCKET_NAME}/private/{namespace}/*",
                                        f"arn:aws:s3:::{S3_BUCKET_NAME}/shared/*",
                                    ],
                                }
                            ],
                        }
                    ),
                )

                self.upsert_lifecycle_policy(
                    S3_BUCKET_NAME, artifact_retention_days
                )

                s3_secret = {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": "mlpipeline-minio-artifact",
                        "namespace": namespace,
                    },
                    "data": {
                        "accesskey": base64.b64encode(
                            s3_access_key["AccessKey"]["AccessKeyId"].encode(
                                "utf-8"
                            )
                        ).decode("utf-8"),
                        "secretkey": base64.b64encode(
                            s3_access_key["AccessKey"][
                                "SecretAccessKey"
                            ].encode("utf-8")
                        ).decode("utf-8"),
                    },
                }
                desired_resources.insert(0, s3_secret)

            desired_status = compute_desired_status(
                attachments, desired_resources
            )

            return {"status": desired_status, "attachments": desired_resources}

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length")))
            )
            desired = self.sync(observed["object"], observed["attachments"])

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), "utf-8"))

    return HTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()

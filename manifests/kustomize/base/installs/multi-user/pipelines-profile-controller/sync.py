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
from kubernetes import client, config
from kubernetes.client.rest import ApiException

# From awscli installed in alpine/k8s image
import botocore.session

S3_BUCKET_NAME = 'mlpipeline'

session = botocore.session.get_session()
# S3 client for lifecycle policy management
s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL", "http://seaweedfs.kubeflow:8333")
s3 = session.create_client('s3', region_name='foobar', endpoint_url=s3_endpoint_url)


def _normalize_domain(domain):
    return domain if domain.startswith('.') else '.' + domain


def create_iam_client():
    # To interact with SeaweedFS user management. Region does not matter.
    endpoint_url = os.environ.get("AWS_ENDPOINT_URL")
    if endpoint_url:
        return session.create_client('iam', region_name='foobar', endpoint_url=endpoint_url)
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
                          artifact_retention_days=None,
                          cluster_domain=None,
                          object_store_host=None):
    """
    Returns a dict of settings from environment variables relevant to the controller

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

    settings["artifact_retention_days"] = \
        artifact_retention_days or \
        os.environ.get("ARTIFACT_RETENTION_DAYS", -1)

    settings["cluster_domain"] = \
        cluster_domain or \
        os.environ.get("CLUSTER_DOMAIN", ".svc.cluster.local")

    settings["object_store_host"] = \
        object_store_host or \
        os.environ.get("OBJECT_STORE_HOST", "seaweedfs")

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
                   cluster_domain=".svc.cluster.local",
                   object_store_host="seaweedfs",
                   url="",
                   controller_port=8080):
    """
    Returns an HTTPServer populated with Handler with customized settings
    """
    class Controller(BaseHTTPRequestHandler):
        def upsert_lifecycle_policy(self, bucket_name, artifact_retention_days):
            """Configures or deletes the lifecycle policy based on the artifact_retention_days string."""
            try:
                retention_days = int(artifact_retention_days)
            except ValueError:
                print(f"ERROR: ARTIFACT_RETENTION_DAYS value '{artifact_retention_days}' is not a valid integer. Aborting policy update.")
                return

            # To disable lifecycle policy we need to delete it
            if retention_days <= 0:
                print(f"ARTIFACT_RETENTION_DAYS is non-positive ({retention_days} days). Attempting to delete lifecycle policy.")
                try:
                    response = s3.get_bucket_lifecycle_configuration(Bucket=bucket_name)
                    # Check if there are any enabled rules
                    has_enabled_rules = any(rule.get('Status') == 'Enabled' for rule in response.get('Rules', []))

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
            print('upsert_lifecycle_policy:', life_cycle_policy)

            try:
                api_response = s3.put_bucket_lifecycle_configuration(
                    Bucket=bucket_name,
                    LifecycleConfiguration = life_cycle_policy
                )
                print('Lifecycle policy configured successfully:', api_response)
            except Exception as exception:
                if hasattr(exception, 'response') and 'Error' in exception.response:
                    print(f"ERROR: Failed to configure lifecycle policy: {exception.response['Error']['Code']} - {exception}")
                else:
                    print(f"ERROR: Failed to configure lifecycle policy: {exception}")

        def upsert_executor_plugin_sa(self, namespace):
            print('create executor plugin SAs for: ', namespace)
            try:
                config.load_incluster_config()
            except:
                config.load_kube_config()
            core_v1 = client.CoreV1Api()
            rbac_v1 = client.RbacAuthorizationV1Api()
            agent_sa_name = "ml-pipeline-driver-agent-executor-plugin"
            try:
                core_v1.create_namespaced_service_account(
                    namespace=namespace,
                    body=client.V1ServiceAccount(
                        metadata=client.V1ObjectMeta(
                            name=agent_sa_name,
                            labels={"application-crd-id": "kubeflow-pipelines"},
                        )
                    )
                )
                print(f"ServiceAccount {agent_sa_name} created in {namespace}")
                role_name = "configmap-reader"
                role = client.V1Role(
                    metadata=client.V1ObjectMeta(
                        name=role_name,
                        namespace=namespace
                    ),
                    rules=[
                        client.V1PolicyRule(
                            api_groups=[""],
                            resources=["configmaps"],
                            verbs=["get", "list", "watch"]
                        )
                    ]
                )
                rbac_v1.create_namespaced_role(namespace=namespace, body=role)
                print(f"Role {role_name} created in {namespace}")

                role_binding_name = "configmap-reader-binding"
                namespace = 'kubeflow-user-example-com'
                agent_sa_name = "ml-pipeline-driver-agent-executor-plugin"
                role_name = "configmap-reader"
                role_binding = client.V1RoleBinding(
                    metadata=client.V1ObjectMeta(
                        name="configmap-reader-binding",
                        namespace=namespace
                    ),
                    subjects=[
                        {
                            "kind": "ServiceAccount",
                            "name": agent_sa_name,
                            "namespace": namespace,
                            "apiGroup": ""
                        }
                    ],
                    role_ref=client.V1RoleRef(
                        kind="Role",
                        name=role_name,
                        api_group="rbac.authorization.k8s.io"
                    )
                )

                rbac_v1.create_namespaced_role_binding(
                    namespace=namespace,
                    body=role_binding
                )
                print(f"RoleBinding {role_binding_name} created in {namespace}")
            except ApiException as e:
                if e.status == 409:
                    print(f"ServiceAccount {agent_sa_name} already exists in {namespace}")
                else:
                    print(f"Failed to create ServiceAccount {agent_sa_name}: {e}")
            except Exception as e:
                print(f"Unexpected error during the update sa in: {namespace} {e}")

        def sync(self, parent, attachments):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = parent.get("metadata", {}).get(
                "labels", {}).get("pipelines.kubeflow.org/enabled")

            if pipeline_enabled != "true":
                return {"status": {}, "attachments": []}

            # Compute status based on observed state.
            desired_status = {
                "kubeflow-pipelines-ready":
                    len(attachments["Secret.v1"]) == 1 and
                    len(attachments["ConfigMap.v1"]) == 3 and
                    len(attachments["Deployment.apps/v1"]) == (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    len(attachments["Service.v1"]) == (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    "True" or "False"
            }


            self.upsert_executor_plugin_sa(namespace)

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
                        "defaultPipelineRoot": f"minio://{S3_BUCKET_NAME}/private-artifacts/{namespace}/v2/artifacts",
                        "clusterDomain": cluster_domain,
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "metadata-grpc-configmap",
                        "namespace": namespace,
                    },
                    "data": {
                        "METADATA_GRPC_SERVICE_HOST":
                            "metadata-grpc-service.kubeflow",
                        "METADATA_GRPC_SERVICE_PORT": "8080",
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "artifact-repositories",
                        "namespace": namespace,
                        "annotations": {
                            "workflows.argoproj.io/default-artifact-repository": "default-namespaced"
                        }
                    },
                    "data": {
                        "default-namespaced": json.dumps({
                            "archiveLogs": True,
                            "s3": {
                                "endpoint": f"{object_store_host}.kubeflow{_normalize_domain(cluster_domain)}:9000",
                                "bucket": S3_BUCKET_NAME,
                                "keyFormat": f"private-artifacts/{namespace}/{{{{workflow.name}}}}/{{{{workflow.creationTimestamp.Y}}}}/{{{{workflow.creationTimestamp.m}}}}/{{{{workflow.creationTimestamp.d}}}}/{{{{pod.name}}}}",
                                "insecure": True,
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

            if artifacts_proxy_enabled.lower() == "true":
                desired_resources.extend([
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
                                        **({"sidecar.istio.io/inject": "false"} if disable_istio_sidecar else {}),
                                        "kubeflow-pipelines/image-spec-hash": hashlib.sha256(f"{frontend_image}:{frontend_tag}".encode()).hexdigest()[:16],
                                    },
                                },
                                "spec": {
                                    "containers": [{
                                        "name":
                                            "ml-pipeline-ui-artifact",
                                        "image": f"{frontend_image}:{frontend_tag}",
                                        "imagePullPolicy":
                                            "IfNotPresent",
                                        "ports": [{
                                            "containerPort": 3000
                                        }],
                                        "env": [
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
                                            }
                                        ],
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
                                    "serviceAccountName":
                                        "default-editor"
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

            print('Creating executor-plugin service accounts')
            # Argo Workflow Executor Plugin Necessary Resources
            agent_sa_name = "ml-pipeline-driver-agent-executor-plugin"
            secret_name = f"{agent_sa_name}.service-account-token"
            desired_resources.extend([
                {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": "default-editor.service-account-token",
                        "namespace": namespace,
                        "annotations": {
                            "kubernetes.io/service-account.name": "default-editor"
                        }
                    },
                    "type": "kubernetes.io/service-account-token"
                },
                {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": secret_name,
                        "namespace": namespace,
                        "annotations": {
                            "kubernetes.io/service-account.name": agent_sa_name,
                        },
                    },
                    "type": "kubernetes.io/service-account-token",
                },
            ])

            print('Received request:\n', json.dumps(parent, sort_keys=True))
            print('Desired resources except secrets:\n', json.dumps(desired_resources, sort_keys=True))

            # Moved after the print argument because this is sensitive data.

            # Check if secret is already there when the controller made the request. If yes, then
            # use it. Else create a new credentials on seaweedfs for the namespace.
            if s3_secret := attachments["Secret.v1"].get(f"{namespace}/mlpipeline-minio-artifact"):
                desired_resources.append(s3_secret)
                print('Using existing secret')
            else:
                print('Creating new access key.')
                s3_access_key = iam.create_access_key(UserName=namespace)
                # Use the AWS IAM API of seaweedfs to manage access policies to bucket.
                # This policy ensures that a user can only access artifacts from his own profile.
                iam.put_user_policy(
                    UserName=namespace,
                    PolicyName=f"KubeflowProject{namespace}",
                    PolicyDocument=json.dumps(
                        {
                            "Version": "2012-10-17",
                            "Statement": [{
                                "Effect": "Allow",
                                "Action": [
                                    "s3:Put*",
                                    "s3:Get*",
                                    "s3:List*"
                                ],
                                "Resource": [
                                    f"arn:aws:s3:::{S3_BUCKET_NAME}/artifacts/*",
                                    f"arn:aws:s3:::{S3_BUCKET_NAME}/private-artifacts/{namespace}/*",
                                    f"arn:aws:s3:::{S3_BUCKET_NAME}/private/{namespace}/*",
                                    f"arn:aws:s3:::{S3_BUCKET_NAME}/shared/*",
                                ]
                            }]
                        })
                )

                self.upsert_lifecycle_policy(S3_BUCKET_NAME, artifact_retention_days)

                desired_resources.insert(
                    0,
                    {
                        "apiVersion": "v1",
                        "kind": "Secret",
                        "metadata": {
                            "name": "mlpipeline-minio-artifact",
                            "namespace": namespace,
                        },
                        "data": {
                            "accesskey": base64.b64encode(s3_access_key["AccessKey"]["AccessKeyId"].encode('utf-8')).decode("utf-8"),
                            "secretkey": base64.b64encode(s3_access_key["AccessKey"]["SecretAccessKey"].encode('utf-8')).decode("utf-8"),
                    },
                })

            return {"status": desired_status, "attachments": desired_resources}

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length"))))
            desired = self.sync(observed["object"], observed["attachments"])

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), 'utf-8'))

    return HTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()

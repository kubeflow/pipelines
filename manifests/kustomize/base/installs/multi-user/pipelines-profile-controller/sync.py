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
import boto3, botocore, secrets, string
from minio import Minio, MinioAdmin
from kubernetes import client, config


def main():
    settings = get_settings_from_env()
    server = server_factory(**settings)
    server.serve_forever()


def get_settings_from_env(controller_port=None,
                          visualization_server_image=None, frontend_image=None,
                          visualization_server_tag=None, frontend_tag=None, disable_istio_sidecar=None,
                          minio_access_key=None, minio_secret_key=None, kfp_default_pipeline_root=None):
    """
    Returns a dict of settings from environment variables relevant to the controller

    Environment settings can be overridden by passing them here as arguments.

    Settings are pulled from the all-caps version of the setting name.  The
    following defaults are used if those environment variables are not set
    to enable backwards compatibility with previous versions of this script:
        visualization_server_image: gcr.io/ml-pipeline/visualization-server
        visualization_server_tag: value of KFP_VERSION environment variable
        frontend_image: gcr.io/ml-pipeline/frontend
        frontend_tag: value of KFP_VERSION environment variable
        disable_istio_sidecar: Required (no default)
        minio_access_key: Required (no default)
        minio_secret_key: Required (no default)
    """
    settings = dict()
    settings["controller_port"] = \
        controller_port or \
        os.environ.get("CONTROLLER_PORT", "8080")

    settings["visualization_server_image"] = \
        visualization_server_image or \
        os.environ.get("VISUALIZATION_SERVER_IMAGE", "gcr.io/ml-pipeline/visualization-server")

    settings["frontend_image"] = \
        frontend_image or \
        os.environ.get("FRONTEND_IMAGE", "gcr.io/ml-pipeline/frontend")

    # Look for specific tags for each image first, falling back to
    # previously used KFP_VERSION environment variable for backwards
    # compatibility
    settings["visualization_server_tag"] = \
        visualization_server_tag or \
        os.environ.get("VISUALIZATION_SERVER_TAG") or \
        os.environ["KFP_VERSION"]

    settings["frontend_tag"] = \
        frontend_tag or \
        os.environ.get("FRONTEND_TAG") or \
        os.environ["KFP_VERSION"]

    settings["disable_istio_sidecar"] = \
        disable_istio_sidecar if disable_istio_sidecar is not None \
            else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"

    settings["minio_access_key"] = \
        minio_access_key or \
        base64.b64encode(bytes(os.environ.get("MINIO_ACCESS_KEY"), 'utf-8')).decode('utf-8')

    settings["minio_secret_key"] = \
        minio_secret_key or \
        base64.b64encode(bytes(os.environ.get("MINIO_SECRET_KEY"), 'utf-8')).decode('utf-8')

    # KFP_DEFAULT_PIPELINE_ROOT is optional
    settings["kfp_default_pipeline_root"] = \
        kfp_default_pipeline_root or \
        os.environ.get("KFP_DEFAULT_PIPELINE_ROOT")

    return settings


def server_factory(visualization_server_image,
                   visualization_server_tag, frontend_image, frontend_tag,
                   disable_istio_sidecar, minio_access_key,
                   minio_secret_key, kfp_default_pipeline_root=None,
                   url="", controller_port=8080):
    """
    Returns an HTTPServer populated with Handler with customized settings
    """

    class Controller(BaseHTTPRequestHandler):
        config.load_incluster_config()
        kubernetes_api_v1 = client.CoreV1Api()
        minio_host = 'minio-service.kubeflow.svc.cluster.local:9000'
        shared_bucket_name = 'mlpipeline' # TODO use the environment variable as the apiserver does
        session = boto3.session.Session()

        admin_minio_access_key = base64.b64decode(minio_access_key).decode('utf-8')
        admin_minio_secret_key = base64.b64decode(minio_secret_key).decode('utf-8')

        s3_client = session.client(
            service_name='s3',
            aws_access_key_id=admin_minio_access_key,
            aws_secret_access_key=admin_minio_secret_key,
            endpoint_url='http://%s' % minio_host
        )

        minio_client = Minio(
            "%s" % minio_host,
            access_key=admin_minio_access_key,
            secret_key=admin_minio_secret_key,
        )
        # init_mc_client()
        mc_config = {
            "version": "8",
            "hosts": {
                "kubeflow-minio": {
                    "url": 'http://%s' % minio_host,
                    "accessKey": admin_minio_access_key,
                    "secretKey": admin_minio_secret_key,
                    "api": "s3v4",
                }
            }
        }
        with open('/tmp/config.json', 'w') as outfile:
            json.dump(mc_config, outfile)
        admin = MinioAdmin(target="kubeflow-minio", binary_path='/app/mc', config_dir='/tmp')

        def upsert_lifecycle_policy(self, bucket_name):
            lfc = {
                "Rules": [
                    {
                        "Status": "Enabled",
                        "Filter": {"Prefix": "private-artifacts/"},
                        "Expiration": {"Days":31},
                        "ID": "private-artifacts",
                    },
                ]
            }
            print('upsert_lifecycle_policy:', lfc)
            api_response = self.s3_client.put_bucket_lifecycle_configuration(Bucket=bucket_name,
                                                                             LifecycleConfiguration=lfc)
            print(api_response)

        def upsert_iam_policy(self, shared_bucket_name):
            namespace_isolation_policy = {
                "Version": "2012-10-17",
                "Statement": [
                    {
                        "Effect": "Allow",
                        "Action": [
                            "s3:ListBucket",
                        ],
                        "Resource": [
                            "arn:aws:s3:::%s"  % shared_bucket_name, # the root is needed to list the bucket
                        ],
                        "Condition":{"StringLike":{"s3:prefix": [
                            "", "shared/*", "artifacts/*","private-artifacts/", "private/",
                            "private-artifacts/${aws:username}/*", "private/${aws:username}/*"
                        ]}}
                    },
                    {
                        "Effect": "Allow",
                        "Action": [
                            "s3:ListBucket",
                            "s3:GetBucketLocation",
                            "s3:GetBucketPolicy",
                            "s3:GetObject",
                            "s3:DeleteObject",
                            "s3:PutObject"
                        ],
                        "Resource": [
                            "arn:aws:s3:::%s/artifacts/*"  % shared_bucket_name, # old shared artifacts for backwards compatibility
                            "arn:aws:s3:::%s/private-artifacts/${aws:username}/*"  % shared_bucket_name, # private artifacts
                            "arn:aws:s3:::%s/private/${aws:username}/*"  % shared_bucket_name, # private storage
                            "arn:aws:s3:::%s/shared/*"  % shared_bucket_name # shared storage for collaboration
                        ]
                    }
                ]
            }
            tmp_policy_filename = '/tmp/namespace_isolation_policy'
            with open(tmp_policy_filename, 'w') as outfile:
                json.dump(namespace_isolation_policy, outfile)
            print(f' upsert_iam_policy: {namespace_isolation_policy}')
            api_response = self.admin.policy_add('namespace-isolation-policy', tmp_policy_filename)
            print(api_response)
            os.remove(tmp_policy_filename)
            policy_arn = api_response['policy']
            return policy_arn

        def get_password(self, namespace):
            secret_name = 'mlpipeline-minio-artifact'
            secret_names = [secret.metadata.name for secret in self.kubernetes_api_v1.list_namespaced_secret(namespace).items]
            print(secret_names)
            if secret_name in secret_names:
                print('found secret %s', secret_name)
                print('Using existing password for %s', namespace)
                secret = str(self.kubernetes_api_v1.read_namespaced_secret(secret_name, namespace).data)
                secret = json.loads(secret.replace('\'', '\"'))
                return base64.b64decode(secret['secretkey']).decode('utf-8')
            else:
                print('Generating new password for %s', namespace)
                alphabet = string.ascii_letters + string.digits
                return ''.join(secrets.choice(alphabet) for i in range(16))

        def create_user(self, user_name, password):
            self.admin.user_add(user_name, password)
            policy_name = self.upsert_iam_policy(self.shared_bucket_name)
            self.attach_managed_policy(user_name, policy_name)

        def attach_managed_policy(self, user_name, policy_name):
            print('attach_managed_policy %s %s', user_name, policy_name)
            print(self.admin.policy_set(policy_name=policy_name, user=user_name))


        def sync(self, parent, children):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = parent.get("metadata", {}).get(
                "labels", {}).get("pipelines.kubeflow.org/enabled")

            if pipeline_enabled != "true":
                return {"status": {}, "children": []}

            user_password = self.get_password(namespace)
            #print('password for ' + namespace + ' is ' + user_password)
            self.create_user(user_name=namespace, password=user_password)
            # TODO lifecycle policy should take the days from the environment variable
            # A negative value should disable it.
            self.upsert_lifecycle_policy(self.shared_bucket_name)

            desired_configmap_count = 1
            desired_resources = []
            if kfp_default_pipeline_root:
                desired_configmap_count += 1
                desired_resources += [{
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "kfp-launcher",
                        "namespace": namespace,
                    },
                    "data": {
                        "defaultPipelineRoot": kfp_default_pipeline_root,
                    },
                }]

            # Compute status based on observed state.
            desired_status = {
                "kubeflow-pipelines-ready":
                    len(children["NetworkPolicy.networking.k8s.io/v1"]) == 1 and \
                    len(children["PodDefault.kubeflow.org/v1alpha1"]) == 1 and \
                    len(children["NetworkAttachmentDefinition.k8s.cni.cncf.io/v1"]) == 1 and \
                    len(children["LimitRange.v1"]) == 1 and \
                    len(children["Secret.v1"]) == 1 and
                    len(children["ConfigMap.v1"]) == desired_configmap_count and
                    len(children["Deployment.apps/v1"]) == 1 and
                    len(children["Service.v1"]) == 1 and
                    "True" or "False"
            }

            # Generate the desired child object(s).
            desired_resources += [
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
                # Artifact fetcher related resources below.
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
                                "annotations": disable_istio_sidecar and {
                                    "sidecar.istio.io/inject": "false"
                                } or {},
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
                {
                    "apiVersion": "k8s.cni.cncf.io/v1",
                    "kind": "NetworkAttachmentDefinition",
                    "metadata": {
                        "name": "istio-cni",
                        "namespace": namespace
                    }
                },
                {
                    "apiVersion": "v1",
                    "kind": "LimitRange",
                    "metadata": {
                        "name": "default-request",
                        "namespace": namespace
                    },
                    "spec": {
                        "limits": [
                            {
                                "type": "Container",
                                "defaultRequest": {
                                    "cpu": "10m",
                                    "memory": "10Mi"
                                }
                            }
                        ]
                    }
                },
                {
                    "apiVersion": "kubeflow.org/v1alpha1",
                    "kind": "PodDefault",
                    "metadata": {
                        "name": "access-ml-pipeline",
                        "namespace": namespace
                    },
                    "spec": {
                        "desc": "Allow access to Kubeflow Pipelines",
                        "selector": {
                            "matchLabels": {
                                "access-ml-pipeline": "true"
                            }
                        },
                        "volumes": [
                            {
                                "name": "volume-kf-pipeline-token",
                                "projected": {
                                    "sources": [
                                        {
                                            "serviceAccountToken": {
                                                "path": "token",
                                                "expirationSeconds": 7200,
                                                "audience": "pipelines.kubeflow.org"
                                            }
                                        }
                                    ]
                                }
                            }
                        ],
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubeflow/pipelines",
                                "name": "volume-kf-pipeline-token",
                                "readOnly": True
                            }
                        ],
                        "env": [
                            {
                                "name": "KF_PIPELINES_SA_TOKEN_PATH",
                                "value": "/var/run/secrets/kubeflow/pipelines/token"
                            }
                        ]
                    }
                },
                {
                    "kind": "NetworkPolicy",
                    "apiVersion": "networking.k8s.io/v1",
                    "metadata": {
                        "name": "allow-non-profile-namespaces",
                        "namespace": namespace
                    },
                    "spec": {
                        "podSelector": {},
                        "ingress": [
                            {
                                "from": [
                                    {
                                        "namespaceSelector": {
                                            "matchExpressions": [
                                                {
                                                    "key": "kubernetes.io/metadata.name",
                                                    "operator": "In",
                                                    "values": [
                                                        "kubeflow",
                                                        "istio-system",
                                                        "knative-serving",
                                                        "knative-eventing"
                                                    ]
                                                }
                                            ]
                                        }
                                    },
                                    {
                                        "podSelector": {}
                                    }
                                ]
                            }],
                        "policyTypes": [
                            "Ingress"
                        ]
                    }
                }
            ]
            print('Received request:\n', json.dumps(parent, indent=2, sort_keys=True))
            print('Desired resources except secrets:\n', json.dumps(desired_resources, indent=2, sort_keys=True))
            # Moved after the print argument because this is sensitive data.
            desired_resources.append({
                "apiVersion": "v1",
                "kind": "Secret",
                "metadata": {
                    "name": "mlpipeline-minio-artifact",
                    "namespace": namespace,
                },
                "data": {
                    "accesskey": base64.b64encode(namespace.encode('ascii')).decode('utf-8'),
                    "secretkey": base64.b64encode(user_password.encode('ascii')).decode('utf-8'),
                },
            })
            return {"status": desired_status, "children": desired_resources}

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length"))))
            desired = self.sync(observed["parent"], observed["children"])

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), 'utf-8'))

    return HTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()

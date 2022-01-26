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


def main():
    settings = get_settings_from_env()
    server = server_factory(**settings)
    server.serve_forever()


def get_settings_from_env(
    controller_port=None,
    visualization_server_image=None,
    frontend_image=None,
    visualization_server_tag=None,
    frontend_tag=None,
    disable_istio_sidecar=None,
    minio_access_key=None,
    minio_secret_key=None,
    kfp_default_pipeline_root=None,
):
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
    settings["controller_port"] = controller_port or os.environ.get(
        "CONTROLLER_PORT", "8080"
    )

    settings[
        "visualization_server_image"
    ] = visualization_server_image or os.environ.get(
        "VISUALIZATION_SERVER_IMAGE", "gcr.io/ml-pipeline/visualization-server"
    )

    settings["frontend_image"] = frontend_image or os.environ.get(
        "FRONTEND_IMAGE", "gcr.io/ml-pipeline/frontend"
    )

    # Look for specific tags for each image first, falling back to
    # previously used KFP_VERSION environment variable for backwards
    # compatibility
    settings["visualization_server_tag"] = (
        visualization_server_tag
        or os.environ.get("VISUALIZATION_SERVER_TAG")
        or os.environ["KFP_VERSION"]
    )

    settings["frontend_tag"] = (
        frontend_tag or os.environ.get("FRONTEND_TAG") or os.environ["KFP_VERSION"]
    )

    settings["disable_istio_sidecar"] = (
        disable_istio_sidecar
        if disable_istio_sidecar is not None
        else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"
    )

    
    settings["minio_access_key"] = minio_access_key or base64.b64encode(
        bytes(os.environ.get("MINIO_ACCESS_KEY"), "utf-8")
    ).decode("utf-8")

    #settings["minio_secret_key"] = minio_secret_key or base64.b64encode(
    #    bytes(os.environ.get("MINIO_SECRET_KEY"), "utf-8")
    #).decode("utf-8")

    if "minio_secret_key" in locals() and minio_secret_key and minio_secret_key != "":
        print("VARIABLE")
        settings["minio_secret_key"] = minio_secret_key
    elif "MINIO_SECRET_KEY" in os.environ:
        print("ENVIRONMENT")
        settings["minio_secret_key"] = base64.b64encode(bytes(
            os.environ.get("MINIO_SECRET_KEY"), "utf-8")).decode("utf-8")
    else:
        print("HASH")
        import hashlib
        service_account_path = '/var/run/secrets/kubernetes.io/serviceaccount'
        with open(os.path.join(service_account_path, 'namespace')) as fp:
            namespace = fp.read().strip()
        with open(os.path.join(service_account_path, 'token')) as fp:
            token = fp.read().strip()
        hasher = hashlib.sha3_256()
        hasher.update((namespace+token).encode('utf-8'))
        settings["minio_secret_key"] = base64.b64encode(bytes(hasher.hexdigest(), "utf-8")).decode('utf-8')

    # KFP_DEFAULT_PIPELINE_ROOT is optional
    settings["kfp_default_pipeline_root"] = kfp_default_pipeline_root or os.environ.get(
        "KFP_DEFAULT_PIPELINE_ROOT"
    )

    return settings


def server_factory(
    visualization_server_image,
    visualization_server_tag,
    frontend_image,
    frontend_tag,
    disable_istio_sidecar,
    minio_access_key,
    minio_secret_key,
    kfp_default_pipeline_root=None,
    url="",
    controller_port=8080,
):
    """
    Returns an HTTPServer populated with Handler with customized settings
    """

    class Controller(BaseHTTPRequestHandler):
        def sync(self, parent, children):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = (
                parent.get("metadata", {})
                .get("labels", {})
                .get("pipelines.kubeflow.org/enabled")
            )

            if pipeline_enabled != "true":
                return {"status": {}, "children": []}

            desired_configmap_count = 2
            desired_resources = []
            if kfp_default_pipeline_root:
                desired_configmap_count += 1
                desired_resources += [
                    {
                        "apiVersion": "v1",
                        "kind": "ConfigMap",
                        "metadata": {
                            "name": "kfp-launcher",
                            "namespace": namespace,
                        },
                        "data": {
                            "defaultPipelineRoot": kfp_default_pipeline_root,
                        },
                    }
                ]
            # Compute status based on observed state.
            desired_status = {
                "kubeflow-pipelines-ready":
                len(children["NetworkPolicy.networking.k8s.io/v1"]) == 1 and
                len(children["PodDefault.kubeflow.org/v1alpha1"]) == 1 and
                len(children["NetworkAttachmentDefinition.k8s.cni.cncf.io/v1"]) == 1 and
                len(children["LimitRange.v1"]) == 1 and
                len(children["Secret.v1"]) == 1 and
                len(children["ConfigMap.v1"]) == desired_configmap_count and
                len(children["Deployment.apps/v1"]) == 2 and
                len(children["Service.v1"]) == 2 and
                len(children["DestinationRule.networking.istio.io/v1beta1"]) == 1 and
                len(children["AuthorizationPolicy.security.istio.io/v1beta1"]) == 1 and
                len(children["PersistentVolumeClaim.v1"]) and
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
                        "METADATA_GRPC_SERVICE_HOST": "metadata-grpc-service.kubeflow",
                        "METADATA_GRPC_SERVICE_PORT": "8080",
                    },
                },
                # Visualization server related manifests below
                # deprecated and therefore commented out
                # {
                #    "apiVersion": "apps/v1",
                #    "kind": "Deployment",
                #    "metadata": {
                #        "labels": {
                #            "app": "ml-pipeline-visualizationserver"
                #        },
                #        "name": "ml-pipeline-visualizationserver",
                #        "namespace": namespace,
                #    },
                #    "spec": {
                #        "selector": {
                #            "matchLabels": {
                #                "app": "ml-pipeline-visualizationserver"
                #            },
                #        },
                #        "template": {
                #            "metadata": {
                #                "labels": {
                #                    "app": "ml-pipeline-visualizationserver"
                #                },
                #                "annotations": disable_istio_sidecar and {
                #                    "sidecar.istio.io/inject": "false"
                #                } or {},
                #            },
                #            "spec": {
                #                "containers": [{
                #                    "image": f"{visualization_server_image}:{visualization_server_tag}",
                #                    "imagePullPolicy":
                #                        "IfNotPresent",
                #                    "name":
                #                        "ml-pipeline-visualizationserver",
                #                    "ports": [{
                #                        "containerPort": 8888
                #                    }],
                #                    "resources": {
                #                        "requests": {
                #                            "cpu": "50m",
                #                            "memory": "200Mi"
                #                        },
                #                        "limits": {
                #                            "cpu": "500m",
                #                            "memory": "1Gi"
                #                        },
                #                    }
                #                }],
                #                "serviceAccountName":
                #                    "default-viewer",
                #            },
                #        },
                #    },
                # },
                # {
                #    "apiVersion": "networking.istio.io/v1alpha3",
                #    "kind": "DestinationRule",
                #    "metadata": {
                #        "name": "ml-pipeline-visualizationserver",
                #        "namespace": namespace,
                #    },
                #    "spec": {
                #        "host": "ml-pipeline-visualizationserver",
                #        "trafficPolicy": {
                #            "tls": {
                #                "mode": "ISTIO_MUTUAL"
                #            }
                #        }
                #    }
                # },
                # {
                #    "apiVersion": "security.istio.io/v1beta1",
                #    "kind": "AuthorizationPolicy",
                #    "metadata": {
                #        "name": "ml-pipeline-visualizationserver",
                #        "namespace": namespace,
                #    },
                #    "spec": {
                #        "selector": {
                #            "matchLabels": {
                #                "app": "ml-pipeline-visualizationserver"
                #            }
                #        },
                #        "rules": [{
                #            "from": [{
                #                "source": {
                #                    "principals": ["cluster.local/ns/kubeflow/sa/ml-pipeline"]
                #                }
                #            }]
                #        }]
                #    }
                # },
                # {
                #    "apiVersion": "v1",
                #    "kind": "Service",
                #    "metadata": {
                #        "name": "ml-pipeline-visualizationserver",
                #        "namespace": namespace,
                #    },
                #    "spec": {
                #        "ports": [{
                #            "name": "http",
                #            "port": 8888,
                #            "protocol": "TCP",
                #            "targetPort": 8888,
                #        }],
                #        "selector": {
                #            "app": "ml-pipeline-visualizationserver",
                #        },
                #    },
                # },
                # Artifact fetcher related resources below.
                {
                    "apiVersion": "apps/v1",
                    "kind": "Deployment",
                    "metadata": {
                        "labels": {"app": "ml-pipeline-ui-artifact"},
                        "name": "ml-pipeline-ui-artifact",
                        "namespace": namespace,
                    },
                    "spec": {
                        "replicas": 1,
                        "selector": {"matchLabels": {"app": "ml-pipeline-ui-artifact"}},
                        "template": {
                            "metadata": {
                                "labels": {"app": "ml-pipeline-ui-artifact"},
                                "annotations": disable_istio_sidecar
                                and {"sidecar.istio.io/inject": "false"}
                                or {},
                            },
                            "spec": {
                                "containers": [
                                    {
                                        "name": "ml-pipeline-ui-artifact",
                                        "image": f"{frontend_image}:{frontend_tag}",
                                        "imagePullPolicy": "IfNotPresent",
                                        "ports": [{"containerPort": 3000}],
                                        "env": [
                                            {
                                                "name": "MINIO_NAMESPACE",
                                                "value": namespace
                                            },
                                            {
                                                "name": "MINIO_ACCESS_KEY",
                                                "valueFrom": {
                                                    "secretKeyRef": {
                                                        "key": "accesskey",
                                                        "name": "mlpipeline-minio-artifact",
                                                    }
                                                },
                                            },
                                            {
                                                "name": "MINIO_SECRET_KEY",
                                                "valueFrom": {
                                                    "secretKeyRef": {
                                                        "key": "secretkey",
                                                        "name": "mlpipeline-minio-artifact",
                                                    }
                                                },
                                            },
                                        ],
                                        "resources": {
                                            "requests": {
                                                "cpu": "10m",
                                                "memory": "70Mi",
                                            },
                                            "limits": {
                                                "cpu": "100m",
                                                "memory": "500Mi",
                                            },
                                        },
                                    }
                                ],
                                "serviceAccountName": "default-viewer",
                            },
                        },
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "Service",
                    "metadata": {
                        "name": "ml-pipeline-ui-artifact",
                        "namespace": namespace,
                        "labels": {"app": "ml-pipeline-ui-artifact"},
                    },
                    "spec": {
                        "ports": [
                            {
                                "name": "http",  # name is required to let istio understand request protocol
                                "port": 80,
                                "protocol": "TCP",
                                "targetPort": 3000,
                            }
                        ],
                        "selector": {"app": "ml-pipeline-ui-artifact"},
                    },
                },
                # Networkattachment for istio-cni on openshift
                {
                    "apiVersion": "k8s.cni.cncf.io/v1",
                    "kind": "NetworkAttachmentDefinition",
                    "metadata": {"name": "istio-cni", "namespace": namespace},
                },
                # Limitrange to workaround the buggy kubeflow profile quotas
                {
                    "apiVersion": "v1",
                    "kind": "LimitRange",
                    "metadata": {"name": "default-request", "namespace": namespace},
                    "spec": {
                        "limits": [
                            {
                                "type": "Container",
                                "defaultRequest": {"cpu": "10m", "memory": "10Mi"},
                            }
                        ]
                    },
                },
                # poddefault such that users can enable ml-pipeline access
                # with a checkbox in the Webinterface
                {
                    "apiVersion": "kubeflow.org/v1alpha1",
                    "kind": "PodDefault",
                    "metadata": {"name": "access-ml-pipeline", "namespace": namespace},
                    "spec": {
                        "desc": "Allow access to Kubeflow Pipelines",
                        "selector": {"matchLabels": {"access-ml-pipeline": "true"}},
                        "volumes": [
                            {
                                "name": "volume-kf-pipeline-token",
                                "projected": {
                                    "sources": [
                                        {
                                            "serviceAccountToken": {
                                                "path": "token",
                                                "expirationSeconds": 7200,
                                                "audience": "pipelines.kubeflow.org",
                                            }
                                        }
                                    ]
                                },
                            }
                        ],
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubeflow/pipelines",
                                "name": "volume-kf-pipeline-token",
                                "readOnly": True,
                            }
                        ],
                        "env": [
                            {
                                "name": "KF_PIPELINES_SA_TOKEN_PATH",
                                "value": "/var/run/secrets/kubeflow/pipelines/token",
                            }
                        ],
                    },
                },
                # Networkpolicy to protect the users namespace
                {
                    "kind": "NetworkPolicy",
                    "apiVersion": "networking.k8s.io/v1",
                    "metadata": {
                        "name": "allow-non-profile-namespaces",
                        "namespace": namespace,
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
                                                        "knative-eventing",
                                                    ],
                                                }
                                            ]
                                        }
                                    },
                                    {"podSelector": {}},
                                ]
                            }
                        ],
                        "policyTypes": ["Ingress"],
                    },
                },
                {
                    "kind": "PersistentVolumeClaim",
                    "apiVersion": "v1",
                    "metadata": {
                        "name": "minio",
                        "namespace": namespace
                    },
                    "spec": {
                        "accessModes": [
                        "ReadWriteOnce"
                        ],
                        "resources": {
                        "requests": {
                            "storage": "20Gi"
                        }
                        }
                    }
                },
                {
                    "apiVersion": "apps/v1",
                    "kind": "Deployment",
                    "metadata": {
                        "labels": {
                            "app": "minio",
                            "application-crd-id": "kubeflow-pipelines",
                        },
                        "name": "minio",
                        "namespace": namespace,
                    },
                    "spec": {
                        #"replicas": 1, we want to be able to scle doen for PVC resize
                        "selector": {
                            "matchLabels": {
                                "app": "minio",
                                "application-crd-id": "kubeflow-pipelines",
                            }
                        },
                        "strategy": {"type": "Recreate"},
                        "template": {
                            "metadata": {
                                "annotations": {"sidecar.istio.io/inject": "true"},
                                "labels": {
                                    "app": "minio",
                                    "application-crd-id": "kubeflow-pipelines",
                                },
                            },
                            "spec": {
                                "serviceAccountName": "default-viewer",
                                "volumes": [
                                    {
                                        "name": "minio",
                                        "persistentVolumeClaim": {
                                            "claimName": "minio"
                                        }
                                    }
                                ],
                                "containers": [
                                    {
                                        "env": [
                                            {
                                                "name": "MINIO_ACCESS_KEY",
                                                "valueFrom": {
                                                    "secretKeyRef": {
                                                        "key": "accesskey",
                                                        "name": "mlpipeline-minio-artifact",
                                                    }
                                                },
                                            },
                                            {
                                                "name": "MINIO_SECRET_KEY",
                                                "valueFrom": {
                                                    "secretKeyRef": {
                                                        "key": "secretkey",
                                                        "name": "mlpipeline-minio-artifact",
                                                    }
                                                },
                                            },
                                            {
                                                "name": "MINIO_DEFAULT_BUCKETS",
                                                "value": "mlpipeline",
                                            },
                                        ],
                                        "volumeMounts": [
                                            {
                                                "name": "minio",
                                                "mountPath": "/data"
                                            }
                                        ],
                                        "image": "quay.io/bitnami/minio:2021",
                                        "name": "minio",
                                        "ports": [{"containerPort": 9000}],
                                        "resources": {
                                            "requests": {
                                                "cpu": "20m",
                                                "memory": "100Mi",
                                            }
                                        },
                                    }
                                ],
                            },
                        },
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "artifact-repositories",
                        "namespace": namespace,
                        "annotations": {
                            "workflows.argoproj.io/default-artifact-repository": "default-v1"
                        },
                    },
                    "data": {
                        "default-v1": 'archiveLogs: true\ns3:\n  endpoint: "minio-service:9000"\n  bucket: "mlpipeline"\n  keyFormat: "artifacts/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}"\n  insecure: true\n  accessKeySecret:\n    name: mlpipeline-minio-artifact\n    key: accesskey\n  secretKeySecret:\n    name: mlpipeline-minio-artifact\n    key: secretkey\n'
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "Service",
                    "metadata": {
                        "name": "minio-service",
                        "namespace": namespace
                    },
                    "spec": {
                        "ports": [
                            {
                                "name": "http",
                                "port": 9000,
                                "protocol": "TCP",
                                "targetPort": 9000,
                            }
                        ],
                        "selector": {"app": "minio"},
                    },
                },
                {
                    "apiVersion": "networking.istio.io/v1beta1",
                    "kind": "DestinationRule",
                    "metadata": {
                        "name": "minio",
                        "namespace": namespace
                    },
                    "spec": {
                        "host": "minio-service",
                        "trafficPolicy": {
                            "tls": {
                                "mode": "ISTIO_MUTUAL"
                            }
                        }
                    }
                },
                {
                    "apiVersion": "security.istio.io/v1beta1",
                    "kind": "AuthorizationPolicy",
                    "metadata": {
                        "labels": {
                        "application-crd-id": "kubeflow-pipelines"
                        },
                        "name": "minio-service",
                        "namespace": namespace
                    },
                    "spec": {
                        "action": "ALLOW",
                        "rules": [
                        {
                            "from": [
                            {
                                "source": {
                                "principals": [
                                    "cluster.local/ns/kubeflow/sa/ml-pipeline"
                                ]
                                }
                            }
                            ]
                        },
                        {
                            "from": [
                            {
                                "source": {
                                "principals": [
                                    "cluster.local/ns/kubeflow/sa/ml-pipeline-ui"
                                ]
                                }
                            }
                            ]
                        },
                        {}
                        ],
                        "selector": {
                        "matchLabels": {
                            "app": "minio"
                        }
                        }
                    }
                }
            ]
            print("Received request:\n", json.dumps(parent, sort_keys=True))
            print(
                "Desired resources except secrets:\n",
                json.dumps(desired_resources, sort_keys=True),
            )
            # Moved after the print argument because this is sensitive data.
            desired_resources.append(
                {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": "mlpipeline-minio-artifact",
                        "namespace": namespace,
                    },
                    "data": {
                        "accesskey": minio_access_key,
                        "secretkey": minio_secret_key,
                    },
                }
            )

            return {"status": desired_status, "children": desired_resources}

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length")))
            )
            desired = self.sync(observed["parent"], observed["children"])

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), "utf-8"))

    return HTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()

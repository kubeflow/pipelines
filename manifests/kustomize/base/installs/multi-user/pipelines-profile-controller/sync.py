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
from functools import partial



class Controller(BaseHTTPRequestHandler):
    def __init__(self, *args, visualization_server_image=None,
        visualization_server_tag=None, frontend_image=None, frontend_tag=None,
        disable_istio_sidecar=None, mlpipeline_minio_access_key=None,
        mlpipeline_minio_secret_key=None, **kwargs):
        """
        Executes a controller syncronization event using an incoming post

        Note that because HTTPServer instantiates a new Handler instance for
        each post at time of receipt (rather than invoking the same Handler
        instance for each post) and because HTTPServer has no interface for
        passing extra data at Handler instantiation (such as frontend_image),
        this Controller is typically prepared for usage with functools.partial.
        An example of this is available at
        Controller.partial_init_from_environment().

        Args:
            visualization_server_image (str): repo/imagename for the ml-pipeline-visualizationserver deployment
            visualization_server_tag (str): tag for the ml-pipeline-visualizationserver deployment
            frontend_image (str):  repo/imagename for the ml-pipeline-ui-artifact deployment
            frontend_tag (str): tag for the ml-pipeline-ui-artifact deployment
            disable_istio_sidecar (str): "true" or "false"
            mlpipeline_minio_access_key (str): access_key for pipelines minio store
            mlpipeline_minio_secret_key (str): secret_key for pipelines minio store

        Returns:
            None
        """
        self.visualization_server_image = visualization_server_image
        self.frontend_image = frontend_image
        self.visualization_server_tag = visualization_server_tag
        self.frontend_tag = frontend_tag
        self.disable_istio_sidecar = disable_istio_sidecar
        self.mlpipeline_minio_access_key = mlpipeline_minio_access_key
        self.mlpipeline_minio_secret_key = mlpipeline_minio_secret_key

        super().__init__(*args, **kwargs)

    def sync(self, parent, children):
        pipeline_enabled = parent.get("metadata", {}).get(
            "labels", {}).get("pipelines.kubeflow.org/enabled")

        if pipeline_enabled != "true":
            return {"status": {}, "children": []}

        # Compute status based on observed state.
        desired_status = {
            "kubeflow-pipelines-ready": \
                len(children["Secret.v1"]) == 1 and \
                len(children["ConfigMap.v1"]) == 1 and \
                len(children["Deployment.apps/v1"]) == 2 and \
                len(children["Service.v1"]) == 2 and \
                len(children["DestinationRule.networking.istio.io/v1alpha3"]) == 1 and \
                len(children["AuthorizationPolicy.security.istio.io/v1beta1"]) == 1 and \
                "True" or "False"
        }

        # Generate the desired child object(s).
        # parent is a namespace
        namespace = parent.get("metadata", {}).get("name")
        desired_resources = [
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
            # Visualization server related manifests below
            {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {
                    "labels": {
                        "app": "ml-pipeline-visualizationserver"
                    },
                    "name": "ml-pipeline-visualizationserver",
                    "namespace": namespace,
                },
                "spec": {
                    "selector": {
                        "matchLabels": {
                            "app": "ml-pipeline-visualizationserver"
                        },
                    },
                    "template": {
                        "metadata": {
                            "labels": {
                                "app": "ml-pipeline-visualizationserver"
                            },
                            "annotations": self.disable_istio_sidecar and {
                                "sidecar.istio.io/inject": "false"
                            } or {},
                        },
                        "spec": {
                            "containers": [{
                                "image": f"{self.visualization_server_image}:{self.visualization_server_tag}",
                                "imagePullPolicy":
                                "IfNotPresent",
                                "name":
                                "ml-pipeline-visualizationserver",
                                "ports": [{
                                    "containerPort": 8888
                                }],
                                "resources": {
                                    "requests": {
                                        "cpu": "50m",
                                        "memory": "200Mi"
                                    },
                                    "limits": {
                                        "cpu": "500m",
                                        "memory": "1Gi"
                                    },
                                }
                            }],
                            "serviceAccountName":
                            "default-editor",
                        },
                    },
                },
            },
            {
                "apiVersion": "networking.istio.io/v1alpha3",
                "kind": "DestinationRule",
                "metadata": {
                    "name": "ml-pipeline-visualizationserver",
                    "namespace": namespace,
                },
                "spec": {
                    "host": "ml-pipeline-visualizationserver",
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
                    "name": "ml-pipeline-visualizationserver",
                    "namespace": namespace,
                },
                "spec": {
                    "selector": {
                        "matchLabels": {
                            "app": "ml-pipeline-visualizationserver"
                        }
                    },
                    "rules": [{
                        "from": [{
                            "source": {
                                "principals": ["cluster.local/ns/kubeflow/sa/ml-pipeline"]
                            }
                        }]
                    }]
                }
            },
            {
                "apiVersion": "v1",
                "kind": "Service",
                "metadata": {
                    "name": "ml-pipeline-visualizationserver",
                    "namespace": namespace,
                },
                "spec": {
                    "ports": [{
                        "name": "http",
                        "port": 8888,
                        "protocol": "TCP",
                        "targetPort": 8888,
                    }],
                    "selector": {
                        "app": "ml-pipeline-visualizationserver",
                    },
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
                            "annotations": self.disable_istio_sidecar and {
                                "sidecar.istio.io/inject": "false"
                            } or {},
                        },
                        "spec": {
                            "containers": [{
                                "name":
                                "ml-pipeline-ui-artifact",
                                "image": f"{self.frontend_image}:{self.frontend_tag}",
                                "imagePullPolicy":
                                "IfNotPresent",
                                "ports": [{
                                    "containerPort": 3000
                                }],
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
                "accesskey": self.mlpipeline_minio_access_key,
                "secretkey": self.mlpipeline_minio_secret_key,
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

    @staticmethod
    def get_settings_from_environment(visualization_server_image=None, frontend_image=None,
            visualization_server_tag=None, frontend_tag=None, disable_istio_sidecar=None,
            mlpipeline_minio_access_key=None, mlpipeline_minio_secret_key=None):
        """
        Returns a dict of settings from environment variables relevant to the controller

        Environment settings can be overridden by passing them here as arguments.

        Settings are pulled from the all-caps version of the setting name.  The
        following defaults are used if those environment variables are not set
        to enable backwards compatability with previous versions of this script:
            visualization_server_image: gcr.io/ml-pipeline/visualization-server
            visualization_server_tag: value of KFP_VERSION environment variable
            frontend_image: gcr.io/ml-pipeline/frontend
            frontend_tag: value of KFP_VERSION environment variable
            disable_istio_sidecar: Required (no default)
            mlpipeline_minio_access_key: Required (no default)
            mlpipeline_minio_secret_key: Required (no default)
        """
        settings = {}
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
            visualization_server_image or \
            os.environ.get("VISUALIZATION_SERVER_TAG") or \
            os.environ["KFP_VERSION"]

        settings["frontend_tag"] = \
            visualization_server_image or \
            os.environ.get("FRONTEND_TAG") or \
            os.environ["KFP_VERSION"]

        settings["disable_istio_sidecar"] = \
            disable_istio_sidecar if disable_istio_sidecar is not None \
            else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"

        settings["mlpipeline_minio_access_key"] = \
            mlpipeline_minio_access_key or \
            base64.b64encode(bytes(os.environ.get("MINIO_ACCESS_KEY"), 'utf-8')).decode('utf-8')

        settings["mlpipeline_minio_secret_key"] = \
            mlpipeline_minio_access_key or \
            base64.b64encode(bytes(os.environ.get("MINIO_SECRET_KEY"), 'utf-8')).decode('utf-8')

        return settings

    @classmethod
    def partial_init_from_environment(cls):
        """
        Returns this class partially instantiated with settings inferred from environment variables

        Useful for building a version of this Controller that can be consumed by
        HTTPServer as a handler, but that includes all settings inferred from
        environment variables, an a testable way.
        """
        settings = cls.get_settings_from_environment()
        return partial(cls, **settings)


def serve_controller_forever(port=8080):
    controller_partial = Controller.partial_init_from_environment()
    server = HTTPServer(("", port), Controller)
    server.serve_forever()


if __name__ == "__main__":
    port = int(os.environ.get("CONTROLLER_PORT", "8080"))
    serve_controller_forever(port)

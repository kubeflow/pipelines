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
        visualization_server_image: ghcr.io/kubeflow/kfp-visualization-server
        visualization_server_tag: value of KFP_VERSION environment variable
        frontend_image: ghcr.io/kubeflow/kfp-frontend
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
        os.environ.get("VISUALIZATION_SERVER_IMAGE", "ghcr.io/kubeflow/kfp-visualization-server")

    settings["frontend_image"] = \
        frontend_image or \
        os.environ.get("FRONTEND_IMAGE", "ghcr.io/kubeflow/kfp-frontend")

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



def fill_json_template(path, **kwargs):
    """
    Fill a JSON template with values provided as keyword arguments.

    :param path: Path to the JSON template file.
    :param kwargs: Keyword arguments representing placeholder values.
    :return: A JSON object with placeholders filled.
    """
    with open(path, "r") as file:
        loaded_template = file.read()

    if loaded_template == "":
        return []

    for key, value in kwargs.items():
        placeholder = f"{{{{{key}}}}}"  # Double curly braces for placeholders
        loaded_template = loaded_template.replace(placeholder, str(value))

    json_object = json.loads(loaded_template)

    return json_object


def load_desired_resources(*args, **kwargs):
    """Load desired resources from a JSON template file, strings with
    double curly brackets will be filled with the values provided through
    kwargs."""

    resources = fill_json_template(
        path="hooks/desired_resources.json", **kwargs
    ) + fill_json_template(
        path="hooks/additional_desired_resources.json", **kwargs
    )
    if kwargs.get("kfp_default_pipeline_root"):
        resources.append(
            {
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "kfp-launcher",
                    "namespace": kwargs.get("namespace"),
                },
                "data": {
                    "defaultPipelineRoot": kwargs.get(
                        "kfp_default_pipeline_root"
                    ),
                },
            }
        )
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


def server_factory(visualization_server_image,
                   visualization_server_tag, frontend_image, frontend_tag,
                   disable_istio_sidecar, minio_access_key,
                   minio_secret_key, kfp_default_pipeline_root=None,
                   url="", controller_port=8080):
    """
    Returns an HTTPServer populated with Handler with customized settings
    """
    class Controller(BaseHTTPRequestHandler):
        def sync(self, parent, attachments):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")
            pipeline_enabled = parent.get("metadata", {}).get(
                "labels", {}).get("pipelines.kubeflow.org/enabled")

            if pipeline_enabled != "true":
                return {"status": {}, "attachments": []}

            if disable_istio_sidecar:
                istio_sidecar_annotation = {"sidecar.istio.io/inject": "false"}
            else:
                istio_sidecar_annotation = {}

            desired_resources = load_desired_resources(
                namespace=namespace,
                kfp_default_pipeline_root=kfp_default_pipeline_root,
                istio_sidecar_annotation=istio_sidecar_annotation,
                visualization_server_image=visualization_server_image,
                visualization_server_tag=visualization_server_tag,
                frontend_image=frontend_image,
                frontend_tag=frontend_tag,
                minio_access_key=minio_access_key,
                minio_secret_key=minio_secret_key,
            )

            # Exclude secrets from the print statement
            non_secret_resources = [
                r for r in desired_resources if r["kind"] != "Secret"
            ]
            print("Received request:\n", json.dumps(parent, sort_keys=True))
            print(
                "Desired resources except secrets:\n",
                json.dumps(non_secret_resources, sort_keys=True),
            )

            desired_status = compute_desired_status(attachments, desired_resources)

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

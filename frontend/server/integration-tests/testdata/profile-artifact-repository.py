# Copyright 2026 The Kubeflow Authors
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
"""Emit an artifact repository from the actual profile controller for frontend
tests."""

from contextlib import redirect_stdout
import importlib.util
from io import StringIO
from pathlib import Path
import sys
from types import ModuleType
from types import SimpleNamespace
from unittest.mock import Mock
from unittest.mock import patch


def main():
    object_store_host, cluster_domain, namespace = sys.argv[1:]
    repository_root = Path(__file__).resolve().parents[4]
    producer_path = repository_root / (
        "manifests/kustomize/base/installs/multi-user/"
        "pipelines-profile-controller/sync.py")

    # Import-time clients are inert; any unexpected storage call must fail.
    botocore = ModuleType("botocore")
    botocore.session = ModuleType("botocore.session")
    botocore.session.get_session = Mock(
        return_value=Mock(create_client=Mock(return_value=object())))
    spec = importlib.util.spec_from_file_location("profile_sync", producer_path)
    producer = importlib.util.module_from_spec(spec)
    sys.dont_write_bytecode = True
    with patch.dict(sys.modules, {
            "botocore": botocore,
            "botocore.session": botocore.session,
    }):
        spec.loader.exec_module(producer)

    controller_env = {
        "KFP_VERSION": "test-version",
        "OBJECT_STORE_HOST": object_store_host,
        "ALLOWED_ARTIFACT_ENDPOINTS": "",
    }
    if cluster_domain:
        controller_env["CLUSTER_DOMAIN"] = cluster_domain

    # Capture the real generated handler without opening an HTTP listener.
    with patch.dict(
            producer.os.environ, controller_env, clear=True), patch.object(
                producer,
                "HTTPServer",
                side_effect=lambda address, handler: SimpleNamespace(
                    RequestHandlerClass=handler),
            ):
        server = producer.server_factory(**producer.get_settings_from_env())
    handler_type = server.RequestHandlerClass
    handler = handler_type.__new__(handler_type)
    parent = {
        "metadata": {
            "name": namespace,
            "labels": {
                "pipelines.kubeflow.org/enabled": "true"
            },
        }
    }
    attachments = {
        "Secret.v1": {
            f"{namespace}/mlpipeline-minio-artifact": {
                "kind": "Secret"
            },
        },
        "ConfigMap.v1": {},
        "Deployment.apps/v1": {},
        "Service.v1": {},
    }
    with redirect_stdout(StringIO()):
        result = handler.sync(parent, attachments)
    repository_config = next(
        child for child in result["attachments"]
        if child.get("kind") == "ConfigMap" and
        child.get("metadata", {}).get("name") == "artifact-repositories")
    print(repository_config["data"]["default-namespaced"])


if __name__ == "__main__":
    main()

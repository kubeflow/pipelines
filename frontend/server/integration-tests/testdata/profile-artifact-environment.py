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
"""Emit the actual profile controller's artifact-server environment for
tests."""

import importlib.util
import json
from pathlib import Path
import sys
from types import ModuleType
from unittest.mock import Mock
from unittest.mock import patch

producer_path = Path(__file__).resolve().parents[4] / (
    "manifests/kustomize/base/installs/multi-user/"
    "pipelines-profile-controller/sync.py")
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
print(
    json.dumps(
        producer.artifact_server_environment("tenant", ".svc.cluster.local",
                                             sys.argv[1])))

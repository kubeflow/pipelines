# Copyright 2023 The Kubeflow Authors
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

from google.protobuf import json_format
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def get_existing_kubernetes_config_as_message(
        task: 'PipelineTask') -> pb.KubernetesExecutorConfig:
    cur_k8_config_dict = task.platform_config.get('kubernetes', {})
    k8_config_msg = pb.KubernetesExecutorConfig()
    return json_format.ParseDict(cur_k8_config_dict, k8_config_msg)

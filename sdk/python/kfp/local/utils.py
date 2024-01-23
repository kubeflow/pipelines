# Copyright 2024 The Kubeflow Authors
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
"""Assorted utilities."""

from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp.pipeline_spec import pipeline_spec_pb2


def struct_to_executor_spec(
    struct: struct_pb2.Struct,
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec:
    executor_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec()
    json_format.ParseDict(json_format.MessageToDict(struct), executor_spec)
    return executor_spec

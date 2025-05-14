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

from typing import Union

from kfp.dsl import pipeline_channel
from kfp.compiler.pipeline_spec_builder import to_protobuf_value
from kfp.dsl import PipelineTask
from google.protobuf import json_format
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb

def get_existing_kubernetes_config_as_message(
        task: 'PipelineTask') -> pb.KubernetesExecutorConfig:
    cur_k8_config_dict = task.platform_config.get('kubernetes', {})
    k8_config_msg = pb.KubernetesExecutorConfig()
    return json_format.ParseDict(cur_k8_config_dict, k8_config_msg)


def parse_k8s_parameter_input(
        input_param: Union[pipeline_channel.PipelineParameterChannel, str, dict],
        task: PipelineTask,
) -> pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec:
    param_spec = pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec()

    if isinstance(input_param, (str, dict)):
        param_spec.runtime_value.constant.CopyFrom(to_protobuf_value(input_param))
    elif isinstance(input_param, pipeline_channel.PipelineParameterChannel):
        if input_param.task_name is None:
            param_spec.component_input_parameter = input_param.full_name

        else:
            param_spec.task_output_parameter.producer_task = input_param.task_name
            param_spec.task_output_parameter.output_parameter_key = input_param.name
            if input_param.task:
                task.after(input_param.task)
    else:
        raise ValueError(
            f'Argument for {"input_param"!r} must be an instance of str, dict, or PipelineChannel. '
            f'Got unknown input type: {type(input_param)!r}.'
        )

    return param_spec

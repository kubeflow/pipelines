# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from kfp.local import io
from kfp.local.orchestrator import orchestrator_utils
from kfp.local.orchestrator import task_spec_utils
from kfp.pipeline_spec import pipeline_spec_pb2


def test_artifact_sources_preserve_order():
    io_store = io.IOStore()
    io_store.put_task_output('train-svc', 'model', 'svc-model')
    io_store.put_task_output('train-xgb', 'model', 'xgb-model')
    io_store.put_parent_input('pretrained-models', ['base-model'])
    task_inputs = pipeline_spec_pb2.TaskInputsSpec()
    sources = task_inputs.artifacts['models'].artifact_sources.artifacts
    sources.add().task_output_artifact.CopyFrom(
        pipeline_spec_pb2.TaskInputsSpec.InputArtifactSpec
        .TaskOutputArtifactSpec(
            producer_task='train-svc', output_artifact_key='model'))
    sources.add().task_output_artifact.CopyFrom(
        pipeline_spec_pb2.TaskInputsSpec.InputArtifactSpec
        .TaskOutputArtifactSpec(
            producer_task='train-xgb', output_artifact_key='model'))
    sources.add().component_input_artifact = 'pretrained-models'

    arguments = orchestrator_utils.OrchestratorUtils.make_task_arguments(
        task_inputs_spec=task_inputs,
        io_store=io_store,
    )

    assert arguments['models'] == ['svc-model', 'xgb-model', 'base-model']


def test_artifact_sources_are_rewritten_for_loop_iterations():
    task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    source = task_spec.inputs.artifacts[
        'models'].artifact_sources.artifacts.add()
    source.component_input_artifact = 'pipelinechannel--models-loop-item'

    iteration = task_spec_utils.create_artifact_loop_iteration_task_spec(
        original_task_spec=task_spec,
        iteration_index=2,
        loop_task_name='models-loop',
        artifact_item_input='pipelinechannel--models-loop-item',
    )

    rewritten = iteration.inputs.artifacts['models'].artifact_sources.artifacts[
        0].component_input_artifact
    assert rewritten == 'pipelinechannel--models-loop-item-iteration-2'

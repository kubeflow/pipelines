# Copyright 2019 Google LLC
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

from ._op_to_template import _process_base_ops, _op_to_template, _outputs_to_json, _inputs_to_json
from ..dsl._container_op import BaseOp
import copy

def _op_to_metadata_templates(op: BaseOp):
    op = _process_base_ops(op)
    exec_op = copy.deepcopy(op)
    exec_op.name = op.name + '-exec'
    return [_op_to_dag_template(op), _op_to_template(exec_op), _op_to_track_template(op)]

def _op_to_track_template(processed_op: BaseOp):
    import json
    template = {'name': processed_op.name + '-track'}
    # inputs = _inputs_to_json(processed_op.outputs.values())
    # if inputs:
    template['inputs'] = {
        'parameters': [{
            'name': 'execution'
        }]
    }
    if processed_op.outputs:
        param_outputs = { output.name: '/tmp/kfp/outputs/%s' % output.name for output in processed_op.outputs.values()}
        template['outputs'] = _outputs_to_json(processed_op, processed_op.outputs,
                                            param_outputs, [])
    # actual_outputs = { output.name : '{{inputs.parameters.%s}}' % output.full_name for output in processed_op.outputs.values()}
    template['container'] = {
        'image': 'gcr.io/hongyes-ml/metadata-tool:latest',
        'args': ['track.py', '{{inputs.parameters.execution}}']
    }
    return template

def _op_to_dag_template(processed_op: BaseOp):
    track_op_name = processed_op.name + '-track'
    exec_op_name = processed_op.name + '-exec'
    template = {'name': processed_op.name}
    inputs = _inputs_to_json(processed_op.inputs)
    if inputs:
        template['inputs'] = inputs

    exec_task = {
        'name': exec_op_name,
        'template': exec_op_name
    }

    if inputs:
        parameters = []
        for param in processed_op.inputs:
            parameters.append({
                'name': param.full_name,
                'value': '{{inputs.parameters.%s}}' % param.full_name
            })
        parameters.sort(key=lambda x: x['name'])
        exec_task['arguments'] = { 'parameters': parameters}
    

    import json
    execution = {
        'inputs': {param.full_name: '{{inputs.parameters.%s}}' % param.full_name for param in processed_op.inputs},
        'outputs': {param.name: '{{tasks.%s.outputs.parameters.%s}}' % (exec_op_name, param.full_name) for param in processed_op.outputs.values()},
        'image': processed_op.container.image,
        'command': processed_op.container.command,
        'args': processed_op.container.args,
        'workflow_id': '{{workflow.id}}'
    }
    track_task = {
        'name': track_op_name,
        'template': track_op_name,
        'dependencies': [exec_op_name],
        'arguments': {'parameters': [{
            'name': 'execution',
            'value': json.dumps(execution)
        }]}
    }
    
    if processed_op.outputs:
        output_parameters = []
        # track_input_parameters = []
        for param in processed_op.outputs.values():
            # track_input_parameters.append({
            #     'name': param.full_name,
            #     'value': '{{tasks.%s.outputs.parameters.%s}}' % (exec_op_name, param.full_name)
            # })
            output_parameters.append({
                'name': param.full_name,
                'valueFrom': {
                    'parameter': '{{tasks.%s.outputs.parameters.%s}}' % (track_op_name, param.full_name)
                }
            })
        # track_input_parameters.sort(key=lambda x: x['name'])
        # track_task['arguments'] = { 'parameters': track_input_parameters }

        output_parameters.sort(key=lambda x: x['name'])
        template['outputs'] = {
            'parameters': output_parameters
        }

    template['dag'] = {'tasks': [exec_task, track_task]}
    return template
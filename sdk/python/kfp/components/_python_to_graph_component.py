# Copyright 2018 Google LLC
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

__all__ = [
    'create_graph_component_from_pipeline_func',
]


import inspect
from collections import OrderedDict

import kfp.components as comp
from kfp.components._structures import TaskSpec, ComponentSpec, TaskOutputReference, InputSpec, OutputSpec, GraphInputArgument, TaskOutputArgument, GraphImplementation, GraphSpec
from kfp.components._naming import _make_name_unique_by_adding_index, generate_unique_name_conversion_table, _sanitize_python_function_name, _convert_to_human_name


def create_graph_component_from_pipeline_func(pipeline_func) -> ComponentSpec:
    '''Converts python pipeline function to a graph component object that can be saved, shared, composed or submitted for execution.

    Example:

        producer_op = load_component(component_with_0_inputs_and_2_outputs)
        processor_op = load_component(component_with_2_inputs_and_2_outputs)

        def pipeline1(pipeline_param_1: int):
            producer_task = producer_op()
            processor_task = processor_op(pipeline_param_1, producer_task.outputs['Output 2'])

            return OrderedDict([
                ('Pipeline output 1', producer_task.outputs['Output 1']),
                ('Pipeline output 2', processor_task.outputs['Output 2']),
            ])
        
        graph_component = create_graph_component_from_pipeline_func(pipeline1)
    '''

    task_map = OrderedDict() #Preserving task order

    def task_construction_handler(task: TaskSpec):
        #Rewriting task ids so that they're same every time
        task_id = task.component_ref.spec.name or "Task"
        task_id = _make_name_unique_by_adding_index(task_id, task_map.keys(), ' ')
        for output_ref in task.outputs.values():
            output_ref.task_output.task_id = task_id
        task_map[task_id] = task

        return task #The handler is a transformation function, so it must pass the task through.

    pipeline_signature = inspect.signature(pipeline_func)
    parameters = list(pipeline_signature.parameters.values())
    parameter_names = [param.name for param in parameters]
    human_parameter_names_map = generate_unique_name_conversion_table(parameter_names, _convert_to_human_name)

    #Preparing the pipeline_func arguments 
    pipeline_func_args = {param.name: GraphInputArgument(input_name=human_parameter_names_map[param.name]) for param in parameters}

    try:
        #Setting the contextmanager to fix and catch the tasks.
        comp._components._created_task_transformation_handler.append(task_construction_handler)
        
        #Calling the pipeline_func with GraphInputArgument instances as arguments 
        pipeline_func_result = pipeline_func(**pipeline_func_args)
    finally:
        comp._components._created_task_transformation_handler.pop()

    graph_output_value_map = OrderedDict()
    if pipeline_func_result is None:
        pass
    elif isinstance(pipeline_func_result, dict):
        graph_output_value_map = pipeline_func_result
    else:
        raise TypeError('Pipeline must return outputs as OrderedDict.')
    
    #Checking the pipeline_func output object types
    for output_name, output_value in graph_output_value_map.items():
        if not isinstance(output_value, TaskOutputArgument):
            raise TypeError('Only TaskOutputArgument instances should be returned from graph component, but got "{output_name}" = "{}".'.format(output_name, str(output_value)))

    graph_output_specs = [OutputSpec(name=output_name, type=output_value.task_output.type) for output_name, output_value in graph_output_value_map.items()]
    
    def convert_inspect_empty_to_none(value):
        return value if value is not inspect.Parameter.empty else None
    graph_input_specs = [InputSpec(name=human_parameter_names_map[param.name], default=convert_inspect_empty_to_none(param.default)) for param in parameters] #TODO: Convert type annotations to component artifact types

    component_name = _convert_to_human_name(pipeline_func.__name__)
    component = ComponentSpec(
        name=component_name,
        inputs=graph_input_specs,
        outputs=graph_output_specs,
        implementation=GraphImplementation(
            graph=GraphSpec(
                tasks=task_map,
                output_values=graph_output_value_map,
            )
        )
    )
    return component

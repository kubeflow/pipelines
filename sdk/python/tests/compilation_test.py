from kfp import compiler
from kfp.dsl import Output
from kfp.dsl import pipeline
import pytest
from test_data.components import add_numbers
from test_data.components import dict_input
from test_data.components.containerized_python_component import concat_message
from test_data.datamodels.node import Node


class TestPipelineCompilation:

    @pytest.mark.parametrize('pipeline_name, output_pipeline_file_name, node', [
        ('Add Numbers', 'add_numbers.yaml',
         Node(data=(add_numbers.add_numbers, {
             'a': 6,
             'b': 4
         }))),
        ('Add Numbers No Input', 'add_numbers.yaml',
         Node(data=add_numbers.add_numbers)),
        ('Dict input', 'dict_input.yaml',
         Node(
             data=(dict_input.dict_input, {
                 'struct': {
                     'number1': 6,
                     'number2': 4
                 }
             }))),
        ('Containerized Component - Concat Message',
         'containerized_concat_message.yaml',
         Node(
             data=(concat_message, {
                 'message1': 'Hello',
                 'message2': 'World'
             }))),
    ])
    def test_compilation(self, pipeline_name, output_pipeline_file_name, node):

        if not node.next and type(node.data) != tuple:
            compiler.Compiler().compile(
                pipeline_func=node.data,
                pipeline_name=pipeline_name,
                package_path=output_pipeline_file_name)
        else:

            @pipeline(display_name=pipeline_name)
            def create_pipeline():
                self.create_components(node=node)

            compiler.Compiler().compile(
                pipeline_func=create_pipeline,
                pipeline_name=pipeline_name,
                package_path=output_pipeline_file_name)
        print('Pipeline Created')

    def create_components(self, node: Node):
        current_node = node
        comp_value = None
        if type(current_node.data) == tuple:
            comp_value = current_node.data[0](**current_node.data[1])
            current_node = current_node.next
        else:
            component = current_node.data()
        while current_node:
            current_data = current_node.data
            inputs: list = current_data.required_inputs
            if inputs:
                arguments = dict()
                for index, input in enumerate(inputs):
                    if len(inputs) == 1:
                        if type(comp_value) in [str, int]:
                            arguments[input] = comp_value
                        elif type(comp_value) == Output:
                            arguments[input] = comp_value.output
                    else:
                        arguments[input] = current_data[index]
                component = current_data(**arguments)
            else:
                component = current_data()
            current_node = current_node.next

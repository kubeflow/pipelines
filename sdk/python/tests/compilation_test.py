from typing import Union

import kfp
from kfp import compiler
from kfp.dsl import Output
from kfp.dsl import pipeline
import pytest
from test_data.components import add_numbers
from test_data.components import dict_input
from test_data.components.containerized_python_component import concat_message
from test_data.datamodels.node import Node
from test_data.datamodels.node_tree import NodeTree
import yaml


class TestPipelineCompilation:

    keys_to_ignore_for_comparison = ['displayName', 'name', 'sdkVersion']

    @pytest.mark.parametrize(
        'pipeline_display_name, pipeline_name, output_pipeline_file_name, node, expected_file',
        [
            ('Add Numbers', 'add_numbers', 'add_numbers.yaml',
             Node(data=(add_numbers.add_numbers, {
                 'a': 6,
                 'b': 4
             })), '../../../data/pipeline_files/valid/add_numbers.yaml'),
            ('Add Numbers no input',
             'add_numbers_no_input', 'add_numbers_no_input.yaml',
             Node(data=add_numbers.add_numbers), 'add_numbers_no_input.yaml'),
            ('Dict input', 'dict_input', 'dict_input.yaml',
             Node(
                 data=(dict_input.dict_input, {
                     'struct': {
                         'number1': 6,
                         'number2': 4
                     }
                 })), 'dict_input.yaml'),
            ('Containerized Component - Concat Message',
             'containerized_concat_message',
             'containerized_concat_message.yaml',
             Node(
                 data=(concat_message, {
                     'message1': 'Hello',
                     'message2': 'World'
                 })), 'containerized_concat_message.yaml'),
        ])
    def test_compilation(self, pipeline_display_name, pipeline_name,
                         output_pipeline_file_name, node, expected_file):

        print('Compiling Pipeline')
        if type(node) == Node and not node.next and type(node.data) != tuple:
            compiler.Compiler().compile(
                pipeline_func=node.data,
                pipeline_name=pipeline_name,
                package_path=output_pipeline_file_name)
        else:

            @pipeline(display_name=pipeline_display_name)
            def create_pipeline():
                self.create_components(component=node)

            compiler.Compiler().compile(
                pipeline_func=create_pipeline,
                pipeline_name=pipeline_name,
                package_path=output_pipeline_file_name)
        print('Pipeline Created')
        print(f'Parsing expected yaml {expected_file} for comparison')
        expected_yaml = self.read_yaml_file(expected_file)
        print(
            f'Parsing expected yaml {output_pipeline_file_name} for comparison')
        generated_yaml = self.read_yaml_file(output_pipeline_file_name)
        print('Verify that the generated yaml matches expected yaml or not')
        self.compare_dict(
            actual=generated_yaml,
            expected=expected_yaml,
            display_name=pipeline_display_name,
            name=pipeline_name)

    def create_single_component(self, node: Node):
        if type(node.data) == tuple:
            return node.data[0](**node.data[1])
        else:
            return node.data()

    def create_components(self, component: Union[Node, NodeTree]):
        current_node = component
        while current_node:
            if type(current_node) == NodeTree:
                for node in current_node.nodes:
                    self.create_components(node)
            comp_value = self.create_single_component(current_node)
            if not current_node.next:
                return
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

    def read_yaml_file(self, filepath) -> dict:
        with open(filepath, 'r') as file:
            try:
                yaml_data: dict = yaml.safe_load(file)
                return yaml_data
            except yaml.YAMLError as ex:
                print(f'Error parsing YAML file: {ex}')
                raise f'Could not load yaml file: {filepath} due to {ex}'

    def compare_dict(self, actual: dict, expected: dict, **kwargs):
        for key, value in expected.items():
            if type(value) == dict:
                self.compare_dict(actual[key], value, **kwargs)
            else:
                if key in self.keys_to_ignore_for_comparison:
                    if key == 'sdkVersion':
                        expected['sdkVersion'] = f'kfp-{kfp.__version__}'
                    elif key == 'displayName':
                        pipeline_display_name = kwargs['name'] if kwargs[
                            'display_name'] is None else kwargs['display_name']
                        expected[key] = pipeline_display_name
                    elif key == 'name':
                        expected[key] = kwargs['name']
                assert value == actual[
                    key], f'Value for "{key}" is not the same'

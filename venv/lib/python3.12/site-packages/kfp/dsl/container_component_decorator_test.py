# Copyright 2022 The Kubeflow Authors
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

from typing import Dict, List
import unittest

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import container_component_class
from kfp.dsl import Input
from kfp.dsl import Output


class TestContainerComponentDecorator(unittest.TestCase):

    def test_func_with_no_arg(self):

        @dsl.container_component
        def hello_world() -> dsl.ContainerSpec:
            """Hello world component."""
            return dsl.ContainerSpec(
                image='python3.9',
                command=['echo', 'hello world'],
                args=[],
            )

        self.assertIsInstance(hello_world,
                              container_component_class.ContainerComponent)
        self.assertIsNone(hello_world.component_spec.inputs)

    def test_func_with_simple_io(self):

        @dsl.container_component
        def hello_world_io(
            text: str,
            text_output_path: dsl.OutputPath(str)) -> dsl.ContainerSpec:
            """Hello world component with input and output."""
            return dsl.ContainerSpec(
                image='python:3.9',
                command=['echo'],
                args=['--text', text, '--output_path', text_output_path])

        self.assertIsInstance(hello_world_io,
                              container_component_class.ContainerComponent)

    def test_func_with_artifact_io(self):

        @dsl.container_component
        def container_comp_with_artifacts(
                dataset: dsl.Input[dsl.Dataset],
                num_epochs: int,  # also as an input
                model: dsl.Output[dsl.Model],
                model_config_path: dsl.OutputPath(str),
        ):
            return dsl.ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'run.sh'],
                args=[
                    '--dataset_location',
                    dataset.path,
                    '--epochs',
                    num_epochs,
                    '--model_path',
                    model.uri,
                    '--model_config_path',
                    model_config_path,
                ])

        self.assertIsInstance(container_comp_with_artifacts,
                              container_component_class.ContainerComponent)


class TestInputValuePlaceholderIrTypeHack(unittest.TestCase):

    def test(self):

        @dsl.container_component
        def comp(
            in_artifact: Input[Artifact],
            out_artifact: Output[Artifact],
            status: dsl.PipelineTaskFinalStatus,
            string: str = 'hello',
            integer: int = 1,
            floating_pt: float = 0.1,
            boolean: bool = True,
            dictionary: Dict = {'key': 'value'},
            array: List = [1, 2, 3],
            hlist: List = [
                {
                    'k': 'v'
                },
                1,
                ['a'],
                'a',
            ],
        ):
            self.assertEqual(status._ir_type, 'STRUCT')
            self.assertEqual(string._ir_type, 'STRING')
            self.assertEqual(integer._ir_type, 'NUMBER_INTEGER')
            self.assertEqual(floating_pt._ir_type, 'NUMBER_DOUBLE')
            self.assertEqual(boolean._ir_type, 'BOOLEAN')
            self.assertEqual(dictionary._ir_type, 'STRUCT')
            self.assertEqual(array._ir_type, 'LIST')
            self.assertEqual(hlist._ir_type, 'LIST')
            self.assertFalse(hasattr(in_artifact, '_ir_type'))
            self.assertFalse(hasattr(out_artifact, '_ir_type'))
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    'echo',
                ],
                args=[
                    string,
                    integer,
                    floating_pt,
                    boolean,
                    dictionary,
                    array,
                    hlist,
                    in_artifact.path,
                    out_artifact.path,
                ],
            )

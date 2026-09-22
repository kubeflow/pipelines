# Copyright 2021 The Kubeflow Authors
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
"""Tests for the public PipelineSpec IR component loaders."""

import os
import tempfile
from typing import Optional
import unittest
from unittest import mock

from absl.testing import parameterized
from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import placeholders
import yaml


@dsl.container_component
def round_trip_component(
    dataset: dsl.Input[dsl.Dataset],
    model: dsl.Output[dsl.Model],
    count: dsl.OutputPath(int),
    message: str = 'hello',
    enabled: bool = False,
    number: int = 7,
    fraction: float = 1.5,
    items: list = ['a', 2],
    config: dict = {'key': True},
    optional_dataset: Optional[dsl.Input[dsl.Dataset]] = None,
):
    return dsl.ContainerSpec(
        image='alpine',
        command=['echo'],
        args=[
            message,
            enabled,
            number,
            fraction,
            items,
            config,
            dataset.path,
            dataset.uri,
            model.path,
            model.uri,
            count,
            dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER,
            dsl.IfPresentPlaceholder(
                input_name='optional_dataset',
                then=[
                    dsl.ConcatPlaceholder(['--dataset=', optional_dataset.uri])
                ],
                else_=['--no-dataset']),
        ])


class LoadYamlTests(parameterized.TestCase):

    def load(self, entrypoint, text, directory):
        if entrypoint == 'text':
            return components.load_component_from_text(text)
        if entrypoint == 'file':
            path = os.path.join(directory, 'input.yaml')
            with open(path, 'w') as f:
                f.write(text)
            return components.load_component_from_file(path)
        response = mock.Mock(content=text.encode('utf-8'))
        url = 'https://example.com/component.yaml'
        auth = ('user', 'password')
        with mock.patch(
                'kfp.components.load_yaml_utilities.requests.get',
                return_value=response) as get:
            try:
                return components.load_component_from_url(url, auth=auth)
            finally:
                get.assert_called_once_with(url, auth=auth)
                response.raise_for_status.assert_called_once_with()

    @parameterized.product(entrypoint=['text', 'file', 'url'])
    def test_rejects_implementation_container_yaml(self, entrypoint):
        text = '''name: old-component
inputs:
- {name: message, type: String, default: hello}
implementation:
  container:
    image: alpine
    args: [{inputValue: message}]
'''
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(
                    ValueError,
                    'Component YAML must use the PipelineSpec IR format.*Recompile'
            ):
                self.load(entrypoint, text, directory)

    @parameterized.product(entrypoint=['text', 'file', 'url'])
    def test_container_component_round_trip(self, entrypoint):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, 'compiled.yaml')
            compiler.Compiler().compile(round_trip_component, path)
            with open(path) as f:
                loaded = self.load(entrypoint, f.read(), directory)

            self.assertEqual(loaded.pipeline_spec,
                             round_trip_component.pipeline_spec)
            spec = loaded.component_spec
            self.assertEqual(spec.inputs,
                             round_trip_component.component_spec.inputs)
            self.assertEqual(spec.outputs,
                             round_trip_component.component_spec.outputs)
            self.assertEqual(spec.inputs['dataset'].type,
                             'system.Dataset@0.0.1')
            self.assertTrue(spec.inputs['optional_dataset'].optional)
            self.assertEqual(spec.outputs['model'].type, 'system.Model@0.0.1')
            self.assertEqual(spec.outputs['count'].type, 'Integer')
            for name, expected in {
                    'message': 'hello',
                    'enabled': False,
                    'number': 7,
                    'fraction': 1.5,
                    'items': ['a', 2],
                    'config': {
                        'key': True
                    },
            }.items():
                self.assertEqual(spec.inputs[name].default, expected)
                self.assertTrue(spec.inputs[name].optional)
            self.assertEqual(spec.implementation.container.args, [
                placeholders.convert_command_line_element_to_string(arg)
                for arg in round_trip_component.component_spec.implementation
                .container.args
            ])
            compiler.Compiler().compile(loaded, path)
            reloaded = components.load_component_from_file(path)
            self.assertEqual(reloaded.pipeline_spec, loaded.pipeline_spec)
            self.assertEqual(reloaded.component_spec, spec)

    @parameterized.product(entrypoint=['text', 'file', 'url'])
    def test_pipeline_with_platform_spec_round_trip(self, entrypoint):

        @dsl.pipeline
        def pipeline(dataset: dsl.Input[dsl.Dataset],
                     message: str = 'hello') -> dsl.Model:
            task = round_trip_component(dataset=dataset, message=message)
            task.set_env_variable('MESSAGE', 'test')
            task.platform_config['kubernetes'] = {
                'podMetadata': {
                    'labels': {
                        'test': 'round-trip'
                    }
                }
            }
            return task.outputs['model']

        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, 'compiled.yaml')
            compiler.Compiler().compile(pipeline, path)
            with open(path) as f:
                text = f.read()
            self.assertLen(list(yaml.safe_load_all(text)), 2)
            loaded = self.load(entrypoint, text, directory)
            self.assertEqual(loaded.pipeline_spec, pipeline.pipeline_spec)
            self.assertEqual(loaded.platform_spec, pipeline.platform_spec)
            self.assertEqual(loaded.component_spec.inputs,
                             pipeline.component_spec.inputs)
            self.assertEqual(loaded.component_spec.outputs,
                             pipeline.component_spec.outputs)
            compiler.Compiler().compile(loaded, path)
            reloaded = components.load_component_from_file(path)
            self.assertEqual(reloaded.pipeline_spec, loaded.pipeline_spec)
            self.assertEqual(reloaded.platform_spec, loaded.platform_spec)

    def test_gcs_url(self):
        response = mock.Mock(content=b'not a pipeline')
        with mock.patch(
                'kfp.components.load_yaml_utilities.requests.get',
                return_value=response
        ) as get, mock.patch(
                'kfp.components.load_yaml_utilities.load_component_from_text'
        ) as load:
            components.load_component_from_url('gs://bucket/component.yaml')
        get.assert_called_once_with(
            'https://storage.googleapis.com/bucket/component.yaml', auth=None)
        load.assert_called_once_with('not a pipeline')

    def test_http_error_is_not_loaded(self):
        response = mock.Mock()
        response.raise_for_status.side_effect = RuntimeError('HTTP error')
        with mock.patch(
                'kfp.components.load_yaml_utilities.requests.get',
                return_value=response):
            with self.assertRaisesRegex(RuntimeError, 'HTTP error'):
                components.load_component_from_url(
                    'https://example.com/component.yaml')


if __name__ == '__main__':
    unittest.main()

# Copyright 2026 The Kubeflow Authors
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
"""Compile shared component fixtures without handwritten component YAML."""

import importlib
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest

from kfp import compiler
from kfp import components
from kfp import dsl


class ComponentMigrationTest(unittest.TestCase):

    def test_container_fixtures_compile_and_reload_ir(self):
        pipelines = {
            'critical.producer_consumer_param':
                'producer_consumer_param_pipeline',
            'critical.pipeline_with_env':
                'my_pipeline',
            'essential.pipeline_with_after':
                'my_pipeline',
            'essential.pipeline_with_if_placeholder':
                'pipeline_none',
            'essential.pipeline_with_nested_conditions_yaml':
                'my_pipeline',
            'parallel_and_nested.pipeline_in_pipeline_complex':
                'my_pipeline',
            'pipeline_with_component_from_text':
                'pipeline_with_env',
            'pipeline_with_concat_placeholder':
                'pipeline_with_concat_placeholder',
            'pipeline_with_importer_and_gcpc_types':
                'my_pipeline',
            'pipeline_with_task_final_status_yaml':
                'my_pipeline',
            'pipeline_with_various_io_types':
                'my_pipeline',
            'two_step_pipeline':
                'my_pipeline',
            'two_step_with_uri_placeholder':
                'two_step_with_uri_placeholder',
            'xgboost_sample_pipeline':
                'xgboost_pipeline',
        }
        with tempfile.TemporaryDirectory() as directory:
            for module_name, pipeline_name in pipelines.items():
                with self.subTest(module=module_name):
                    module = importlib.import_module(
                        'test_data.sdk_compiled_pipelines.valid.' + module_name)
                    pipeline = getattr(module, pipeline_name)
                    output = Path(directory) / 'pipeline.yaml'
                    compiler.Compiler().compile(pipeline, str(output))
                    loaded = components.load_component_from_file(str(output))
                    self.assertEqual(loaded.component_spec.name,
                                     pipeline.component_spec.name)
                    self.assertTrue(loaded.pipeline_spec.root.dag.tasks)

    def test_browser_fixture_emits_the_expected_log_message(self):
        source = (
            Path(__file__).resolve().parents[1] / 'test' /
            'frontend-integration-test' / 'compile_fixtures.py')
        spec = importlib.util.spec_from_file_location('browser_fixtures',
                                                      source)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        container = module.helloworld.pipeline_spec.deployment_spec[
            'executors']['exec-echo']['container']
        for message in ('Hello world in test', 'spaces and "quotes"; $literal'):
            with self.subTest(message=message):
                args = [
                    arg.replace("{{$.inputs.parameters['message']}}",
                                message).replace(
                                    "{{$.inputs.parameters['node']}}", 'A')
                    for arg in container['args']
                ]
                result = subprocess.run(
                    list(container['command']) + args,
                    check=True,
                    capture_output=True,
                    text=True)
                self.assertEqual(result.stdout, message + ' from node: A\n')

    def test_env_overrides_survive_container_conversion(self):
        from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_env import my_pipeline
        executors = my_pipeline.pipeline_spec.deployment_spec['executors']
        env = executors['exec-print-env']['container']['env']
        self.assertEqual({entry['name']: entry['value'] for entry in env}, {
                              'ENV2': 'val2',
                              'ENV3': 'val3'
                          })

    def test_gcpc_containers_preserve_typed_optional_artifacts(self):
        from google_cloud_pipeline_components._implementation.starry_net.evaluation.component import evaluation
        from google_cloud_pipeline_components._implementation.starry_net.upload_model.component import upload_model
        self.assertEqual(evaluation.component_spec.inputs['model'].type,
                         'google.VertexModel@0.0.1')
        self.assertTrue(evaluation.component_spec.inputs['model'].optional)
        self.assertEqual(upload_model.component_spec.outputs['model'].type,
                         'google.VertexModel@0.0.1')
        self.assertTrue(
            any(
                isinstance(arg, dsl.IfPresentPlaceholder) for arg in
                upload_model.component_spec.implementation.container.args))


if __name__ == '__main__':
    unittest.main()

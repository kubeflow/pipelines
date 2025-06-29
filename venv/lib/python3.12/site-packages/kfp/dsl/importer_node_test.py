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
import unittest

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import importer_node


class TestImporterSupportsDynamicMetadata(unittest.TestCase):

    def test_dynamic_dict_element_from_pipeline_param(self):

        @dsl.pipeline()
        def my_pipeline(meta_inp: str):

            dataset = importer_node.importer(
                'gs://my_bucket',
                Dataset,
                reimport=True,
                metadata={
                    'string': meta_inp,
                    'string-2': meta_inp
                })

        pipeline_spec = my_pipeline.pipeline_spec
        input_keys = list(pipeline_spec.components['comp-importer']
                          .input_definitions.parameters.keys())
        self.assertIn('metadata', input_keys)
        deployment_spec = pipeline_spec.deployment_spec.fields[
            'executors'].struct_value.fields['exec-importer']
        metadata = deployment_spec.struct_value.fields[
            'importer'].struct_value.fields['metadata']
        self.assertEqual(metadata.struct_value.fields['string'].string_value,
                         "{{$.inputs.parameters['metadata']}}")
        self.assertEqual(metadata.struct_value.fields['string-2'].string_value,
                         "{{$.inputs.parameters['metadata']}}")

    def test_dynamic_list_element_from_pipeline_param(self):

        @dsl.pipeline()
        def my_pipeline(meta_inp1: str, meta_inp2: int):

            dataset = importer_node.importer(
                'gs://my_bucket',
                Dataset,
                reimport=True,
                metadata={
                    'outer_key': [meta_inp1, meta_inp2],
                    meta_inp1: meta_inp1
                })

        pipeline_spec = my_pipeline.pipeline_spec
        input_keys = list(pipeline_spec.components['comp-importer']
                          .input_definitions.parameters.keys())
        self.assertIn('metadata', input_keys)
        deployment_spec = pipeline_spec.deployment_spec.fields[
            'executors'].struct_value.fields['exec-importer']
        metadata = deployment_spec.struct_value.fields[
            'importer'].struct_value.fields['metadata']
        self.assertEqual(
            metadata.struct_value.fields['outer_key'].list_value.values[0]
            .string_value, "{{$.inputs.parameters['metadata']}}")
        self.assertEqual(
            metadata.struct_value.fields['outer_key'].list_value.values[1]
            .string_value, "{{$.inputs.parameters['metadata-2']}}")
        self.assertEqual(
            metadata.struct_value.fields["{{$.inputs.parameters['metadata']}}"]
            .string_value, "{{$.inputs.parameters['metadata']}}")

    def test_dynamic_dict_element_from_task_output(self):

        @dsl.component
        def string_task(string: str) -> str:
            return 'string'

        @dsl.pipeline()
        def my_pipeline():
            task1 = string_task(string='string1')
            task2 = string_task(string='string2')
            dataset = importer_node.importer(
                'gs://my_bucket',
                Dataset,
                reimport=True,
                metadata={
                    'string-1': task1.output,
                    'string-2': task2.output
                })

        pipeline_spec = my_pipeline.pipeline_spec
        input_keys = list(pipeline_spec.components['comp-importer']
                          .input_definitions.parameters.keys())
        self.assertIn('metadata', input_keys)
        self.assertIn('metadata-2', input_keys)

        deployment_spec = pipeline_spec.deployment_spec.fields[
            'executors'].struct_value.fields['exec-importer']
        metadata = deployment_spec.struct_value.fields[
            'importer'].struct_value.fields['metadata']
        self.assertEqual(metadata.struct_value.fields['string-1'].string_value,
                         "{{$.inputs.parameters['metadata']}}")
        self.assertEqual(metadata.struct_value.fields['string-2'].string_value,
                         "{{$.inputs.parameters['metadata-2']}}")

    def test_dynamic_list_element_from_task_output(self):

        @dsl.component
        def string_task() -> str:
            return 'string'

        @dsl.pipeline()
        def my_pipeline():
            task = string_task()
            dataset = importer_node.importer(
                'gs://my_bucket',
                Dataset,
                reimport=True,
                metadata={
                    'outer_key': [task.output, task.output],
                    task.output: task.output
                })

        pipeline_spec = my_pipeline.pipeline_spec
        input_keys = list(pipeline_spec.components['comp-importer']
                          .input_definitions.parameters.keys())
        self.assertIn('metadata', input_keys)
        self.assertNotIn('metadata-2', input_keys)
        deployment_spec = pipeline_spec.deployment_spec.fields[
            'executors'].struct_value.fields['exec-importer']
        metadata = deployment_spec.struct_value.fields[
            'importer'].struct_value.fields['metadata']
        self.assertEqual(
            metadata.struct_value.fields['outer_key'].list_value.values[0]
            .string_value, "{{$.inputs.parameters['metadata']}}")
        self.assertEqual(
            metadata.struct_value.fields['outer_key'].list_value.values[1]
            .string_value, "{{$.inputs.parameters['metadata']}}")
        self.assertEqual(
            metadata.struct_value.fields["{{$.inputs.parameters['metadata']}}"]
            .string_value, "{{$.inputs.parameters['metadata']}}")

    def test_dynamic_fstring_in_metadata(self):

        @dsl.component
        def string_task() -> str:
            return 'string'

        @dsl.pipeline()
        def my_pipeline(integer: int = 1):
            task = string_task()
            dataset = importer_node.importer(
                'gs://my_bucket',
                Dataset,
                reimport=True,
                metadata={
                    f'prefix1-{integer}': f'prefix2-{task.output}',
                    'key': 'value'
                })

        pipeline_spec = my_pipeline.pipeline_spec
        input_keys = list(pipeline_spec.components['comp-importer']
                          .input_definitions.parameters.keys())
        self.assertIn('metadata', input_keys)
        self.assertIn('metadata-2', input_keys)
        self.assertNotIn('metadata-3', input_keys)
        deployment_spec = pipeline_spec.deployment_spec.fields[
            'executors'].struct_value.fields['exec-importer']
        metadata = deployment_spec.struct_value.fields[
            'importer'].struct_value.fields['metadata']
        self.assertEqual(
            metadata.struct_value.fields[
                "prefix1-{{$.inputs.parameters[\'metadata\']}}"].string_value,
            "prefix2-{{$.inputs.parameters[\'metadata-2\']}}")
        self.assertEqual(metadata.struct_value.fields['key'].string_value,
                         'value')

    def test_uri_from_loop(self):

        @dsl.component
        def make_args() -> list:
            return [{'uri': 'gs://foo', 'key': 'foo'}]

        @dsl.pipeline
        def my_pipeline():
            with dsl.ParallelFor(make_args().output) as data:
                dsl.importer(
                    artifact_uri=data.uri,
                    artifact_class=Dataset,
                    metadata={'metadata_key': data.key})

        self.assertEqual(
            my_pipeline.pipeline_spec.deployment_spec['executors']
            ['exec-importer']['importer']['artifactUri']['runtimeParameter'],
            'uri')
        self.assertEqual(
            my_pipeline.pipeline_spec.deployment_spec['executors']
            ['exec-importer']['importer']['metadata']['metadata_key'],
            "{{$.inputs.parameters[\'metadata\']}}")
        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-for-loop-1'].dag
            .tasks['importer'].inputs.parameters['metadata']
            .component_input_parameter,
            'pipelinechannel--make-args-Output-loop-item')
        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-for-loop-1'].dag
            .tasks['importer'].inputs.parameters['metadata']
            .parameter_expression_selector,
            'parseJson(string_value)["key"]',
        )

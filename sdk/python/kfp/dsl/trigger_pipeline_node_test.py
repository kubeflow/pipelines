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
import unittest

from kfp import compiler
from kfp import dsl
from kfp.dsl import trigger_pipeline_node


class TestTriggerPipeline(unittest.TestCase):

    def test_rejects_empty_pipeline_name(self):
        with self.assertRaisesRegex(ValueError, 'pipeline_name'):
            trigger_pipeline_node.trigger_pipeline(pipeline_name='')

    def test_rejects_non_positive_poke_interval(self):
        with self.assertRaisesRegex(ValueError, 'poke_interval'):
            trigger_pipeline_node.trigger_pipeline(
                pipeline_name='child', poke_interval_seconds=0)

    def test_compiled_ir_contains_trigger_pipeline(self):

        @dsl.pipeline(name='parent-with-trigger')
        def parent_pipeline(model_name: str = 'm1'):
            task = dsl.trigger_pipeline(
                pipeline_name='get-sasrec-recommendations',
                arguments={
                    'model_name': model_name,
                    'batch_size': 16,
                },
                pipeline_version_id='',
                wait_for_completion=True,
                poke_interval_seconds=30,
            )
            # Ensure outputs are wired
            assert task.outputs['run_id'] is not None
            assert task.outputs['state'] is not None

        pipeline_spec = parent_pipeline.pipeline_spec
        deployment = pipeline_spec.deployment_spec
        executors = deployment.fields['executors'].struct_value.fields
        trigger_executors = [
            name for name, spec in executors.items()
            if 'triggerPipeline' in spec.struct_value.fields or
            'trigger_pipeline' in spec.struct_value.fields
        ]
        self.assertTrue(
            trigger_executors,
            msg=f'expected triggerPipeline executor, got keys={list(executors)}')

        exec_name = trigger_executors[0]
        trigger_spec = executors[exec_name].struct_value.fields.get(
            'triggerPipeline') or executors[exec_name].struct_value.fields.get(
                'trigger_pipeline')
        fields = trigger_spec.struct_value.fields
        self.assertEqual(fields['pipelineName'].string_value,
                         'get-sasrec-recommendations')
        self.assertTrue(fields['waitForCompletion'].bool_value)
        self.assertEqual(fields['pokeIntervalSeconds'].number_value, 30)

        # Component has parameter inputs from arguments + string outputs
        comp_keys = [
            k for k in pipeline_spec.components.keys()
            if 'trigger' in k
        ]
        self.assertTrue(comp_keys)
        comp = pipeline_spec.components[comp_keys[0]]
        self.assertIn('model_name',
                      comp.input_definitions.parameters)
        self.assertIn('batch_size',
                      comp.input_definitions.parameters)
        self.assertIn('run_id',
                      comp.output_definitions.parameters)
        self.assertIn('state',
                      comp.output_definitions.parameters)

    def test_export_via_dsl(self):
        self.assertTrue(callable(dsl.trigger_pipeline))

    def test_compiler_yaml_smoke(self):
        @dsl.pipeline(name='parent-compile-smoke')
        def parent():
            dsl.trigger_pipeline(
                pipeline_name='child-pipeline',
                arguments={'x': 'y'},
            )

        import tempfile
        import os
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, 'p.yaml')
            compiler.Compiler().compile(parent, package_path=path)
            with open(path) as f:
                text = f.read()
            self.assertIn('triggerPipeline', text)
            self.assertIn('child-pipeline', text)
            self.assertNotIn('system-importer', text)


if __name__ == '__main__':
    unittest.main()

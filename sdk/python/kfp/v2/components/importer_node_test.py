# Copyright 2020 The Kubeflow Authors
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

import os
import tempfile
import unittest

import yaml
from absl.testing import parameterized
from google.protobuf import json_format
from kfp.dsl import _pipeline_param
from kfp.pipeline_spec import pipeline_spec_pb2 as pb
from kfp.v2 import compiler, dsl
from kfp.v2.components import importer_node


class ImporterNodeTest(parameterized.TestCase):

    @parameterized.parameters(
        {
            # artifact_uri is a constant value
            'input_uri':
                'gs://artifact',
            'artifact_type_schema':
                pb.ArtifactTypeSchema(schema_title='system.Dataset'),
            'metadata': {
                'key': 'value'
            },
            'expected_result': {
                'artifactUri': {
                    'constantValue': {
                        'stringValue': 'gs://artifact'
                    }
                },
                'metadata': {
                    'key': 'value'
                },
                'typeSchema': {
                    'schemaTitle': 'system.Dataset'
                }
            }
        },
        {
            # artifact_uri is from PipelineParam
            'input_uri':
                _pipeline_param.PipelineParam(name='uri_to_import'),
            'artifact_type_schema':
                pb.ArtifactTypeSchema(schema_title='system.Model'),
            'metadata': {},
            'expected_result': {
                'artifactUri': {
                    'runtimeParameter': 'uri'
                },
                'typeSchema': {
                    'schemaTitle': 'system.Model'
                }
            },
        })
    def test_build_importer_spec(self, input_uri, artifact_type_schema,
                                 metadata, expected_result):
        expected_importer_spec = pb.PipelineDeploymentConfig.ImporterSpec()
        json_format.ParseDict(expected_result, expected_importer_spec)
        importer_spec = importer_node._build_importer_spec(
            artifact_uri=input_uri,
            artifact_type_schema=artifact_type_schema,
            metadata_with_placeholders=metadata)

        self.maxDiff = None
        self.assertEqual(expected_importer_spec, importer_spec)

    @parameterized.parameters(
        {
            # artifact_uri is a constant value
            'importer_name': 'importer-1',
            'input_uri': 'gs://artifact',
            'expected_result': {
                'inputs': {
                    'parameters': {
                        'uri': {
                            'runtimeValue': {
                                'constantValue': {
                                    'stringValue': 'gs://artifact'
                                }
                            }
                        }
                    }
                },
                'componentRef': {
                    'name': 'comp-importer-1'
                },
            }
        },
        {
            # artifact_uri is from PipelineParam
            'importer_name': 'importer-2',
            'input_uri': _pipeline_param.PipelineParam(name='uri_to_import'),
            'expected_result': {
                'inputs': {
                    'parameters': {
                        'uri': {
                            'componentInputParameter': 'uri_to_import'
                        }
                    }
                },
                'componentRef': {
                    'name': 'comp-importer-2'
                },
            },
        })
    def test_build_importer_task_spec(self, importer_name, input_uri,
                                      expected_result):
        expected_task_spec = pb.PipelineTaskSpec()
        json_format.ParseDict(expected_result, expected_task_spec)

        task_spec = importer_node._build_importer_task_spec(
            importer_base_name=importer_name,
            artifact_uri=input_uri,
            metadata_inputs={})

        self.maxDiff = None
        self.assertEqual(expected_task_spec, task_spec)

    def test_build_importer_component_spec(self):
        expected_importer_component = {
            'inputDefinitions': {
                'parameters': {
                    'uri': {
                        'type': 'STRING'
                    }
                }
            },
            'outputDefinitions': {
                'artifacts': {
                    'artifact': {
                        'artifactType': {
                            'schemaTitle': 'system.Artifact'
                        }
                    }
                }
            },
            'executorLabel': 'exec-importer-1'
        }
        expected_importer_comp_spec = pb.ComponentSpec()
        json_format.ParseDict(expected_importer_component,
                              expected_importer_comp_spec)
        importer_comp_spec = importer_node._build_importer_component_spec(
            importer_base_name='importer-1',
            artifact_type_schema=pb.ArtifactTypeSchema(
                schema_title='system.Artifact'),
            metadata_inputs={})

        self.maxDiff = None
        self.assertEqual(expected_importer_comp_spec, importer_comp_spec)

    def test_import_with_invalid_artifact_uri_value_should_fail(self):
        from kfp.v2.components.types.artifact_types import Dataset
        with self.assertRaisesRegex(
                ValueError,
                "Importer got unexpected artifact_uri: 123 of type: <class 'int'>."
        ):
            importer_node.importer(artifact_uri=123, artifact_class=Dataset)

    def test_dynamic_importer_metadata(self):

        @dsl.component
        def func() -> str:
            return 'string'

        @dsl.component
        def func2() -> int:
            return 1

        @dsl.pipeline(name='pipeline')
        def my_pipeline(uri: str = 'my_uri'):
            task = func()
            task2 = func2()
            dsl.importer(
                artifact_uri=uri,
                artifact_class=dsl.Artifact,
                metadata={
                    task.output: task2.output,
                    'other': [task.output, task2.output, uri]
                })

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'output.json')
            compiler.Compiler().compile(my_pipeline, package_path=package_path)
            with open(package_path) as f:
                pipeline_spec_dict = yaml.safe_load(f.read())['pipelineSpec']

        pipeline_spec = pb.PipelineSpec()
        json_format.ParseDict(pipeline_spec_dict, pipeline_spec)

        # test uri parameter
        self.assertEqual(
            pipeline_spec.deployment_spec['executors']['exec-importer']
            ['importer']['artifactUri'].fields['runtimeParameter'].string_value,
            'uri')

        # test executor contents
        self.assertEqual(
            pipeline_spec.deployment_spec['executors']['exec-importer']
            ['importer']['metadata']["{{$.inputs.parameters[\'metadata\']}}"],
            "{{$.inputs.parameters['metadata-2']}}")
        self.assertEqual(
            pipeline_spec.deployment_spec['executors']['exec-importer']
            ['importer']['metadata']["other"].values[0].string_value,
            "{{$.inputs.parameters[\'metadata\']}}")
        self.assertEqual(
            pipeline_spec.deployment_spec['executors']['exec-importer']
            ['importer']['metadata']["other"].values[1].string_value,
            "{{$.inputs.parameters[\'metadata-2\']}}")

        # test another uri parameter in metadata to ensure no name collisions
        self.assertEqual(
            pipeline_spec.deployment_spec['executors']['exec-importer']
            ['importer']['metadata']["other"].values[2].string_value,
            "{{$.inputs.parameters[\'metadata-3\']}}")


        self.assertEqual(
            pipeline_spec.components['comp-importer'].input_definitions
            .parameters['metadata'].type, 3)
        self.assertEqual(
            pipeline_spec.components['comp-importer'].input_definitions
            .parameters['metadata-2'].type, 1)
        self.assertEqual(
            pipeline_spec.components['comp-importer'].input_definitions
            .parameters['metadata-3'].type, 3)
        self.assertEqual(
            pipeline_spec.components['comp-importer'].input_definitions
            .parameters['uri'].type, 3)

    def test_importer_with_none_metadata(self):
        x = dsl.importer(
            artifact_uri='', artifact_class=dsl.Artifact, metadata=None)
        self.assertLen(x.component_spec.input_definitions.parameters, 1)
        self.assertEqual(
            x.component_spec.input_definitions.parameters['uri'].type, 3)


if __name__ == '__main__':
    unittest.main()

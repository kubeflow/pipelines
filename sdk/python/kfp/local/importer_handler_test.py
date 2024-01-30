# Copyright 2024 The Kubeflow Authors
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
"""Tests for importer_handler_test.py."""
import unittest

from google.protobuf import json_format
from kfp import dsl
from kfp.local import importer_handler
from kfp.local import status
from kfp.local import testing_utilities
from kfp.pipeline_spec import pipeline_spec_pb2


class TestRunImporter(testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_uri_from_upstream(self):
        component_spec_dict = {
            'inputDefinitions': {
                'parameters': {
                    'metadata': {
                        'parameterType': 'STRING'
                    },
                    'uri': {
                        'parameterType': 'STRING'
                    }
                }
            },
            'outputDefinitions': {
                'artifacts': {
                    'artifact': {
                        'artifactType': {
                            'schemaTitle': 'system.Dataset',
                            'schemaVersion': '0.0.1'
                        }
                    }
                }
            },
            'executorLabel': 'exec-importer'
        }
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(component_spec_dict, component_spec)

        executor_spec_dict = {
            'importer': {
                'artifactUri': {
                    'runtimeParameter': 'uri'
                },
                'typeSchema': {
                    'schemaTitle': 'system.Dataset',
                    'schemaVersion': '0.0.1'
                },
                'metadata': {
                    'foo': "{{$.inputs.parameters['metadata']}}"
                }
            }
        }
        executor_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
        )
        json_format.ParseDict(executor_spec_dict, executor_spec)

        outputs, task_status = importer_handler.run_importer(
            pipeline_resource_name='my-pipeline-2024-01-24-15-16-30-586674',
            component_name='comp-importer',
            component_spec=component_spec,
            executor_spec=executor_spec,
            arguments={
                'metadata': 'bar',
                'uri': '/fizz/buzz'
            },
            pipeline_root='/foo/bar',
            unique_pipeline_id='19024073',
        )
        expected_artifact = dsl.Dataset(
            name='artifact',
            uri='/fizz/buzz',
            metadata={'foo': 'bar'},
        )
        self.assertEqual(outputs['artifact'].schema_title,
                         expected_artifact.schema_title)
        self.assertEqual(outputs['artifact'].name, expected_artifact.name)
        self.assertEqual(outputs['artifact'].uri, expected_artifact.uri)
        self.assertEqual(outputs['artifact'].metadata,
                         expected_artifact.metadata)
        self.assertEqual(task_status, status.Status.SUCCESS)

    def test_uri_constant(self):
        component_spec_dict = {
            'inputDefinitions': {
                'parameters': {
                    'metadata': {
                        'parameterType': 'STRING'
                    },
                    'uri': {
                        'parameterType': 'STRING'
                    }
                }
            },
            'outputDefinitions': {
                'artifacts': {
                    'artifact': {
                        'artifactType': {
                            'schemaTitle': 'system.Artifact',
                            'schemaVersion': '0.0.1'
                        }
                    }
                }
            },
            'executorLabel': 'exec-importer'
        }
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(component_spec_dict, component_spec)

        executor_spec_dict = {
            'importer': {
                'artifactUri': {
                    'constant': 'gs://path'
                },
                'typeSchema': {
                    'schemaTitle': 'system.Artifact',
                    'schemaVersion': '0.0.1'
                },
                'metadata': {
                    'foo': [
                        "{{$.inputs.parameters['metadata']}}",
                        "{{$.inputs.parameters['metadata']}}"
                    ]
                }
            }
        }
        executor_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
        )
        json_format.ParseDict(executor_spec_dict, executor_spec)

        outputs, task_status = importer_handler.run_importer(
            pipeline_resource_name='my-pipeline-2024-01-24-15-16-30-586674',
            component_name='comp-importer',
            component_spec=component_spec,
            executor_spec=executor_spec,
            arguments={
                'metadata': 'text',
                'uri': 'gs://path'
            },
            pipeline_root='/foo/bar',
            unique_pipeline_id='19024073',
        )
        expected_artifact = dsl.Artifact(
            name='artifact',
            uri='gs://path',
            metadata={'foo': ['text', 'text']},
        )
        self.assertEqual(outputs['artifact'].schema_title,
                         expected_artifact.schema_title)
        self.assertEqual(outputs['artifact'].name, expected_artifact.name)
        self.assertEqual(outputs['artifact'].uri, expected_artifact.uri)
        self.assertEqual(outputs['artifact'].metadata,
                         expected_artifact.metadata)
        self.assertEqual(task_status, status.Status.SUCCESS)


class TestGetImporterUri(unittest.TestCase):

    def test_constant(self):
        importer_spec_dict = {
            'artifactUri': {
                'constant': '/foo/path'
            },
            'typeSchema': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            },
            'metadata': {
                'foo': 'bar'
            }
        }
        importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec(
        )
        json_format.ParseDict(importer_spec_dict, importer_spec)
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'uri': '/foo/path'
                }
            },
            'outputs': {
                'artifacts': {
                    'artifact': {
                        'artifacts': [{
                            'name': 'artifact',
                            'type': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            },
                            'uri': '/pipeline_root/task/artifact',
                            'metadata': {}
                        }]
                    }
                },
                'outputFile': '/pipeline_root/task/executor_output.json'
            }
        }
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(executor_input_dict, executor_input)
        uri = importer_handler.get_importer_uri(
            importer_spec=importer_spec,
            executor_input=executor_input,
        )
        self.assertEqual(uri, '/foo/path')

    def test_runtime_parameter(self):
        importer_spec_dict = {
            'artifactUri': {
                'runtimeParameter': 'uri'
            },
            'typeSchema': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            },
            'metadata': {
                'foo': 'bar'
            }
        }
        importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec(
        )
        json_format.ParseDict(importer_spec_dict, importer_spec)
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'uri': '/fizz/buzz'
                }
            },
            'outputs': {
                'artifacts': {
                    'artifact': {
                        'artifacts': [{
                            'name': 'artifact',
                            'type': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            },
                            'uri': '/pipeline_root/foo/task/artifact',
                            'metadata': {}
                        }]
                    }
                },
                'outputFile': '/pipeline_root/task/executor_output.json'
            }
        }
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(executor_input_dict, executor_input)
        uri = importer_handler.get_importer_uri(
            importer_spec=importer_spec,
            executor_input=executor_input,
        )
        self.assertEqual(uri, '/fizz/buzz')

    def test_constant_warns(self):
        importer_spec_dict = {
            'artifactUri': {
                'constant': 'gs://foo/bar'
            },
            'typeSchema': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            },
            'metadata': {
                'foo': 'bar'
            }
        }
        importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec(
        )
        json_format.ParseDict(importer_spec_dict, importer_spec)
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'uri': 'gs://foo/bar'
                }
            },
            'outputs': {
                'artifacts': {
                    'artifact': {
                        'artifacts': [{
                            'name': 'artifact',
                            'type': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            },
                            'uri': '/pipeline_root/foo/task/artifact',
                            'metadata': {}
                        }]
                    }
                },
                'outputFile': '/pipeline_root/task/executor_output.json'
            }
        }
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(executor_input_dict, executor_input)
        with self.assertWarnsRegex(
                UserWarning,
                r"It looks like you're using the remote file 'gs://foo/bar' in a 'dsl\.importer'\. Note that you will only be able to read and write to/from local files using 'artifact\.path' in local executed pipelines\."
        ):
            uri = importer_handler.get_importer_uri(
                importer_spec=importer_spec,
                executor_input=executor_input,
            )
        self.assertEqual(uri, 'gs://foo/bar')

    def test_runtime_parameter_warns(self):
        importer_spec_dict = {
            'artifactUri': {
                'runtimeParameter': 'uri'
            },
            'typeSchema': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            },
            'metadata': {
                'foo': 'bar'
            }
        }
        importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec(
        )
        json_format.ParseDict(importer_spec_dict, importer_spec)
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'uri': 's3://fizz/buzz'
                }
            },
            'outputs': {
                'artifacts': {
                    'artifact': {
                        'artifacts': [{
                            'name': 'artifact',
                            'type': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            },
                            'uri': '/pipeline_root/foo/task/artifact',
                            'metadata': {}
                        }]
                    }
                },
                'outputFile': '/pipeline_root/task/executor_output.json'
            }
        }
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(executor_input_dict, executor_input)
        with self.assertWarnsRegex(
                UserWarning,
                r"It looks like you're using the remote file 's3://fizz/buzz' in a 'dsl\.importer'\. Note that you will only be able to read and write to/from local files using 'artifact\.path' in local executed pipelines\."
        ):
            uri = importer_handler.get_importer_uri(
                importer_spec=importer_spec,
                executor_input=executor_input,
            )
        self.assertEqual(uri, 's3://fizz/buzz')


class TestGetArtifactClassForSchemaTitle(unittest.TestCase):

    def test_artifact(self):
        actual = importer_handler.get_artifact_class_from_schema_title(
            'system.Artifact')
        expected = dsl.Artifact
        self.assertEqual(actual, expected)

    def test_classification_metrics(self):
        actual = importer_handler.get_artifact_class_from_schema_title(
            'system.ClassificationMetrics')
        expected = dsl.ClassificationMetrics
        self.assertEqual(actual, expected)

    def test_not_system_type(self):
        actual = importer_handler.get_artifact_class_from_schema_title(
            'unknown.Type')
        expected = dsl.Artifact
        self.assertEqual(actual, expected)


if __name__ == '__main__':
    unittest.main()

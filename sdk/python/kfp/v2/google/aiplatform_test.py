# Copyright 2021 Google LLC
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
"""Tests of kfp.google.aiplatform module."""
from google.protobuf import json_format
import unittest
from kfp.v2.google import aiplatform
from kfp import dsl
from kfp.dsl import io_types
from kfp.components import _structures as structures

_EXPECTED_COMPONENT_SPEC = {
    'inputDefinitions': {
        'artifacts': {
            'examples': {
                'artifactType': {
                    'schemaTitle': 'system.Dataset'}}},
        'parameters': {
            'optimizer': {'type': 'STRING'}
        }
    },
    'outputDefinitions': {
        'artifacts': {
            'model': {
                'artifactType': {
                    'schemaTitle': 'system.Model'}}},
        'parameters': {
            'out_param': {
                'type': 'STRING'}}},
    'executorLabel': 'exec-my-custom-job'}

_EXPECTED_TASK_SPEC = {
    'taskInfo': {'name': 'task-my-custom-job'},
    'inputs': {
        'parameters': {
            'optimizer': {
                'runtimeValue': {
                    'constantValue': {
                        'stringValue': 'sgd'}}}},
        'artifacts': {
            'examples': {
                'taskOutputArtifact': {
                    'producerTask': 'task-ingestor',
                    'outputArtifactKey': 'output'}}}},
    'componentRef': {
        'name': 'comp-my-custom-job'}}

class AiplatformTest(unittest.TestCase):

  def testSingleNodeCustomContainer(self):
    self.maxDiff = None
    expected_custom_job_spec = {
        'name': 'my-custom-job',
        'jobSpec': {
            'workerPoolSpecs': [
                {
                    'machineSpec': {'machineType': 'n1-standard-4'},
                    'replicaCount': '1',
                    'containerSpec': {
                        'imageUri': 'my_image:latest',
                        'command': ['python', 'entrypoint.py'],
                        'args': ['--input_path',
                                 "{{$.inputs.artifacts['examples'].uri}}",
                                 '--output_path',
                                 "{{$.outputs.artifacts['model'].uri}}",
                                 '--optimizer',
                                 "{{$.inputs.parameters['optimizer']}}",
                                 '--output_param_path',
                                 "{{$.outputs.parameters['out_param'].output_file}}"
                                 ]}}]
        }}

    task = aiplatform.custom_job(
        name='my-custom-job',
        input_artifacts={
            'examples': dsl.PipelineParam(
                name='output',
                op_name='ingestor',
                param_type='Dataset')},
        input_parameters={'optimizer': 'sgd'},
        output_artifacts={
            'model': io_types.Model},
        output_parameters={
            'out_param': str},
        image_uri='my_image:latest',
        commands=['python', 'entrypoint.py'],
        args=[
            '--input_path', structures.InputUriPlaceholder('examples'),
            '--output_path', structures.OutputUriPlaceholder('model'),
            '--optimizer', structures.InputValuePlaceholder('optimizer'),
            '--output_param_path',
            structures.OutputPathPlaceholder('out_param')
        ])
    self.assertDictEqual(expected_custom_job_spec, task.custom_job_spec)
    self.assertDictEqual(_EXPECTED_COMPONENT_SPEC,
                         json_format.MessageToDict(task.component_spec))
    self.assertDictEqual(_EXPECTED_TASK_SPEC,
                         json_format.MessageToDict(task.task_spec))

  def testSingleNodeCustomPython(self):
    expected_custom_job_spec = {
        'name': 'my-custom-job',
        'jobSpec': {
            'workerPoolSpecs': [{
                'machineSpec': {'machineType': 'n1-standard-8'},
                'replicaCount': '1',
                'pythonPackageSpec': {
                    'executorImageUri': 'my_image:latest',
                    'packageUris': ['gs://my-bucket/my-training-prgram.tar.gz'],
                    'pythonModule': 'my_trainer',
                    'args': [
                        '--input_path',
                        "{{$.inputs.artifacts['examples'].path}}",
                        '--output_path',
                        "{{$.outputs.artifacts['model'].path}}",
                        '--optimizer', "{{$.inputs.parameters['optimizer']}}",
                        '--output_param_path',
                        "{{$.outputs.parameters['out_param'].output_file}}"]}}]}}

    task = aiplatform.custom_job(
        name='my-custom-job',
        input_artifacts={
            'examples': dsl.PipelineParam(
                name='output',
                op_name='ingestor',
                param_type='Dataset')},
        input_parameters={'optimizer': 'sgd'},
        output_artifacts={
            'model': io_types.Model},
        output_parameters={
            'out_param': str},
        executor_image_uri='my_image:latest',
        package_uris=['gs://my-bucket/my-training-prgram.tar.gz'],
        python_module='my_trainer',
        args=[
            '--input_path', structures.InputPathPlaceholder('examples'),
            '--output_path', structures.OutputPathPlaceholder('model'),
            '--optimizer', structures.InputValuePlaceholder('optimizer'),
            '--output_param_path',
            structures.OutputPathPlaceholder('out_param')
        ],
        machine_type='n1-standard-8',
    )
    self.assertDictEqual(expected_custom_job_spec, task.custom_job_spec)
    self.assertDictEqual(_EXPECTED_COMPONENT_SPEC ,
                         json_format.MessageToDict(task.component_spec))
    self.assertDictEqual(_EXPECTED_TASK_SPEC,
                         json_format.MessageToDict(task.task_spec))

  def testAdvancedTrainingSpec(self):
    expected_custom_job_spec = {
        'name': 'my-custom-job',
        'jobSpec': {
            "workerPoolSpecs": [
                {
                    "replicaCount": 1,
                    "containerSpec": {
                        "imageUri": "my-master-image:latest",
                        "command": [
                            "python3",
                            "master_entrypoint.py"
                        ],
                        "args": [
                            "--input_path",
                            "{{$.inputs.artifacts['examples'].path}}",
                            "--output_path",
                            "{{$.outputs.artifacts['model'].path}}",
                            "--optimizer",
                            "{{$.inputs.parameters['optimizer']}}",
                            "--output_param_path",
                            "{{$.outputs.parameters['out_param'].output_file}}"
                        ]
                    },
                    "machineSpec": {
                        "machineType": "n1-standard-4"
                    }
                },
                {
                    "replicaCount": 4,
                    "containerSpec": {
                        "imageUri": "gcr.io/my-project/my-worker-image:latest",
                        "command": [
                            "python3",
                            "worker_entrypoint.py"
                        ],
                        "args": [
                            "--input_path",
                            "{{$.inputs.artifacts['examples'].path}}",
                            "--output_path",
                            "{{$.outputs.artifacts['model'].path}}",
                            "--optimizer",
                            "{{$.inputs.parameters['optimizer']}}"
                        ]
                    },
                    "machineSpec": {
                        "machineType": "n1-standard-4",
                        "acceleratorType": "NVIDIA_TESLA_K80",
                        "acceleratorCount": 1
                    }
                }
            ]
        }}
    task = aiplatform.custom_job(
        name='my-custom-job',
        input_artifacts={
            'examples': dsl.PipelineParam(
                name='output',
                op_name='ingestor',
                param_type='Dataset')},
        input_parameters={'optimizer': 'sgd'},
        output_artifacts={
            'model': io_types.Model},
        output_parameters={
            'out_param': str},
        additional_job_spec={
            'workerPoolSpecs': [
                {
                    'replicaCount': 1,
                    'containerSpec': {
                        'imageUri': 'my-master-image:latest',
                        'command': ['python3', 'master_entrypoint.py'],
                        'args': [
                            '--input_path',
                            structures.InputPathPlaceholder('examples'),
                            '--output_path',
                            structures.OutputPathPlaceholder('model'),
                            '--optimizer',
                            structures.InputValuePlaceholder('optimizer'),
                            '--output_param_path',
                            structures.OutputPathPlaceholder('out_param')
                        ],
                    },
                    'machineSpec': {'machineType': 'n1-standard-4'}
                },
                {
                    'replicaCount': 4,
                    'containerSpec': {
                        'imageUri': 'gcr.io/my-project/my-worker-image:latest',
                        'command': ['python3', 'worker_entrypoint.py'],
                        'args': [
                            '--input_path',
                            structures.InputPathPlaceholder('examples'),
                            '--output_path',
                            structures.OutputPathPlaceholder('model'),
                            '--optimizer',
                            structures.InputValuePlaceholder('optimizer')
                        ]
                    },
                    # Optionally one can also attach accelerators.
                    'machineSpec': {
                        'machineType': 'n1-standard-4',
                        'acceleratorType': 'NVIDIA_TESLA_K80',
                        'acceleratorCount': 1
                    }}]})
    self.assertDictEqual(expected_custom_job_spec, task.custom_job_spec)
    self.assertDictEqual(_EXPECTED_COMPONENT_SPEC ,
                         json_format.MessageToDict(task.component_spec))
    self.assertDictEqual(_EXPECTED_TASK_SPEC,
                         json_format.MessageToDict(task.task_spec))

  def testScaffoldProgramToSpecs(self):
    expected_custom_job_spec = {
        'name': 'my-custom-job',
        'jobSpec': {
            "workerPoolSpecs": [
                {
                    "replicaCount": 1,
                    "machineSpec": {
                        "machineType": "n1-standard-4"
                    },
                    "containerSpec": {
                        "imageUri": "my_image:latest",
                        "command": [
                            "python",
                            "entrypoint.py"
                        ],
                        "args": [
                            "--input_path",
                            "{{$.inputs.artifacts['examples'].uri}}",
                            "--output_path",
                            "{{$.outputs.artifacts['model'].uri}}",
                            "--optimizer",
                            "{{$.inputs.parameters['optimizer']}}",
                            "--output_param_path",
                            "{{$.outputs.parameters['out_param'].output_file}}"
                        ]
                    }
                },
                {
                    "replicaCount": 4,
                    "containerSpec": {
                        "imageUri": "gcr.io/my-project/my-worker-image:latest",
                        "command": [
                            "python3",
                            "override_entrypoint.py"
                        ],
                        "args": [
                            "--arg1",
                            "param1"
                        ]
                    },
                    "machineSpec": {
                        "machineType": "n1-standard-8",
                        "acceleratorType": "NVIDIA_TESLA_K80",
                        "acceleratorCount": 1
                    }
                }
            ]
        }}
    task = aiplatform.custom_job(
        name='my-custom-job',
        input_artifacts={
            'examples': dsl.PipelineParam(
                name='output',
                op_name='ingestor',
                param_type='Dataset')},
        input_parameters={'optimizer': 'sgd'},
        output_artifacts={
            'model': io_types.Model},
        output_parameters={
            'out_param': str},
        image_uri='my_image:latest',
        commands=['python', 'entrypoint.py'],
        args=[
            '--input_path', structures.InputUriPlaceholder('examples'),
            '--output_path', structures.OutputUriPlaceholder('model'),
            '--optimizer', structures.InputValuePlaceholder('optimizer'),
            '--output_param_path',
            structures.OutputPathPlaceholder('out_param')
        ],
        additional_job_spec={
            'workerPoolSpecs': [
                {
                    'replicaCount': 1,
                    'machineSpec': {'machineType': 'n1-standard-4'}
                },
                {
                    'replicaCount': 4,
                    'containerSpec': {
                        'imageUri': 'gcr.io/my-project/my-worker-image:latest',
                        'command': ['python3', 'override_entrypoint.py'],
                        'args': ['--arg1', 'param1']
                    },
                    # Optionally one can also attach accelerators.
                    'machineSpec': {
                        'machineType': 'n1-standard-8',
                        'acceleratorType': 'NVIDIA_TESLA_K80',
                        'acceleratorCount': 1
                    }}]
        }
    )
    self.assertDictEqual(expected_custom_job_spec, task.custom_job_spec)
    self.assertDictEqual(_EXPECTED_COMPONENT_SPEC ,
                         json_format.MessageToDict(task.component_spec))
    self.assertDictEqual(_EXPECTED_TASK_SPEC,
                         json_format.MessageToDict(task.task_spec))

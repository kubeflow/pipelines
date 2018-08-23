# Copyright 2018 Google LLC
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


import mlp
import datetime


@mlp.pipeline(
  name='TFMA Taxi Cab Classification Pipeline Example',
  description='Example pipeline that does classification with model analysis based on a public BigQuery dataset.'
)
def taxi_cab_classification(
    output: mlp.PipelineParam,
    project: mlp.PipelineParam,

    schema: mlp.PipelineParam=mlp.PipelineParam(
        name='schema',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json'),
    train: mlp.PipelineParam=mlp.PipelineParam(
        name='train',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv'),
    evaluation: mlp.PipelineParam=mlp.PipelineParam(
        name='evaluation',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv'),
    preprocess_mode: mlp.PipelineParam=mlp.PipelineParam(
        name='preprocess-mode', value='local'),
    preprocess_module: mlp.PipelineParam=mlp.PipelineParam(
        name='preprocess-module',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py'),
    target: mlp.PipelineParam=mlp.PipelineParam(
        name='target', value='tips'),
    learning_rate: mlp.PipelineParam=mlp.PipelineParam(name='learning-rate', value=0.1),
    hidden_layer_size: mlp.PipelineParam=mlp.PipelineParam(name='hidden-layer-size', value='1500'),
    steps: mlp.PipelineParam=mlp.PipelineParam(name='steps', value=3000),
    workers: mlp.PipelineParam=mlp.PipelineParam(name='workers', value=0),
    pss: mlp.PipelineParam=mlp.PipelineParam(name='pss', value=0),
    predict_mode: mlp.PipelineParam=mlp.PipelineParam(name='predict-mode', value='local'),
    analyze_mode: mlp.PipelineParam=mlp.PipelineParam(name='analyze-mode', value='local'),
    analyze_slice_column: mlp.PipelineParam=mlp.PipelineParam(
        name='analyze-slice-column', value='trip_start_hour')):
  transform_output = '%s/{{workflow.name}}/transformed' % output
  training_output = '%s/{{workflow.name}}/train' % output
  analysis_output = '%s/{{workflow.name}}/analysis' % output
  prediction_output = '%s/{{workflow.name}}/predict' % output

  preprocess = mlp.ContainerOp(
      name = 'preprocess',
      image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft',
      arguments = ['--train', train, '--eval', evaluation, '--schema', schema, '--output', transform_output,
                   '--project', project, '--mode', preprocess_mode, '--preprocessing-module', preprocess_module,
      ],
      file_outputs = {'transformed': '/output.txt'})

  training = mlp.ContainerOp(
      name = 'training',
      image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf',
      arguments = ['--job-dir', training_output, '--transformed-data-dir', preprocess.output,
                   '--schema', schema, '--learning-rate', learning_rate, '--hidden-layer-size', hidden_layer_size,
                   '--steps', steps, '--target', target, '--workers', workers, '--pss', pss,
                   '--preprocessing-module', preprocess_module, '--tfjob-timeout-minutes', 60
      ],
      file_outputs = {'train': '/output.txt'})

  analysis = mlp.ContainerOp(
      name = 'analysis',
      image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma',
      arguments = ['--output', analysis_output, '--model', training.output, '--eval', evaluation, '--schema', schema,
                   '--project', project, '--mode', analyze_mode, '--slice-columns', analyze_slice_column,
      ],
      file_outputs = {'analysis': '/output.txt'})

  prediction = mlp.ContainerOp(
      name = 'prediction',
      image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict',
      arguments = ['--output', prediction_output, '--data', evaluation, '--schema', schema,
                   '--target', target, '--model',  training.output, '--mode', predict_mode, '--project', project],
      file_outputs = {'predict': '/output.txt'})

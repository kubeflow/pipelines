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
  name='Pipeline TFJob',
  description='Demonstrate the DSL for TFJob'
)
def kubeflow_training( output: mlp.PipelineParam, project: mlp.PipelineParam,
  evaluation: mlp.PipelineParam=mlp.PipelineParam(name='evaluation', value='gs://ml-pipeline-playground/flower/eval100.csv'),
  train: mlp.PipelineParam=mlp.PipelineParam(name='train', value='gs://ml-pipeline-playground/flower/train200.csv'),
  schema: mlp.PipelineParam=mlp.PipelineParam(name='schema', value='gs://ml-pipeline-playground/flower/schema.json'),
  learning_rate: mlp.PipelineParam=mlp.PipelineParam(name='learningrate', value=0.1),
  hidden_layer_size: mlp.PipelineParam=mlp.PipelineParam(name='hiddenlayersize', value='100,50'),
  steps: mlp.PipelineParam=mlp.PipelineParam(name='steps', value=2000),
  target: mlp.PipelineParam=mlp.PipelineParam(name='target', value='label'),
  workers: mlp.PipelineParam=mlp.PipelineParam(name='workers', value=0),
  pss: mlp.PipelineParam=mlp.PipelineParam(name='pss', value=0),
  preprocess_mode: mlp.PipelineParam=mlp.PipelineParam(name='preprocessmode', value='local'),
  predict_mode: mlp.PipelineParam=mlp.PipelineParam(name='predictmode', value='local')):
  # TODO: use the argo job name as the workflow
  workflow = '{{workflow.name}}'
  preprocess = mlp.ContainerOp(
      name = 'preprocess',
      image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:0.0.16',
      arguments = ['--train', train, '--eval', evaluation, '--schema', schema, '--output', '%s/%s/transformed' % (output, workflow),
        '--project', project, '--mode', preprocess_mode],
      file_outputs = {'transformed': '/output.txt'})

  training = mlp.ContainerOp(
      name = 'training',
      image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf:0.0.16',
      arguments = ['--job-dir', '%s/%s/train' % (output, workflow), '--transformed-data-dir', preprocess.output,
        '--schema', schema, '--learning-rate', learning_rate, '--hidden-layer-size', hidden_layer_size, '--steps', steps,
        '--target', target, '--workers', workers, '--pss', pss],
      file_outputs = {'trained': '/output.txt'}
      )

  prediction = mlp.ContainerOp(
      name = 'prediction',
      image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:0.0.16',
      arguments = ['--output', '%s/%s/predict' % (output, workflow), '--data', evaluation, '--schema', schema,
        '--target', target, '--model',  training.output, '--mode', predict_mode, '--project', project],
      file_outputs = {'prediction': '/output.txt'})

  confusion_matrix = mlp.ContainerOp(
      name = 'confusionmatrix',
      image = 'gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:0.0.16',
      arguments = ['--output', '%s/%s/confusionmatrix' % (output, workflow), '--predictions', prediction.output])

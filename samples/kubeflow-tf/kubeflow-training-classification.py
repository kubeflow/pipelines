#!/usr/bin/env python3
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

def dataflow_tf_transform_op(train_data: 'GcsUri', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', preprocess_mode, preprocess_module: 'GcsUri[text/code/python]', transform_output: 'GcsUri[Directory]', step_name='preprocess'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:0.0.18',
        arguments = [
            '--train', train_data,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', preprocess_mode,
            '--preprocessing-module', preprocess_module,
            '--output', transform_output,
        ],
        file_outputs = {'transformed': '/output.txt'}
    )

def kubeflow_tf_training_launcher_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target, workers: int, pss, preprocess_module: 'GcsUri[text/code/python]', tfjob_timeout_minutes: int, training_output: 'GcsUri[Directory]', step_name='training'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf:0.0.18',
        arguments = [
            '--transformed-data-dir', transformed_data_dir,
            '--schema', schema,
            '--learning-rate', learning_rate,
            '--hidden-layer-size', hidden_layer_size,
            '--steps', steps,
            '--target', target,
            '--workers', workers,
            '--pss', pss,
            '--preprocessing-module', preprocess_module,
            '--tfjob-timeout-minutes', tfjob_timeout_minutes,
            '--job-dir', training_output,
        ],
        file_outputs = {'train': '/output.txt'}
    )

def dataflow_tf_predict_op(evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', target: str, model: 'TensorFlow model', predict_mode, project: 'GcpProject', prediction_output: 'GcsUri', step_name='prediction'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:0.0.18',
        arguments = [
            '--data', evaluation_data,
            '--schema', schema,
            '--target', target,
            '--model',  model,
            '--mode', predict_mode,
            '--project', project,
            '--output', prediction_output,
        ],
        file_outputs = {'prediction': '/output.txt'}
    )

def confusion_matrix_op(predictions, output, step_name='confusionmatrix'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:0.0.18',
        arguments = [
          '--predictions', predictions,
          '--output', output,
        ]
    )

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

  preprocess = dataflow_tf_transform_op(train, evaluation, schema, project, preprocess_mode, '', '%s/%s/transformed' % (output, workflow))
  training = kubeflow_tf_training_launcher_op(preprocess.output, schema, learning_rate, hidden_layer_size, steps, target, workers, pss, '', 10, '%s/%s/train' % (output, workflow))
  prediction = dataflow_tf_predict_op(evaluation, schema, target,  training.output, predict_mode, project, '%s/%s/predict' % (output, workflow))
  confusion_matrix = confusion_matrix_op(prediction.output, '%s/%s/confusionmatrix' % (output, workflow))

if __name__ == '__main__':
  import mlpc.main as compiler
  compiler._compile_pipeline_function(None, __file__ + '.yaml')

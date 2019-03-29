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


import kfp.dsl as dsl
import kfp.gcp as gcp
import datetime

def dataflow_tf_transform_op(train_data: 'GcsUri', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', preprocess_mode, preprocess_module: 'GcsUri[text/code/python]', transform_output: 'GcsUri[Directory]', step_name='preprocess'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:2c2445df83fa879387a200747cc20f72a7ee9727',
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


def kubeflow_tf_training_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target, preprocess_module: 'GcsUri[text/code/python]', training_output: 'GcsUri[Directory]', step_name='training', use_gpu=False):
    kubeflow_tf_training_op = dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:2c2445df83fa879387a200747cc20f72a7ee9727',
        arguments = [
            '--transformed-data-dir', transformed_data_dir,
            '--schema', schema,
            '--learning-rate', learning_rate,
            '--hidden-layer-size', hidden_layer_size,
            '--steps', steps,
            '--target', target,
            '--preprocessing-module', preprocess_module,
            '--job-dir', training_output,
        ],
        file_outputs = {'train': '/output.txt'}
    )
    if use_gpu:
        kubeflow_tf_training_op.image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer-gpu:2c2445df83fa879387a200747cc20f72a7ee9727'
        kubeflow_tf_training_op.set_gpu_limit(1)
    
    return kubeflow_tf_training_op

def dataflow_tf_predict_op(evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', target: str, model: 'TensorFlow model', predict_mode, project: 'GcpProject', prediction_output: 'GcsUri', step_name='prediction'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:2c2445df83fa879387a200747cc20f72a7ee9727',
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
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:2c2445df83fa879387a200747cc20f72a7ee9727',
        arguments = [
          '--predictions', predictions,
          '--output', output,
        ]
    )

@dsl.pipeline(
  name='Pipeline TFJob',
  description='Demonstrate the DSL for TFJob'
)
def kubeflow_training(output, project,
  evaluation='gs://ml-pipeline-playground/flower/eval100.csv',
  train='gs://ml-pipeline-playground/flower/train200.csv',
  schema='gs://ml-pipeline-playground/flower/schema.json',
  learning_rate=0.1,
  hidden_layer_size='100,50',
  steps=2000,
  target='label',
  workers=0,
  pss=0,
  preprocess_mode='local',
  predict_mode='local'):
  # TODO: use the argo job name as the workflow
  workflow = '{{workflow.name}}'
  # set the flag to use GPU trainer
  use_gpu = False

  preprocess = dataflow_tf_transform_op(train, evaluation, schema, project, preprocess_mode, '', '%s/%s/transformed' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))
  training = kubeflow_tf_training_op(preprocess.output, schema, learning_rate, hidden_layer_size, steps, target, '', '%s/%s/train' % (output, workflow), use_gpu=use_gpu).apply(gcp.use_gcp_secret('user-gcp-sa'))
  prediction = dataflow_tf_predict_op(evaluation, schema, target,  training.output, predict_mode, project, '%s/%s/predict' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))
  confusion_matrix = confusion_matrix_op(prediction.output, '%s/%s/confusionmatrix' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))


if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(kubeflow_training, __file__ + '.zip')

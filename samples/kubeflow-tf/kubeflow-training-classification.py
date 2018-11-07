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
import datetime

def dataflow_tf_transform_op(train_data: 'GcsUri', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', preprocess_mode, preprocess_module: 'GcsUri[text/code/python]', transform_output: 'GcsUri[Directory]', step_name='preprocess'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:0.1.0',
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


def kubeflow_tf_training_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target, preprocess_module: 'GcsUri[text/code/python]', training_output: 'GcsUri[Directory]', step_name='training'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:0.1.0',
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

def dataflow_tf_predict_op(evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', target: str, model: 'TensorFlow model', predict_mode, project: 'GcpProject', prediction_output: 'GcsUri', step_name='prediction'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:0.1.0',
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
        image = 'gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:0.1.0',
        arguments = [
          '--predictions', predictions,
          '--output', output,
        ]
    )

@dsl.pipeline(
  name='Pipeline TFJob',
  description='Demonstrate the DSL for TFJob'
)
def kubeflow_training( output: dsl.PipelineParam, project: dsl.PipelineParam,
  evaluation: dsl.PipelineParam=dsl.PipelineParam(name='evaluation', value='gs://ml-pipeline-playground/flower/eval100.csv'),
  train: dsl.PipelineParam=dsl.PipelineParam(name='train', value='gs://ml-pipeline-playground/flower/train200.csv'),
  schema: dsl.PipelineParam=dsl.PipelineParam(name='schema', value='gs://ml-pipeline-playground/flower/schema.json'),
  learning_rate: dsl.PipelineParam=dsl.PipelineParam(name='learningrate', value=0.1),
  hidden_layer_size: dsl.PipelineParam=dsl.PipelineParam(name='hiddenlayersize', value='100,50'),
  steps: dsl.PipelineParam=dsl.PipelineParam(name='steps', value=2000),
  target: dsl.PipelineParam=dsl.PipelineParam(name='target', value='label'),
  workers: dsl.PipelineParam=dsl.PipelineParam(name='workers', value=0),
  pss: dsl.PipelineParam=dsl.PipelineParam(name='pss', value=0),
  preprocess_mode: dsl.PipelineParam=dsl.PipelineParam(name='preprocessmode', value='local'),
  predict_mode: dsl.PipelineParam=dsl.PipelineParam(name='predictmode', value='local')):
  # TODO: use the argo job name as the workflow
  workflow = '{{workflow.name}}'

  preprocess = dataflow_tf_transform_op(train, evaluation, schema, project, preprocess_mode, '', '%s/%s/transformed' % (output, workflow))
  training = kubeflow_tf_training_op(preprocess.output, schema, learning_rate, hidden_layer_size, steps, target, '', '%s/%s/train' % (output, workflow))
  prediction = dataflow_tf_predict_op(evaluation, schema, target,  training.output, predict_mode, project, '%s/%s/predict' % (output, workflow))
  confusion_matrix = confusion_matrix_op(prediction.output, '%s/%s/confusionmatrix' % (output, workflow))


if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(kubeflow_training, __file__ + '.tar.gz')

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

def dataflow_tf_data_validation_op(inference_data: 'GcsUri', validation_data: 'GcsUri', column_names: 'GcsUri[text/json]', key_columns, project: 'GcpProject', mode, validation_output: 'GcsUri[Directory]', step_name='validation'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfdv:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--csv-data-for-inference', inference_data,
            '--csv-data-to-validate', validation_data,
            '--column-names', column_names,
            '--key-columns', key_columns,
            '--project', project,
            '--mode', mode,
            '--output', '%s/{{workflow.name}}/validation' % validation_output,
        ],
        file_outputs = {
            'schema': '/schema.txt',
            'validation': '/output_validation_result.txt',
        }
    )

def dataflow_tf_transform_op(train_data: 'GcsUri', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', preprocess_mode, preprocess_module: 'GcsUri[text/code/python]', transform_output: 'GcsUri[Directory]', step_name='preprocess'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--train', train_data,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', preprocess_mode,
            '--preprocessing-module', preprocess_module,
            '--output', '%s/{{workflow.name}}/transformed' % transform_output,
        ],
        file_outputs = {'transformed': '/output.txt'}
    )


def tf_train_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target: str, preprocess_module: 'GcsUri[text/code/python]', training_output: 'GcsUri[Directory]', step_name='training'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--transformed-data-dir', transformed_data_dir,
            '--schema', schema,
            '--learning-rate', learning_rate,
            '--hidden-layer-size', hidden_layer_size,
            '--steps', steps,
            '--target', target,
            '--preprocessing-module', preprocess_module,
            '--job-dir', '%s/{{workflow.name}}/train' % training_output,
        ],
        file_outputs = {'train': '/output.txt'}
    )

def dataflow_tf_model_analyze_op(model: 'TensorFlow model', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', analyze_mode, analyze_slice_column, analysis_output: 'GcsUri', step_name='analysis'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--model', model,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', analyze_mode,
            '--slice-columns', analyze_slice_column,
            '--output', '%s/{{workflow.name}}/analysis' % analysis_output,
        ],
        file_outputs = {'analysis': '/output.txt'}
    )


def dataflow_tf_predict_op(evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', target: str, model: 'TensorFlow model', predict_mode, project: 'GcpProject', prediction_output: 'GcsUri', step_name='prediction'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--data', evaluation_data,
            '--schema', schema,
            '--target', target,
            '--model',  model,
            '--mode', predict_mode,
            '--project', project,
            '--output', '%s/{{workflow.name}}/predict' % prediction_output,
        ],
        file_outputs = {'prediction': '/output.txt'}
    )


def confusion_matrix_op(predictions: 'GcsUri', output: 'GcsUri', step_name='confusion_matrix'):
  return dsl.ContainerOp(
      name=step_name,
      image='gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
      arguments=[
        '--output', '%s/{{workflow.name}}/confusionmatrix' % output,
        '--predictions', predictions,
        '--target_lambda', """lambda x: (x['target'] > x['fare'] * 0.2)""",
     ])


def roc_op(predictions: 'GcsUri', output: 'GcsUri', step_name='roc'):
  return dsl.ContainerOp(
      name=step_name,
      image='gcr.io/ml-pipeline/ml-pipeline-local-roc:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
      arguments=[
        '--output', '%s/{{workflow.name}}/roc' % output,
        '--predictions', predictions,
        '--target_lambda', """lambda x: 1 if (x['target'] > x['fare'] * 0.2) else 0""",
     ])


def kubeflow_deploy_op(model: 'TensorFlow model', tf_server_name, step_name='deploy'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:65d0f6a1a3b1a4c2254a4398cc6b92550803fe62',
        arguments = [
            '--model-path', model,
            '--server-name', tf_server_name
        ]
    )


@dsl.pipeline(
  name='TFX Taxi Cab Classification Pipeline Example',
  description='Example pipeline that does classification with model analysis based on a public BigQuery dataset.'
)
def taxi_cab_classification(
    output,
    project,
    column_names='gs://ml-pipeline-playground/tfx/taxi-cab-classification/column-names.json',
    key_columns='trip_start_timestamp',
    train='gs://ml-pipeline-playground/tfx/taxi-cab-classification/train.csv',
    evaluation='gs://ml-pipeline-playground/tfx/taxi-cab-classification/eval.csv',
    mode='local',
    preprocess_module='gs://ml-pipeline-playground/tfx/taxi-cab-classification/preprocessing.py',
    learning_rate=0.1,
    hidden_layer_size='1500',
    steps=3000,
    analyze_slice_column='trip_start_hour'):

  tf_server_name = 'taxi-cab-classification-model-{{workflow.name}}'
  validation = dataflow_tf_data_validation_op(train, evaluation, column_names, 
      key_columns, project, mode, output
  ).apply(gcp.use_gcp_secret('user-gcp-sa'))
  preprocess = dataflow_tf_transform_op(train, evaluation, validation.outputs['schema'],
      project, mode, preprocess_module, output
  ).apply(gcp.use_gcp_secret('user-gcp-sa'))
  training = tf_train_op(preprocess.output, validation.outputs['schema'], learning_rate,
      hidden_layer_size, steps, 'tips', preprocess_module, output
  ).apply(gcp.use_gcp_secret('user-gcp-sa'))
  analysis = dataflow_tf_model_analyze_op(training.output, evaluation,
      validation.outputs['schema'], project, mode, analyze_slice_column, output
  ).apply(gcp.use_gcp_secret('user-gcp-sa'))
  prediction = dataflow_tf_predict_op(evaluation, validation.outputs['schema'], 'tips',
      training.output, mode, project, output
  ).apply(gcp.use_gcp_secret('user-gcp-sa'))
  cm = confusion_matrix_op(prediction.output, output).apply(gcp.use_gcp_secret('user-gcp-sa'))
  roc = roc_op(prediction.output, output).apply(gcp.use_gcp_secret('user-gcp-sa'))
  deploy = kubeflow_deploy_op(training.output, tf_server_name).apply(gcp.use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(taxi_cab_classification, __file__ + '.tar.gz')

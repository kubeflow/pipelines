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
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfdv:dev', #TODO-release: update the release tag for the next release
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
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

def dataflow_tf_transform_op(train_data: 'GcsUri', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', preprocess_mode, preprocess_module: 'GcsUri[text/code/python]', transform_output: 'GcsUri[Directory]', step_name='preprocess'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:0.1.3-rc.2', #TODO-release: update the release tag for the next release
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
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))


def tf_train_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target: str, preprocess_module: 'GcsUri[text/code/python]', training_output: 'GcsUri[Directory]', step_name='training'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:0.1.3-rc.2', #TODO-release: update the release tag for the next release
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
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

def dataflow_tf_model_analyze_op(model: 'TensorFlow model', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', analyze_mode, analyze_slice_column, analysis_output: 'GcsUri', step_name='analysis'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:0.1.3-rc.2', #TODO-release: update the release tag for the next release
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
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))


def dataflow_tf_predict_op(evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', target: str, model: 'TensorFlow model', predict_mode, project: 'GcpProject', prediction_output: 'GcsUri', step_name='prediction'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:0.1.3-rc.2', #TODO-release: update the release tag for the next release
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
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))


def confusion_matrix_op(predictions: 'GcsUri', output: 'GcsUri', step_name='confusion_matrix'):
  return dsl.ContainerOp(
      name=step_name,
      image='gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:dev', #TODO-release: update the release tag for the next release
      arguments=[
        '--output', '%s/{{workflow.name}}/confusionmatrix' % output,
        '--predictions', predictions,
        '--target_lambda', """lambda x: (x['target'] > x['fare'] * 0.2)""",
     ])


def roc_op(predictions: 'GcsUri', output: 'GcsUri', step_name='roc'):
  return dsl.ContainerOp(
      name=step_name,
      image='gcr.io/ml-pipeline/ml-pipeline-local-roc:dev', #TODO-release: update the release tag for the next release
      arguments=[
        '--output', '%s/{{workflow.name}}/roc' % output,
        '--predictions', predictions,
        '--target_lambda', """lambda x: 1 if (x['target'] > x['fare'] * 0.2) else 0""",
     ])


def kubeflow_deploy_op(model: 'TensorFlow model', tf_server_name, step_name='deploy'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:0.1.3-rc.2', #TODO-release: update the release tag for the next release
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
  validation = dataflow_tf_data_validation_op(train, evaluation, column_names, key_columns, project, mode, output)
  preprocess = dataflow_tf_transform_op(train, evaluation, validation.outputs['schema'],
      project, mode, preprocess_module, output)
  training = tf_train_op(preprocess.output, validation.outputs['schema'], learning_rate,
      hidden_layer_size, steps, 'tips', preprocess_module, output)
  analysis = dataflow_tf_model_analyze_op(training.output, evaluation,
      validation.outputs['schema'], project, mode, analyze_slice_column, output)
  prediction = dataflow_tf_predict_op(evaluation, validation.outputs['schema'], 'tips',
      training.output, mode, project, output)
  cm = confusion_matrix_op(prediction.output, output)
  roc = roc_op(prediction.output, output)
  deploy = kubeflow_deploy_op(training.output, tf_server_name)

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(taxi_cab_classification, __file__ + '.tar.gz')

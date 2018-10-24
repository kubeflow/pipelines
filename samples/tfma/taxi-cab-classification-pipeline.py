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

def dataflow_tf_data_validation_op(inference_data: 'GcsUri', validation_data: 'GcsUri', column_names: 'GcsUri[text/json]', key_columns, project: 'GcpProject', mode, validation_output: 'GcsUri[Directory]', step_name='validation'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/omars-test-kubeflow-project/ml-pipeline-dataflow-tfdv:0.0.23',
        arguments = [
            '--csv-data-for-inference', inference_data,
            '--csv-data-to-validate', validation_data,
            '--column-names', column_names,
            '--key-columns', key_columns,
            '--project', project,
            '--mode', mode,
            '--output', validation_output,
        ],
        file_outputs = {
            'validation': '/output.txt',
            'schema': '/output_schema.json',
        }
    )

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

def kubeflow_tfjob_launcher_op(container_image, command, number_of_workers: int, number_of_parameter_servers: int, tfjob_timeout_minutes: int, output_dir=None, step_name='TFJob-launcher'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf:0.0.18', #TODO: Update the name in next release.
        arguments = [
            '--workers', number_of_workers,
            '--pss', number_of_parameter_servers,
            '--tfjob-timeout-minutes', tfjob_timeout_minutes,
            '--container-image', container_image,
            '--output-dir', output_dir,
            '--ui-metadata-type', 'tensorboard',
            '--',
        ] + command,
        file_outputs = {'train': '/output.txt'}
    )

def tf_train_dnn_on_kubeflow_op(transformed_data_dir, schema: 'GcsUri[text/json]', learning_rate: float, hidden_layer_size: int, steps: int, target: str, preprocess_module: 'GcsUri[text/code/python]', training_output: 'GcsUri[Directory]', number_of_workers: int = 1, number_of_parameter_servers: int = 1, tfjob_timeout_minutes: int = 10, step_name='training'):
    return kubeflow_tfjob_launcher_op(
        container_image='gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:0.0.18',
        command=[
            'python', '-m', 'trainer.task',
            '--transformed-data-dir', transformed_data_dir,
            '--schema', schema,
            '--learning-rate', learning_rate,
            '--hidden-layer-size', hidden_layer_size,
            '--steps', steps,
            '--target', target,
            '--preprocessing-module', preprocess_module,
            '--job-dir', training_output,
        ],
        number_of_workers=number_of_workers,
        number_of_parameter_servers=number_of_parameter_servers,
        tfjob_timeout_minutes=tfjob_timeout_minutes,
        output_dir=training_output,
        step_name=step_name + '-on-kubeflow'
    )

def dataflow_tf_model_analyze_op(model: 'TensorFlow model', evaluation_data: 'GcsUri', schema: 'GcsUri[text/json]', project: 'GcpProject', analyze_mode, analyze_slice_column, analysis_output: 'GcsUri', step_name='analysis'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:0.0.18',
        arguments = [
            '--model', model,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', analyze_mode,
            '--slice-columns', analyze_slice_column,
            '--output', analysis_output,
        ],
        file_outputs = {'analysis': '/output.txt'}
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

def kubeflow_deploy_op(model: 'TensorFlow model', tf_server_name, step_name='deploy'):
    return mlp.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:0.0.18',
        arguments = [
            '--model-path', model,
            '--server-name', tf_server_name
        ]
    )


@mlp.pipeline(
  name='TFMA Taxi Cab Classification Pipeline Example',
  description='Example pipeline that does classification with model analysis based on a public BigQuery dataset.'
)
def taxi_cab_classification(
    output: mlp.PipelineParam,
    project: mlp.PipelineParam,

    column_names: mlp.PipelineParam=mlp.PipelineParam(
        name='column-names',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/column-names.json'),
    key_columns: mlp.PipelineParam=mlp.PipelineParam(
        name='key-columns',
        value='trip_start_timestamp'),
    train: mlp.PipelineParam=mlp.PipelineParam(
        name='train',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv'),
    evaluation: mlp.PipelineParam=mlp.PipelineParam(
        name='evaluation',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv'),
    validation_mode: mlp.PipelineParam=mlp.PipelineParam(
        name='validation-mode', value='local'),
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
    workers: mlp.PipelineParam=mlp.PipelineParam(name='workers', value=1), #Must be 1 or greater
    pss: mlp.PipelineParam=mlp.PipelineParam(name='pss', value=1), #Must be 1 or greater
    predict_mode: mlp.PipelineParam=mlp.PipelineParam(name='predict-mode', value='local'),
    analyze_mode: mlp.PipelineParam=mlp.PipelineParam(name='analyze-mode', value='local'),
    analyze_slice_column: mlp.PipelineParam=mlp.PipelineParam(
        name='analyze-slice-column', value='trip_start_hour')):
  validation_output = '%s/{{workflow.name}}/validation' % output
  transform_output = '%s/{{workflow.name}}/transformed' % output
  training_output = '%s/{{workflow.name}}/train' % output
  analysis_output = '%s/{{workflow.name}}/analysis' % output
  prediction_output = '%s/{{workflow.name}}/predict' % output
  tf_server_name = 'taxi-cab-classification-model-{{workflow.name}}'

  validation = dataflow_tf_data_validation_op(train, evaluation, column_names, key_columns, project, validation_mode, validation_output)
  preprocess = dataflow_tf_transform_op(train, evaluation, validation.outputs['schema'], project, preprocess_mode, preprocess_module, transform_output)
  training = tf_train_dnn_on_kubeflow_op(preprocess.output, validation.outputs['schema'], learning_rate, hidden_layer_size, steps, target, preprocess_module, training_output, workers, pss, 60)
  analysis = dataflow_tf_model_analyze_op(training.output, evaluation, validation.outputs['schema'], project, analyze_mode, analyze_slice_column, analysis_output)
  prediction = dataflow_tf_predict_op(evaluation, validation.outputs['schema'], target, training.output, predict_mode, project, prediction_output)
  deploy = kubeflow_deploy_op(training.output, tf_server_name)

if __name__ == '__main__':
  import mlpc
  mlpc.Compiler().compile(taxi_cab_classification, __file__ + '.tar.gz')

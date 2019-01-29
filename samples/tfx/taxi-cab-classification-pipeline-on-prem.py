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
from kubernetes import client as k8s_client


def dataflow_tf_data_validation_op(inference_data, validation_data,
                                   column_names, key_columns, project, mode,
                                   validation_output, step_name='validation'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-dataflow-tfdv:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--csv-data-for-inference', inference_data,
            '--csv-data-to-validate', validation_data,
            '--column-names', column_names,
            '--key-columns', key_columns,
            '--project', project,
            '--mode', mode,
            '--output', '%s/{{workflow.name}}/validation' % validation_output,
        ],
        file_outputs={
            'schema': '/schema.txt',
            'validation': '/output_validation_result.txt',
        }
    )


def dataflow_tf_transform_op(train_data, evaluation_data, schema,
                             project, preprocess_mode, preprocess_module,
                             transform_output, step_name='preprocess'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--train', train_data,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', preprocess_mode,
            '--preprocessing-module', preprocess_module,
            '--output', '%s/{{workflow.name}}/transformed' % transform_output,
        ],
        file_outputs={'transformed': '/output.txt'}
    )


def tf_train_op(transformed_data_dir, schema, learning_rate: float, hidden_layer_size: int,
                steps: int, target: str, preprocess_module,
                training_output, step_name='training'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--transformed-data-dir', transformed_data_dir,
            '--schema', schema,
            '--learning-rate', learning_rate,
            '--hidden-layer-size', hidden_layer_size,
            '--steps', steps,
            '--target', target,
            '--preprocessing-module', preprocess_module,
            '--job-dir', '%s/{{workflow.name}}/train' % training_output,
        ],
        file_outputs={'train': '/output.txt'}
    )


def dataflow_tf_model_analyze_op(model: 'TensorFlow model', evaluation_data, schema,
                                 project, analyze_mode, analyze_slice_column, analysis_output,
                                 step_name='analysis'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--model', model,
            '--eval', evaluation_data,
            '--schema', schema,
            '--project', project,
            '--mode', analyze_mode,
            '--slice-columns', analyze_slice_column,
            '--output', '%s/{{workflow.name}}/analysis' % analysis_output,
        ],
        file_outputs={'analysis': '/output.txt'}
    )


def dataflow_tf_predict_op(evaluation_data, schema, target: str,
                           model: 'TensorFlow model', predict_mode, project, prediction_output,
                           step_name='prediction'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--data', evaluation_data,
            '--schema', schema,
            '--target', target,
            '--model', model,
            '--mode', predict_mode,
            '--project', project,
            '--output', '%s/{{workflow.name}}/predict' % prediction_output,
        ],
        file_outputs={'prediction': '/output.txt'}
    )


def confusion_matrix_op(predictions, output, step_name='confusion_matrix'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--output', '%s/{{workflow.name}}/confusionmatrix' % output,
            '--predictions', predictions,
            '--target_lambda', """lambda x: (x['target'] > x['fare'] * 0.2)""",
        ])


def roc_op(predictions, output, step_name='roc'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-local-roc:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--output', '%s/{{workflow.name}}/roc' % output,
            '--predictions', predictions,
            '--target_lambda', """lambda x: 1 if (x['target'] > x['fare'] * 0.2) else 0""",
        ])


def kubeflow_deploy_op(model: 'TensorFlow model', tf_server_name, pvc_name, step_name='deploy'):
    return dsl.ContainerOp(
        name=step_name,
        image='gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:be19cbc2591a48d2ef5ca715c34ecae8223cf454',
        arguments=[
            '--cluster-name', 'tfx-taxi-pipeline-on-prem',
            '--model-path', model,
            '--server-name', tf_server_name,
            '--model-storage-type', 'nfs',
            '--pvc-name', pvc_name,
        ]
    )


@dsl.pipeline(
    name='TFX Taxi Cab Classification Pipeline Example',
    description='Example pipeline that does classification with model analysis based on a public BigQuery dataset.'
)
def taxi_cab_classification(
        pvc_name='pipeline-pvc',
        project='tfx-taxi-pipeline-on-prem',
        column_names='taxi-cab-classification/column-names.json',
        key_columns='trip_start_timestamp',
        train='taxi-cab-classification/train.csv',
        evaluation='taxi-cab-classification/eval.csv',
        mode='local',
        preprocess_module='taxi-cab-classification/preprocessing.py',
        learning_rate=0.1,
        hidden_layer_size=1500,
        steps=3000,
        analyze_slice_column='trip_start_hour'):
    tf_server_name = 'taxi-cab-classification-model-{{workflow.name}}'
    validation = dataflow_tf_data_validation_op('/mnt/%s' % train, '/mnt/%s' % evaluation, '/mnt/%s' % column_names,
                                                key_columns, project, mode, '/mnt').add_volume(
        k8s_client.V1Volume(name='pipeline-nfs', persistent_volume_claim=k8s_client.V1PersistentVolumeClaimVolumeSource(
            claim_name='pipeline-pvc'))).add_volume_mount(k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    preprocess = dataflow_tf_transform_op('/mnt/%s' % train, '/mnt/%s' % evaluation, validation.outputs['schema'],
                                          project, mode, '/mnt/%s' % preprocess_module, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    training = tf_train_op(preprocess.output, validation.outputs['schema'], learning_rate, hidden_layer_size, steps,
                           'tips', '/mnt/%s' % preprocess_module, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    analysis = dataflow_tf_model_analyze_op(training.output, '/mnt/%s' % evaluation, validation.outputs['schema'],
                                            project, mode, analyze_slice_column, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    prediction = dataflow_tf_predict_op('/mnt/%s' % evaluation, validation.outputs['schema'], 'tips', training.output,
                                        mode, project, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    cm = confusion_matrix_op(prediction.output, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    roc = roc_op(prediction.output, '/mnt').add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))
    deploy = kubeflow_deploy_op(training.output, tf_server_name, pvc_name).add_volume_mount(
        k8s_client.V1VolumeMount(mount_path='/mnt', name='pipeline-nfs'))


if __name__ == '__main__':
    import kfp.compiler as compiler

    compiler.Compiler().compile(taxi_cab_classification, __file__ + '.tar.gz')

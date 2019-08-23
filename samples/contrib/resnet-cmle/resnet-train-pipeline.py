#!/usr/bin/env python3
# Copyright 2019 Google LLC
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

# flake8: noqa TODO

import kfp.dsl as dsl
import kfp.gcp as gcp
import kfp.components as comp
import datetime
import json
import os

dataflow_python_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/a97f1d0ad0e7b92203f35c5b0b9af3a314952e05/components/gcp/dataflow/launch_python/component.yaml')
cloudml_train_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/a97f1d0ad0e7b92203f35c5b0b9af3a314952e05/components/gcp/ml_engine/train/component.yaml')
cloudml_deploy_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/a97f1d0ad0e7b92203f35c5b0b9af3a314952e05/components/gcp/ml_engine/deploy/component.yaml')


def resnet_preprocess_op(project_id: 'GcpProject', output: 'GcsUri', staging_dir: 'GcsUri', train_csv: 'GcsUri[text/csv]',
                         validation_csv: 'GcsUri[text/csv]', labels, train_size: 'Integer', validation_size: 'Integer',
                         step_name='preprocess'):
    return dataflow_python_op(
        python_file_path='gs://ml-pipeline-playground/samples/ml_engine/resnet-cmle/preprocess/preprocess.py',
        project_id=project_id,
        requirements_file_path='gs://ml-pipeline-playground/samples/ml_engine/resnet-cmle/preprocess/requirements.txt',
        staging_dir=staging_dir,
        args=json.dumps([
            '--train_csv', str(train_csv),
            '--validation_csv', str(validation_csv),
            '--labels', str(labels),
            '--output_dir', str(output),
            '--train_size', str(train_size),
            '--validation_size', str(validation_size)
        ])
    )


def resnet_train_op(project_id, data_dir, output: 'GcsUri', region: 'GcpRegion', depth: int, train_batch_size: int,
                    eval_batch_size: int, steps_per_eval: int, train_steps: int, num_train_images: int,
                    num_eval_images: int, num_label_classes: int, tf_version, step_name='train'):
    return cloudml_train_op(
        project_id=project_id,
        region='us-central1',
        python_module='trainer.resnet_main',
        package_uris=json.dumps(
            ['gs://ml-pipeline-playground/samples/ml_engine/resnet-cmle/trainer/trainer-1.0.tar.gz']),
        job_dir=output,
        args=json.dumps([
            '--data_dir', str(data_dir),
            '--model_dir', str(output),
            '--use_tpu', 'True',
            '--resnet_depth', str(depth),
            '--train_batch_size', str(train_batch_size),
            '--eval_batch_size', str(eval_batch_size),
            '--steps_per_eval', str(steps_per_eval),
            '--train_steps', str(train_steps),
            '--num_train_images', str(num_train_images),
            '--num_eval_images', str(num_eval_images),
            '--num_label_classes', str(num_label_classes),
            '--export_dir', '{}/export'.format(str(output))
        ]),
        runtime_version=tf_version,
        training_input=json.dumps({
            'scaleTier': 'BASIC_TPU'
        })
    )


def resnet_deploy_op(model_dir, model, version, project_id: 'GcpProject', region: 'GcpRegion',
                     tf_version, step_name='deploy'):
    # TODO(hongyes): add region to model payload.
    return cloudml_deploy_op(
        model_uri=model_dir,
        project_id=project_id,
        model_id=model,
        version_id=version,
        runtime_version=tf_version,
        replace_existing_version='True'
    )


@dsl.pipeline(
    name='ResNet_Train_Pipeline',
    description='Demonstrate the ResNet50 predict.'
)
def resnet_train(
        project_id,
        output,
        region='us-central1',
        model='bolts',
        version='beta1',
        tf_version='1.12',
        train_csv='gs://bolts_image_dataset/bolt_images_train.csv',
        validation_csv='gs://bolts_image_dataset/bolt_images_validate.csv',
        labels='gs://bolts_image_dataset/labels.txt',
        depth=50,
        train_batch_size=1024,
        eval_batch_size=1024,
        steps_per_eval=250,
        train_steps=10000,
        num_train_images=218593,
        num_eval_images=54648,
        num_label_classes=10):
    output_dir = os.path.join(str(output), '{{workflow.name}}')
    preprocess_staging = os.path.join(output_dir, 'staging')
    preprocess_output = os.path.join(output_dir, 'preprocessed_output')
    train_output = os.path.join(output_dir, 'model')
    preprocess = resnet_preprocess_op(project_id, preprocess_output, preprocess_staging, train_csv,
                                      validation_csv, labels, train_batch_size, eval_batch_size).apply(gcp.use_gcp_secret())
    train = resnet_train_op(project_id, preprocess_output, train_output, region, depth, train_batch_size,
                            eval_batch_size, steps_per_eval, train_steps, num_train_images, num_eval_images,
                            num_label_classes, tf_version).apply(gcp.use_gcp_secret())
    train.after(preprocess)
    export_output = os.path.join(str(train.outputs['job_dir']), 'export')
    deploy = resnet_deploy_op(export_output, model, version, project_id, region,
                              tf_version).apply(gcp.use_gcp_secret())


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(resnet_train, __file__ + '.zip')

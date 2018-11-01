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

def resnet_preprocess_op(project_id: 'GcpProject', output: 'GcsUri', train_csv: 'GcsUri[text/csv]', validation_csv: 'GcsUri[text/csv]', labels, step_name='preprocess'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/resnet-preprocess:dev', # TODO: update it with a version number after a new release.
        arguments = [
            '--project_id', project_id,
            '--output', output,
            '--train_csv', train_csv,
            '--validation_csv', validation_csv,
            '--labels', labels,
        ],
        file_outputs = {'preprocessed': '/output.txt'}
    )

def resnet_train_op(data_dir, output: 'GcsUri', region: 'GcpRegion', depth: int, train_batch_size: int, eval_batch_size: int, steps_per_eval: int, train_steps: int, num_train_images: int, num_eval_images: int, num_label_classes: int, tf_version, step_name='train'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/resnet-train:0.0.18',
        arguments = [
            '--data_dir', data_dir,
            '--output', output,
            '--region', region,
            '--depth', depth,
            '--train_batch_size', train_batch_size,
            '--eval_batch_size', eval_batch_size,
            '--steps_per_eval', steps_per_eval,
            '--train_steps', train_steps,
            '--num_train_images', num_train_images,
            '--num_eval_images', num_eval_images,
            '--num_label_classes', num_label_classes,
            '--TFVERSION', tf_version
        ],
        file_outputs = {'trained': '/output.txt'}
    )

def resnet_deploy_op(model_dir, model, version, project_id: 'GcpProject', region: 'GcpRegion', tf_version, step_name='deploy'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/resnet-deploy:0.0.18',
        arguments = [
            '--model', model,
            '--version', version,
            '--project_id', project_id,
            '--region', region,
            '--model_dir', model_dir,
            '--TFVERSION', tf_version
        ]
    )


@dsl.pipeline(
  name='ResNet_Train_Pipeline',
  description='Demonstrate the ResNet50 predict.'
)
def resnet_train(project_id: dsl.PipelineParam,
  output: dsl.PipelineParam,
  region: dsl.PipelineParam=dsl.PipelineParam(name='region', value='us-central1'),
  model: dsl.PipelineParam=dsl.PipelineParam(name='model', value='bolts'),
  version: dsl.PipelineParam=dsl.PipelineParam(name='version', value='beta1'),
  tf_version: dsl.PipelineParam=dsl.PipelineParam(name='tf-version', value='1.8'),
  train_csv: dsl.PipelineParam=dsl.PipelineParam(name='train-csv', value='gs://bolts_image_dataset/bolt_images_train.csv'),
  validation_csv: dsl.PipelineParam=dsl.PipelineParam(name='validation-csv', value='gs://bolts_image_dataset/bolt_images_validate.csv'),
  labels: dsl.PipelineParam=dsl.PipelineParam(name='labels', value='gs://bolts_image_dataset/labels.txt'),
  depth: dsl.PipelineParam=dsl.PipelineParam(name='depth', value=50),
  train_batch_size: dsl.PipelineParam=dsl.PipelineParam(name='train-batch-size', value=1024),
  eval_batch_size: dsl.PipelineParam=dsl.PipelineParam(name='eval-batch-size', value=1024),
  steps_per_eval: dsl.PipelineParam=dsl.PipelineParam(name='steps-per-eval', value=250),
  train_steps: dsl.PipelineParam=dsl.PipelineParam(name='train-steps', value=10000),
  num_train_images: dsl.PipelineParam=dsl.PipelineParam(name='num-train-images', value=218593),
  num_eval_images: dsl.PipelineParam=dsl.PipelineParam(name='num-eval-images', value=54648),
  num_label_classes: dsl.PipelineParam=dsl.PipelineParam(name='num-label-classes', value=10)):

  preprocess = resnet_preprocess_op(project_id, output, train_csv, validation_csv, labels)
  train = resnet_train_op(preprocess.output, output, region, depth, train_batch_size, eval_batch_size, steps_per_eval, train_steps, num_train_images, num_eval_images, num_label_classes, tf_version)
  deploy = resnet_deploy_op(train.output, model, version, project_id, region, tf_version)

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(resnet_train, __file__ + '.tar.gz')

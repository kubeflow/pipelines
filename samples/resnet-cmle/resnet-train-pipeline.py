# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import mlp
import datetime


@mlp.pipeline(
  name='ResNet_Train_Pipeline',
  description='Demonstrate the ResNet50 predict.'
)
def resnet_train(project_id: mlp.PipelineParam=mlp.PipelineParam(name='project-id', value=''),
  bucket: mlp.PipelineParam=mlp.PipelineParam(name='bucket', value='bolts_resnet'),
  region: mlp.PipelineParam=mlp.PipelineParam(name='region', value='us-central1'),
  model: mlp.PipelineParam=mlp.PipelineParam(name='model', value='bolts'),
  version: mlp.PipelineParam=mlp.PipelineParam(name='version', value='resnet'),
  model_dir: mlp.PipelineParam=mlp.PipelineParam(name='model-dir', value='gs://bolts_resnet/tpu/resnet/trained'),
  tf_version: mlp.PipelineParam=mlp.PipelineParam(name='tf-version', value='1.8'),
  train_csv: mlp.PipelineParam=mlp.PipelineParam(name='train-csv', value='gs://bolts_image_dataset/bolt_images_train.csv'),
  validation_csv: mlp.PipelineParam=mlp.PipelineParam(name='validation-csv', value='gs://bolts_image_dataset/bolt_images_validate.csv'),
  labels: mlp.PipelineParam=mlp.PipelineParam(name='labels', value='gs://bolts_image_dataset/labels.txt'),
  depth: mlp.PipelineParam=mlp.PipelineParam(name='depth', value=50),
  train_batch_size: mlp.PipelineParam=mlp.PipelineParam(name='train-batch-size', value=1024),
  eval_batch_size: mlp.PipelineParam=mlp.PipelineParam(name='eval-batch-size', value=1024),
  steps_per_eval: mlp.PipelineParam=mlp.PipelineParam(name='steps-per-eval', value=250),
  train_steps: mlp.PipelineParam=mlp.PipelineParam(name='train-steps', value=1000),
  num_train_images: mlp.PipelineParam=mlp.PipelineParam(name='num-train-images', value=218593),
  num_eval_images: mlp.PipelineParam=mlp.PipelineParam(name='num-eval-images', value=54648),
  num_label_classes: mlp.PipelineParam=mlp.PipelineParam(name='num-label-classes', value=10)):

  preprocess = mlp.ContainerOp( 
      name = 'preprocess', 
      image = 'gcr.io/ml-pipeline/resnet-preprocess',
      arguments = ['--project_id', project_id, '--bucket', bucket, '--region', region, '--train_csv', train_csv,
        '--validation_csv', validation_csv,'--labels', labels,'--version', version],
      file_outputs = {'preprocessed': '/output.txt'})

  train = mlp.ContainerOp( 
      name = 'train', 
      image = 'gcr.io/ml-pipeline/resnet-train',
      arguments = ['--project_id', project_id, '--bucket', bucket, '--region', region, '--depth', depth,
        '--train_batch_size', train_batch_size,'--eval_batch_size', eval_batch_size,'--steps_per_eval', steps_per_eval,
        '--train_steps', train_steps,'--num_train_images', num_train_images,'--num_eval_images', num_eval_images,
        '--num_label_classes', num_label_classes,'--data_dir',preprocess.output,'--TFVERSION',tf_version,'--output_dir',model_dir],
      file_outputs = {'trained': '/output.txt'})

  deploy = mlp.ContainerOp( 
      name = 'deploy', 
      image = 'gcr.io/ml-pipeline/resnet-deploy',
      arguments = ['--model', model, '--version', version,'--project_id', project_id, '--bucket', bucket, 
        '--region', region,'--data_dir',train.output,'--TFVERSION',tf_version])

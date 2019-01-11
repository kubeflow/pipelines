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

import sys
from pathlib import Path

import kfp.dsl as dsl
import kfp.gcp as gcp

from kfp.components import ComponentStore

cs = ComponentStore()
cs.local_search_paths.append(str(Path(__file__).parent.joinpath('../../components/'))) #local repo checkout path
cs.url_search_prefixes.append('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/')
cs.url_search_prefixes.append('https://raw.githubusercontent.com/Ark-kun/pipelines/Added-component-definitions-to-our-components/components/')

dataflow_tf_transform_op = cs.load_component('dataflow/tft')
kubeflow_tf_training_op  = cs.load_component('kubeflow/dnntrainer')
dataflow_tf_predict_op   = cs.load_component('dataflow/predict')
confusion_matrix_op      = cs.load_component('local/confusion_matrix')

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
  training = kubeflow_tf_training_op(preprocess.output, schema, learning_rate, hidden_layer_size, steps, target, '', '%s/%s/train' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))
  if use_gpu:
    training.image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer-gpu:d3c4add0a95e930c70a330466d0923827784eb9a',
    training.set_gpu_limit(1)

  prediction = dataflow_tf_predict_op(evaluation, schema, target,  training.output, predict_mode, project, '%s/%s/predict' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))
  confusion_matrix = confusion_matrix_op(prediction.output, '%s/%s/confusionmatrix' % (output, workflow)).apply(gcp.use_gcp_secret('user-gcp-sa'))


if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(kubeflow_training, __file__ + '.tar.gz')

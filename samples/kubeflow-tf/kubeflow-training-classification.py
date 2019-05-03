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


import kfp
from kfp import components
from kfp import dsl
from kfp import gcp

dataflow_tf_transform_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e8524eefb138725fc06600d1956da0f4dd477178/components/dataflow/tft/component.yaml')
kubeflow_tf_training_op  = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e8524eefb138725fc06600d1956da0f4dd477178/components/kubeflow/dnntrainer/component.yaml')
dataflow_tf_predict_op   = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e8524eefb138725fc06600d1956da0f4dd477178/components/dataflow/predict/component.yaml')
confusion_matrix_op      = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e8524eefb138725fc06600d1956da0f4dd477178/components/local/confusion_matrix/component.yaml')

@dsl.pipeline(
    name='TF training and prediction pipeline',
    description=''
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
    predict_mode='local',
):
    output_template = str(output) + '/{{workflow.uid}}/{{pod.name}}/data'

    # set the flag to use GPU trainer
    use_gpu = False

    preprocess = dataflow_tf_transform_op(
        training_data_file_pattern=train,
        evaluation_data_file_pattern=evaluation,
        schema=schema,
        gcp_project=project,
        run_mode=preprocess_mode,
        preprocessing_module='',
        transformed_data_dir=output_template
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

    training = kubeflow_tf_training_op(
        transformed_data_dir=preprocess.output,
        schema=schema,
        learning_rate=learning_rate,
        hidden_layer_size=hidden_layer_size,
        steps=steps,
        target=target,
        preprocessing_module='',
        training_output_dir=output_template
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

    if use_gpu:
        training.image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer-gpu:727c48c690c081b505c1f0979d11930bf1ef07c0',
        training.set_gpu_limit(1)

    prediction = dataflow_tf_predict_op(
        data_file_pattern=evaluation,
        schema=schema,
        target_column=target,
        model=training.output,
        run_mode=predict_mode,
        gcp_project=project,
        predictions_dir=output_template
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

    confusion_matrix = confusion_matrix_op(
        predictions=prediction.output,
        output_dir=output_template
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(kubeflow_training, __file__ + '.zip')

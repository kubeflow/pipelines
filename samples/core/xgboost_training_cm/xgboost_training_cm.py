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

confusion_matrix_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e598176c02f45371336ccaa819409e8ec83743df/components/local/confusion_matrix/component.yaml')
roc_op =              components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e598176c02f45371336ccaa819409e8ec83743df/components/local/roc/component.yaml')

# ! Please do not forget to enable the Dataproc API in your cluster https://console.developers.google.com/apis/api/dataproc.googleapis.com/overview

# ================================================================
# The following classes should be provided by components provider.

def dataproc_create_cluster_op(
    project,
    region,
    staging,
    cluster_name='xgb-{{workflow.name}}'
):
    return dsl.ContainerOp(
        name='Dataproc - Create cluster',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--name', cluster_name,
            '--staging', staging,
        ],
        file_outputs={
            'output': '/output.txt',
        }
    )


def dataproc_delete_cluster_op(
    project,
    region,
    cluster_name='xgb-{{workflow.name}}'
):
    return dsl.ContainerOp(
        name='Dataproc - Delete cluster',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--name', cluster_name,
        ],
        is_exit_handler=True
    )


def dataproc_analyze_op(
    project,
    region,
    cluster_name,
    schema,
    train_data,
    output
):
    return dsl.ContainerOp(
        name='Dataproc - Analyze',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-analyze:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--cluster', cluster_name,
            '--schema', schema,
            '--train', train_data,
            '--output', output,
        ],
        file_outputs={
            'output': '/output.txt',
        }
    )


def dataproc_transform_op(
    project,
    region,
    cluster_name,
    train_data,
    eval_data,
    target,
    analysis,
    output
):
    return dsl.ContainerOp(
        name='Dataproc - Transform',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-transform:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--cluster', cluster_name,
            '--train', train_data,
            '--eval', eval_data,
            '--analysis', analysis,
            '--target', target,
            '--output', output,
        ],
        file_outputs={
            'train': '/output_train.txt',
            'eval': '/output_eval.txt',
        }
    )


def dataproc_train_op(
    project,
    region,
    cluster_name,
    train_data,
    eval_data,
    target,
    analysis,
    workers,
    rounds,
    output,
    is_classification=True
):
    if is_classification:
      config='gs://ml-pipeline-playground/trainconfcla.json'
    else:
      config='gs://ml-pipeline-playground/trainconfreg.json'

    return dsl.ContainerOp(
        name='Dataproc - Train XGBoost model',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-train:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--cluster', cluster_name,
            '--train', train_data,
            '--eval', eval_data,
            '--analysis', analysis,
            '--target', target,
            '--package', 'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            '--workers', workers,
            '--rounds', rounds,
            '--conf', config,
            '--output', output,
        ],
        file_outputs={
            'output': '/output.txt',
        }
    )


def dataproc_predict_op(
    project,
    region,
    cluster_name,
    data,
    model,
    target,
    analysis,
    output
):
    return dsl.ContainerOp(
        name='Dataproc - Predict with XGBoost model',
        image='gcr.io/ml-pipeline/ml-pipeline-dataproc-predict:57d9f7f1cfd458e945d297957621716062d89a49',
        arguments=[
            '--project', project,
            '--region', region,
            '--cluster', cluster_name,
            '--predict', data,
            '--analysis', analysis,
            '--target', target,
            '--package', 'gs://ml-pipeline-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar',
            '--model', model,
            '--output', output,
        ],
        file_outputs={
            'output': '/output.txt',
        },
        output_artifact_paths={
            'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json',
        },
    )

# =======================================================================

@dsl.pipeline(
    name='XGBoost Trainer',
    description='A trainer that does end-to-end distributed training for XGBoost models.'
)
def xgb_train_pipeline(
    output,
    project,
    region='us-central1',
    train_data='gs://ml-pipeline-playground/sfpd/train.csv',
    eval_data='gs://ml-pipeline-playground/sfpd/eval.csv',
    schema='gs://ml-pipeline-playground/sfpd/schema.json',
    target='resolution',
    rounds=200,
    workers=2,
    true_label='ACTION',
):
    output_template = str(output) + '/' + dsl.EXECUTION_ID_PLACEHOLDER + '/data'

    delete_cluster_op = dataproc_delete_cluster_op(
        project,
        region
    ).apply(gcp.use_gcp_secret('user-gcp-sa'))

    with dsl.ExitHandler(exit_op=delete_cluster_op):
        create_cluster_op = dataproc_create_cluster_op(
            project,
            region,
            output
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        analyze_op = dataproc_analyze_op(
            project,
            region,
            create_cluster_op.output,
            schema,
            train_data,
            output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        transform_op = dataproc_transform_op(
            project,
            region,
            create_cluster_op.output,
            train_data,
            eval_data,
            target,
            analyze_op.output,
            output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        train_op = dataproc_train_op(
            project,
            region,
            create_cluster_op.output,
            transform_op.outputs['train'],
            transform_op.outputs['eval'],
            target,
            analyze_op.output,
            workers,
            rounds,
            output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        predict_op = dataproc_predict_op(
            project,
            region,
            create_cluster_op.output,
            transform_op.outputs['eval'],
            train_op.output,
            target,
            analyze_op.output,
            output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        confusion_matrix_task = confusion_matrix_op(
            predict_op.output,
            output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

        roc_task = roc_op(
            predictions_dir=predict_op.output,
            true_class=true_label,
            true_score_column=true_label,
            output_dir=output_template
        ).apply(gcp.use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(xgb_train_pipeline, __file__ + '.yaml')

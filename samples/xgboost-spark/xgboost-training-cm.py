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

from kfp import components

confusion_matrix_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/local/confusion_matrix/component.yaml')
roc_op =              components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/local/roc/component.yaml')

# ================================================================
# The following classes should be provided by components provider.

class CreateClusterOp(dsl.ContainerOp):

  def __init__(self, name, project, region, staging):
    super(CreateClusterOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
      arguments=[
          '--project', project,
          '--region', region,
          '--name', 'xgb-{{workflow.name}}',
          '--staging', staging
     ],
     file_outputs={'output': '/output.txt'})


class DeleteClusterOp(dsl.ContainerOp):

  def __init__(self, name, project, region):
    super(DeleteClusterOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
      arguments=[
          '--project', project,
          '--region', region,
          '--name', 'xgb-{{workflow.name}}',
      ],
      is_exit_handler=True)


class AnalyzeOp(dsl.ContainerOp):

  def __init__(self, name, project, region, cluster_name, schema, train_data, output):
    super(AnalyzeOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-analyze:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
      arguments=[
          '--project', project,
          '--region', region,
          '--cluster', cluster_name,
          '--schema', schema,
          '--train', train_data,
          '--output', output,
     ],
     file_outputs={'output': '/output.txt'})


class TransformOp(dsl.ContainerOp):

  def __init__(self, name, project, region, cluster_name, train_data, eval_data,
               target, analysis, output):
    super(TransformOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-transform:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
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
     file_outputs={'train': '/output_train.txt', 'eval': '/output_eval.txt'})


class TrainerOp(dsl.ContainerOp):

  def __init__(self, name, project, region, cluster_name, train_data, eval_data,
               target, analysis, workers, rounds, output, is_classification=True):
    if is_classification:
      config='gs://ml-pipeline-playground/trainconfcla.json'
    else:
      config='gs://ml-pipeline-playground/trainconfreg.json'

    super(TrainerOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-train:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
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
      file_outputs={'output': '/output.txt'})


class PredictOp(dsl.ContainerOp):

  def __init__(self, name, project, region, cluster_name, data, model, target, analysis, output):
    super(PredictOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-dataproc-predict:a277f87ea1d4707bf860d080d06639b7caf9a1cf',
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
      file_outputs={'output': '/output.txt'})

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
  delete_cluster_op = DeleteClusterOp('delete-cluster', project, region).apply(gcp.use_gcp_secret('user-gcp-sa'))
  with dsl.ExitHandler(exit_op=delete_cluster_op):
    create_cluster_op = CreateClusterOp('create-cluster', project, region, output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    analyze_op = AnalyzeOp('analyze', project, region, create_cluster_op.output, schema,
                           train_data, '%s/{{workflow.name}}/analysis' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    transform_op = TransformOp('transform', project, region, create_cluster_op.output,
                               train_data, eval_data, target, analyze_op.output,
                               '%s/{{workflow.name}}/transform' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    train_op = TrainerOp('train', project, region, create_cluster_op.output, transform_op.outputs['train'],
                         transform_op.outputs['eval'], target, analyze_op.output, workers,
                         rounds, '%s/{{workflow.name}}/model' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    predict_op = PredictOp('predict', project, region, create_cluster_op.output, transform_op.outputs['eval'],
                           train_op.output, target, analyze_op.output, '%s/{{workflow.name}}/predict' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    confusion_matrix_task = confusion_matrix_op(predict_op.output,
                                            '%s/{{workflow.name}}/confusionmatrix' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

    roc_task = roc_op(predict_op.output, true_label, '%s/{{workflow.name}}/roc' % output).apply(gcp.use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(xgb_train_pipeline, __file__ + '.zip')

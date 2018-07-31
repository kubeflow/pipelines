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


# ================================================================
# The following classes should be provided by components provider.

class CreateClusterOp(mlp.ContainerOp):

  def __init__(self, name, project, region, staging):
    super(CreateClusterOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-create-cluster',
      arguments=[
          '--project', project,
          '--region', region,
          '--name', 'xgboost-spark-{{workflow.name}}',
          '--staging', staging
     ],
     file_outputs={'output': '/output.txt'})


class DeleteClusterOp(mlp.ContainerOp):

  def __init__(self, name, project, region):
    super(DeleteClusterOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-delete-cluster',
      arguments=[
          '--project', project,
          '--region', region,
          '--name', 'xgboost-spark-{{workflow.name}}',
      ],
      is_exit_handler=True)


class AnalyzeOp(mlp.ContainerOp):

  def __init__(self, name, project, region, cluster_name, schema, train_data, output):
    super(AnalyzeOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-analyze',
      arguments=[
          '--project', project,
          '--region', region,
          '--cluster', cluster_name,
          '--schema', schema,
          '--train', train_data,
          '--output', output,
     ],
     file_outputs={'output': '/output.txt'})


class TransformOp(mlp.ContainerOp):

  def __init__(self, name, project, region, cluster_name, train_data, eval_data,
               target, analysis, output):
    super(TransformOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-transform',
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


class TrainerOp(mlp.ContainerOp):

  def __init__(self, name, project, region, cluster_name, train_data, eval_data,
               target, analysis, workers, rounds, output, is_classification=True):
    if is_classification:
      config='gs://ml-pipeline-playground/trainconfcla.json'
    else:
      config='gs://ml-pipeline-playground/trainconfreg.json'

    super(TrainerOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-train',
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


class PredictOp(mlp.ContainerOp):

  def __init__(self, name, project, region, cluster_name, data, model, target, analysis, output):
    super(PredictOp, self).__init__(
      name=name,
      image='gcr.io/bradley-playground/ml-pipeline-dataproc-predict',
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


class ConfusionMatrixOp(mlp.ContainerOp):

  def __init__(self, name, predictions, output):
    super(ConfusionMatrixOp, self).__init__(
      name=name,
      image='gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix',
      arguments=[
        '--output', output,
        '--predictions', predictions
     ])

# =======================================================================

@mlp.pipeline(
  name='XGBoost Trainer',
  description='A trainer that does end-to-end training for XGBoost models.'
)
def xgb_train_pipeline(
    output,
    project,
    region=mlp.PipelineParam('region', value='us-central1'),
    train_data=mlp.PipelineParam('train-data', value='gs://ml-pipeline-playground/sfpd/train.csv'),
    eval_data=mlp.PipelineParam('eval-data', value='gs://ml-pipeline-playground/sfpd/eval.csv'),
    schema=mlp.PipelineParam('schema', value='gs://ml-pipeline-playground/sfpd/schema.json'),
    target=mlp.PipelineParam('target', value='resolution'),
    rounds=mlp.PipelineParam('rounds', value=200),
    workers=mlp.PipelineParam('workers', value=2),
):
  delete_cluster_op = DeleteClusterOp('delete-cluster', project, region)
  with mlp.ExitHandler(exit_op=delete_cluster_op):
    create_cluster_op = CreateClusterOp('create-cluster', project, region, output)

    analyze_op = AnalyzeOp('analyze', project, region, create_cluster_op.output, schema,
                           train_data, '%s/{{workflow.name}}/analysis' % output)
    transform_op = TransformOp('transform', project, region, create_cluster_op.output,
                               train_data, eval_data, target, analyze_op.output,
                               '%s/{{workflow.name}}/transform' % output)

    train_op = TrainerOp('train', project, region, create_cluster_op.output, transform_op.outputs['train'],
                         transform_op.outputs['eval'], target, analyze_op.output, workers,
                         rounds, '%s/{{workflow.name}}/model' % output)
    predict_op = PredictOp('predict', project, region, create_cluster_op.output, transform_op.outputs['eval'],
                           train_op.output, target, analyze_op.output, '%s/{{workflow.name}}/predict' % output)

    confusion_matrix_op = ConfusionMatrixOp('confusion-matrix', predict_op.output,
                                            '%s/{{workflow.name}}/confusionmatrix' % output)



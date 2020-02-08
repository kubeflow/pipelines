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

import os

from typing import Text

import kfp
from kfp import dsl
from tfx.components.evaluator.component import Evaluator
from tfx.components.example_gen.csv_example_gen.component import CsvExampleGen
from tfx.components.example_validator.component import ExampleValidator
from tfx.components.model_validator.component import ModelValidator
from tfx.components.pusher.component import Pusher
from tfx.components.schema_gen.component import SchemaGen
from tfx.components.statistics_gen.component import StatisticsGen
from tfx.components.trainer.component import Trainer
from tfx.components.transform.component import Transform
from tfx.orchestration import data_types
from tfx.orchestration import pipeline
from tfx.orchestration.kubeflow import kubeflow_dag_runner
from tfx.proto import evaluator_pb2
from tfx.utils.dsl_utils import external_input
from tfx.proto import pusher_pb2
from tfx.proto import trainer_pb2

# Define pipeline params used for pipeline execution.
# Path to the module file, should be a GCS path.
_taxi_module_file_param = data_types.RuntimeParameter(
    name='module-file',
    default=
    'gs://ml-pipeline-playground/tfx_taxi_simple/modules/tfx_taxi_utils_1205.py',
    ptype=Text,
)

# Path to the CSV data file, under which their should be a data.csv file.
_data_root_param = data_types.RuntimeParameter(
    name='data-root',
    default='gs://ml-pipeline-playground/tfx_taxi_simple/data',
    ptype=Text,
)

# Path of pipeline root, should be a GCS path.
pipeline_root = os.path.join(
    'gs://<your-gcs-bucket>', 'tfx_taxi_simple', kfp.dsl.RUN_ID_PLACEHOLDER
)


def _create_test_pipeline(
    pipeline_root: Text, csv_input_location: data_types.RuntimeParameter,
    taxi_module_file: data_types.RuntimeParameter, enable_cache: bool
):
  """Creates a simple Kubeflow-based Chicago Taxi TFX pipeline.

  Args:
    pipeline_root: The root of the pipeline output.
    csv_input_location: The location of the input data directory.
    taxi_module_file: The location of the module file for Transform/Trainer.
    enable_cache: Whether to enable cache or not.

  Returns:
    A logical TFX pipeline.Pipeline object.
  """
  examples = external_input(csv_input_location)

  example_gen = CsvExampleGen(input=examples)
  statistics_gen = StatisticsGen(examples=example_gen.outputs['examples'])
  infer_schema = SchemaGen(
      statistics=statistics_gen.outputs['statistics'],
      infer_feature_shape=False,
  )
  validate_stats = ExampleValidator(
      statistics=statistics_gen.outputs['statistics'],
      schema=infer_schema.outputs['schema'],
  )
  transform = Transform(
      examples=example_gen.outputs['examples'],
      schema=infer_schema.outputs['schema'],
      module_file=taxi_module_file,
  )
  trainer = Trainer(
      module_file=taxi_module_file,
      transformed_examples=transform.outputs['transformed_examples'],
      schema=infer_schema.outputs['schema'],
      transform_graph=transform.outputs['transform_graph'],
      train_args=trainer_pb2.TrainArgs(num_steps=10),
      eval_args=trainer_pb2.EvalArgs(num_steps=5),
  )
  model_analyzer = Evaluator(
      examples=example_gen.outputs['examples'],
      model=trainer.outputs['model'],
      feature_slicing_spec=evaluator_pb2.FeatureSlicingSpec(
          specs=[
              evaluator_pb2.SingleSlicingSpec(
                  column_for_slicing=['trip_start_hour']
              )
          ]
      ),
  )
  model_validator = ModelValidator(
      examples=example_gen.outputs['examples'], model=trainer.outputs['model']
  )

  # Hack: ensuring push_destination can be correctly parameterized and interpreted.
  # pipeline root will be specified as a dsl.PipelineParam with the name
  # pipeline-root, see:
  # https://github.com/tensorflow/tfx/blob/1c670e92143c7856f67a866f721b8a9368ede385/tfx/orchestration/kubeflow/kubeflow_dag_runner.py#L226
  _pipeline_root_param = dsl.PipelineParam(name='pipeline-root')
  pusher = Pusher(
      model=trainer.outputs['model'],
      model_blessing=model_validator.outputs['blessing'],
      push_destination=pusher_pb2.PushDestination(
          filesystem=pusher_pb2.PushDestination.Filesystem(
              base_directory=os.path.
              join(str(_pipeline_root_param), 'model_serving')
          )
      ),
  )

  return pipeline.Pipeline(
      pipeline_name='parameterized_tfx_oss',
      pipeline_root=pipeline_root,
      components=[
          example_gen, statistics_gen, infer_schema, validate_stats, transform,
          trainer, model_analyzer, model_validator, pusher
      ],
      enable_cache=enable_cache,
  )


if __name__ == '__main__':

  enable_cache = True
  pipeline = _create_test_pipeline(
      pipeline_root,
      _data_root_param,
      _taxi_module_file_param,
      enable_cache=enable_cache,
  )
  # Make sure the version of TFX image used is consistent with the version of
  # TFX SDK. Here we use tfx:0.15.0 image.
  config = kubeflow_dag_runner.KubeflowDagRunnerConfig(
      kubeflow_metadata_config=kubeflow_dag_runner.
      get_default_kubeflow_metadata_config(),
      # TODO: remove this override when KubeflowDagRunnerConfig doesn't default to use_gcp_secret op.
      pipeline_operator_funcs=list(
          filter(
              lambda operator: operator.__name__.find('gcp_secret') == -1,
              kubeflow_dag_runner.get_default_pipeline_operator_funcs()
          )
      ),
      tfx_image='tensorflow/tfx:0.21.0rc0',
  )
  kfp_runner = kubeflow_dag_runner.KubeflowDagRunner(
      output_filename=__file__ + '.yaml', config=config
  )

  kfp_runner.run(pipeline)

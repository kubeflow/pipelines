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
from tfx.orchestration import pipeline
from tfx.orchestration.kubeflow import kubeflow_dag_runner
from tfx.proto import evaluator_pb2
from tfx.utils.dsl_utils import csv_input
from tfx.proto import pusher_pb2
from tfx.proto import trainer_pb2

# Define pipeline params used for pipeline execution.
# Path to the module file, should be a GCS path.
_taxi_module_file_param = dsl.PipelineParam(
    name='module-file',
    value='gs://ml-pipeline-playground/tfx_taxi_simple/modules/taxi_utils.py'
)

# Path to the CSV data file, under which their should be a data.csv file.
_data_root_param = dsl.PipelineParam(
    name='data-root', value='gs://ml-pipeline-playground/tfx_taxi_simple/data'
)

# Path of pipeline root, should be a GCS path.
pipeline_root = os.path.join(
    'gs://your-bucket', 'tfx_taxi_simple', kfp.dsl.RUN_ID_PLACEHOLDER
)


def _create_test_pipeline(
    pipeline_root: Text, csv_input_location: Text, taxi_module_file: Text,
    enable_cache: bool
):
  """Creates a simple Kubeflow-based Chicago Taxi TFX pipeline.

  Args:
    pipeline_name: The name of the pipeline.
    pipeline_root: The root of the pipeline output.
    csv_input_location: The location of the input data directory.
    taxi_module_file: The location of the module file for Transform/Trainer.
    enable_cache: Whether to enable cache or not.

  Returns:
    A logical TFX pipeline.Pipeline object.
  """
  examples = csv_input(csv_input_location)

  example_gen = CsvExampleGen(input_base=examples)
  statistics_gen = StatisticsGen(input_data=example_gen.outputs.examples)
  infer_schema = SchemaGen(
      stats=statistics_gen.outputs.output, infer_feature_shape=False,
  )
  validate_stats = ExampleValidator(
      stats=statistics_gen.outputs.output, schema=infer_schema.outputs.output,
  )
  transform = Transform(
      input_data=example_gen.outputs.examples,
      schema=infer_schema.outputs.output,
      module_file=taxi_module_file,
  )
  trainer = Trainer(
      module_file=taxi_module_file,
      transformed_examples=transform.outputs.transformed_examples,
      schema=infer_schema.outputs.output,
      transform_output=transform.outputs.transform_output,
      train_args=trainer_pb2.TrainArgs(num_steps=10),
      eval_args=trainer_pb2.EvalArgs(num_steps=5),
  )
  model_analyzer = Evaluator(
      examples=example_gen.outputs.examples,
      model_exports=trainer.outputs.output,
      feature_slicing_spec=evaluator_pb2.FeatureSlicingSpec(
          specs=[
              evaluator_pb2.SingleSlicingSpec(
                  column_for_slicing=['trip_start_hour']
              )
          ]
      ),
  )
  model_validator = ModelValidator(
      examples=example_gen.outputs.examples, model=trainer.outputs.output
  )

  # Hack: ensuring push_destination can be correctly parameterized and interpreted.
  # pipeline root will be specified as a dsl.PipelineParam with the name
  # pipeline-root, see:
  # https://github.com/tensorflow/tfx/blob/1c670e92143c7856f67a866f721b8a9368ede385/tfx/orchestration/kubeflow/kubeflow_dag_runner.py#L226
  _pipeline_root_param = dsl.PipelineParam(name='pipeline-root')
  pusher = Pusher(
      model_export=trainer.outputs.output,
      model_blessing=model_validator.outputs.blessing,
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
      str(_data_root_param),
      str(_taxi_module_file_param),
      enable_cache=enable_cache,
  )
  config = kubeflow_dag_runner.KubeflowDagRunnerConfig(
      kubeflow_metadata_config=kubeflow_dag_runner.
      get_default_kubeflow_metadata_config(),
      tfx_image='tensorflow/tfx:0.16.0.dev20191101',
  )
  kfp_runner = kubeflow_dag_runner.KubeflowDagRunner(config=config)
  # Make sure kfp_runner recognizes those parameters.
  kfp_runner._params.extend([_data_root_param, _taxi_module_file_param])

  kfp_runner.run(pipeline)

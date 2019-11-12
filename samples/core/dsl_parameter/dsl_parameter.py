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
"""This sample demonstrates how to use pipeline parameter in TFX DSL."""

import os

from typing import Optional, Text

import kfp
from kfp import dsl

from tfx.components.example_gen.csv_example_gen.component import CsvExampleGen
from tfx.orchestration import pipeline
from tfx.orchestration.experimental.parameter import runtime_parameter
from tfx.orchestration.kubeflow import kubeflow_dag_runner
from tfx.utils.dsl_utils import csv_input
from tfx.proto import example_gen_pb2

# Path of pipeline root, should be a GCS path.
pipeline_root = os.path.join(
    'gs://your-bucket', 'tfx_taxi_simple', kfp.dsl.RUN_ID_PLACEHOLDER
)

# Path to the CSV data file, under which their should be a data.csv file.
# Note: this is still digested as raw PipelineParam b/c parameterization of
# ExternalArtifact attributes has not been implemented yet in TFX.
_data_root_param = dsl.PipelineParam(
    name='data-root',
    value='gs://ml-pipeline-playground/tfx_taxi_simple/data')

# Name of the output split from ExampleGen. Specified as a RuntimeParameter.
_example_split_name = runtime_parameter.RuntimeParameter(
    name='split-name', default='train'
)


def _create_one_step_pipeline(
    pipeline_name: Text,
    pipeline_root: Text,
    enable_cache: Optional[bool] = True
) -> pipeline.Pipeline:
  """Creates a simple TFX pipeline including only an ExampleGen.

  Args:
    pipeline_name: The name of the pipeline.
    pipeline_root: The root of the pipeline output.
    csv_input_location: The location of the input data directory.

  Returns:
    A logical TFX pipeline.Pipeline object.
  """
  example_split_name = runtime_parameter.RuntimeParameter(
      name='split-name', default='train'
  )
  examples = csv_input(str(_data_root_param))
  example_gen = CsvExampleGen(
      input=examples,
      output_config=example_gen_pb2.Output(
          split_config=example_gen_pb2.SplitConfig(
              splits=[
                  example_gen_pb2.SplitConfig.
                  Split(name=example_split_name, hash_buckets=10)
              ]
          )
      )
  )
  return pipeline.Pipeline(
      pipeline_name=pipeline_name,
      pipeline_root=pipeline_root,
      components=[example_gen],
      enable_cache=enable_cache,
  )


if __name__ == '__main__':

  enable_cache = True
  pipeline = _create_one_step_pipeline(
      'dsl_parameter', pipeline_root, enable_cache=enable_cache
  )
  config = kubeflow_dag_runner.KubeflowDagRunnerConfig(
      kubeflow_metadata_config=kubeflow_dag_runner.
      get_default_kubeflow_metadata_config(),
      tfx_image='tensorflow/tfx:latest'
  )
  kfp_runner = kubeflow_dag_runner.KubeflowDagRunner(config=config)
  # Make sure kfp_runner recognizes those parameters.
  kfp_runner._params.extend([_data_root_param])

  kfp_runner.run(pipeline)

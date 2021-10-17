"""Library used to run model evaluation service backend."""

from typing import Any, Dict, Optional

import apache_beam
from apache_beam.options import pipeline_options

from lib import beam as beam_lib
from lib import config
from lib import constants
from lib.proto import model_evaluation_pb2
from lib.proto import configuration_pb2
from google.protobuf import json_format

JsonType = Dict[str, Any]


def _run_backend(
    be_config: model_evaluation_pb2.EvaluationConfig,
    pipeline_opts: pipeline_options.PipelineOptions,
    problem_type: constants.ProblemType,
    tfma_format: Optional[bool] = False,
    json_mode: Optional[bool] = False
) -> apache_beam.runners.runner.PipelineState:
  """Build and execute model evaluation pipeline."""
  pipeline = apache_beam.Pipeline(options=pipeline_opts)
  beam_lib.append_tfma_pipeline(
      pipeline=pipeline,
      me_eval_config=be_config,
      problem_type=problem_type,
      tfma_format=tfma_format,
      json_mode=json_mode)
  result = pipeline.run()
  return result.wait_until_finish()


def run_backend_from_model_eval_service_json_config(
    mes_config_json: JsonType,
    tfma_format: Optional[bool] = False,
    json_mode: Optional[bool] = False
) -> apache_beam.runners.runner.PipelineState:
  """Builds and executes model evaluation pipeline from service config.

  Args:
    mes_config_json: Model Evaluation Service configuration serialized as a JSON
      string.
    tfma_format: Whether to output metrics in the same format as Tensorflow
      Model Analysis library.
    json_mode: Whether to output metrics in JSON format.

  Returns:
    PipelineResult object with the results of the model evaluation
    pipeline execution.
  """
  mes_config = configuration_pb2.EvaluationRunConfig()
  json_format.ParseDict(mes_config_json, mes_config)
  be_config = config.get_evaluation_config_from_service_config(mes_config)
  pipeline_opts = config.get_pipeline_options_from_service_config(mes_config)
  problem_type = config.get_problem_type_from_service_config(mes_config)
  return _run_backend(
      be_config,
      pipeline_opts,
      problem_type,
      tfma_format=tfma_format,
      json_mode=json_mode)

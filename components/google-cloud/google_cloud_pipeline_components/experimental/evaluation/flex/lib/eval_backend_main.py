"""Backend binary of the Model Evaluation Service."""

import argparse
import json
import sys

from absl import app
from apache_beam.runners import runner
import tensorflow.compat.v1 as tf

from lib import eval_backend_lib
from lib import pipeline_options

_USAGE = """
  eval_backend_main --config=<serialized_json>
"""


def main(argv):
  if not argv:
    raise app.UsageError('USAGE: \n {}'.format(_USAGE))

  _, pipeline_args = argparse.ArgumentParser().parse_known_args(argv)
  pipeline_opts = pipeline_options.ModelEvaluationServiceOptions(pipeline_args)
  model_eval_json_config = pipeline_opts.config

  # Use GCS as input of the model_eval_json_config.
  if pipeline_opts.config_gcs_uri:
    with tf.io.gfile.GFile(pipeline_opts.config_gcs_uri, 'rb') as f:
      model_eval_json_config = json.loads(f.read())

  if pipeline_opts.output_gcs_uri:
    model_eval_json_config['output'] = {
        'gcs_sink': {
            'path': pipeline_opts.output_gcs_uri
        }
    }

  devel_pipeline_opts = pipeline_options.DevelModelEvaluationServiceOptions(
      pipeline_args)
  tfma_format = devel_pipeline_opts.tfma_format
  json_mode = devel_pipeline_opts.json_mode
  pipeline_state = (
      eval_backend_lib.run_backend_from_model_eval_service_json_config(
          mes_config_json=model_eval_json_config,
          tfma_format=tfma_format,
          json_mode=json_mode))
  if pipeline_state == runner.PipelineState.FAILED:
    sys.exit(1)


if __name__ == '__main__':
  app.run(main)

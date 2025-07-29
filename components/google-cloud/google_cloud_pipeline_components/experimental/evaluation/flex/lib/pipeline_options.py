"""Definition for the Model Eval Service pipeline options."""

import json

from apache_beam.options.pipeline_options import PipelineOptions


class ModelEvaluationServiceOptions(PipelineOptions):
  """Options supplied by model evaluation service."""

  @classmethod
  def _add_argparse_args(cls, parser):
    parser.add_argument(
        '--config',
        type=json.loads,
        help=(
            'Configuration for the model evaluation backend represented as a '
            'serialized JSON string. The configuration message is defined in '
            'cloud/ai/platform/evaluation/metrics/proto/configuration.proto.'))
    parser.add_argument(
        '--config_gcs_uri',
        type=str,
        default='',
        help=(
            'GCS URI of the GCS object that contains the content of'
            'configuration for the model evaluation backend represented as a '
            'serialized JSON string. The configuration message is defined in '
            'cloud/ai/platform/evaluation/metrics/proto/configuration.proto.'))
    parser.add_argument(
        '--output_gcs_uri',
        type=str,
        default='',
        help=(
            'GCS URI of the directory where the metrics file will be outputted'
        ))


def _my_boolean_parser(argument):
  """Support both --flag and --flag=<bool_value>."""
  bool_arg = True
  if not argument or argument.lower() == 'false':
    bool_arg = False
  return bool_arg


class DevelModelEvaluationServiceOptions(PipelineOptions):
  """Developer options."""

  @classmethod
  def _add_argparse_args(cls, parser):
    parser.add_argument(
        '--json_mode',
        type=_my_boolean_parser,
        default=False,
        help=('If true, output as JSON parseable format instead of TFRecord.'))
    parser.add_argument(
        '--tfma_format',
        type=_my_boolean_parser,
        default=False,
        help=('If true, output TFMA-compatible metrics instead of MES.'))

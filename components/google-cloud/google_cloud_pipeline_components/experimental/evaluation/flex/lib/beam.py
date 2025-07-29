"""Beam-specific functionality for Cloud AI Evluation Metrics."""

import os.path
from typing import Any, Iterable, List, Optional

import apache_beam as beam
import tensorflow_model_analysis as tfma
from tensorflow_model_analysis.proto import metrics_for_slice_pb2
from tfx_bsl.tfxio import tensor_adapter
from tfx_bsl.tfxio import tf_example_record

from lib import column_spec
from lib import constants
from lib import evaluation_column_specs as ecs
from lib import io
from lib import tfma_adapter
from lib.proto import model_evaluation_pb2 as me_proto

from google.protobuf import json_format, message

ColumnSpec = column_spec.ColumnSpec
EvaluationColumnSpecs = ecs.EvaluationColumnSpecs


class JSONToSerializedExample(beam.DoFn):
  """Deserialize JSON data and return serialized tf.Example."""

  def __init__(self,
               eval_column_specs: EvaluationColumnSpecs,
               class_list: Optional[List[str]] = None,
               quantile_list: Optional[List[float]] = None,
               quantile_index: Optional[int] = None):
    """Construct a transformation.

    Args:
      eval_column_specs: The set of specs for columns needed for model
        evaluation.
      class_list: For classification task, a list of class names.
      quantile_list: For quantile forecasting task, a list of quantiles.
      quantile_index: For quantile forecasting task, the quantile index.
    """
    self._eval_column_specs = eval_column_specs
    self.class_list = class_list
    self.quantile_list = quantile_list
    self.quantile_index = quantile_index

  def process(self, input_data: bytes) -> Iterable[bytes]:
    """Convert a single row of data.

    Args:
      input_data: A JSON serialized dict of features, labels, and predictions.

    Yields:
      a tensorflow.Example serialized as bytes.
    """
    tf_example = io.parse_row_as_example(
        json_data=input_data,
        evaluation_column_specs=self._eval_column_specs,
        class_list=self.class_list,
        quantile_list=self.quantile_list,
        quantile_index=self.quantile_index)
    yield tf_example.SerializeToString()


_TFMA_METRICS_KEY = 'metrics'


def _proto_as_json(msg: message.Message) -> str:
  return json_format.MessageToJson(msg, indent=0).replace('\n', '')


class MakeSliceMetricsSet(beam.CombineFn):
  """A Combiner that groups a list of SliceMetricsSet into a SlicedMetricSet."""

  def create_accumulator(self):
    return me_proto.SlicedMetricsSet()

  def add_input(self, accumulator, value):
    accumulator.sliced_metrics.extend(value.sliced_metrics)
    return accumulator

  def merge_accumulators(self, accumulators):
    result = me_proto.SlicedMetricsSet()
    for accumulator in accumulators:
      result.sliced_metrics.extend(accumulator.sliced_metrics)
    return result

  def extract_output(self, accumulator):
    return accumulator


@beam.ptransform_fn
# TODO(b/148788775): These type hints fail Beam type checking. Failing test:
# //cloud/ai/platform/evaluation/metrics/lib:beam_test
# @beam.typehints.with_input_types(tfma.evaluators.Evaluation)
# @beam.typehints.with_output_types(beam.pvalue.PDone)
def _write_metrics(evaluation: tfma.evaluators.Evaluation,
                   output_file: str,
                   problem_type: constants.ProblemType,
                   class_labels: Optional[List[str]] = None,
                   tfma_format: Optional[bool] = False,
                   json_mode: Optional[bool] = False):
  """PTransform to write metrics in different formats."""

  def _convert_metrics(tfma_metrics):
    return tfma_adapter.TFMAToME.model_evaluation_metrics(
        tfma_metrics, problem_type, class_labels)

  extract = (
      evaluation[_TFMA_METRICS_KEY] | 'ConvertSliceMetricsToProto' >> beam.Map(
          tfma.writers.metrics_plots_and_validations_writer
          .convert_slice_metrics_to_proto,
          add_metrics_callbacks=[]))

  if tfma_format:
    metrics = extract
  else:
    metrics = (
        extract | 'TFMAMetricsToME' >> beam.Map(_convert_metrics) |
        'Combine' >> beam.CombineGlobally(MakeSliceMetricsSet()))

  if json_mode:
    _ = (
        metrics | 'ToJSON' >> beam.Map(_proto_as_json) |
        'WriteToText' >> beam.io.WriteToText(
            file_path_prefix=output_file,
            shard_name_template='',
            file_name_suffix=''))
  elif tfma_format:
    _ = (
        metrics | 'WriteToTFRecord' >> beam.io.WriteToTFRecord(
            file_path_prefix=output_file,
            shard_name_template='',
            file_name_suffix='',
            coder=beam.coders.ProtoCoder(metrics_for_slice_pb2.MetricsForSlice))
    )
  else:
    _ = (
        metrics | 'Write' >> beam.io.WriteToText(
            file_path_prefix=output_file,
            shard_name_template='',
            file_name_suffix='',
            append_trailing_newlines=False,
            compression_type='uncompressed',
            coder=beam.coders.ProtoCoder(me_proto.SlicedMetricsSet)))


def append_tfma_pipeline(pipeline: beam.Pipeline,
                         me_eval_config: me_proto.EvaluationConfig,
                         problem_type: constants.ProblemType,
                         tfma_format: Optional[bool] = False,
                         json_mode: Optional[bool] = False,
                         schema: Optional[Any] = None):
  """Extend a beam pipeline to add TFMA evaluation given a configuration.

  Args:
    pipeline: A beam pipeline.
    me_eval_config: A ME Evaluation Configuration.
    problem_type: Defines what type of problem to expect.
    tfma_format: If true, use TFMA format, if false use Model Evaluation.
    json_mode: Output metrics in a plain text mode.
    schema: Optional tf.metadata schema. If you need to pass multi-tensor input
      to the model, you need to pass the schema.
  """
  input_files = (
      me_eval_config.data_spec.input_source_spec.jsonl_file_spec.file_names)
  output_path = me_eval_config.output_spec.gcs_sink.path
  data_spec = me_eval_config.data_spec
  weight_column_spec = ColumnSpec(
      me_eval_config.data_spec.example_weight_key_spec
  ) if me_eval_config.data_spec.HasField('example_weight_key_spec') else None
  eval_column_specs = EvaluationColumnSpecs(
      ground_truth_column_spec=ColumnSpec(
          me_eval_config.data_spec.label_key_spec),
      example_weight_column_spec=weight_column_spec,
      predicted_score_column_spec=ColumnSpec(data_spec.predicted_score_key_spec)
      if data_spec.HasField('predicted_score_key_spec') else None,
      predicted_label_column_spec=ColumnSpec(data_spec.predicted_label_key_spec)
      if data_spec.HasField('predicted_label_key_spec') else None,
      predicted_label_id_column_spec=ColumnSpec(
          data_spec.predicted_label_id_key_spec)
      if data_spec.HasField('predicted_label_id_key_spec') else None)
  class_name_list = list(me_eval_config.data_spec.labels) or None
  quantile_list = list(data_spec.quantiles) or None
  quantile_index = data_spec.quantile_index if data_spec.quantile_index >= 0 else None
  tfma_eval_config = tfma_adapter.METoTFMA(class_name_list).eval_config(
      me_eval_config)
  me_writers = [
      tfma.writers.Writer(
          stage_name='WriteMetrics',
          # pylint:disable=no-value-for-parameter
          ptransform=_write_metrics(
              output_file=os.path.join(output_path,
                                       constants.Pipeline.METRICS_KEY),
              problem_type=problem_type,
              class_labels=class_name_list,
              tfma_format=tfma_format,
              json_mode=json_mode)),
  ]

  coder = tf_example_record.TFExampleBeamRecord(
      physical_format='inmem',
      schema=schema,
      raw_record_column_name=tfma.ARROW_INPUT_COLUMN,
      telemetry_descriptors=None)
  tensor_adapter_config = None
  if schema is not None:
    tensor_adapter_config = tensor_adapter.TensorAdapterConfig(
        arrow_schema=coder.ArrowSchema(),
        tensor_representations=coder.TensorRepresentations())
  _ = (
      pipeline |
      'InputFileList' >> beam.Create(input_files) |
      'ReadText' >> beam.io.textio.ReadAllFromText() |
      'ParseData' >> beam.ParDo(
          JSONToSerializedExample(
              eval_column_specs=eval_column_specs,
              class_list=class_name_list,
              quantile_list=quantile_list,
              quantile_index=quantile_index)) |
      'ExamplesToRecordBatch' >> coder.BeamSource() |
      'ExtractEvaluateAndWriteResults' >> tfma.ExtractEvaluateAndWriteResults(
          eval_config=tfma_eval_config,
          writers=me_writers,
          tensor_adapter_config=tensor_adapter_config))

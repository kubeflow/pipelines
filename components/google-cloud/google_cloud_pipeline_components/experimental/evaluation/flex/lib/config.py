"""Builds configurations for running metrics."""

import os
from typing import List, Optional, Text

from apache_beam.options import pipeline_options as pipeline_opts_lib
from tensorflow_model_analysis.proto import config_pb2

from lib import column_spec
from lib import constants
from lib import evaluation_column_specs as ecs
from lib import tfma_adapter
from lib import tfma_metrics
from lib.constants import Metric
from lib.proto import model_evaluation_pb2
from lib.proto import configuration_pb2
from google.protobuf import message

ColumnSpec = column_spec.ColumnSpec
EvaluationColumnSpecs = ecs.EvaluationColumnSpecs

CLASSIFICATION_METRICS = [
    Metric.AUC,
    Metric.AUC_PRECISION_RECALL,
    Metric.CONFUSION_MATRIX,
    Metric.EXAMPLE_COUNT,
    Metric.WEIGHTED_EXAMPLE_COUNT,
    Metric.PRECISION,
    Metric.RECALL,
    Metric.BINARY_ACCURACY,
    Metric.BINARY_CROSSENTROPY,
    Metric.CATEGORICAL_ACCURACY,
    Metric.CATEGORICAL_CROSSENTROPY,
    Metric.MULTI_CLASS_CONFUSION_MATRIX,
]

REGRESSION_METRICS = [
    Metric.ACCURACY,
    Metric.EXAMPLE_COUNT,
    Metric.MEAN_ABSOLUTE_ERROR,
    Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR,
    Metric.MEAN_SQUARED_LOG_ERROR,
    Metric.ROOT_MEAN_SQUARED_ERROR,
    Metric.R_SQUARED,
    Metric.WEIGHTED_EXAMPLE_COUNT,
]

POINT_FORECASTING_METRICS = (
    Metric.ACCURACY,
    Metric.EXAMPLE_COUNT,
    Metric.MEAN_ABSOLUTE_ERROR,
    Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR,
    Metric.MEAN_SQUARED_LOG_ERROR,
    Metric.ROOT_MEAN_SQUARED_ERROR,
    Metric.R_SQUARED,
    Metric.WEIGHTED_EXAMPLE_COUNT,
)

QUANTILE_FORECASTING_METRICS = (Metric.QUANTILE_ACCURACY,)

ENV_SETUP_FILE = os.environ.get(constants.Pipeline.ENV_VAR_SETUP, None)


def _get_metric_specs(
    problem_type: constants.ProblemType,
    class_names: Optional[List[Text]] = None,
    positive_class_names: Optional[List[Text]] = None,
    top_k_list: Optional[List[int]] = None) -> List[config_pb2.MetricsSpec]:
  """Build predfined list of metrics for particular problem types.

  Args:
    problem_type: One of the ProblemType enum.
    class_names: For classification problems, the names of the classes.
    positive_class_names: For classification problems, the positively valued
      classes.
    top_k_list: For classification problems, the list of top-k values.

  Returns:
    A list of MetricSpec values.
  Some combinations of problem_type, class_names, and top_k_list are illegal.
  """
  if problem_type == constants.ProblemType.MULTICLASS:
    if not class_names:
      raise ValueError('Multiclass problem requires a list of class names: {}')
    metric_names = CLASSIFICATION_METRICS
    is_multi_label = False
    class_weights = None
  elif problem_type == constants.ProblemType.MULTILABEL:
    if not class_names or len(class_names) < 2:
      raise ValueError(
          'Multilabel problem requires 2 or more class names: {}'.format(
              class_names))
    metric_names = CLASSIFICATION_METRICS
    is_multi_label = True
    # TODO(b/171963359): Permit user-configured class weights other than flat.
    class_weights = {class_id: 1.0 for class_id in range(0, len(class_names))}
  elif problem_type == constants.ProblemType.REGRESSION:
    if class_names:
      raise ValueError(
          'Regression problem incompatible with class_names: {}'.format(
              class_names))
    if positive_class_names:
      raise ValueError(
          'Regression problem incompatible with positive_class_names: {}'
          .format(positive_class_names))
    if top_k_list:
      raise ValueError(
          'Regression problem incompatible with top_k_list: {}'.format(
              top_k_list))
    metric_names = REGRESSION_METRICS
    is_multi_label = False
    class_weights = None
  else:
    assert False, 'Unknown problem type %r' % problem_type

  if class_names and len(class_names) == 1:
    # Binary classification
    # "True" (index 1) represents positive class.
    positive_class_ids = [constants.Data.BINARY_CLASSIFICATION_LABEL_IDS[1]]
  elif class_names and positive_class_names:
    positive_class_ids = [
        class_names.index(class_name) for class_name in positive_class_names
    ]
  else:
    positive_class_ids = None
  metric_specs = tfma_metrics.get_metric_specs(
      metric_names=metric_names,
      is_multi_label=is_multi_label,
      positive_class_ids=positive_class_ids,
      top_k_list=top_k_list,
      custom_metric_map=None,
      class_weights=class_weights)
  return metric_specs


def _get_tfma_slicing_specs(
    slice_features: List[List[ColumnSpec]]) -> List[config_pb2.SlicingSpec]:
  """Build TFMA Slicing Spec values from feature lists.

  Args:
    slice_features: List of slices, each being a list of feature keys.

  Returns:
    A list of SlicingSpec values, including the overall slice.
  """
  # Overall Slice:
  slicing_specs = [config_pb2.SlicingSpec()]
  # Per specified slice_features:
  for feature_list in slice_features:
    if feature_list:
      slicing_spec = config_pb2.SlicingSpec(
          feature_keys=[spec.as_string() for spec in feature_list])
      slicing_specs.append(slicing_spec)
  return slicing_specs


def get_evaluation_config(
    problem_type: constants.ProblemType,
    evaluation_column_specs: EvaluationColumnSpecs,
    slice_features: List[List[ColumnSpec]],
    class_names: Optional[List[Text]] = None,
    positive_class_names: Optional[List[Text]] = None,
    top_k_list: Optional[List[int]] = None
) -> model_evaluation_pb2.EvaluationConfig:
  """Build a Model Evaluation Configuration.

  Args:
    problem_type: One of the ProblemType enum.
    evaluation_column_specs: column specs necessary for parsing evaluation data.
    slice_features: List of slice specs, each a list of keys to slice. The
      default slice over all values will automatically be added.
    class_names: For classification-type problems, a list of string names for
      classes.
    positive_class_names: For classification-type problems, a list of string
      names for classes to be treated as positively valued.
    top_k_list: For classification-type problems, if specified, a list of top-k
      aggregations.

  Returns:
    An EvaluationConfig.
  """
  tfma_eval_config = config_pb2.EvalConfig()

  tfma_eval_config.model_specs.append(
      config_pb2.ModelSpec(
          prediction_key=evaluation_column_specs.predicted_score_column_spec
          .as_string(),
          prediction_keys=None,
          label_key=evaluation_column_specs.ground_truth_column_spec.as_string(
          ),
          label_keys=None))

  metric_specs = _get_metric_specs(problem_type, class_names,
                                   positive_class_names, top_k_list)
  assert metric_specs, 'At least one metric_spec must be defined %r' % metric_specs
  tfma_eval_config.metrics_specs.extend(metric_specs)

  slicing_specs = _get_tfma_slicing_specs(slice_features)
  assert slicing_specs, 'At least one slicing_spec must be defined %r' % slicing_specs
  tfma_eval_config.slicing_specs.extend(slicing_specs)

  adapter = tfma_adapter.TFMAToME(
      class_name_list=class_names,
      predicted_label_column_spec=evaluation_column_specs
      .predicted_label_column_spec,
      predicted_label_id_column_spec=evaluation_column_specs
      .predicted_label_id_column_spec)
  return adapter.eval_config(tfma_eval_config)


def _get_eval_config_from_service_classification(  # pylint: disable=missing-docstring
    classification: configuration_pb2.ClassificationProblemSpec,
    eval_config: model_evaluation_pb2.EvaluationConfig) -> None:
  if classification.HasField('ground_truth_column_spec'):
    eval_config.data_spec.label_key_spec.CopyFrom(
        classification.ground_truth_column_spec)
  if classification.HasField('example_weight_column_spec'):
    eval_config.data_spec.example_weight_key_spec.CopyFrom(
        classification.example_weight_column_spec)
  if classification.HasField('prediction_score_column_spec'):
    eval_config.data_spec.predicted_score_key_spec.CopyFrom(
        classification.prediction_score_column_spec)
  if classification.HasField('prediction_label_column_spec'):
    eval_config.data_spec.predicted_label_key_spec.CopyFrom(
        classification.prediction_label_column_spec)
  if classification.HasField('prediction_id_column_spec'):
    eval_config.data_spec.predicted_label_id_key_spec.CopyFrom(
        classification.prediction_id_column_spec)

  eval_config.data_spec.labels.extend(classification.class_names)

  num_classes = len(classification.class_names)
  if classification.type == configuration_pb2.ClassificationProblemSpec.MULTICLASS:
    problem_type = constants.ProblemType.MULTICLASS
  elif classification.type == configuration_pb2.ClassificationProblemSpec.MULTILABEL:
    problem_type = constants.ProblemType.MULTILABEL
  else:
    raise NotImplementedError('Classification type %r not implemented' %
                              classification.type)
  # TODO(b/160182662): Include evaluation_options.decision_threshold.
  adapter = tfma_adapter.TFMAToME(
      class_name_list=list(classification.class_names))

  tfma_metric_specs = _get_metric_specs(
      problem_type, list(classification.class_names),
      list(classification.evaluation_options.positive_classes),
      list(classification.evaluation_options.top_k_list))
  for tfma_metric_spec in tfma_metric_specs:
    eval_config.metrics_specs.append(adapter.metrics_spec(tfma_metric_spec))


# TODO(b/165386270): Remove when the old way to specify various evaluation
# dataset components is not longer used.
def _ensure_name_spec(name_spec: configuration_pb2.ColumnSpec,
                      name: str) -> None:
  """Sets name_spec using name, unless already set."""
  if name_spec != configuration_pb2.ColumnSpec():
    # name_spec is already set; nothing to do.
    return

  if not name:
    # name is empty; nothing to do.
    return

  name_spec.name = name


def _get_eval_config_from_service_regression(  # pylint: disable=missing-docstring
    regression: configuration_pb2.RegressionProblemSpec,
    eval_config: model_evaluation_pb2.EvaluationConfig) -> None:

  if regression.HasField('ground_truth_column_spec'):
    eval_config.data_spec.label_key_spec.CopyFrom(
        regression.ground_truth_column_spec)
  if regression.HasField('example_weight_column_spec'):
    eval_config.data_spec.example_weight_key_spec.CopyFrom(
        regression.example_weight_column_spec)
  if regression.HasField('prediction_score_column_spec'):
    eval_config.data_spec.predicted_score_key_spec.CopyFrom(
        regression.prediction_score_column_spec)

  adapter = tfma_adapter.TFMAToME()

  tfma_metric_specs = _get_metric_specs(constants.ProblemType.REGRESSION)
  for tfma_metric_spec in tfma_metric_specs:
    eval_config.metrics_specs.append(adapter.metrics_spec(tfma_metric_spec))


def _get_eval_config_from_service_forecasting(
    forecasting: configuration_pb2.ForecastingProblemSpec,
    eval_config: model_evaluation_pb2.EvaluationConfig) -> None:
  """Convert forecasting problem spec to pipeline evaluation config."""
  if forecasting.HasField('ground_truth_column_spec'):
    eval_config.data_spec.label_key_spec.CopyFrom(
        forecasting.ground_truth_column_spec)
  if forecasting.HasField('example_weight_column_spec'):
    eval_config.data_spec.example_weight_key_spec.CopyFrom(
        forecasting.example_weight_column_spec)
  if forecasting.HasField('prediction_score_column_spec'):
    eval_config.data_spec.predicted_score_key_spec.CopyFrom(
        forecasting.prediction_score_column_spec)

  if forecasting.type == configuration_pb2.ForecastingProblemSpec.POINT:
    adapter = tfma_adapter.TFMAToME()
    tfma_metric_specs = tfma_metrics.get_metric_specs(
        metric_names=list(POINT_FORECASTING_METRICS), is_multi_label=False)
  elif forecasting.type == configuration_pb2.ForecastingProblemSpec.QUANTILE:
    problem_type = constants.ProblemType.QUANTILE_FORECASTING
    adapter = tfma_adapter.TFMAToME()
    eval_config.data_spec.quantiles.extend(forecasting.quantiles)
    custom_metric_map = {
        Metric.QUANTILE_ACCURACY:
            tfma_metrics.MetricSpec(
                class_name='QuantileAccuracy',
                module_name=Metric.CUSTOM + '.quantile_accuracy',
                config={
                    'name': Metric.QUANTILE_ACCURACY,
                    'quantiles': list(forecasting.quantiles)
                },
                aggregatable=False)
    }
    if forecasting.options.enable_point_evaluation:
      eval_config.data_spec.quantile_index = forecasting.options.point_evaluation_quantile_index
      tfma_metric_specs = tfma_metrics.get_metric_specs(
          metric_names=list(POINT_FORECASTING_METRICS),
          model_names=[
              constants.Pipeline.MODEL_KEY + constants.Data.POINT_KEY_SUFFIX
          ],
          is_multi_label=False)
      tfma_metric_specs.extend(
          tfma_metrics.get_metric_specs(
              metric_names=list(QUANTILE_FORECASTING_METRICS),
              model_names=[
                  constants.Pipeline.MODEL_KEY +
                  constants.Data.QUANTILE_KEY_SUFFIX
              ],
              is_multi_label=False,
              custom_metric_map=custom_metric_map))
    else:
      eval_config.data_spec.quantile_index = -1
      tfma_metric_specs = tfma_metrics.get_metric_specs(
          metric_names=list(QUANTILE_FORECASTING_METRICS),
          is_multi_label=False,
          custom_metric_map=custom_metric_map)
  else:
    raise NotImplementedError('Forecasting type %r not implemented' %
                              forecasting.type)

  for tfma_metric_spec in tfma_metric_specs:
    eval_config.metrics_specs.append(adapter.metrics_spec(tfma_metric_spec))


def _is_field_present(msg: message.Message,
                      field_name: str,
                      error_msgs: List[str],
                      error_prefix: str = '') -> bool:
  """Checks whether a message field in a proto is present."""
  if not msg.HasField(field_name):
    error_msgs.append('{}{} field is missing'.format(error_prefix, field_name))
    return False
  return True


def _verify_field_specified(msg: message.Message,
                            field_name: str,
                            error_msgs: List[str],
                            error_prefix: Optional[str] = '') -> None:
  """Verifies that a field in a proto is specified."""
  if not getattr(msg, field_name):
    error_msgs.append('{}{} is not specified'.format(error_prefix, field_name))


def _validate_service_config_data_source(
    data_source: configuration_pb2.DataSource, error_msgs: List[str]) -> None:
  """Validates data_source specification within service configuration."""
  if not _is_field_present(data_source, 'gcs_source', error_msgs):
    return

  gcs_source = data_source.gcs_source
  _verify_field_specified(gcs_source, 'files', error_msgs, 'gcs_source: ')
  _verify_field_specified(gcs_source, 'format', error_msgs, 'gcs_source: ')


def _validate_service_config_problem(problem: configuration_pb2.ProblemSpec,
                                     error_msgs: List[str]) -> None:
  """Validates problem specification within service configuration."""
  problem_kind = problem.WhichOneof('problems')
  if problem_kind is None:
    error_msgs.append('problem type is not specified.')
    return

  if problem_kind == 'classification':
    classification = problem.classification
    if not classification.class_names:
      error_msgs.append(
          'classification: class_names must be specified and non-empty.')

    if _is_field_present(classification, 'ground_truth_column_spec', error_msgs,
                         'classification: '):
      _verify_field_specified(classification.ground_truth_column_spec, 'name',
                              error_msgs,
                              'classification.ground_truth_column_spec: ')

    if _is_field_present(classification, 'prediction_score_column_spec',
                         error_msgs, 'classification: '):
      _verify_field_specified(classification.prediction_score_column_spec,
                              'name', error_msgs,
                              'classification.prediction_score_column_spec: ')
    if classification.type == configuration_pb2.ClassificationProblemSpec.UNKNOWN:
      error_msgs.append('classification: type must be specified.')
    return

  if problem_kind == 'regression':
    regression = problem.regression
    if _is_field_present(regression, 'ground_truth_column_spec', error_msgs,
                         'regression: '):
      _verify_field_specified(regression.ground_truth_column_spec, 'name',
                              error_msgs,
                              'regression.ground_truth_column_spec: ')

    if _is_field_present(regression, 'prediction_score_column_spec', error_msgs,
                         'regression: '):
      _verify_field_specified(regression.prediction_score_column_spec, 'name',
                              error_msgs,
                              'regression.prediction_score_column_spec: ')
    return

  if problem_kind == 'forecasting':
    forecasting = problem.forecasting
    if _is_field_present(forecasting, 'ground_truth_column_spec', error_msgs,
                         'forecasting: '):
      _verify_field_specified(forecasting.ground_truth_column_spec, 'name',
                              error_msgs,
                              'forecasting.ground_truth_column_spec: ')

    if _is_field_present(forecasting, 'prediction_score_column_spec',
                         error_msgs, 'forecasting: '):
      _verify_field_specified(forecasting.prediction_score_column_spec, 'name',
                              error_msgs,
                              'forecasting.prediction_score_column_spec: ')

    if forecasting.type == configuration_pb2.ForecastingProblemSpec.UNKNOWN:
      error_msgs.append('forecasting: type must be specified.')

    if forecasting.type == configuration_pb2.ForecastingProblemSpec.QUANTILE:
      if not forecasting.quantiles:
        error_msgs.append(
            'quantile forecasting: quantiles must be specified and non-empty.')
      if not all((v >= 0 and v <= 1) for v in forecasting.quantiles):
        raise TypeError(
            '{}: all values are expected to be between 0 and 1 but are not.'
            .format(forecasting.quantiles))
      if forecasting.HasField(
          'options') and forecasting.options.enable_point_evaluation:
        index = forecasting.options.point_evaluation_quantile_index
        if index < 0 or index >= len(forecasting.quantiles):
          error_msgs.append('quantile forecasting: invalid quantile index.')


def _validate_service_config_output(output: configuration_pb2.OutputSpec,
                                    error_msgs: List[str]) -> None:
  """Validates output specification within service configuration."""
  if not _is_field_present(output, 'gcs_sink', error_msgs):
    return

  _verify_field_specified(output.gcs_sink, 'path', error_msgs,
                          'output.gcs_sink: ')


def _validate_service_config_execution(
    execution: configuration_pb2.ExecutionSpec, error_msgs: List[str]) -> None:
  """Validates execution specification within service configuration."""
  execution_kind = execution.WhichOneof('spec')
  if execution_kind is None:
    error_msgs.append('execution spec is missing.')
    return

  if execution_kind == 'dataflow_beam':
    dataflow_beam = execution.dataflow_beam
    _verify_field_specified(dataflow_beam, 'dataflow_job_prefix', error_msgs,
                            'dataflow_beam: ')
    _verify_field_specified(dataflow_beam, 'project_id', error_msgs,
                            'dataflow_beam: ')
    _verify_field_specified(dataflow_beam, 'service_account', error_msgs,
                            'dataflow_beam: ')
    _verify_field_specified(dataflow_beam, 'dataflow_staging_dir', error_msgs,
                            'dataflow_beam: ')
    _verify_field_specified(dataflow_beam, 'dataflow_temp_dir', error_msgs,
                            'dataflow_beam: ')
    _verify_field_specified(dataflow_beam, 'region', error_msgs,
                            'dataflow_beam: ')


def _validate_service_config(
    mes_config: configuration_pb2.EvaluationRunConfig) -> None:
  """Validate model evaluation service config.

  Args:
    mes_config: A Model Evaluation Service configuration.

  Raises:
    ValueError if mes_config is invalid.
  """
  error_msgs = []
  if not mes_config.name:
    error_msgs.append('name is not specified.')

  if _is_field_present(mes_config, 'data_source', error_msgs):
    _validate_service_config_data_source(mes_config.data_source, error_msgs)

  if _is_field_present(mes_config, 'problem', error_msgs):
    _validate_service_config_problem(mes_config.problem, error_msgs)

  if _is_field_present(mes_config, 'output', error_msgs):
    _validate_service_config_output(mes_config.output, error_msgs)

  if _is_field_present(mes_config, 'execution', error_msgs):
    _validate_service_config_execution(mes_config.execution, error_msgs)

  if error_msgs:
    error_msgs_str = '\n  *'.join(error_msgs)
    raise ValueError('Model Evaluation Service configuration for backend ' +
                     f'{mes_config} has following errors: {error_msgs_str}')


def get_evaluation_config_from_service_config(
    mes_config: configuration_pb2.EvaluationRunConfig
) -> model_evaluation_pb2.EvaluationConfig:
  """Convert Model Evaluation Service into CAIMQ Evaluation Config."""
  if not mes_config:
    return None

  _validate_service_config(mes_config)

  eval_config = model_evaluation_pb2.EvaluationConfig()
  eval_config.name = mes_config.name

  gcs_source = mes_config.data_source.gcs_source
  if gcs_source.format != configuration_pb2.GcsSource.JSONL:
    raise ValueError('Unsupported format: {}'.format(gcs_source.format))
  eval_config.data_spec.input_source_spec.jsonl_file_spec.file_names.extend(
      gcs_source.files)
  eval_config.output_spec.gcs_sink.path = mes_config.output.gcs_sink.path

  if mes_config.problem.HasField('classification'):
    _get_eval_config_from_service_classification(
        mes_config.problem.classification, eval_config)
  elif mes_config.problem.HasField('regression'):
    _get_eval_config_from_service_regression(mes_config.problem.regression,
                                             eval_config)
  elif mes_config.problem.HasField('forecasting'):
    _get_eval_config_from_service_forecasting(mes_config.problem.forecasting,
                                              eval_config)
  else:
    raise NotImplementedError('Problem %r not implemented' % mes_config.problem)

  eval_config.execution_spec.CopyFrom(mes_config.execution)

  return eval_config


def _get_dataflow_job_name(dataflow_job_prefix, eval_run_name):
  return '{}-{}'.format(dataflow_job_prefix, eval_run_name)


def get_pipeline_options_from_service_config(
    mes_config: configuration_pb2.EvaluationRunConfig
) -> pipeline_opts_lib.PipelineOptions:
  """Specifies Beam pipeline options based on model eval service config."""
  _validate_service_config(mes_config)

  pipeline_options = pipeline_opts_lib.PipelineOptions()
  if mes_config.execution.HasField('dataflow_beam'):
    dataflow_beam_spec = mes_config.execution.dataflow_beam
    google_cloud_options = pipeline_options.view_as(
        pipeline_opts_lib.GoogleCloudOptions)
    google_cloud_options.project = dataflow_beam_spec.project_id
    google_cloud_options.service_account_email = dataflow_beam_spec.service_account
    google_cloud_options.region = dataflow_beam_spec.region
    google_cloud_options.job_name = _get_dataflow_job_name(
        dataflow_beam_spec.dataflow_job_prefix, mes_config.name)
    google_cloud_options.staging_location = dataflow_beam_spec.dataflow_staging_dir
    google_cloud_options.temp_location = dataflow_beam_spec.dataflow_temp_dir

    worker_options = pipeline_options.view_as(pipeline_opts_lib.WorkerOptions)
    if dataflow_beam_spec.num_workers:
      worker_options.num_workers = dataflow_beam_spec.num_workers
    if dataflow_beam_spec.max_num_workers:
      worker_options.max_num_workers = dataflow_beam_spec.max_num_workers
    if dataflow_beam_spec.machine_type:
      worker_options.machine_type = dataflow_beam_spec.machine_type
    pipeline_options.view_as(
        pipeline_opts_lib.SetupOptions
    ).setup_file = dataflow_beam_spec.setup_file or ENV_SETUP_FILE
    pipeline_options.view_as(pipeline_opts_lib.StandardOptions
                            ).runner = constants.Pipeline.DATAFLOW_RUNNER
  else:
    local_beam_spec = mes_config.execution.local_beam
    pipeline_options.view_as(pipeline_opts_lib.DirectOptions
                            ).direct_num_workers = local_beam_spec.num_workers
  return pipeline_options


def get_problem_type_from_service_config(
    mes_config: configuration_pb2.EvaluationRunConfig) -> constants.ProblemType:
  """Gets evaluation problem type from model evaluation service config.

  Args:
    mes_config: Model evaluation service config.

  Returns:
    Type of the model evaluation problem.

  Raises:
    ValueError if the specified evaluation problem type is invalid.
  """
  if mes_config.problem.HasField('regression'):
    return constants.ProblemType.REGRESSION

  if mes_config.problem.HasField('classification'):
    classification = mes_config.problem.classification
    if classification.type == configuration_pb2.ClassificationProblemSpec.MULTICLASS:
      return constants.ProblemType.MULTICLASS
    if classification.type == configuration_pb2.ClassificationProblemSpec.MULTILABEL:
      return constants.ProblemType.MULTILABEL

    raise ValueError('Unsupported classification problem type: {}'.format(
        classification.type))

  if mes_config.problem.HasField('forecasting'):
    forecasting = mes_config.problem.forecasting
    if forecasting.type == configuration_pb2.ForecastingProblemSpec.POINT:
      return constants.ProblemType.POINT_FORECASTING
    if forecasting.type == configuration_pb2.ForecastingProblemSpec.QUANTILE:
      return constants.ProblemType.QUANTILE_FORECASTING

    raise ValueError('Unsupported forecasting problem type: {}'.format(
        forecasting.type))

  raise ValueError('Evaluation problem type either unspecified or unsupported '
                   'in the configuration \n {}'.format(mes_config))

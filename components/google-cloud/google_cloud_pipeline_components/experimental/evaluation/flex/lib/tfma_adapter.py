"""Conversion adapters for metrics library.

This includes conversion between Tensorflow Model Analysis (TFMA) and
Cloud AI Platform Model Evaluation Metrics (ME) formats.
"""

import math
from typing import List, Optional, Tuple

from tensorflow_model_analysis.proto import config_pb2
from tensorflow_model_analysis.proto import metrics_for_slice_pb2

from lib import column_spec, constants
from lib.proto import model_evaluation_pb2 as me_proto

ColumnSpec = column_spec.ColumnSpec

# Convenience alias.
Metric = constants.Metric

# To find near-miss floating point values.
DEFAULT_TOL = 1e-5


def _is_binary_classification(class_list: List[str]) -> bool:
  """Returns true for binary classification problems."""
  if not class_list:
    return False

  return len(class_list) == 1


def f1(p: float, r: float) -> float:
  """F1 metric computed from precision and recall."""
  return 2 * p * r / (p + r) if p + r else 0.0


def fpr(fp: int, tn: int) -> float:
  """False Positive Rate computed from false positives and true negatives."""
  return fp / (fp + tn) if fp + tn else 0.0


class TFMAToME(object):
  """Adapter converting TFMA configurations to ME."""

  def __init__(self,
               class_name_list: Optional[List[str]] = None,
               predicted_label_column_spec: Optional[ColumnSpec] = None,
               predicted_label_id_column_spec: Optional[ColumnSpec] = None):
    """Create an adapter.

    Args:
      class_name_list: Optional class name list, for classification problems.
        The order of the items is used as an index here and elsewhere.
      predicted_label_column_spec: The spec for a column containing labels for
        classes scored by a model, if model returns them.
      predicted_label_id_column_spec: The spec for a column containing label ids
        for classes scored by a model, if model returns them.
    """
    self._class_name_list = class_name_list
    self._predicted_label_column_spec = predicted_label_column_spec
    self._predicted_label_id_column_spec = predicted_label_id_column_spec

  def get_class_names(self, class_id_list: List[int]) -> List[str]:
    """Return class names as strings.

    Args:
      class_id_list: Input class ids.

    Returns:
      List of class names as strings.
    """
    if _is_binary_classification(self._class_name_list):
      # In binary classification case positive class is fixed.
      return [constants.Data.BINARY_CLASSIFICATION_LABELS[1]]

    return [self._class_name_list[class_id] for class_id in class_id_list]

  def eval_config(
      self, tfma_config: config_pb2.EvalConfig) -> me_proto.EvaluationConfig:
    """Convert TFMA EvalConfig into ME EvaluationConfig.

    Args:
      tfma_config: Input TFMA EvalConfig.

    Returns:
      ME EvaluationConfig.
    """
    if not tfma_config:
      return None
    if len(tfma_config.model_specs) != 1:
      raise ValueError('Exactly one model_spec must be defined.')

    me_config = me_proto.EvaluationConfig()
    me_config.data_spec.CopyFrom(
        self._model_spec_to_data_spec(tfma_config.model_specs[0]))
    for slicing_spec in tfma_config.slicing_specs:
      me_config.slicing_specs.append(self._slicing_spec(slicing_spec))
    for metrics_spec in tfma_config.metrics_specs:
      me_config.metrics_specs.append(self.metrics_spec(metrics_spec))
    # Options ignored.
    return me_config

  def _model_spec_to_data_spec(
      self, tfma_model_spec: config_pb2.ModelSpec) -> me_proto.DataSpec:
    """Convert TFMA ModelSpec into ME DataSpec.

    Args:
      tfma_model_spec: Input TFMA ModelSpec.

    Returns:
      ME DataSPec.
    """
    if not tfma_model_spec:
      return None
    me_data_spec = me_proto.DataSpec()
    me_data_spec.label_key_spec.CopyFrom(
        ColumnSpec(
            tfma_model_spec.label_key.replace(
                constants.Data.K_HOT_KABEL_KEY_SUFFIX, '')).as_proto())
    if tfma_model_spec.example_weight_key:
      me_data_spec.example_weight_key_spec.CopyFrom(
          ColumnSpec(tfma_model_spec.example_weight_key).as_proto())
    if tfma_model_spec.prediction_key:
      me_data_spec.predicted_score_key_spec.CopyFrom(
          ColumnSpec(tfma_model_spec.prediction_key).as_proto())

    if self._predicted_label_column_spec:
      me_data_spec.predicted_label_key_spec.CopyFrom(
          self._predicted_label_column_spec.as_proto())
    if self._predicted_label_id_column_spec:
      me_data_spec.predicted_label_id_key_spec.CopyFrom(
          self._predicted_label_id_column_spec.as_proto())

    if self._class_name_list:
      me_data_spec.labels.extend(self._class_name_list)
    return me_data_spec

  def _slicing_spec(
      self, tfma_slicing_spec: config_pb2.SlicingSpec) -> me_proto.SlicingSpec:
    """Convert TFMA SlicingSPec into ME SlicingSpec.

    Args:
      tfma_slicing_spec: Input TFMA SlicingSpec.

    Returns:
      ME SlicingSpec.
    """
    if not tfma_slicing_spec:
      return None
    me_slicing_spec = me_proto.SlicingSpec()
    for tfma_feature_key in tfma_slicing_spec.feature_keys:
      me_slicing_spec.feature_key_specs.append(
          ColumnSpec(tfma_feature_key).as_proto())
    for name in tfma_slicing_spec.feature_values:
      me_feature_value = me_proto.SlicingSpec.FeatureValueSpec(
          name_spec=ColumnSpec([name]).as_proto(),
          value=tfma_slicing_spec.feature_values[name])

      me_slicing_spec.feature_values.append(me_feature_value)
    return me_slicing_spec

  def metrics_spec(
      self, tfma_metrics_spec: config_pb2.MetricsSpec) -> me_proto.MetricsSpec:
    """Convert TFMA MetricsSpec into ME MetricsSpec.

    Args:
      tfma_metrics_spec: Input TFMA MetricsSpec.

    Returns:
      ME MetricsSpec.
    """
    if not tfma_metrics_spec:
      return None
    me_metrics_spec = me_proto.MetricsSpec()
    for metric_config in tfma_metrics_spec.metrics:
      me_metrics_spec.metrics.append(self._metric_config(metric_config))
    if tfma_metrics_spec.HasField('binarize'):
      me_metrics_spec.binarize.CopyFrom(
          self._binarization_options(tfma_metrics_spec.binarize))
    if tfma_metrics_spec.HasField('aggregate'):
      me_metrics_spec.aggregate.CopyFrom(
          self._aggregation_options(tfma_metrics_spec.aggregate))
    me_metrics_spec.model_names.extend(tfma_metrics_spec.model_names)
    return me_metrics_spec

  def _metric_config(
      self,
      tfma_metric_config: config_pb2.MetricConfig) -> me_proto.MetricConfig:
    """Convert TFMA MetricConfig into ME MetricConfig.

    Args:
      tfma_metric_config: Input TFMA MetricConfig.

    Returns:
      ME MetricConfig.
    """
    if not tfma_metric_config:
      return None
    me_metric_config = me_proto.MetricConfig()
    class_name = tfma_metric_config.class_name
    if not class_name:
      raise ValueError('The class_name must be defined for tfma_metric_config')
    me_metric_config.name = class_name
    me_metric_config.tfma_metric_config.class_name = class_name
    if tfma_metric_config.module:
      me_metric_config.tfma_metric_config.module_name = tfma_metric_config.module
    if tfma_metric_config.config:
      me_metric_config.tfma_metric_config.config = tfma_metric_config.config
    return me_metric_config

  def _binarization_options(
      self, tfma_binarization: config_pb2.BinarizationOptions
  ) -> Optional[me_proto.Binarization]:
    """Convert TFMA BinarizationOptions into ME Binarization.

    Args:
      tfma_binarization: Input TFMA BinarizationOptions.

    Returns:
      ME Binarization.
    """
    if not tfma_binarization:
      return None
    me_binarization = me_proto.Binarization()
    if tfma_binarization.class_ids.values:
      me_binarization.class_ids.extend(
          self.get_class_names(list(tfma_binarization.class_ids.values)))
    if tfma_binarization.top_k_list.values:
      me_binarization.top_k_list.extend(tfma_binarization.top_k_list.values)
    return me_binarization

  def _aggregation_options(
      self, tfma_aggregation: config_pb2.AggregationOptions
  ) -> Optional[me_proto.Aggregation]:
    """Convert TFMA AggregationOptions into ME Aggregation.

    Args:
      tfma_aggregation: Input TFMA AggregationOptions.

    Returns:
      ME Aggregation.
    """
    if not tfma_aggregation:
      return None

    me_aggregation = me_proto.Aggregation()
    if tfma_aggregation.micro_average:
      me_aggregation.micro_average = True
    if tfma_aggregation.macro_average:
      me_aggregation.macro_average = True
    if tfma_aggregation.class_weights:
      me_aggregation.class_weights.update(tfma_aggregation.class_weights)
    return me_aggregation

  @classmethod
  def _get_display_slice_key(cls,
                             slice_key: metrics_for_slice_pb2.SliceKey) -> str:
    keys = []
    for slice_key_1 in slice_key.single_slice_keys:
      keys.append('%s:%r' %
                  (slice_key_1.column, slice_key_1.float_value or
                   slice_key_1.int64_value or slice_key_1.bytes_value))
    return ','.join(keys)

  @classmethod
  def single_output_slicing_spec(
      cls, tfma_slice_key: metrics_for_slice_pb2.SliceKey
  ) -> me_proto.OutputSlicingSpec:
    """Convert TFMA SliceKey into an OutputSlicingSpec."""
    out_spec = me_proto.OutputSlicingSpec()
    if not tfma_slice_key.single_slice_keys:
      return out_spec
    if len(tfma_slice_key.single_slice_keys) > 1:
      raise ValueError('Too many slice keys in %r' %
                       tfma_slice_key.single_slice_keys)
    tfma_single_slice_key = tfma_slice_key.single_slice_keys[0]
    out_spec.feature_name_spec.CopyFrom(
        ColumnSpec(tfma_single_slice_key.column).as_proto())
    if tfma_single_slice_key.WhichOneof('kind') == 'bytes_value':
      out_spec.bytes_value = tfma_single_slice_key.bytes_value
    elif tfma_single_slice_key.WhichOneof('kind') == 'float_value':
      out_spec.float_value = tfma_single_slice_key.float_value
    elif tfma_single_slice_key.WhichOneof('kind') == 'int64_value':
      out_spec.int64_value = tfma_single_slice_key.int64_value
    return out_spec

  @classmethod
  def _sparse_to_dense_matrix(cls, entries: List[Tuple[int, int, float]],
                              rank: int) -> List[List[int]]:
    """Convert sparse list of triples into a 2d array."""
    dense = [[0] * rank for _ in range(0, rank)]
    for (i, j, c) in entries:
      # TODO(b/173103299): Do not discard deletions.
      if j >= 0:
        dense[i][j] = int(math.ceil(c))

    return dense

  @classmethod
  def _tfma_multi_class_confusion_matrix_to_caimq_confidence_metrics(
      cls,
      tfma_cm_set: metrics_for_slice_pb2.MultiClassConfusionMatrixAtThresholds,
      class_labels: List[str]) -> Optional[me_proto.ConfusionMatrix]:
    """Convert the sparse representation in TFMA to the dense one in CAIMQ."""
    for tfma_cm in tfma_cm_set.matrices:
      if math.isclose(
          tfma_cm.threshold,
          constants.Thresholds.DEFAULT_DECISION_THRESHOLD,
          abs_tol=DEFAULT_TOL):
        caimq_confusion = me_proto.ConfusionMatrix()
        sparse_entries = [(entry.actual_class_id, entry.predicted_class_id,
                           entry.num_weighted_examples)
                          for entry in tfma_cm.entries]
        dense_matrix = TFMAToME._sparse_to_dense_matrix(sparse_entries,
                                                        len(class_labels))
        for (idx, row) in enumerate(dense_matrix):
          caimq_confusion.annotation_specs.append(
              me_proto.ConfusionMatrix.AnnotationSpecRef(
                  id=str(idx), display_name=class_labels[idx]))
          caimq_confusion.rows.append(
              me_proto.ConfusionMatrix.Row(data_item_counts=row))
        return caimq_confusion
    return None

  @classmethod
  def _tfma_confusion_matrices_to_caimq_confidence_metrics(
      cls, tfma_cm_set: metrics_for_slice_pb2.ConfusionMatrixAtThresholds,
      tfma_cm_set_top_1: metrics_for_slice_pb2.ConfusionMatrixAtThresholds
  ) -> List[me_proto.ClassificationEvaluationMetrics.ConfidenceMetrics]:
    """Build a ConfidenceMetrics out of ConfusionMatrices."""

    if not tfma_cm_set:
      raise ValueError('Overall confusion matrix must have values.')

    tfma_cm_map = {matrix.threshold: matrix for matrix in tfma_cm_set.matrices}
    tfma_cm_map_top_1 = {
        matrix.threshold: matrix for matrix in tfma_cm_set_top_1.matrices
    }

    tfma_cm_thresholds = sorted(tfma_cm_map.keys())
    tfma_cm_top_1_thresholds = sorted(tfma_cm_map_top_1.keys())
    if tfma_cm_thresholds and tfma_cm_top_1_thresholds and tfma_cm_thresholds != tfma_cm_top_1_thresholds:
      raise ValueError(
          'Keys in overall and top 1 confusion matrices mismatch:{}, {}'.format(
              tfma_cm_thresholds, tfma_cm_top_1_thresholds))

    confidence_metrics = []
    for threshold in tfma_cm_thresholds:
      tfma_cm = tfma_cm_map[threshold]
      tfma_cm_top_1 = tfma_cm_map_top_1.get(threshold, None)
      caimq_cm = me_proto.ClassificationEvaluationMetrics.ConfidenceMetrics(
      )
      caimq_cm.confidence_threshold = threshold
      caimq_cm.false_negative_count = math.ceil(tfma_cm.false_negatives)
      caimq_cm.true_negative_count = math.ceil(tfma_cm.true_negatives)
      caimq_cm.false_positive_count = math.ceil(tfma_cm.false_positives)
      caimq_cm.true_positive_count = math.ceil(tfma_cm.true_positives)
      caimq_cm.false_positive_rate = fpr(caimq_cm.false_positive_count,
                                         caimq_cm.true_negative_count)
      caimq_cm.precision = tfma_cm.precision
      caimq_cm.recall = tfma_cm.recall
      caimq_cm.f1_score = f1(caimq_cm.precision, caimq_cm.recall)

      if tfma_cm_top_1:
        false_positive_count = math.ceil(tfma_cm_top_1.false_positives)
        true_negative_count = math.ceil(tfma_cm_top_1.true_negatives)
        caimq_cm.precision_at1 = tfma_cm_top_1.precision
        caimq_cm.recall_at1 = tfma_cm_top_1.recall
        caimq_cm.false_positive_rate_at1 = fpr(false_positive_count,
                                               true_negative_count)
        caimq_cm.f1_score_at1 = f1(caimq_cm.precision_at1, caimq_cm.recall_at1)

      confidence_metrics.append(caimq_cm)

    return confidence_metrics

  @classmethod
  def model_evaluation_metrics(
      cls,
      tfma_metrics: metrics_for_slice_pb2.MetricsForSlice,
      problem_type: constants.ProblemType,
      class_labels: Optional[List[str]] = None) -> me_proto.SlicedMetricsSet:
    """Convert TFMA metrics into Model Evaluation metrics.

    Args:
      tfma_metrics: The metrics to convert, classification or regression.
      problem_type: Which type of problem to consider.
      class_labels: Optional list of classification labels in index order.

    Returns:
      Model Evaluation metric which has a oneof for classification and
      regression depending on problem_type.
    """
    sliced_metrics_set = me_proto.SlicedMetricsSet()

    if problem_type == constants.ProblemType.MULTICLASS or problem_type == constants.ProblemType.MULTILABEL:
      # Group relevant metrics from tfma metrics based on sub_key.class_id
      # into tfma_metrics_dict with key be either -1 (overall
      # metrics) or sub_key.class_id (binary classification metrics per class).
      tfma_metrics_dict = {}
      for metric_key_and_value in list(tfma_metrics.metric_keys_and_values):
        metric_key = metric_key_and_value.key
        metric_value = metric_key_and_value.value
        if metric_key.HasField('sub_key') and metric_key.sub_key.HasField(
            'class_id'):
          if (metric_key.sub_key.class_id.value) not in tfma_metrics_dict:
            tfma_metrics_dict[metric_key.sub_key.class_id.value] = []
          tfma_metrics_dict[metric_key.sub_key.class_id.value].append(
              metric_key_and_value)
        else:
          if -1 not in tfma_metrics_dict:
            tfma_metrics_dict[-1] = []
          tfma_metrics_dict[-1].append(metric_key_and_value)

      for key, tfma_metrics_list in tfma_metrics_dict.items():
        sliced_metrics = me_proto.SlicedMetrics()
        sliced_metrics.single_output_slicing_spec.CopyFrom(
            TFMAToME.single_output_slicing_spec(tfma_metrics.slice_key))
        # For per-class binary classification metrics, update the slicing specs
        # to reflect the class label if class_labels is provided or class id.
        if key != -1:
          sliced_metrics.single_output_slicing_spec.bytes_value = class_labels[
              key].encode() if class_labels else str(key).encode()
        metrics = me_proto.ModelEvaluationMetrics()
        class_metrics = me_proto.ClassificationEvaluationMetrics()
        cm_set = metrics_for_slice_pb2.ConfusionMatrixAtThresholds()
        cm_set_top_1 = metrics_for_slice_pb2.ConfusionMatrixAtThresholds()

        for metric_key_and_value in tfma_metrics_list:
          metric_key = metric_key_and_value.key
          metric_value = metric_key_and_value.value

          if metric_key.name == Metric.AUC:
            class_metrics.au_roc = metric_value.double_value.value
          elif metric_key.name == Metric.AUC_PRECISION_RECALL:
            class_metrics.au_prc = metric_value.double_value.value
          elif metric_key.name == Metric.CATEGORICAL_CROSSENTROPY:
            class_metrics.log_loss = metric_value.double_value.value
          elif metric_key.name == Metric.BINARY_CROSSENTROPY:
            if key != -1:
              class_metrics.log_loss = metric_value.double_value.value
          elif metric_key.name == Metric.CONFUSION_MATRIX:
            if metric_key.sub_key.top_k.value == 1:
              cm_set_top_1.CopyFrom(metric_value.confusion_matrix_at_thresholds)
            else:
              cm_set.CopyFrom(metric_value.confusion_matrix_at_thresholds)
          elif metric_key.name == Metric.MULTI_CLASS_CONFUSION_MATRIX:
            class_metrics.confusion_matrix.CopyFrom(
                TFMAToME
                ._tfma_multi_class_confusion_matrix_to_caimq_confidence_metrics(
                    metric_value.multi_class_confusion_matrix_at_thresholds,
                    class_labels))
        if cm_set or cm_set_top_1:
          class_metrics.confidence_metrics.extend(
              TFMAToME._tfma_confusion_matrices_to_caimq_confidence_metrics(
                  cm_set, cm_set_top_1))
        metrics.classification.CopyFrom(class_metrics)
        sliced_metrics.metrics.CopyFrom(metrics)
        sliced_metrics_set.sliced_metrics.append(sliced_metrics)

    elif problem_type == constants.ProblemType.REGRESSION:
      sliced_metrics = me_proto.SlicedMetrics()
      sliced_metrics.single_output_slicing_spec.CopyFrom(
          TFMAToME.single_output_slicing_spec(tfma_metrics.slice_key))
      metrics = me_proto.ModelEvaluationMetrics()
      reg_metrics = me_proto.RegressionEvaluationMetrics()
      for metric_key_and_value in list(tfma_metrics.metric_keys_and_values):
        metric_key = metric_key_and_value.key
        metric_value = metric_key_and_value.value

        if metric_key.name == Metric.MEAN_ABSOLUTE_ERROR:
          reg_metrics.mean_absolute_error = metric_value.double_value.value
        elif metric_key.name == Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR:
          reg_metrics.mean_absolute_percentage_error = metric_value.double_value.value
        elif metric_key.name == Metric.R_SQUARED:
          reg_metrics.r_squared = metric_value.double_value.value
        elif metric_key.name == Metric.MEAN_SQUARED_LOG_ERROR:
          reg_metrics.root_mean_squared_log_error = math.sqrt(
              metric_value.double_value.value)
        elif metric_key.name == Metric.ROOT_MEAN_SQUARED_ERROR:
          reg_metrics.root_mean_squared_error = metric_value.double_value.value
      metrics.regression.CopyFrom(reg_metrics)
      sliced_metrics.metrics.CopyFrom(metrics)
      sliced_metrics_set.sliced_metrics.append(sliced_metrics)

    elif problem_type in [
        constants.ProblemType.QUANTILE_FORECASTING,
        constants.ProblemType.POINT_FORECASTING
    ]:
      sliced_metrics = me_proto.SlicedMetrics()
      sliced_metrics.single_output_slicing_spec.CopyFrom(
          TFMAToME.single_output_slicing_spec(tfma_metrics.slice_key))
      metrics = me_proto.ModelEvaluationMetrics()
      forecast_metrics = me_proto.ForecastingEvaluationMetrics()
      for metric_key_and_value in tfma_metrics.metric_keys_and_values:
        metric_key = metric_key_and_value.key
        metric_value = metric_key_and_value.value

        if metric_key.name == Metric.MEAN_ABSOLUTE_ERROR:
          forecast_metrics.mean_absolute_error = metric_value.double_value.value
        elif metric_key.name == Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR:
          forecast_metrics.mean_absolute_percentage_error = metric_value.double_value.value
        elif metric_key.name == Metric.R_SQUARED:
          forecast_metrics.r_squared = metric_value.double_value.value
        elif metric_key.name == Metric.MEAN_SQUARED_LOG_ERROR:
          forecast_metrics.root_mean_squared_log_error = math.sqrt(
              metric_value.double_value.value)
        elif metric_key.name == Metric.ROOT_MEAN_SQUARED_ERROR:
          forecast_metrics.root_mean_squared_error = metric_value.double_value.value
        elif metric_key.name == Metric.QUANTILE_ACCURACY:
          quantile_forecast_metrics = me_proto.ForecastingEvaluationMetrics(
          )
          quantile_forecast_metrics.ParseFromString(metric_value.bytes_value)
          forecast_metrics.MergeFrom(quantile_forecast_metrics)
      metrics.forecasting.CopyFrom(forecast_metrics)
      sliced_metrics.metrics.CopyFrom(metrics)
      sliced_metrics_set.sliced_metrics.append(sliced_metrics)

    else:
      raise NotImplementedError('Problem Type %r is not implemented.' %
                                problem_type)
    return sliced_metrics_set


class METoTFMA(object):
  """Adapter converting ME configurations to TFMA."""

  # TODO(b/165386270): Get rid of class_name_list argument. This field is
  # already present in the EvaluationConfig that we are converting from.
  def __init__(self, class_name_list: Optional[List[str]] = None):
    """Create an adapter.

    Args:
      class_name_list: Optional class name list, for classification problems.
        The order of the items is used as an index here and elsewhere.
    """
    self._class_name_list = class_name_list

  def get_class_ids(self, class_names: List[str]):
    """Return class names as ids.

    Args:
      class_names: Input class names.

    Returns:
      List of class ids.
    """
    if not self._class_name_list:
      raise ValueError('Cannot index class names without a class_name_list.')

    if _is_binary_classification(self._class_name_list):
      # In binary classification case positive class is fixed.
      return [constants.Data.BINARY_CLASSIFICATION_LABEL_IDS[1]]

    return [
        self._class_name_list.index(class_name) for class_name in class_names
    ]

  def eval_config(
      self, me_config: me_proto.EvaluationConfig) -> config_pb2.EvalConfig:
    """Convert ME EvaluationConfig into TFMA.

    Args:
      me_config: Input ME EvaluationConfig.

    Returns:
      TFMA EvalConfig.
    """
    if not me_config:
      return None
    tfma_config = config_pb2.EvalConfig()
    for model_spec in self._data_spec_to_model_specs(me_config.data_spec):
      tfma_config.model_specs.append(model_spec)
    for slicing_spec in me_config.slicing_specs:
      tfma_config.slicing_specs.append(self._slicing_spec(slicing_spec))
    for metrics_spec in me_config.metrics_specs:
      tfma_config.metrics_specs.append(self.metrics_spec(metrics_spec))
    # Options ignored.
    return tfma_config

  @classmethod
  def _is_quantile_forecasting_with_point(
      cls, me_data_spec: me_proto.DataSpec) -> bool:
    """Check if it's the quantile forecasting problem with point value."""
    if me_data_spec.quantiles:
      return me_data_spec.quantile_index >= 0
    return False

  def _data_spec_to_model_specs(
      self, me_data_spec: me_proto.DataSpec) -> List[config_pb2.ModelSpec]:
    """Convert ME DataSpec into a list of TFMA ModelSpecs.

    Args:
      me_data_spec: Input ME DataSpec.

    Returns:
      A list of TFMA ModelSpec.
    """
    if self._is_quantile_forecasting_with_point(me_data_spec):
      return [
          self._data_spec_to_model_spec(me_data_spec,
                                        (constants.Pipeline.MODEL_KEY +
                                         constants.Data.QUANTILE_KEY_SUFFIX)),
          self._data_spec_to_model_spec(
              me_data_spec,
              (constants.Pipeline.MODEL_KEY + constants.Data.POINT_KEY_SUFFIX))
      ]
    else:
      return [self._data_spec_to_model_spec(me_data_spec)]

  def _data_spec_to_model_spec(
      self,
      me_data_spec: me_proto.DataSpec,
      model_name: Optional[str] = None) -> config_pb2.ModelSpec:
    """Convert ME DataSpec into TFMA ModelSpec.

    Args:
      me_data_spec: Input ME DataSpec.
      model_name: Input model name.

    Returns:
      TFMA ModelSpec.
    """
    if not me_data_spec:
      return None
    tfma_model_spec = config_pb2.ModelSpec()
    if model_name:
      tfma_model_spec.name = model_name
    if me_data_spec.HasField('label_key_spec'):
      tfma_model_spec.label_key = ColumnSpec(
          me_data_spec.label_key_spec).as_string() + (
              constants.Data.K_HOT_KABEL_KEY_SUFFIX
              if self._class_name_list else '')
    if me_data_spec.HasField('predicted_score_key_spec'):
      tfma_model_spec.prediction_key = ColumnSpec(
          me_data_spec.predicted_score_key_spec).as_string() + (
              constants.Data.POINT_KEY_SUFFIX if model_name and
              model_name.endswith(constants.Data.POINT_KEY_SUFFIX) else '')
    if me_data_spec.HasField('example_weight_key_spec'):
      tfma_model_spec.example_weight_key = ColumnSpec(
          me_data_spec.example_weight_key_spec).as_string()
    return tfma_model_spec

  def _slicing_spec(
      self, me_slicing_spec: me_proto.SlicingSpec) -> config_pb2.SlicingSpec:
    """Convert ME SlicingSpec into TFMA.

    Args:
      me_slicing_spec: Input ME SlicingSpec.

    Returns:
      TFMA SlicingSpec.
    """
    if not me_slicing_spec:
      return None
    tfma_slicing_spec = config_pb2.SlicingSpec()
    if me_slicing_spec.feature_key_specs:
      tfma_slicing_spec.feature_keys.extend([
          ColumnSpec(spec).as_string()
          for spec in me_slicing_spec.feature_key_specs
      ])
    for feature_value in me_slicing_spec.feature_values:
      tfma_slicing_spec.feature_values.update({
          ColumnSpec(feature_value.name_spec).as_string(): feature_value.value
      })
    return tfma_slicing_spec

  def metrics_spec(
      self, me_metrics_spec: me_proto.MetricsSpec) -> config_pb2.MetricsSpec:
    """Convert ME MetricsSpec into TFMA.

    Args:
      me_metrics_spec: Input ME MetricsSpec.

    Returns:
      TFMA MetricsSpec.
    """
    if not me_metrics_spec:
      return None
    tfma_metrics_spec = config_pb2.MetricsSpec()
    for metric_config in me_metrics_spec.metrics:
      tfma_metrics_spec.metrics.append(self._metric_config(metric_config))
    if me_metrics_spec.HasField('binarize'):
      tfma_metrics_spec.binarize.CopyFrom(
          self._binarization_options(me_metrics_spec.binarize))
    if me_metrics_spec.HasField('aggregate'):
      tfma_metrics_spec.aggregate.CopyFrom(
          self._aggregation_options(me_metrics_spec.aggregate))
    tfma_metrics_spec.model_names.extend(me_metrics_spec.model_names)
    return tfma_metrics_spec

  def _metric_config(
      self, me_metric_config: me_proto.MetricConfig) -> config_pb2.MetricConfig:
    """Convert ME MetricConfig into TFMA.

    Args:
      me_metric_config: Input ME MetricConfig.

    Returns:
      TFMA MetricConfig.
    """
    if not me_metric_config:
      return None
    tfma_metric_config = config_pb2.MetricConfig()
    if me_metric_config.tfma_metric_config.class_name:
      tfma_metric_config.class_name = me_metric_config.tfma_metric_config.class_name
    if me_metric_config.tfma_metric_config.module_name:
      tfma_metric_config.module = me_metric_config.tfma_metric_config.module_name
    if me_metric_config.tfma_metric_config.config:
      tfma_metric_config.config = me_metric_config.tfma_metric_config.config
    # TODO(b/159642889): Error out if messages cannot be converted.
    return tfma_metric_config

  def _binarization_options(
      self,
      me_binarization: me_proto.Binarization) -> config_pb2.BinarizationOptions:
    """Convert ME Binarization into TFMA BinarizationOptions.

    Args:
      me_binarization: Input ME Binarization.

    Returns:
      TFMA BinarizationOptions.
    """
    if not me_binarization:
      return None
    tfma_binarization = config_pb2.BinarizationOptions()
    if me_binarization.class_ids:
      tfma_binarization.class_ids.values.extend(
          self.get_class_ids(list(me_binarization.class_ids)))
    if me_binarization.top_k_list:
      tfma_binarization.top_k_list.values.extend(me_binarization.top_k_list)
    return tfma_binarization

  def _aggregation_options(
      self, me_aggregation: me_proto.Aggregation
  ) -> Optional[config_pb2.AggregationOptions]:
    """Convert ME Aggregation into TFMA AggregationOptions.

    Args:
      me_aggregation: Input ME Aggregation.

    Returns:
      TFMA AggregationOptions.
    """
    if not me_aggregation:
      return None
    tfma_aggregation = config_pb2.AggregationOptions()
    if me_aggregation.micro_average:
      tfma_aggregation.micro_average = True
    if me_aggregation.macro_average:
      tfma_aggregation.macro_average = True
    if me_aggregation.class_weights:
      tfma_aggregation.class_weights.update(me_aggregation.class_weights)

    return tfma_aggregation

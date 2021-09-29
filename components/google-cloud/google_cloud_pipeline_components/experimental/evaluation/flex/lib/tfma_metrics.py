"""Provides consolidated access to metrics for Tensorflow Model Analysis."""

import json
from typing import Any, Dict, List, Optional

import tensorflow_model_analysis as tfma
from lib import constants

# Convenience aliases.
Metric = constants.Metric
Thresholds = constants.Thresholds


class MetricSpec(object):
  """Describes a model quality metric specification.

  Attributes:
    class_name: Name of the class that implements the metric
    module_name: Name of the module containing the class that implements the
      metric.
    config: JSON encoded configuration string for this metric.
    multiclass_only: True is the metric must not be binarized.
    aggregatable: True if the metric can be aggregated. Some metrics, like
      number of examples ot confusion matrix cannot be.
  """

  def __init__(self,
               class_name=None,
               module_name=None,
               config=None,
               multiclass_only=False,
               aggregatable=True):
    self.__class_name = class_name
    self.__module_name = module_name
    self.__config = config
    self.__multiclass_only = multiclass_only
    self.__aggregatable = aggregatable

  @property
  def class_name(self):
    return self.__class_name

  @property
  def module_name(self):
    return self.__module_name

  @property
  def config(self):
    return self.__config

  @property
  def multiclass_only(self):
    return self.__multiclass_only

  @property
  def aggregatable(self):
    return self.__aggregatable

  def as_metric_config(self):
    """Get a TFMA MetricConfig.

    Returns:
      MetricConfig.
    """
    return tfma.config.MetricConfig(
        class_name=self.class_name,
        module=self.module_name,
        config=json.dumps(self.config) if self.config else None)


_TFMAM = "tensorflow_model_analysis.metrics"
_KERAS = "tensorflow.keras.metrics"
_DEFAULT_THRESHOLDS = [x * 0.01 for x in range(0, 100)]
_METRIC_MAP = {
    Metric.AUC:
        MetricSpec(
            class_name="AUC",
            module_name=_KERAS,
            config={
                "name": Metric.AUC,
                "num_thresholds": Thresholds.NUM_THRESHOLDS
            }),
    Metric.AUC_PRECISION_RECALL:
        MetricSpec(
            class_name="AUC",
            module_name=_KERAS,
            config={
                "name": Metric.AUC_PRECISION_RECALL,
                "curve": "PR",
                "num_thresholds": Thresholds.NUM_THRESHOLDS
            }),
    Metric.ACCURACY:
        MetricSpec(
            class_name="Accuracy",
            module_name=_KERAS,
            config={"name": Metric.ACCURACY},
            aggregatable=False),
    Metric.BINARY_ACCURACY:
        MetricSpec(
            class_name="BinaryAccuracy",
            module_name=_KERAS,
            config={"name": Metric.BINARY_ACCURACY}),
    Metric.BINARY_CROSSENTROPY:
        MetricSpec(
            class_name="BinaryCrossentropy",
            module_name=_KERAS,
            config={"name": Metric.BINARY_CROSSENTROPY}),
    Metric.CALIBRATION:
        MetricSpec(
            class_name="Calibration",
            module_name=_TFMAM + ".calibration",
            config={"name": Metric.CALIBRATION}),
    Metric.CALIBRATION_PLOT:
        MetricSpec(
            class_name="CalibrationPlot",
            module_name=_TFMAM + ".calibration_plot",
            config={"name": Metric.CALIBRATION_PLOT
                   }),  # TODO(msiegler)  left=min_value, right=max_value
    Metric.CATEGORICAL_ACCURACY:
        MetricSpec(
            class_name="CategoricalAccuracy",
            module_name=_KERAS,
            config={"name": Metric.CATEGORICAL_ACCURACY},
            multiclass_only=True),
    Metric.CATEGORICAL_CROSSENTROPY:
        MetricSpec(
            class_name="CategoricalCrossentropy",
            module_name=_KERAS,
            config={"name": Metric.CATEGORICAL_CROSSENTROPY},
            multiclass_only=True),
    Metric.CONFUSION_MATRIX:
        MetricSpec(
            class_name="ConfusionMatrixAtThresholds",
            module_name=_TFMAM + ".confusion_matrix_metrics",
            config={
                "name": Metric.CONFUSION_MATRIX,
                "thresholds": Thresholds.THRESHOLD_LIST
            },
            aggregatable=False),
    Metric.EXAMPLE_COUNT:
        MetricSpec(
            class_name="ExampleCount",
            module_name=_TFMAM + ".example_count",
            config={"name": Metric.EXAMPLE_COUNT},
            aggregatable=False),
    Metric.MEAN_ABSOLUTE_ERROR:
        MetricSpec(
            class_name="MeanAbsoluteError",
            module_name=_KERAS,
            config={"name": Metric.MEAN_ABSOLUTE_ERROR},
            aggregatable=False),
    Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR:
        MetricSpec(
            class_name="MeanAbsolutePercentageError",
            module_name=_KERAS,
            config={"name": Metric.MEAN_ABSOLUTE_PERCENTAGE_ERROR},
            aggregatable=False),
    Metric.MEAN_LABEL:
        MetricSpec(
            class_name="MeanLabel",
            module_name=_TFMAM + ".calibration",
            config={"name": Metric.MEAN_LABEL}),
    Metric.MEAN_PREDICTION:
        MetricSpec(
            class_name="MeanPrediction",
            module_name=_TFMAM + ".calibration",
            config={"name": Metric.MEAN_PREDICTION}),
    Metric.MEAN_SQUARED_LOG_ERROR:
        MetricSpec(
            class_name="MeanSquaredLogarithmicError",
            module_name=_KERAS,
            config={"name": Metric.MEAN_SQUARED_LOG_ERROR},
            aggregatable=False),
    Metric.MULTI_CLASS_CONFUSION_MATRIX:
        MetricSpec(
            class_name="MultiClassConfusionMatrixAtThresholds",
            module_name=_TFMAM + ".multi_class_confusion_matrix_metrics",
            config={
                "name": Metric.MULTI_CLASS_CONFUSION_MATRIX,
                "thresholds": [Thresholds.DEFAULT_DECISION_THRESHOLD],
            },
            aggregatable=False),
    Metric.PRECISION:
        MetricSpec(
            class_name="Precision",
            module_name=_KERAS,
            config={
                "name": Metric.PRECISION,
                "thresholds": [Thresholds.DEFAULT_DECISION_THRESHOLD]
            },
        ),
    Metric.R_SQUARED:
        MetricSpec(
            class_name="SquaredPearsonCorrelation",
            module_name=_TFMAM,
            config={"name": Metric.R_SQUARED},
            aggregatable=False),
    Metric.RECALL:
        MetricSpec(
            class_name="Recall",
            module_name=_KERAS,
            config={
                "name": Metric.RECALL,
                "thresholds": [Thresholds.DEFAULT_DECISION_THRESHOLD]
            },
        ),
    Metric.ROOT_MEAN_SQUARED_ERROR:
        MetricSpec(
            class_name="RootMeanSquaredError",
            module_name=_KERAS,
            config={"name": Metric.ROOT_MEAN_SQUARED_ERROR},
            aggregatable=False),
    Metric.WEIGHTED_EXAMPLE_COUNT:
        MetricSpec(
            class_name="WeightedExampleCount",
            module_name=_TFMAM + ".weighted_example_count",
            config={"name": Metric.WEIGHTED_EXAMPLE_COUNT},
            aggregatable=False),
}


def _lookup(key: str, *dicts: Dict[str, Any]) -> Any:
  """Look up a key in multiple dictionaries in order."""
  for dict1 in dicts:
    if key in dict1:
      return dict1.get(key)
  return None


def get_metric_specs(
    metric_names: List[str],
    model_names: Optional[List[str]] = None,
    is_multi_label: bool = False,
    positive_class_ids: Optional[List[int]] = None,
    top_k_list: Optional[List[int]] = None,
    class_weights: Optional[Dict[int, float]] = None,
    custom_metric_map: Optional[Dict[str, MetricSpec]] = None,
) -> List[tfma.config.MetricsSpec]:
  """Given a list of metric names, return them as tfma MetricSpec values.

  This separates metrics into multiple specs if required.

  Args:
    metric_names: A list of metrics, from constants.Metric.
    model_names: A list of models evaluated by the metrics.
    is_multi_label: True for multi-label case.
    positive_class_ids: A list of positive class ids for binarization.
    top_k_list: A list of k values for top-k eval.
    class_weights: Optional mapping from class id to class weight. Required for
      multilabel problems.
    custom_metric_map: Optional custom metrics.

  Returns:
    A list of TFMA MetricSpecs.

  Raises:
    KeyError: if a metric is not defined.
  """
  aggregation = tfma.AggregationOptions()
  if is_multi_label:
    aggregation.macro_average = True
    if class_weights:
      aggregation.class_weights.update(class_weights)
  else:
    aggregation.micro_average = True

  if positive_class_ids or top_k_list:
    binarization_options = tfma.BinarizationOptions()
    if positive_class_ids:
      binarization_options.class_ids.values.extend(positive_class_ids)
    if top_k_list:
      binarization_options.top_k_list.values.extend(top_k_list)
    # A separate metric computation will be computed for each option
    # and be distinguished via their subkey argument.
  else:
    binarization_options = None

  unaggregated_unbinarized_spec = tfma.config.MetricsSpec(
      binarize=None, aggregate=None)
  unaggregated_binarized_spec = tfma.config.MetricsSpec(
      binarize=binarization_options, aggregate=None)
  aggregated_unbinarized_spec = tfma.config.MetricsSpec(
      binarize=None, aggregate=aggregation)

  for name in metric_names:
    metric = _lookup(name, _METRIC_MAP, custom_metric_map)
    if not metric:
      raise KeyError("Metric named %r is undefined." % name)
    # Note, do not add a metric to both aggregated and unaggregated specs as
    # these will not be disambiguated in the output.
    if metric.multiclass_only:
      unaggregated_unbinarized_spec.metrics.append(metric.as_metric_config())
    else:
      if metric.aggregatable:
        aggregated_unbinarized_spec.metrics.append(metric.as_metric_config())
      else:
        unaggregated_unbinarized_spec.metrics.append(metric.as_metric_config())
      unaggregated_binarized_spec.metrics.append(metric.as_metric_config())

  all_specs = []
  # These are returned in a specific order such that later specs are preferred
  # if the disambiguation between them fails.
  for spec in [
      unaggregated_unbinarized_spec, unaggregated_binarized_spec,
      aggregated_unbinarized_spec
  ]:
    if spec.metrics:
      if model_names:
        spec.model_names.extend(model_names)
      all_specs.append(spec)
  return all_specs

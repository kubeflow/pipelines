"""Library-wide constants for metrics."""

from enum import Enum  # pylint: disable=g-importing-member


class Slack(object):
  """Namespace for slack constants."""
  # TFMA does not always return exact counts (side effect of computing many
  # confusion  matrtices simultaneously efficiently). See the thread for more
  # context:
  # https://groups.google.com/a/google.com/g/ucaip-model-eval-eng/c/E-JrccsGiAU
  # We allow confusion matrix entries vary by this amount (should only be used
  # for large test sets).
  # This is tracked in b/173621912.
  FULLSIZE_CONFUSION_MATRIX_COUNT = 4

  # Tolerance for AUC metrics to use for small datasets.
  SMALL_AUC_DELTA = 0.02

  # Tolerance for AUC metrics to use for large datasets.
  AUC_DELTA = 0.005

  # Tolerances for general purpose metric value comparisons.
  METRIC_DELTA = 0.001


_NUM_THRESHOLDS = 100


class Thresholds(object):
  DEFAULT_DECISION_THRESHOLD = 0.5
  NUM_THRESHOLDS = _NUM_THRESHOLDS
  THRESHOLD_LIST = [x / _NUM_THRESHOLDS for x in range(-1, _NUM_THRESHOLDS + 2)]


class Metric(object):
  """Namespace for Metrics."""
  AUC = 'auc'
  AUC_PRECISION_RECALL = 'auc_precision_recall'
  ACCURACY = 'accuracy'
  BINARY_ACCURACY = 'binary_accuracy'
  BINARY_CROSSENTROPY = 'binary_loss'
  CALIBRATION = 'calibration'
  CALIBRATION_PLOT = 'calibration_plot'
  CATEGORICAL_ACCURACY = 'categorical_accuracy'
  CATEGORICAL_CROSSENTROPY = 'categorical_loss'
  CONFUSION_MATRIX = 'confusion_matrix_metrics'
  MEAN_LABEL = 'mean_label'
  EXAMPLE_COUNT = 'ExampleCount'
  MEAN_ABSOLUTE_ERROR = 'mae'
  MEAN_ABSOLUTE_PERCENTAGE_ERROR = 'mape'
  MEAN_PREDICTION = 'mean_prediction'
  MEAN_SQUARED_LOG_ERROR = 'msle'
  MULTI_CLASS_CONFUSION_MATRIX = 'multi_class_confusion_matrix'
  PRECISION = 'precision'
  RECALL = 'recall'
  ROOT_MEAN_SQUARED_ERROR = 'rmse'
  R_SQUARED = 'squared_pearson_correlation'
  WEIGHTED_EXAMPLE_COUNT = 'weighted_example_count'
  QUANTILE_ACCURACY = 'quantile_accuracy'
  CUSTOM = 'google3.cloud.ai.platform.evaluation.metrics.lib.custom_metrics'


class Data(object):
  """Namespace for Data."""
  # Standard labels used for binary classification. If the model only scores the
  # positive class in binary classification we convert this problem to and
  # equivalent multiclass classification with two classes: 'true' and 'false'.
  BINARY_CLASSIFICATION_LABELS = ['false', 'true']
  # Same as above, but for indices. 0 - 'false'; 1 - 'true'.
  BINARY_CLASSIFICATION_LABEL_IDS = [0, 1]
  # Used to append k-hot labels in classification tasks with sparse labels.
  K_HOT_KABEL_KEY_SUFFIX = '__k_hot'
  # Used to append point predictions in forecasting tasks with quantiles.
  POINT_KEY_SUFFIX = '__point'
  # Used to append quantile predictions in forecasting tasks with quantiles.
  QUANTILE_KEY_SUFFIX = '__quantile'


class ProblemType(Enum):
  # TODO(b/178198082): Consider renaming the constants below
  MULTICLASS = 1
  MULTILABEL = 2
  REGRESSION = 3
  POINT_FORECASTING = 4
  QUANTILE_FORECASTING = 5


class Pipeline(object):
  """Namespace for Beam Pipeline related constants."""
  DATAFLOW_RUNNER = 'DataflowRunner'
  ENV_VAR_SETUP = 'FLEX_TEMPLATE_PYTHON_SETUP_FILE'
  METRICS_KEY = 'metrics'
  MODEL_KEY = 'model'

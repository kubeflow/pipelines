"""Module for input and output processing for Cloud AI Metrics."""

import copy
import dataclasses
import json
import numbers
from typing import Any, Dict, List, Optional, Tuple, Type, Union
import tensorflow.compat.v2 as tf

from lib import column_spec, constants, evaluation_column_specs as ecs

ColumnSpec = column_spec.ColumnSpec
EvaluationColumnSpecs = ecs.EvaluationColumnSpecs


def _k_hot_from_label_ids(label_ids: List[int], n_classes: int) -> List[int]:
  """Converts a list of label ids into a k-hot encoding."""
  _validate_label_ids(label_ids, n_classes)
  # Preallocate the encoding and set all bits to 0.
  k_hot = [0] * n_classes
  # Turn on the bits that correspond to label ids.
  for label_id in label_ids:
    k_hot[label_id] = 1

  return k_hot


def _k_hot_from_label_names(labels: List[str], symbols: List[str]) -> List[int]:
  """Converts text labels into symbol list index as k-hot."""
  k_hot = [0] * len(symbols)
  for label in labels:
    try:
      k_hot[symbols.index(label)] = 1
    except IndexError:
      raise ValueError(
          'Label %s did not appear in the list of defined symbols %r' %
          (label, symbols))
  return k_hot


def _validate_no_repeats(values: List[Any], name: str) -> None:
  """Validates that all te elements in the list are unique."""
  if not values:
    return

  n = len(values)
  n_unique = len(set(values))
  if n != n_unique:
    raise ValueError('{}: all values must be unique.'.format(name))


def _validate_list(values: Union[List[int], List[float], List[str]],
                   allowed_types: List[Type[Any]], name: str) -> None:
  """Validates that the list is non-empty and homogeneous."""
  if not values:
    raise ValueError('{}: values list is empty.'.format(name))

  if not isinstance(values, list):
    raise TypeError('{}: values are in a {} but expected a list.'.format(
        name, type(values)))

  value_type = type(values[0])
  if value_type not in allowed_types:
    raise TypeError(
        '{}: values are expected to be one of {} but are {}.'.format(
            name, allowed_types, value_type))
  if not all(isinstance(value, value_type) for value in values):
    raise TypeError(
        '{}: all value types are expected to be {} but are not.'.format(
            name, value_type))


def _validate_binary_classification_labels(labels: List[str]) -> None:
  """Validates label specification for binary classification."""
  if not labels:
    raise ValueError('labels must not be empty.')

  n = len(labels)
  if n > 1:
    raise ValueError(
        'Binary classification requires exactly one label. Got {}'.format(n))

  if labels[0].lower() not in constants.Data.BINARY_CLASSIFICATION_LABELS:
    raise ValueError(
        'Labels for binary classification must be one of ' +
        '{} (not case sensitive). Got {}'.format(
            constants.Data.BINARY_CLASSIFICATION_LABELS, labels[0]))


def _validate_classification_labels(labels: List[str],
                                    class_list: List[str]) -> None:
  """Validates that the labels are specified correctly for classification."""
  _validate_list(values=labels, allowed_types=[str], name='labels')
  _validate_no_repeats(values=labels, name='labels')

  if not all((label in class_list) for label in labels):
    raise ValueError('labels: some labels are not recognized. Got labels: ' +
                     '{}. Allowed labels: {}'.format(labels, class_list))


def _validate_label_ids(label_ids: List[int], n_classes: int) -> None:
  """Validates that label ids are compatible with the list of labels."""
  _validate_list(values=label_ids, allowed_types=[int], name='label_ids')
  _validate_no_repeats(values=label_ids, name='label_ids')

  max_id = max(label_ids)
  if max_id >= n_classes:
    raise ValueError(
        'Label index {} is out of range. There are only {} labels'.format(
            max_id, n_classes))

  min_id = min(label_ids)
  if min_id < 0:
    raise ValueError(
        'Label indices must be non-negative. Got {}'.format(min_id))


def _validate_binary_label_ids(label_ids: List[int]) -> None:
  """Validates that binary label ids are specified correctly."""
  if not label_ids:
    raise ValueError('label_ids must be specified.')

  n = len(label_ids)
  if n > 1:
    raise ValueError(
        'Exactly one label must be provided for binary classification. Got {}'
        .format(n))

  if label_ids[0] not in constants.Data.BINARY_CLASSIFICATION_LABEL_IDS:
    raise ValueError(
        'Label id for binary classification must be one of {}. Got {}'.format(
            constants.Data.BINARY_CLASSIFICATION_LABEL_IDS, label_ids[0]))


# pyformat: disable
def _validate_classification_ground_truth(
    ground_truth: Union[List[str], List[int]],
    class_list: List[str]) -> None:
  # pyformat: enable
  """Validates that ground truth is specified correctly for classification."""
  if not ground_truth:
    raise ValueError('Ground truth must not be empty.')

  if isinstance(ground_truth[0], int):
    # Ground truth specified as label id.
    _validate_label_ids(label_ids=ground_truth, n_classes=len(class_list))
  else:
    # Ground truth specified as label names.
    _validate_classification_labels(labels=ground_truth, class_list=class_list)


# pyformat: disable
def _validate_binary_classification_ground_truth(
    ground_truth: Union[List[str], List[int]],
    class_list: List[str]) -> None:
  # pyformat: enable
  """Validates ground truth specification for binary classification."""
  if not ground_truth:
    raise ValueError('Ground truth must not be empty.')

  if isinstance(ground_truth[0], int):
    # Ground truth specified as label id.
    _validate_binary_label_ids(label_ids=ground_truth)
  else:
    # Ground truth specified as label names.
    _validate_binary_classification_labels(labels=ground_truth)


def _validate_binary_classification_predictions(predictions: List[Any],
                                                labels: List[Any],
                                                label_ids: List[Any]) -> None:
  """Validates predictions specification for binary classification."""
  _validate_list(
      values=predictions, allowed_types=[float], name='binary predictions')
  prediction_count = len(predictions) if predictions else 0
  if prediction_count != 1:
    raise ValueError(
        'binary predictions must contain exactly one value. Got {}'.format(
            prediction_count))

  if labels:
    _validate_binary_classification_labels(labels)

  if label_ids:
    _validate_binary_label_ids(label_ids)


def _validate_classification_predictions(predictions: List[Any],
                                         labels: List[Any],
                                         label_ids: List[Any],
                                         class_list: List[str]) -> None:
  """Validates that predictions are specified correctly for classification."""
  _validate_list(values=predictions, allowed_types=[float], name='predictions')
  n_predictions = len(predictions) if predictions else 0

  if labels:
    _validate_classification_labels(labels, class_list)
    n_labels = len(labels)
    if n_labels != n_predictions:
      raise ValueError(
          'labels and predictions must be of the same size, but {} != {}'
          .format(n_labels, n_predictions))

  if label_ids:
    _validate_label_ids(label_ids, n_classes=len(class_list))
    n_label_ids = len(label_ids)
    if n_label_ids != n_predictions:
      raise ValueError(
          'label_ids and predictions must be of the same size, but {} != {}'
          .format(n_label_ids, n_predictions))

  if labels and label_ids:
    if not all((class_list[label_id] == label)
               for (label, label_id) in zip(labels, label_ids)):
      raise ValueError(
          'label_ids and labels are inconsistent. Labels indicated by ' +
          'label_ids are: {}. Classes indicated by labels are {}.'.format(
              [class_list[label_id] for label_id in label_ids], labels))


def _validate_regression_labels(labels: List[Any]) -> None:
  """Validates that the labels are specified correctly for regression."""
  if not labels:
    raise ValueError('labels: values list is empty.')

  if not isinstance(labels, list):
    raise TypeError('labels: values are in a {} but expected a list.'.format(
        type(labels)))


def _validate_regression_or_point_forecasting_predictions(
    prediction_values: List[Any]) -> None:
  _validate_list(
      values=prediction_values, allowed_types=[float], name='predictions')


def _validate_quantile_forecasting_predictions(
    prediction_values: List[Any], quantile_list: List[float]) -> None:
  """Validates that predictions and quantiles are specified correctly."""
  _validate_list(
      values=prediction_values, allowed_types=[float], name='predictions')
  if len(quantile_list) != len(prediction_values):
    raise ValueError('predictions: values {} unmatch with quantiles {}.'.format(
        prediction_values, quantile_list))


def _is_binary_classification(class_list: List[str]) -> bool:
  if not class_list:
    return False

  return len(class_list) == 1


def _ensure_list(value: Any) -> List[Any]:
  """If value is a scalar, converts it to a list of size 1."""
  if isinstance(value, list):
    return value

  if isinstance(value, str) or isinstance(value, numbers.Number):
    return [value]

  raise TypeError(
      f'Value must be a list, number or a string. Got {type(value)}')


def _pop_labels(input_dict: Dict[str, Any], label_column_spec: ColumnSpec,
                class_list: List[str]) -> Tuple[Any, Optional[List[int]]]:
  """Pops the labels off the input dict and formats accordingly."""
  labels = _ensure_list(label_column_spec.pop_value_from_dict(input_dict))

  if not class_list:
    # REGRESSION PROBLEM
    _validate_regression_labels(labels)
    for i, value in enumerate(labels):
      try:
        labels[i] = float(value)
      except:  # pylint: disable=broad-except
        raise ValueError(
            'label value: {} failed to cast to float.'.format(value))
    return (labels, None)

  # If input label is boolean, convert it into str, as the class are strings.
  if labels and isinstance(labels[0], bool):
    labels = ['true' if label else 'false' for label in labels]
  k_hot_label_values = _classification_k_hot(labels, class_list)
  return (labels, k_hot_label_values)


def _classification_k_hot(labels: List[int],
                          class_list: List[str]) -> List[int]:
  """Pops the labels off the input dict and return a k_hot label vector."""

  if _is_binary_classification(class_list):
    # BINARY CLASSIFICATION PROBLEM
    _validate_binary_classification_ground_truth(labels, class_list)
    if isinstance(labels[0], str):
      # Ground truth is represented as label names.
      return _k_hot_from_label_names(
          [labels[0].lower()], constants.Data.BINARY_CLASSIFICATION_LABELS)

    # Ground truth is represented as label ids.
    return _k_hot_from_label_ids(labels, 2)

  # NON_BINARY CLASSIFICATION PROBLEM
  _validate_classification_ground_truth(labels, class_list)
  if isinstance(labels[0], str):
    # Ground truth is represented as label names.
    return _k_hot_from_label_names(labels, class_list)

  # Ground truth is represented as label ids.
  return _k_hot_from_label_ids(labels, len(class_list))


def _k_hot_to_sparse(k_hot: List[int]) -> List[int]:
  """Converts k-hot embedding to sparse representation."""
  return [idx for idx, val in enumerate(k_hot) if val != 0]


def _pop_predictions(
    input_dict: Dict[str, Any],
    evaluation_column_specs: EvaluationColumnSpecs,
    class_list: Optional[List[str]] = None,
    quantile_list: Optional[List[float]] = None,
    quantile_index: Optional[int] = None
) -> Tuple[List[float], Optional[List[float]]]:
  """Pops the predictions off the input dict and formats accordingly."""

  score_spec = evaluation_column_specs.predicted_score_column_spec
  prediction_values = _ensure_list(score_spec.pop_value_from_dict(input_dict))

  if not class_list:
    if quantile_list:
      # QUANTILE FORECASTING PROBLEM
      _validate_quantile_forecasting_predictions(prediction_values,
                                                 quantile_list)
      if (quantile_index is not None) and quantile_index >= 0:
        return (prediction_values,
                prediction_values[quantile_index:quantile_index + 1])
      else:
        return (prediction_values, None)
    else:
      # REGRESSION/ POINT FORECASTING PROBLEM
      _validate_regression_or_point_forecasting_predictions(prediction_values)
      return (prediction_values, None)

  label_ids = None
  label_id_spec = evaluation_column_specs.predicted_label_id_column_spec
  if label_id_spec and label_id_spec.exists_in_dict(input_dict):
    # Model outputs label IDs, so we just use them directly.
    label_ids = label_id_spec.pop_value_from_dict(input_dict)

  labels = None
  label_spec = evaluation_column_specs.predicted_label_column_spec
  if label_spec and label_spec.exists_in_dict(input_dict):
    # Model outputs labels but not label ids.
    labels = label_spec.pop_value_from_dict(input_dict)

  if _is_binary_classification(class_list):
    # BINARY CLASSIFICATION PROBLEM
    _validate_binary_classification_predictions(prediction_values, labels,
                                                label_ids)
    # We are representing binary classification problem as a multiclass problem.
    # For binary classification problem, the model outputs probability (p)
    # of the positive class only. We amend it with the probability of the
    # negative class, which is 1 - p. The order of probabilities is negative
    # class first, then positive class.
    return ([1 - prediction_values[0], prediction_values[0]], None)

  # NON_BINARY CLASSIFICATION PROBLEM
  _validate_classification_predictions(prediction_values, labels, label_ids,
                                       class_list)
  if not labels:
    # Model outputs neither labels nor label ids. Use class_list as output
    # labels.
    labels = class_list

  # Convert labels into label ids if necessary.
  if not label_ids:
    label_ids = [class_list.index(label) for label in labels]

  reordered_prediction_values = [0.] * len(class_list)
  for (label_id, prediction_value) in zip(label_ids, prediction_values):
    reordered_prediction_values[label_id] = prediction_value
  return (reordered_prediction_values, None)


def _pop_weight(input_dict: Dict[str, Any],
                in_weight_key_spec: ColumnSpec) -> float:
  """Pops the weight value off the input dict."""
  weight = in_weight_key_spec.pop_value_from_dict(input_dict)
  if not isinstance(weight, float):
    raise TypeError('Weight is a %s but expected a float.' % type(weight))
  return weight


def _as_feature(
    value_list: Union[List[int], List[str], List[float]]) -> tf.train.Feature:
  """Converts a plain list into a tf.train.Feature."""
  if not value_list:
    return None
  if isinstance(value_list[0], int):
    return tf.train.Feature(int64_list=tf.train.Int64List(value=value_list))
  if isinstance(value_list[0], str):
    return tf.train.Feature(
        bytes_list=tf.train.BytesList(
            value=[value.encode('utf-8') for value in value_list]))
  if isinstance(value_list[0], float):
    return tf.train.Feature(float_list=tf.train.FloatList(value=value_list))
  # TODO(b/179309868)
  return None


# pyformat: disable
def create_example(feature_dict: Dict[str, Any],
                   evaluation_column_specs: EvaluationColumnSpecs,
                   label_values: Union[List[int], List[float]],
                   prediction_values: Union[List[int], List[float]],
                   weight: Optional[float],
                   k_hot_key: Optional[str],
                   k_hot_values: Optional[List[int]],
                   point_key: Optional[str],
                   point_values: Optional[List[float]]) -> tf.train.Example:
  # pyformat: enable
  """Creates a tf.Train.Example from split inputs."""
  feature_map = {}
  for (k, v) in feature_dict.items():
    feature = _as_feature([v])
    if feature:
      feature_map[k] = feature
  feature_map[evaluation_column_specs.ground_truth_column_spec.as_string(
  )] = _as_feature(label_values)
  feature_map[evaluation_column_specs.predicted_score_column_spec.as_string(
  )] = _as_feature(prediction_values)
  if evaluation_column_specs.example_weight_column_spec:
    feature_map[evaluation_column_specs.example_weight_column_spec.as_string(
    )] = _as_feature([float(weight)])
  if k_hot_key and k_hot_values:
    feature_map[k_hot_key] = _as_feature(k_hot_values)
  if point_key and point_values:
    feature_map[point_key] = _as_feature(point_values)
  example = tf.train.Example(features=tf.train.Features(feature=feature_map))
  return example


@dataclasses.dataclass
class _Example:
  feature_dict: Dict[str, Any]
  label_values: Union[List[int], List[float]]
  prediction_values: List[float]
  weight_values: Union[float, None]
  k_hot_label_values: Optional[List[int]]
  point_prediction_values: Optional[List[float]]


def _split_dict(
    input_dict: Dict[str, Any],
    evaluation_column_specs: EvaluationColumnSpecs,
    class_list: Optional[List[str]] = None,
    quantile_list: Optional[List[float]] = None,
    quantile_index: Optional[int] = None,
) -> _Example:
  """Splits the input dictionary into a tuple for feature conversion."""
  if not input_dict:
    raise ValueError('No contents(%r) to split' % input_dict)
  # Permit manipulation of the input dictionary.
  feature_dict = copy.deepcopy(input_dict)
  (label_values, k_hot_label_values) = _pop_labels(
      feature_dict, evaluation_column_specs.ground_truth_column_spec,
      class_list)
  (prediction_values, point_prediction_values) = _pop_predictions(
      feature_dict, evaluation_column_specs, class_list, quantile_list,
      quantile_index)
  weight = _pop_weight(
      feature_dict, evaluation_column_specs.example_weight_column_spec
  ) if evaluation_column_specs.example_weight_column_spec else None

  return _Example(
      _flatten_dict(d=feature_dict, processed_keys=[]), label_values,
      prediction_values, weight, k_hot_label_values, point_prediction_values)


def parse_row_as_example(
    json_data: bytes,
    evaluation_column_specs: EvaluationColumnSpecs,
    class_list: Optional[List[str]] = None,
    quantile_list: Optional[List[float]] = None,
    quantile_index: Optional[int] = None) -> tf.train.Example:
  """Parses a row of json data (as bytes) into a tf.train.Example.

  Args:
    json_data: The raw data. (See format below.)
    evaluation_column_specs: column specs necessary for parsing evaluation data.
    class_list: Optional list of classification labels. Presumed regression
      problem type if this is missing or empty.
    quantile_list: Optional quantile list. Only valid for forecasting problem.
    quantile_index: Optional quantile index. Only valid for forecasting problem.

  Returns:
    A tf.train.Example. (See format below)

  json_data input format:

    Problem Type: CLASSIFICATION
      given class_list ["class1", "class2", ... , "classN] of size N
      Truths of count K
      Predictions of count M:

      { "<labels_key>" : [ "<label1>", "<label2>", ..., "<labelK>"],
        "<predicted_label_key>": ["<label1>", "<label2>", ..., "<labelM>"],
        "<predicted_label_id_key>": [<int1>, <int2>, ..., <intM>],
        "<predicted_score_key": [<float1>, <float2>, ..., "<labelM"> ],
        "<weight_key>" : <float>,
        "<float_feature_key>" : <float>,
        "<int_feature_key>" : <int>,
        "<string_feature_key>" : <string>,
       }

       Label ids are 0-based indices into class_list.
       If label_ids are present in predictions, then they are used to determine
       correspondence between prediction scores and classes; otherwise labels
       are used instead. If neither label ids nor labels are present, the
       prediction scores are assumed to appear in the same order as classes
       in the class_list.

    Problem Type: REGRESSION or FORECASTING (POINT)
      { "<labels_key>" : [<float>],
        "<predicted_score_key>": [<float>],
        "<weight_key>" : <float>,
        "<float_feature_key>" : <float>,
        "<int_feature_key>" : <int>,
        "<string_feature_key>" : <string>,
      }

    Problem Type: FORECASTING (QUANTILE)
      { "<labels_key>" : [<float>],
        "<predicted_score_key>": [<float1>, <float2>, <float3>, ..., <floatK>],
        "<weight_key>" : <float>,
        "<float_feature_key>" : <float>,
        "<int_feature_key>" : <int>,
        "<string_feature_key>" : <string>,
      }

  tf.Example output format (proto)

    Problem Type: CLASSIFICATION
      given class_list ["class1", "class2", ... , "classN]
      {
        "<labels_key>":
          # k-hot 0/1 indication of false/true for each label.
          int32_list: { value: <int1>    value: <int2> ... value: <intN>}
        "<predictions_key>":
          # Prediction probabilites of every class label.
          float_list: { value: <float1>  value: <value2> ...  value: <floatN>}
        "<weight_key>" :
          float_lsit: { <float> }
        "<float_feature_key>"> :
          float_list: { value: <float1>  value: <float2> ... }
        "int_feature_key>"> :
          int32_list: { value: <int1>    value: <int2> ... }
        "<string_feature_key>"> :
          bytes_list: { value: <bytes1>  value: <bytes2> ... }
      }

    Problem Type: REGRESSION or FORECASTING (POINT)
        "<labels_key>":
          float_list: { value: <float> }
        "<predictions_key>":
          float_list: { value: <float> }
        "<weight_key>" :
          float_lsit: { <float> }
        "<float_feature_key>"> :
          float_list: { value: <float1>  value: <float2> ... }
        "int_feature_key>"> :
          int32_list: { value: <int1>    value: <int2> ... }
        "<string_feature_key>"> :
          bytes_list: { value: <bytes1>  value: <bytes2> ... }
      }

   Problem Type: FORECASTING (QUANTILE)
        "<labels_key>":
          float_list: { value: <float> }
        "<predictions_key>":
          float_list: { value: <float1>  value: <value2> ...  value: <floatK>}
        <point_prediction_key>:
          float_list: { value: <float> }
        "<weight_key>" :
          float_lsit: { <float> }
        "<float_feature_key>"> :
          float_list: { value: <float1>  value: <float2> ... }
        "int_feature_key>"> :
          int32_list: { value: <int1>    value: <int2> ... }
        "<string_feature_key>"> :
          bytes_list: { value: <bytes1>  value: <bytes2> ... }
      }
  """
  try:
    json_dict = json.loads(json_data)
  except json.decoder.JSONDecodeError:
    raise ValueError('Cannot decode line %r' % json_data)

  example = _split_dict(json_dict, evaluation_column_specs, class_list,
                        quantile_list, quantile_index)

  # Remaining keys in input_dict are features.
  return create_example(
      feature_dict=example.feature_dict,
      evaluation_column_specs=evaluation_column_specs,
      label_values=example.label_values,
      prediction_values=example.prediction_values,
      weight=example.weight_values,
      k_hot_key=evaluation_column_specs.ground_truth_column_spec.as_string() +
      constants.Data.K_HOT_KABEL_KEY_SUFFIX,
      k_hot_values=example.k_hot_label_values,
      point_key=evaluation_column_specs.predicted_score_column_spec.as_string() +
      constants.Data.POINT_KEY_SUFFIX,
      point_values=example.point_prediction_values)


def _get_flattened_name(names: List[str]) -> str:
  return ColumnSpec(names).as_string()


def _flatten_dict(d: Dict[str, Any],
                  processed_keys: List[str]) -> Dict[str, Any]:
  """Recursively flattens the dictionary.

  This function is used to pre-process the dictionary that remains after
  special fields (ground truth, predicted scores, etc.) are removed from it.
  The flattened dictionary will contain features that are getting passed to
  TFMA.
  For example,
  {
    'a': {
      'b': 2
    }
  }

  will be transformed into
  {
    'a__b': 2
  }

  Args:
    d: Dictionary to be flattened.
    processed_keys: Dictionary keys that had already been processed.

  Returns:
    Flattened dictionary.
  """
  flattened_dict = {}
  for key in d:
    current_processed_keys = processed_keys + [key]
    if isinstance(d[key], Dict):
      flattened_dict.update(
          _flatten_dict(d=d[key], processed_keys=current_processed_keys))
    else:
      flattened_dict[_get_flattened_name(current_processed_keys)] = d[key]

  return flattened_dict

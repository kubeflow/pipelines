# Copyright 2018 Google LLC
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


import argparse
import json
import os
import tensorflow as tf
import tensorflow_transform as tft
import tensorflow_model_analysis as tfma

from tensorflow.python.lib.io import file_io
from tensorflow_transform.beam.tft_beam_io import transform_fn_io
from tensorflow_transform.saved import input_fn_maker
from tensorflow_transform.saved import saved_transform_io
from tensorflow_transform.tf_metadata import dataset_metadata
from tensorflow_transform.tf_metadata import dataset_schema
from tensorflow_transform.tf_metadata import metadata_io


IMAGE_EMBEDDING_SIZE = 2048
CLASSIFICATION_TARGET_TYPES = [tf.bool, tf.int32, tf.int64]
REGRESSION_TARGET_TYPES = [tf.float32, tf.float64]
TARGET_TYPES = CLASSIFICATION_TARGET_TYPES + REGRESSION_TARGET_TYPES


def parse_arguments():
  parser = argparse.ArgumentParser()
  parser.add_argument('--job-dir',
                      type=str,
                      required=True,
                      help='GCS or local directory.')
  parser.add_argument('--transformed-data-dir',
                      type=str,
                      required=True,
                      help='GCS path containing tf-transformed training and eval data.')
  parser.add_argument('--schema',
                      type=str,
                      required=True,
                      help='GCS json schema file path.')
  parser.add_argument('--target',
                      type=str,
                      required=True,
                      help='The name of the column to predict in training data.')
  parser.add_argument('--learning-rate',
                      type=float,
                      default=0.1,
                      help='Learning rate for training.')
  parser.add_argument('--optimizer',
                      choices=['Adam', 'SGD', 'Adagrad'],
                      default='Adagrad',
                      help='Optimizer for training. If not provided, '
                           'tf.estimator default will be used.')
  parser.add_argument('--hidden-layer-size',
                      type=str,
                      default='100',
                      help='comma separated hidden layer sizes. For example "200,100,50".')
  parser.add_argument('--steps',
                      type=int,
                      help='Maximum number of training steps to perform. If unspecified, will '
                           'honor epochs.')
  parser.add_argument('--epochs',
                      type=int,
                      help='Maximum number of training data epochs on which to train. If '
                           'both "steps" and "epochs" are specified, the training '
                           'job will run for "steps" or "epochs", whichever occurs first.')
  parser.add_argument('--preprocessing-module',
                      type=str,
                      required=False,
                      help=('GCS path to a python file defining '
                            '"preprocess" and "get_feature_columns" functions.'))

  args = parser.parse_args()
  args.hidden_layer_size = [int(x.strip()) for x in args.hidden_layer_size.split(',')]
  return args


def is_classification(transformed_data_dir, target):
  """Whether the scenario is classification (vs regression).

  Returns:
    The number of classes if the target represents a classification
    problem, or None if it does not.
  """
  transformed_metadata = metadata_io.read_metadata(
      os.path.join(transformed_data_dir, transform_fn_io.TRANSFORMED_METADATA_DIR))
  transformed_feature_spec = transformed_metadata.schema.as_feature_spec()
  if target not in transformed_feature_spec:
    raise ValueError('Cannot find target "%s" in transformed data.' % target)

  feature = transformed_feature_spec[target]
  if (not isinstance(feature, tf.FixedLenFeature) or feature.shape != [] or
      feature.dtype not in TARGET_TYPES):
    raise ValueError('target "%s" is of invalid type.' % target)

  if feature.dtype in CLASSIFICATION_TARGET_TYPES:
    if feature.dtype == tf.bool:
      return 2
    return get_vocab_size(transformed_data_dir, target)

  return None


def make_tft_input_metadata(schema):
  """Create tf-transform metadata from given schema."""
  tft_schema = {}

  for col_schema in schema:
    col_type = col_schema['type']
    col_name = col_schema['name']
    if col_type == 'NUMBER':
      tft_schema[col_name] = dataset_schema.ColumnSchema(
          tf.float32, [], dataset_schema.FixedColumnRepresentation(default_value=0.0))
    elif col_type in ['CATEGORY', 'TEXT', 'IMAGE_URL', 'KEY']:
      tft_schema[col_name] = dataset_schema.ColumnSchema(
          tf.string, [], dataset_schema.FixedColumnRepresentation(default_value=''))
  return dataset_metadata.DatasetMetadata(dataset_schema.Schema(tft_schema))


def make_training_input_fn(transformed_data_dir, mode, batch_size, target_name, num_epochs=None):
  """Creates an input function reading from transformed data.
  Args:
    transformed_data_dir: Directory to read transformed data and metadata from.
    mode: 'train' or 'eval'.
    batch_size: Batch size.
    target_name: name of the target column.
    num_epochs: number of training data epochs.
  Returns:
    The input function for training or eval.
  """
  transformed_metadata = metadata_io.read_metadata(
      os.path.join(transformed_data_dir, transform_fn_io.TRANSFORMED_METADATA_DIR))
  transformed_feature_spec = transformed_metadata.schema.as_feature_spec()

  def _input_fn():
    """Input function for training and eval."""
    epochs = 1 if mode == 'eval' else num_epochs
    transformed_features = tf.contrib.learn.io.read_batch_features(
        os.path.join(transformed_data_dir, mode + '-*'),
        batch_size, transformed_feature_spec, tf.TFRecordReader, num_epochs=epochs)

    # Extract features and label from the transformed tensors.
    transformed_labels = transformed_features.pop(target_name)
    return transformed_features, transformed_labels

  return _input_fn


def make_serving_input_fn(transformed_data_dir, schema, target_name):
  """Creates an input function reading from transformed data.
  Args:
    transformed_data_dir: Directory to read transformed data and metadata from.
    schema: the raw data schema.
    target_name: name of the target column.
  Returns:
    The input function for serving.
  """
  raw_metadata = make_tft_input_metadata(schema)
  raw_feature_spec = raw_metadata.schema.as_feature_spec()

  raw_keys = [x['name'] for x in schema]
  raw_keys.remove(target_name)
  serving_input_fn = input_fn_maker.build_csv_transforming_serving_input_receiver_fn(
            raw_metadata=raw_metadata,
            transform_savedmodel_dir=transformed_data_dir + '/transform_fn',
            raw_keys=raw_keys)

  return serving_input_fn


def get_vocab_size(transformed_data_dir, feature_name):
  """Get vocab size of a given text or category column."""
  vocab_file = os.path.join(transformed_data_dir,
                            transform_fn_io.TRANSFORM_FN_DIR,
                            'assets',
                            'vocab_' + feature_name)
  with file_io.FileIO(vocab_file, 'r') as f:
    return sum(1 for _ in f)


def build_feature_columns(schema, transformed_data_dir, target):
  """Build feature columns that tf.estimator expects."""

  feature_columns = []
  for entry in schema:
    name = entry['name']
    datatype = entry['type']
    if name == target:
      continue

    if datatype == 'NUMBER':
      feature_columns.append(tf.feature_column.numeric_column(name, shape=()))
    elif datatype == 'IMAGE_URL':
      feature_columns.append(tf.feature_column.numeric_column(name, shape=(2048)))
    elif datatype == 'CATEGORY':
      vocab_size = get_vocab_size(transformed_data_dir, name)
      category_column = tf.feature_column.categorical_column_with_identity(name, num_buckets=vocab_size)
      indicator_column = tf.feature_column.indicator_column(category_column)
      feature_columns.append(indicator_column)
    elif datatype == 'TEXT':
      vocab_size = get_vocab_size(transformed_data_dir, name)
      indices_column = tf.feature_column.categorical_column_with_identity(name + '_indices', num_buckets=vocab_size + 1)
      weighted_column = tf.feature_column.weighted_categorical_column(indices_column, name + '_weights')
      indicator_column = tf.feature_column.indicator_column(weighted_column)
      feature_columns.append(indicator_column)

  return feature_columns


def get_estimator(schema, transformed_data_dir, target_name, output_dir, hidden_units,
                  optimizer, learning_rate, feature_columns):
  """Get proper tf.estimator (DNNClassifier or DNNRegressor)."""
  optimizer = tf.train.AdagradOptimizer(learning_rate)
  if optimizer == 'Adam':
    optimizer = tf.train.AdamOptimizer(learning_rate)
  elif optimizer == 'SGD':
    optimizer = tf.train.GradientDescentOptimizer(learning_rate)

  # Set how often to run checkpointing in terms of steps.
  config = tf.contrib.learn.RunConfig(save_checkpoints_steps=1000)
  n_classes = is_classification(transformed_data_dir, target_name)
  if n_classes:
    estimator = tf.estimator.DNNClassifier(
        feature_columns=feature_columns,
        hidden_units=hidden_units,
        n_classes=n_classes,
        config=config,
        model_dir=output_dir)
  else:
    estimator = tf.estimator.DNNRegressor(
        feature_columns=feature_columns,
        hidden_units=hidden_units,
        config=config,
        model_dir=output_dir,
        optimizer=optimizer)

  return estimator


def eval_input_receiver_fn(tf_transform_dir, schema, target):
  """Build everything needed for the tf-model-analysis to run the model.
  Args:
    tf_transform_dir: directory in which the tf-transform model was written
      during the preprocessing step.
    schema: the raw data schema.
    target: name of the target column.
  Returns:
    EvalInputReceiver function, which contains:
      - Tensorflow graph which parses raw untranformed features, applies the
        tf-transform preprocessing operators.
      - Set of raw, untransformed features.
      - Label against which predictions will be compared.
  """
  raw_metadata = make_tft_input_metadata(schema)
  raw_feature_spec = raw_metadata.schema.as_feature_spec()
  serialized_tf_example = tf.placeholder(
      dtype=tf.string, shape=[None], name='input_example_tensor')
  features = tf.parse_example(serialized_tf_example, raw_feature_spec)
  _, transformed_features = (
      saved_transform_io.partially_apply_saved_transform(
          os.path.join(tf_transform_dir, transform_fn_io.TRANSFORM_FN_DIR),
          features))
  receiver_tensors = {'examples': serialized_tf_example}
  return tfma.export.EvalInputReceiver(
      features=transformed_features,
      receiver_tensors=receiver_tensors,
      labels=transformed_features[target])


def main():
  # configure the TF_CONFIG such that the tensorflow recoginzes the MASTER in the yaml file as the chief.
  # TODO: kubeflow is working on fixing the problem and this TF_CONFIG can be
  # removed then.

  args = parse_arguments()
  tf.logging.set_verbosity(tf.logging.INFO)

  schema = json.loads(file_io.read_file_to_string(args.schema))
  feature_columns = None
  if args.preprocessing_module:
    module_dir = os.path.abspath(os.path.dirname(__file__))
    preprocessing_module_path = os.path.join(module_dir, 'preprocessing.py')
    with open(preprocessing_module_path, 'w+') as preprocessing_file:
      preprocessing_file.write(
          file_io.read_file_to_string(args.preprocessing_module))
    import preprocessing
    feature_columns = preprocessing.get_feature_columns(args.transformed_data_dir)
  else:
    feature_columns = build_feature_columns(schema, args.transformed_data_dir, args.target)

  estimator = get_estimator(schema, args.transformed_data_dir, args.target, args.job_dir,
                            args.hidden_layer_size, args.optimizer, args.learning_rate,
                            feature_columns)

  # TODO: Expose batch size.
  train_input_fn = make_training_input_fn(
      args.transformed_data_dir,
      'train',
      32,
      args.target,
      num_epochs=args.epochs)

  eval_input_fn = make_training_input_fn(
      args.transformed_data_dir,
      'eval',
      32,
      args.target)
  serving_input_fn = make_serving_input_fn(
      args.transformed_data_dir,
      schema,
      args.target)

  exporter = tf.estimator.FinalExporter('export', serving_input_fn)
  train_spec = tf.estimator.TrainSpec(input_fn=train_input_fn, max_steps=args.steps)
  eval_spec = tf.estimator.EvalSpec(input_fn=eval_input_fn, exporters=[exporter])
  tf.estimator.train_and_evaluate(estimator, train_spec, eval_spec)

  eval_model_dir = os.path.join(args.job_dir, 'tfma_eval_model_dir')
  tfma.export.export_eval_savedmodel(
      estimator=estimator,
      export_dir_base=eval_model_dir,
      eval_input_receiver_fn=(
          lambda: eval_input_receiver_fn(
              args.transformed_data_dir, schema, args.target)))


  metadata = {
    'outputs' : [{
      'type': 'tensorboard',
      'source': args.job_dir,
    }]
  }
  with open('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)

  with open('/output.txt', 'w') as f:
    f.write(args.job_dir)

if __name__ == '__main__':
  main()

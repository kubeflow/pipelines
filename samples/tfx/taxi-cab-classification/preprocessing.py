#!/usr/bin/env python3
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

"""Preprocessor applying tf.transform to the chicago_taxi data."""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import tensorflow as tf

import tensorflow_transform as transform

# Categorical features are assumed to each have a maximum value in the dataset.
MAX_CATEGORICAL_FEATURE_VALUES = [24, 31, 12]
CATEGORICAL_FEATURE_KEYS = [
    'trip_start_hour',
    'trip_start_day',
    'trip_start_month'
]

DENSE_FLOAT_FEATURE_KEYS = [
    'trip_miles',
    'fare',
    'trip_seconds'
]

# Number of buckets used by tf.transform for encoding each feature.
FEATURE_BUCKET_COUNT = 10

BUCKET_FEATURE_KEYS = [
    'pickup_latitude', 'pickup_longitude', 'dropoff_latitude',
    'dropoff_longitude'
]

# Number of vocabulary terms used for encoding VOCAB_FEATURES by tf.transform
VOCAB_SIZE = 1000

# Count of out-of-vocab buckets in which unrecognized VOCAB_FEATURES are hashed.
OOV_SIZE = 10

VOCAB_FEATURE_KEYS = [
    'pickup_census_tract',
    'dropoff_census_tract',
    'payment_type',
    'company',
    'pickup_community_area',
    'dropoff_community_area'
]
LABEL_KEY = 'tips'
FARE_KEY = 'fare'


def preprocess(inputs):
  """tf.transform's callback function for preprocessing inputs.
  Args:
    inputs: map from feature keys to raw not-yet-transformed features.
  Returns:
    Map from string feature key to transformed feature operations.
  """
  outputs = {}
  for key in DENSE_FLOAT_FEATURE_KEYS:
    # Preserve this feature as a dense float, setting nan's to the mean.
    outputs[key] = transform.scale_to_z_score(inputs[key])

  for key in VOCAB_FEATURE_KEYS:
    # Build a vocabulary for this feature.
    if inputs[key].dtype == tf.string:
        vocab_tensor = inputs[key]
    else:
        vocab_tensor = tf.as_string(inputs[key])
    outputs[key] = transform.string_to_int(
        vocab_tensor, vocab_filename='vocab_' + key,
        top_k=VOCAB_SIZE, num_oov_buckets=OOV_SIZE)

  for key in BUCKET_FEATURE_KEYS:
    outputs[key] = transform.bucketize(inputs[key], FEATURE_BUCKET_COUNT)

  for key in CATEGORICAL_FEATURE_KEYS:
    outputs[key] = tf.to_int64(inputs[key])

  taxi_fare = inputs[FARE_KEY]
  taxi_tip = inputs[LABEL_KEY]
  # Test if the tip was > 20% of the fare.
  tip_threshold = tf.multiply(taxi_fare, tf.constant(0.2))
  outputs[LABEL_KEY] = tf.logical_and(
      tf.logical_not(tf.is_nan(taxi_fare)),
      tf.greater(taxi_tip, tip_threshold))

  return outputs


def get_feature_columns(transformed_data_dir):
  """Callback that returns a list of feature columns for building a tf.estimator.
  Args:
    transformed_data_dir: The GCS directory holding the output of the tft transformation.
  Returns:
    A list of tf.feature_column.
  """
  return (
    [tf.feature_column.numeric_column(key, shape=()) for key in DENSE_FLOAT_FEATURE_KEYS] +
    [tf.feature_column.indicator_column(
        tf.feature_column.categorical_column_with_identity(
            key, num_buckets=VOCAB_SIZE+OOV_SIZE))
     for key in VOCAB_FEATURE_KEYS] +
    [tf.feature_column.indicator_column(
        tf.feature_column.categorical_column_with_identity(
            key, num_buckets=FEATURE_BUCKET_COUNT, default_value=0))
     for key in BUCKET_FEATURE_KEYS] +
    [tf.feature_column.indicator_column(
        tf.feature_column.categorical_column_with_identity(
            key, num_buckets=num_buckets, default_value=0))
     for key, num_buckets in zip(CATEGORICAL_FEATURE_KEYS, MAX_CATEGORICAL_FEATURE_VALUES)])

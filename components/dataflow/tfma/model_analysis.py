#!/bin/env python

# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import datetime
import json
import os

import apache_beam as beam

import tensorflow as tf
from tensorflow.python.lib.io import file_io

import tensorflow_model_analysis as tfma
from tensorflow_model_analysis.eval_saved_model.post_export_metrics import post_export_metrics
from tensorflow_model_analysis.slicer import slicer

from tensorflow_transform import coders as tft_coders
from tensorflow_transform.tf_metadata import dataset_schema


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='GCS or local directory.')
  parser.add_argument('--model',
                      type=str,
                      required=True,
                      help='GCS path to the model which will be evaluated.')
  parser.add_argument('--eval',
                      type=str,
                      required=True,
                      help='GCS path of eval files.')
  parser.add_argument('--schema',
                      type=str,
                      required=True,
                      help='GCS json schema file path.')
  parser.add_argument('--mode',
                      choices=['local', 'cloud'],
                      required=True,
                      help='whether to run the job locally or in Cloud Dataflow.')
  parser.add_argument('--project',
                      type=str,
                      help='The GCP project to run the dataflow job, if running in the `cloud` mode.')
  parser.add_argument('--slice-columns',
                      type=str,
                      nargs='+',
                      required=True,
                      help='one or more columns on which to slice for analysis.')

  return parser.parse_args()


def get_raw_feature_spec(schema):
  feature_spec = {}
  for column in schema:
    column_name = column['name']
    column_type = column['type']

    feature = tf.FixedLenFeature(shape=[], dtype=tf.string, default_value='')
    if column_type == 'NUMBER':
      feature = tf.FixedLenFeature(shape=[], dtype=tf.float32, default_value=0.0)
    feature_spec[column_name] = feature
  return feature_spec


def clean_raw_data_dict(raw_feature_spec):
  def clean_method(input_dict):
    output_dict = {}

    for key in raw_feature_spec:
      if key not in input_dict or not input_dict[key]:
        output_dict[key] = raw_feature_spec[key].default_value
      else:
        output_dict[key] = input_dict[key]
    return output_dict

  return clean_method


def run_analysis(output_dir, model_dir, eval_path, schema, project, mode, slice_columns):
  if mode == 'local':
    pipeline_options = None
    runner = 'DirectRunner'
  elif mode == 'cloud':
    tmp_location = os.path.join(output_dir, 'tmp')
    options = {
      'job_name': 'pipeline-tfma-' + datetime.datetime.now().strftime('%y%m%d-%H%M%S'),
      'setup_file': './analysis/setup.py',
      'project': project,
      'temp_location': tmp_location,
    }
    pipeline_options = beam.pipeline.PipelineOptions(flags=[], **options)
    runner = 'DataFlowRunner'
  else:
    raise ValueError("Invalid mode %s." % mode)

  column_names = [x['name'] for x in schema]
  for slice_column in slice_columns:
    if slice_column not in column_names:
      raise ValueError("Unknown slice column: %s" % slice_column)

  slice_spec = [
      slicer.SingleSliceSpec(),  # An empty spec is required for the 'Overall' slice
      slicer.SingleSliceSpec(columns=slice_columns)
  ]

  with beam.Pipeline(runner=runner, options=pipeline_options) as pipeline:
    raw_feature_spec = get_raw_feature_spec(schema)
    raw_schema = dataset_schema.from_feature_spec(raw_feature_spec)
    example_coder = tft_coders.example_proto_coder.ExampleProtoCoder(raw_schema)
    csv_coder = tft_coders.CsvCoder(column_names, raw_schema)

    raw_data = (
        pipeline
        | 'ReadFromText' >> beam.io.ReadFromText(eval_path)
        | 'ParseCSV' >> beam.Map(csv_coder.decode)
        | 'CleanData' >> beam.Map(clean_raw_data_dict(raw_feature_spec))
        | 'ToSerializedTFExample' >> beam.Map(example_coder.encode)
        | 'EvaluateAndWriteResults' >> tfma.EvaluateAndWriteResults(
            eval_saved_model_path=model_dir,
            slice_spec=slice_spec,
            output_path=output_dir))


def main():
  tf.logging.set_verbosity(tf.logging.INFO)
  args = parse_arguments()
  schema = json.loads(file_io.read_file_to_string(args.schema))
  eval_model_parent_dir = os.path.join(args.model, 'tfma_eval_model_dir')
  model_export_dir = os.path.join(eval_model_parent_dir, file_io.list_directory(eval_model_parent_dir)[0])
  run_analysis(args.output, model_export_dir, args.eval, schema,
               args.project, args.mode, args.slice_columns)


if __name__== "__main__":
  main()

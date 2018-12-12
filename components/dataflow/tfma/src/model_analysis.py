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


from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import datetime
import json
import os

import apache_beam as beam
from ipywidgets.embed import embed_data

import tensorflow as tf
from tensorflow.python.lib.io import file_io

import tensorflow_model_analysis as tfma
from tensorflow_model_analysis.eval_saved_model.post_export_metrics import post_export_metrics
from tensorflow_model_analysis.slicer import slicer

from tensorflow_transform import coders as tft_coders
from tensorflow_transform.tf_metadata import dataset_schema


_OUTPUT_HTML_FILE = 'output_display.html'
_STATIC_HTML_TEMPLATE = """
<html>
  <head>
    <title>TFMA Slicing Metrics</title>

    <!-- Load RequireJS, used by the IPywidgets for dependency management -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/require.js/2.3.4/require.min.js"
            integrity="sha256-Ae2Vz/4ePdIu6ZyI/5ZGsYnb+m0JlOmKPjt6XZ9JJkA="
            crossorigin="anonymous">
    </script>

    <!-- Load IPywidgets bundle for embedding. -->
    <script src="https://unpkg.com/@jupyter-widgets/html-manager@^0.12.0/dist/embed-amd.js"
            crossorigin="anonymous">
    </script>

    <!-- Load IPywidgets bundle for embedding. -->
    <script>
      require.config({{
        paths: {{
          "tfma_widget_js": "https://cdn.rawgit.com/tensorflow/model-analysis/v0.6.0/tensorflow_model_analysis/static/index"
        }}
      }});
    </script>

    <link rel="import" href="https://cdn.rawgit.com/tensorflow/model-analysis/v0.6.0/tensorflow_model_analysis/static/vulcanized_template.html">

    <!-- The state of all the widget models on the page -->
    <script type="application/vnd.jupyter.widget-state+json">
      {manager_state}
    </script>
  </head>

  <body>
    <h1>TFMA Slicing Metrics</h1>
    {widget_views}
  </body>
</html>
"""
_SINGLE_WIDGET_TEMPLATE = """
    <div id="slicing-metrics-widget-{0}">
      <script type="application/vnd.jupyter.widget-view+json">
        {1}
      </script>
    </div>
"""


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
                      action='append',
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


def generate_static_html_output(output_dir, slicing_columns):
  result = tfma.load_eval_result(output_path=output_dir)
  slicing_metrics_views = [
      tfma.view.render_slicing_metrics(result, slicing_column=slicing_column)
      for slicing_column in slicing_columns
  ]
  data = embed_data(views=slicing_metrics_views)
  manager_state = json.dumps(data['manager_state'])
  widget_views = [json.dumps(view) for view in data['view_specs']]
  views_html = ""
  for idx, view in enumerate(widget_views):
      views_html += _SINGLE_WIDGET_TEMPLATE.format(idx, view)
  rendered_template = _STATIC_HTML_TEMPLATE.format(
      manager_state=manager_state, widget_views=views_html)
  static_html_path = os.path.join(output_dir, _OUTPUT_HTML_FILE)
  file_io.write_string_to_file(static_html_path, rendered_template)

  metadata = {
    'outputs' : [{
      'type': 'web-app',
      'storage': 'gcs',
      'source': static_html_path,
    }]
  }
  with file_io.FileIO('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)


def main():
  tf.logging.set_verbosity(tf.logging.INFO)
  args = parse_arguments()
  schema = json.loads(file_io.read_file_to_string(args.schema))
  eval_model_parent_dir = os.path.join(args.model, 'tfma_eval_model_dir')
  model_export_dir = os.path.join(eval_model_parent_dir, file_io.list_directory(eval_model_parent_dir)[0])
  run_analysis(args.output, model_export_dir, args.eval, schema,
               args.project, args.mode, args.slice_columns)
  generate_static_html_output(args.output, args.slice_columns)
  with open('/output.txt', 'w') as f:
    f.write(args.output)

if __name__== "__main__":
  main()

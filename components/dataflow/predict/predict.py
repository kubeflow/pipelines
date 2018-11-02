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


# TODO: Add Unit or Integration Test

import apache_beam as beam
import argparse
from apache_beam.options.pipeline_options import PipelineOptions
import datetime
import json
import logging
import os
from tensorflow.python.lib.io import file_io


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='GCS or local directory.')
  parser.add_argument('--data',
                      type=str,
                      required=True,
                      help='GCS or local path of test file patterns.')
  parser.add_argument('--schema',
                      type=str,
                      required=True,
                      help='GCS or local json schema file path.')
  parser.add_argument('--model',
                      type=str,
                      required=True,
                      help='GCS or local path of model trained with tft preprocessed data.')
  parser.add_argument('--target',
                      type=str,
                      required=True,
                      help='Name of the column for prediction target.')
  parser.add_argument('--project',
                      type=str,
                      required=True,
                      help='The GCP project to run the dataflow job.')
  parser.add_argument('--mode',
                      choices=['local', 'cloud'],
                      help='whether to run the job locally or in Cloud Dataflow.')
  parser.add_argument('--batchsize',
                      type=int,
                      default=32,
                      help='Batch size used in prediction.')

  args = parser.parse_args()
  return args


class EmitAsBatchDoFn(beam.DoFn):
  """A DoFn that buffers the records and emits them batch by batch."""

  def __init__(self, batch_size):
    self._batch_size = batch_size
    self._cached = []

  def process(self, element):
    from apache_beam.transforms import window
    from apache_beam.utils.windowed_value import WindowedValue
    self._cached.append(element)
    if len(self._cached) >= self._batch_size:
      emit = self._cached
      self._cached = []
      yield emit

  def finish_bundle(self, context=None):
    from apache_beam.transforms import window
    from apache_beam.utils.windowed_value import WindowedValue
    if len(self._cached) > 0:
      yield WindowedValue(self._cached, -1, [window.GlobalWindow()])


class TargetToLastDoFn(beam.DoFn):
  """A DoFn that moves specified target column to last."""

  def __init__(self, names, target_name):
    self._names = names
    self._target_name = target_name
    self._names_no_target = list(names)
    self._names_no_target.remove(target_name)

  def process(self, element):
    import csv
    content = csv.DictReader([element], fieldnames=self._names).next()
    target = content.pop(self._target_name)
    yield [content[x] for x in self._names_no_target] + [target]


class PredictDoFn(beam.DoFn):
  """A DoFn that performs predictions with given trained model."""

  def __init__(self, model_export_dir):
    self._model_export_dir = model_export_dir

  def start_bundle(self):
    from tensorflow.contrib import predictor

    # We need to import the tensorflow_transform library in order to
    # register all of the ops that might be used by a saved model that
    # incorporates TFT transformations.
    import tensorflow_transform

    self._predict_fn = predictor.from_saved_model(self._model_export_dir)

  def process(self, element):
    import csv
    import StringIO

    prediction_inputs = []
    for instance in element:
      instance_copy = list(instance)
      instance_copy.pop() # remove target
      buf = StringIO.StringIO()
      writer = csv.writer(buf, lineterminator='')
      writer.writerow(instance_copy)
      prediction_inputs.append(buf.getvalue())
    
    return_dict = self._predict_fn({"inputs": prediction_inputs})
    return_dict['source'] = element
    yield return_dict


class ListToCsvDoFn(beam.DoFn):
  """A DoFn function that convert list to csv line."""

  def process(self, element):
    import csv
    import StringIO
    buf = StringIO.StringIO()
    writer = csv.writer(buf, lineterminator='')
    writer.writerow(element)
    yield buf.getvalue()
  

def run_predict(output_dir, data_path, schema, target_name, model_export_dir,
                project, mode, batch_size):
  """Run predictions with given model using DataFlow.
  Args:
    output_dir: output folder
    data_path: test data file path.
    schema: schema list.
    target_name: target column name.
    model_export_dir: GCS or local path of exported model trained with tft preprocessed data.
    project: the project to run dataflow in.
    local: whether the job should be local or cloud.
    batch_size: batch size when running prediction.
  """

  target_type = next(x for x in schema if x['name']==target_name)['type']
  labels_file = os.path.join(model_export_dir, 'assets', 'vocab_' + target_name)
  is_classification = file_io.file_exists(labels_file)

  output_file_prefix = os.path.join(output_dir, 'prediction_results')
  output_schema_file = os.path.join(output_dir, 'schema.json')
  names = [x['name'] for x in schema]

  output_schema = filter(lambda x: x['name'] != target_name, schema)
  if is_classification:
    with file_io.FileIO(labels_file, mode='r') as f:
      labels = [x.strip() for x in f.readlines()]

    output_schema.append({'name': 'target', 'type': 'CATEGORY'})
    output_schema.append({'name': 'predicted', 'type': 'CATEGORY'})
    output_schema.extend([{'name': x, 'type': 'NUMBER'} for x in labels])
  else:
    output_schema.append({'name': 'target', 'type': 'NUMBER'})
    output_schema.append({'name': 'predicted', 'type': 'NUMBER'})

  if mode == 'local':
    pipeline_options = None
    runner = 'DirectRunner'
  elif mode == 'cloud':
    options = {
      'job_name': 'pipeline-predict-' + datetime.datetime.now().strftime('%y%m%d-%H%M%S'),
      'temp_location': os.path.join(output_dir, 'tmp'),
      'project': project,
      'setup_file': './setup.py',
    }
    pipeline_options = beam.pipeline.PipelineOptions(flags=[], **options)
    runner = 'DataFlowRunner'
  else:
    raise ValueError("Invalid mode %s." % mode)

  with beam.Pipeline(runner, options=pipeline_options) as p:
    raw_results = (p
    | 'read data' >> beam.io.ReadFromText(data_path)
    | 'move target to last' >> beam.ParDo(TargetToLastDoFn(names, target_name))
    | 'batch' >> beam.ParDo(EmitAsBatchDoFn(batch_size))
    | 'predict' >> beam.ParDo(PredictDoFn(model_export_dir)))

    if is_classification:
      processed_results = (raw_results
        | 'unbatch' >> beam.FlatMap(lambda x: zip(x['source'], x['scores']))
        | 'get predicted' >> beam.Map(lambda x: x[0] + [labels[x[1].argmax()]] + list(x[1])))
    else:
      processed_results = (raw_results
        | 'unbatch' >> beam.FlatMap(lambda x: zip(x['source'], x['outputs']))
        | 'get predicted' >> beam.Map(lambda x: x[0] + list(x[1])))

    results_save = (processed_results
      | 'write csv lines' >> beam.ParDo(ListToCsvDoFn())
      | 'write file' >> beam.io.WriteToText(output_file_prefix))

    (results_save
      | 'fixed one' >> beam.transforms.combiners.Sample.FixedSizeGlobally(1)
      | 'set schema' >> beam.Map(lambda path: json.dumps(output_schema))
      | 'write schema file' >> beam.io.WriteToText(output_schema_file, shard_name_template=''))


def main():
  logging.getLogger().setLevel(logging.INFO)
  args = parse_arguments()
  # Models trained with estimator are exported to base/export/export/123456781 directory.
  # Our trainer export only one model.
  export_parent_dir = os.path.join(args.model, 'export', 'export')
  model_export_dir = os.path.join(export_parent_dir, file_io.list_directory(export_parent_dir)[0])
  schema = json.loads(file_io.read_file_to_string(args.schema))

  run_predict(args.output, args.data, schema, args.target, model_export_dir,
              args.project, args.mode, args.batchsize)
  prediction_results = os.path.join(args.output, 'prediction_results-*')
  with open('/output.txt', 'w') as f:
    f.write(prediction_results)

  with file_io.FileIO(os.path.join(args.output, 'schema.json'), 'r') as f:
    schema = json.load(f)

  metadata = {
    'outputs' : [{
      'type': 'table',
      'storage': 'gcs',
      'format': 'csv',
      'header': [x['name'] for x in schema],
      'source': prediction_results
    }]
  }
  with open('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)


if __name__== "__main__":
  main()

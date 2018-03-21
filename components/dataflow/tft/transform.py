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


import apache_beam as beam
import argparse
import datetime
import csv
import json
import logging
import os
import tensorflow as tf
import tensorflow_transform as tft


from apache_beam.io import textio
from apache_beam.io import tfrecordio
from apache_beam.options.pipeline_options import PipelineOptions

from tensorflow.contrib.slim.python.slim.nets.inception_v3 import inception_v3
from tensorflow.contrib.slim.python.slim.nets.inception_v3 import inception_v3_arg_scope
from tensorflow.python.lib.io import file_io
from tensorflow_transform.beam import impl as beam_impl
from tensorflow_transform.beam.tft_beam_io import transform_fn_io
from tensorflow_transform.coders.csv_coder import CsvCoder
from tensorflow_transform.coders.example_proto_coder import ExampleProtoCoder
from tensorflow_transform.tf_metadata import dataset_metadata
from tensorflow_transform.tf_metadata import dataset_schema
from tensorflow_transform.tf_metadata import metadata_io


# Inception Checkpoint
INCEPTION_V3_CHECKPOINT = 'gs://cloud-ml-data/img/flower_photos/inception_v3_2016_08_28.ckpt'
INCEPTION_EXCLUDED_VARIABLES = ['InceptionV3/AuxLogits', 'InceptionV3/Logits', 'global_step']

DELIMITERS = '.,!?() '
VOCAB_SIZE = 100000


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='GCS or local directory.')
  parser.add_argument('--train',
                      type=str,
                      required=True,
                      help='GCS path of train file patterns.')
  parser.add_argument('--eval',
                      type=str,
                      required=True,
                      help='GCS path of eval file patterns.')
  parser.add_argument('--schema',
                      type=str,
                      required=True,
                      help='GCS json schema file path.')
  parser.add_argument('--project',
                      type=str,
                      required=True,
                      help='The GCP project to run the dataflow job.')

  args = parser.parse_args()
  return args


def _image_to_vec(image_str_tensor):

  def _decode_and_resize(image_str_tensor):
    """Decodes jpeg string, resizes it and returns a uint8 tensor."""

    # These constants are set by Inception v3's expectations.
    height = 299
    width = 299
    channels = 3

    image = tf.read_file(image_str_tensor)
    image = tf.image.decode_jpeg(image, channels=channels)
    image = tf.expand_dims(image, 0)
    image = tf.image.resize_bilinear(image, [height, width], align_corners=False)
    image = tf.squeeze(image, squeeze_dims=[0])
    image = tf.cast(image, dtype=tf.uint8)
    return image

  image = tf.map_fn(_decode_and_resize, image_str_tensor, back_prop=False, dtype=tf.uint8)
  image = tf.image.convert_image_dtype(image, dtype=tf.float32)
  image = tf.subtract(image, 0.5)
  inception_input = tf.multiply(image, 2.0)

  # Build Inception layers, which expect a tensor of type float from [-1, 1)
  # and shape [batch_size, height, width, channels].
  with tf.contrib.slim.arg_scope(inception_v3_arg_scope()):
    _, end_points = inception_v3(inception_input, is_training=False)
 
  embeddings = end_points['PreLogits']
  inception_embeddings = tf.squeeze(embeddings, [1, 2], name='SpatialSqueeze')
  return inception_embeddings


def make_preprocessing_fn(schema, tmp_dir):
  """Makes a preprocessing function.
  Args:
    schema: the schema of the training data.
    tmp_dir: temp dir to hold inception checkpoints.
  Returns:
    a preprocessing_fn function used by tft.
  """

  def preprocessing_fn(inputs):
    """TFT preprocessing function.
    Args:
      inputs: dictionary of input `tensorflow_transform.Column`.
    Returns:
      A dictionary of `tensorflow_transform.Column` representing the transformed
          columns.
    """

    features_dict = {}
    for col_schema in schema:
      col_name = col_schema['name']
      if col_schema['type'] == 'NUMBER':
        features_dict[col_name] = tft.scale_to_0_1(inputs[col_name])
      elif col_schema['type'] == 'CATEGORY':
        features_dict[col_name] = tft.string_to_int(inputs[col_name],
                                                    vocab_filename='vocab_' + col_name)
      elif col_schema['type'] == 'TEXT':
        tokens = tf.string_split(inputs[col_name], DELIMITERS)
        indices = tft.string_to_int(tokens,
                                    vocab_filename='vocab_' + col_name,
                                    top_k=VOCAB_SIZE)
        # Add one for the oov bucket created by string_to_int.
        bow_indices, bow_weights = tft.tfidf(indices, VOCAB_SIZE + 1)
        features_dict[col_name + '_indices'] = bow_indices
        features_dict[col_name + '_weights'] = bow_weights
      elif col_schema['type'] == 'IMAGE_URL':
        features_dict[col_name] = tft.apply_function_with_checkpoint(
            _image_to_vec,
            [inputs[col_name]],
            INCEPTION_V3_CHECKPOINT,
            exclude=INCEPTION_EXCLUDED_VARIABLES)
    return features_dict

  return preprocessing_fn


def make_tft_input_metadata(schema):
  """Make a TFT Schema object
  In the tft framework, this is where default values are recoreded for training.
  Args:
    schema: schema list of training data.
  Returns:
    TFT metadata object.
  """
  tft_schema = {}

  for col_schema in schema:
    col_type = col_schema['type']
    col_name = col_schema['name']
    if col_type == 'NUMBER':
      tft_schema[col_name] = dataset_schema.ColumnSchema(
          tf.float32, [], dataset_schema.FixedColumnRepresentation(default_value=0.0))
    elif col_type in ['CATEGORY', 'TEXT', 'IMAGE_URL']:
      tft_schema[col_name] = dataset_schema.ColumnSchema(
          tf.string, [], dataset_schema.FixedColumnRepresentation(default_value=''))
  return dataset_metadata.DatasetMetadata(dataset_schema.Schema(tft_schema))


def run_transform(output_dir, schema, train_data_file, eval_data_file, project):
  """Writes a tft transform fn, and metadata files.
  Args:
    output_dir: output folder
    schema: schema list.
    train_data_file: training data file pattern.
    eval_data_file: eval data file pattern.
    project: the project to run dataflow in.
  """

  tft_input_metadata = make_tft_input_metadata(schema)
  temp_dir = os.path.join(output_dir, 'tmp')
  preprocessing_fn = make_preprocessing_fn(schema, temp_dir)

  options = {
    'job_name': 'pipeline-tft-' + datetime.datetime.now().strftime('%y%m%d-%H%M%S'),
    'temp_location': temp_dir,
    'project': project,
    'extra_packages': ['gs://ml-pipeline-playground/tensorflow-transform-0.6.0.dev0.tar.gz']
  }
  pipeline_options = beam.pipeline.PipelineOptions(flags=[], **options)
  with beam.Pipeline('DataflowRunner', options=pipeline_options) as p:
    with beam_impl.Context(temp_dir=temp_dir):
      names = [x['name'] for x in schema]
      converter = CsvCoder(names, tft_input_metadata.schema)
      train_data = (
          p
          | 'ReadTrainData' >> textio.ReadFromText(train_data_file)
          | 'DecodeTrainData' >> beam.Map(converter.decode))

      train_dataset = (train_data, tft_input_metadata)
      transformed_dataset, transform_fn = (
          train_dataset | beam_impl.AnalyzeAndTransformDataset(preprocessing_fn))
      transformed_data, transformed_metadata = transformed_dataset

      # Writes transformed_metadata and transfrom_fn folders
      _ = (transform_fn | 'WriteTransformFn' >> transform_fn_io.WriteTransformFn(output_dir))

      # Write the raw_metadata
      metadata_io.write_metadata(
          metadata=tft_input_metadata,
          path=os.path.join(output_dir, 'metadata'))

      _ = transformed_data | 'WriteTrainData' >> tfrecordio.WriteToTFRecord(
          os.path.join(output_dir, 'train'),
          coder=ExampleProtoCoder(transformed_metadata.schema))

      eval_data = (
          p
          | 'ReadEvalData' >> textio.ReadFromText(eval_data_file)
          | 'DecodeEvalData' >> beam.Map(converter.decode))

      eval_dataset = (eval_data, tft_input_metadata)

      transformed_eval_dataset = (
          (eval_dataset, transform_fn) | beam_impl.TransformDataset())
      # Don't need transformed data schema, it's the same as before.
      transformed_eval_data, _ = transformed_eval_dataset

      _ = transformed_eval_data | 'WriteEvalData' >> tfrecordio.WriteToTFRecord(
          os.path.join(output_dir, 'eval'),
          coder=ExampleProtoCoder(transformed_metadata.schema))


def main():
  logging.getLogger().setLevel(logging.INFO)
  args = parse_arguments()
  schema = json.loads(file_io.read_file_to_string(args.schema))
  run_transform(args.output, schema, args.train, args.eval, args.project)


if __name__== "__main__":
  main()

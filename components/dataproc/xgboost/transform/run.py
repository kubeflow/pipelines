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
from collections import Counter
import six
from pyspark.sql import Row
from pyspark.mllib.linalg import SparseVector
from pyspark.mllib.util import MLUtils
from pyspark.mllib.regression import LabeledPoint
from pyspark.sql.types import StructType, StructField
from pyspark.sql.types import DoubleType, IntegerType, StringType
import pandas as pd
from tensorflow.python.lib.io import file_io
from pyspark.sql.session import SparkSession


VOCAB_ANALYSIS_FILE = "vocab_%s.csv"
STATS_FILE = 'stats.json'


def load_schema(analysis_path):
  type_map = {
    'KEY': StringType(),
    'NUMBER': DoubleType(),
    'CATEGORY': StringType(),
    'TEXT': StringType(),
    'IMAGE_URL': StringType()
  }
  schema_file = os.path.join(analysis_path, 'schema.json')
  schema_json = json.loads(file_io.read_file_to_string(schema_file))
  fields = [StructField(x['name'], type_map[x['type']]) for x in schema_json]
  return schema_json, StructType(fields)


def get_columns_of_type(datatype, schema_json):
  return [x['name'] for x in schema_json if x['type'] == datatype]


parser = argparse.ArgumentParser(description='ML')
parser.add_argument('--output', type=str)
parser.add_argument('--train', type=str)
parser.add_argument('--eval', type=str)
parser.add_argument('--target', type=str)
parser.add_argument('--analysis', type=str)
args = parser.parse_args()

schema_json, schema = load_schema(args.analysis)
text_columns = get_columns_of_type('TEXT', schema_json)
category_columns = get_columns_of_type('CATEGORY', schema_json)
number_columns = get_columns_of_type('NUMBER', schema_json)
classification = False
if args.target in number_columns:
  number_columns.remove(args.target)
elif args.target in category_columns:
  category_columns.remove(args.target)
  classification = True
else:
  raise ValueError(
      'Specified target "%s" is neither in numeric or categorical columns' % args.target)

stats = json.loads(file_io.read_file_to_string(os.path.join(args.analysis, STATS_FILE)))

# Load vocab
vocab = {}
columns_require_vocab = text_columns + category_columns
if classification:
  columns_require_vocab.append(args.target)

for col in columns_require_vocab:
  with file_io.FileIO(os.path.join(args.analysis, VOCAB_ANALYSIS_FILE % col), 'r') as f:
    vocab_df = pd.read_csv(f, header=None, names=['vocab', 'count'], dtype=str, na_filter=False)
    vocab[col] = list(vocab_df.vocab)

# Calculate the feature size
feature_size = 0
for col in text_columns + category_columns:
  feature_size += stats['column_stats'][col]['vocab_size']

for col in number_columns:
  feature_size += 1

spark = SparkSession.builder.appName("ML Transformer").getOrCreate()


def make_process_rows_fn(
    classification, target_col, text_cols, category_cols, number_cols, vocab, stats):
  
  def process_rows(row):
    feature_indices = []
    feature_values = []
    start_index = 0
    for col in schema.names:
      col_value = getattr(row, col)
      if col in number_cols:
        v_max = stats['column_stats'][col]['max']
        v_min = stats['column_stats'][col]['min']
        value = -1 + (col_value - v_min) * 2 / (v_max - v_min)
        feature_indices.append(start_index)
        feature_values.append(value)
        start_index += 1
      if col in category_cols:
        if col_value in vocab[col]:
          value_index = vocab[col].index(col_value)
          feature_indices.append(start_index + value_index)
          feature_values.append(1.0)
        start_index += len(vocab[col])
      if col in text_cols:
        if col_value is not None:
          values = col_value.split()
          word_indices = []
          for v in values:
            if v in vocab[col]:
              word_indices.append(start_index + vocab[col].index(v))
          for k, v in sorted(six.iteritems(Counter(word_indices))):
            feature_indices.append(k)
            feature_values.append(float(v))
        start_index += len(vocab[col])
      if col == target_col:
        label = vocab[col].index(col_value) if classification else col_value
    return {"label": label, "indices": feature_indices, "values": feature_values}
  
  return process_rows


process_row_fn = make_process_rows_fn(
    classification, args.target, text_columns, category_columns, number_columns, vocab, stats)

dfs = []
if args.train:
  dfTrain = spark.read.schema(schema).csv(args.train)
  dfs.append(("train", dfTrain))
if args.eval:
  dfEval = spark.read.schema(schema).csv(args.eval)
  dfs.append(("eval", dfEval))

for name, df in dfs:
  rdd = df.rdd.map(process_row_fn).map(
      lambda row: LabeledPoint(row["label"],
                               SparseVector(feature_size, row["indices"], row["values"])))
  MLUtils.saveAsLibSVMFile(rdd, os.path.join(args.output, name))

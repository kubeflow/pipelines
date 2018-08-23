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
from pyspark.sql.types import StructType, StructField
from pyspark.sql.types import DoubleType, IntegerType, StringType
import pandas as pd
from tensorflow.python.lib.io import file_io
from pyspark.sql.session import SparkSession
import json
import os


VOCAB_ANALYSIS_FILE = 'vocab_%s.csv'
STATS_FILE = 'stats.json'


def load_schema(schema_file):
  type_map = {
    'KEY': StringType(),
    'NUMBER': DoubleType(),
    'CATEGORY': StringType(),
    'TEXT': StringType(),
    'IMAGE_URL': StringType()
  }
  
  schema_json = json.loads(file_io.read_file_to_string(schema_file))
  fields = [StructField(x['name'], type_map[x['type']]) for x in schema_json]
  return schema_json, StructType(fields)


def get_columns_of_type(datatype, schema_json):
  return [x['name'] for x in schema_json if x['type'] == datatype]


parser = argparse.ArgumentParser(description='ML')
parser.add_argument('--output', type=str)
parser.add_argument('--train', type=str)
parser.add_argument('--schema', type=str)
args = parser.parse_args()

schema_json, schema = load_schema(args.schema)
text_columns = get_columns_of_type('TEXT', schema_json)
category_columns = get_columns_of_type('CATEGORY', schema_json)
number_columns = get_columns_of_type('NUMBER', schema_json)

spark = SparkSession.builder.appName("MLAnalyzer").getOrCreate()
df = spark.read.schema(schema).csv(args.train)
df.createOrReplaceTempView("train")

num_examples = df.sql_ctx.sql(
    'SELECT COUNT(*) AS num_examples FROM train').collect()[0].num_examples
stats = {'column_stats': {}, 'num_examples': num_examples}

for col in text_columns:
  col_data = df.sql_ctx.sql("""
SELECT token, COUNT(token) AS token_count
FROM (SELECT EXPLODE(SPLIT({name}, \' \')) AS token FROM train)
GROUP BY token 
ORDER BY token_count DESC, token ASC""".format(name=col))
  token_counts = [(r.token, r.token_count) for r in col_data.collect()]
  csv_string = pd.DataFrame(token_counts).to_csv(index=False, header=False)
  file_io.write_string_to_file(os.path.join(args.output, VOCAB_ANALYSIS_FILE % col), csv_string)
  stats['column_stats'][col] = {'vocab_size': len(token_counts)}

for col in category_columns:
  col_data = df.sql_ctx.sql("""
SELECT {name} as token, COUNT({name}) AS token_count
FROM train
GROUP BY token
ORDER BY token_count DESC, token ASC
""".format(name=col))
  token_counts = [(r.token, r.token_count) for r in col_data.collect()]
  csv_string = pd.DataFrame(token_counts).to_csv(index=False, header=False)
  file_io.write_string_to_file(os.path.join(args.output, VOCAB_ANALYSIS_FILE % col), csv_string)
  stats['column_stats'][col] = {'vocab_size': len(token_counts)}

for col in number_columns:
  col_stats = df.sql_ctx.sql("""
SELECT MAX({name}) AS max_value, MIN({name}) AS min_value, AVG({name}) AS mean_value
FROM train""".format(name=col)).collect()
  stats['column_stats'][col] = {'min': col_stats[0].min_value, 'max': col_stats[0].max_value, 'mean': col_stats[0].mean_value}

file_io.write_string_to_file(os.path.join(args.output, STATS_FILE), json.dumps(stats, indent=2, separators=(',', ': ')))
file_io.write_string_to_file(os.path.join(args.output, 'schema.json'), json.dumps(schema_json, indent=2, separators=(',', ': ')))

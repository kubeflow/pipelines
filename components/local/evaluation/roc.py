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


# A program to generate ROC data out of prediction results.
# Usage:
# python roc.py  \
#   --predictions=gs://bradley-playground/sfpd/predictions/part-* \
#   --trueclass=ACTION \
#   --output=gs://bradley-playground/sfpd/roc/ \


import argparse
import json
import os
import pandas as pd
from sklearn.metrics import roc_curve
from tensorflow.python.lib.io import file_io


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Trainer')
  parser.add_argument('--predictions', type=str, help='GCS path of prediction file pattern.')
  parser.add_argument('--trueclass', type=str, help='The name of the class as true value.')
  parser.add_argument('--output', type=str, help='GCS path of the output directory.')
  args = parser.parse_args()

  schema_file = os.path.join(os.path.dirname(args.predictions), 'schema.json')
  schema = json.loads(file_io.read_file_to_string(schema_file))
  names = [x['name'] for x in schema]
  dfs = []
  files = file_io.get_matching_files(args.predictions)
  for file in files:
    with file_io.FileIO(file, 'r') as f:
      dfs.append(pd.read_csv(f, names=names))
    
  df = pd.concat(dfs)
  df['target'] = df['target'].apply(lambda x: 1 if x == args.trueclass else 0)
  fpr, tpr, thresholds = roc_curve(df['target'], df[args.trueclass])
  df_roc = pd.DataFrame({'fpr': fpr, 'tpr': tpr, 'thresholds': thresholds})
  roc_file = os.path.join(args.output, 'roc.csv')
  with file_io.FileIO(roc_file, 'w') as f:
    df_roc.to_csv(f, columns=['fpr', 'tpr', 'thresholds'], header=False, index=False)
  
  metadata = {
    'outputs': [{
      'type': 'roc',
      'storage': 'gcs',
      'format': 'csv',
      'schema': [
        {'name': 'fpr', 'type': 'NUMBER'},
        {'name': 'tpr', 'type': 'NUMBER'},
        {'name': 'thresholds', 'type': 'NUMBER'},
      ],
      'source': roc_file
    }]
  }
  with file_io.FileIO('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)


if __name__== "__main__":
  main()

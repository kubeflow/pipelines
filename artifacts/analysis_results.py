# Copyright 2018 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under
# the License.


class AnalysisResults(object):
  """Defines directory structure and file formats for training data analysis results.

  An example of the directory structure:
  /stats.json
  /schema.json
  /col1_vocab.csv
  /col2_vocab.csv

  Files:
    stats.json: Contains global stats, column specific stats such as numeric stats, vocab size, etc.
      Example:
      {
        "column_stats": {
          "fare": {
            "max": 184.25,
            "mean": 11.2135084657688,
            "min": 2.25
          },
          "dropoff_longitude": {
            "max": -87.534902901,
            "mean": -87.65432881808832,
            "min": -87.913624596
          },
          "dropoff_latitude": {
            "max": 42.021223593,
            "mean": 41.902932609443006,
            "min": 41.660136051
          }
          "weekday": {
            "vocab_size": 7
          },
        },
        "num_examples": 203998
      }

      For number columns, max/min/mean are required. For text or category columns, vocab_size is required.
      Everything else is optional and allowed.

    schema.json: contains names and types for each column. Example:
      [
        {"name": "trip_id", "type": "key"},
        {"name": "fare", "type": "float"},
        {"name": "dropoff_longitude", "type": "float"},
        {"name": "dropoff_latitude", "type": "float"},
        {"name": "weekday", "type": "category"},
      ]

      The support types are: key, integer, float, text, category, image_url.
      The order of the list should match the order of columns in csv, if format is csv.

    [column_name]_vocab.csv: csv containing word and frequency for text and category columns. For example:

      Sunday,34063
      Saturday,33688
      Friday,29680
      Thursday,28105
      Monday,27244
      Wednesday,26489
      Tuesday,24729

      They are reverse ordered by frequency. The file should be loadable to pandas dataframe's read_csv.
  """

  def __init__(self, storage, source):
    """
    Args
      storage: one of ['local', 'gs'].
      source: the path to the directory holding files.
    """

    if storage == 'gs' and not source.startswith('gs://'):
      raise ValueError('Source does not look like a gcs path but storage is gs.')
      
    self._storage = storage
    self._source = source

  @property
  def storage(self):
    return self._storage

  @property
  def source(self):
    return self._source

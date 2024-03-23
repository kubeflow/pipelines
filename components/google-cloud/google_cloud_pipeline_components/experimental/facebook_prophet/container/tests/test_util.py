# Copyright 2021 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from pathlib import Path
import sys
import tempfile
import unittest
import numpy as np
import pandas as pd

sys.path.append(str(Path(__file__).parent.parent.absolute()))
from fit_model import DataSource


class TestDataSource(unittest.TestCase):

  def test_local_file(self):
    with tempfile.NamedTemporaryFile() as f:
      orig = pd.DataFrame({'col1': np.arange(0, 10)})
      orig.to_csv(f.name)
      read = DataSource(f.name).load()
      self.assertIn('col1', read)
      self.assertTrue(orig['col1'].equals(read['col1']))

  def test_remote_csv(self):
    iris = DataSource('gs://automl-tables-quickstart/Iris.csv').load()
    self.assertIn('Species', iris)
    self.assertEqual(len(iris), 150)

  def test_bq_table(self):
    blackhole = DataSource('bigquery-public-data.blackhole_database.sdss_dr7').load()
    self.assertIn('PLATE', blackhole)
    self.assertEqual(len(blackhole), 7955)

  def test_invalid_data_source(self):
    with self.assertRaises(Exception):
      DataSource('{"foo":"bar"}')

    with self.assertRaises(Exception):
      DataSource('hello')


if __name__ == '__main__':
  unittest.main()

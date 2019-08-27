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
# limitations under the License

import os
import shutil
import sys
import unittest
import yaml


# Need to adjust sys path to find utils.py
_PACKAGE_PARENT = '..'
_SCRIPT_DIR = os.path.dirname(os.path.realpath(os.path.join(os.getcwd(), os.path.expanduser(__file__))))
sys.path.append(os.path.normpath(os.path.join(_SCRIPT_DIR, _PACKAGE_PARENT)))

import utils


_DATAPATH = 'testdata/'
_WORK_DIR = 'workdir/'


class TestUtils(unittest.TestCase):
  """Unit tests for utility functions defined in test/sample-test/utils.py"""
  def setUp(self) -> None:
    """Prepare unit test environment."""

    os.mkdir(_WORK_DIR)
    # Copy file for test_file_injection because the function works inplace.
    shutil.copyfile(os.path.join(_DATAPATH, 'test_file_injection_input.yaml'),
                    os.path.join(_WORK_DIR, 'test_file_injection.yaml'))

  def tearDown(self) -> None:
    """Clean up."""
    shutil.rmtree(_WORK_DIR)

  def test_file_injection(self):
    """Test file_injection function."""
    subs = {
        'gcr\.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\w+':'gcr.io/ml-pipeline/LOCAL_CONFUSION_MATRIX_IMAGE',
        'gcr\.io/ml-pipeline/ml-pipeline-dataproc-analyze:\w+':'gcr.io/ml-pipeline/DATAPROC_ANALYZE_IMAGE',
        'gcr\.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:\w+':'gcr.io/ml-pipeline/DATAPROC_CREATE_IMAGE',
        'gcr\.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:\w+':'gcr.io/ml-pipeline/DATAPROC_DELETE_IMAGE',
    }
    utils.file_injection(
        os.path.join(_WORK_DIR, 'test_file_injection.yaml'),
        os.path.join(_WORK_DIR, 'test_file_injection_tmp.yaml'),
        subs)
    with open(os.path.join(_DATAPATH,
                           'test_file_injection_output.yaml'), 'r') as f:
      golden = yaml.safe_load(f)

    with open(os.path.join(_WORK_DIR,
                           'test_file_injection.yaml'), 'r') as f:
      injected = yaml.safe_load(f)
    self.assertEqual(golden, injected)
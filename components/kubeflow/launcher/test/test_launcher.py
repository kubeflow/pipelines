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


import launcher
from launcher import train
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
import yaml


class TestLauncher(unittest.TestCase):

  def test_yaml_generation_basic(self):
    """Test generating train yaml from templates"""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    train_template_file = os.path.join(test_data_dir, 'train.template.yaml')
    tfjob_ns = 'default'
    worker = 2
    pss = 1
    args_list = []
    args_list.append('--learning-rate=0.1')
    generated_yaml = train._generate_train_yaml(train_template_file, tfjob_ns, worker, pss, args_list)
    with open(os.path.join(test_data_dir, 'train_basic.yaml'), 'r') as f:
      golden = yaml.safe_load(f)
    self.assertEqual(golden, generated_yaml)

  def test_yaml_generation_advanced(self):
    """Test generating train yaml with zero worker and specified tfjob namespace"""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    train_template_file = os.path.join(test_data_dir, 'train.template.yaml')
    worker = 0
    pss = 0
    args_list = []
    tfjob_ns = 'kubeflow'
    args_list.append('--learning-rate=0.1')
    generated_yaml = train._generate_train_yaml(train_template_file, tfjob_ns, worker, pss, args_list)
    with open(os.path.join(test_data_dir, 'train_zero_worker.yaml'), 'r') as f:
      golden = yaml.safe_load(f)
    self.assertEqual(golden, generated_yaml)

if __name__ == '__main__':
  unittest.main()

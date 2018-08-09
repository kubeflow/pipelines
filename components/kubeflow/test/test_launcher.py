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

  def test_yaml_generation_one(self):
    """Test generating train yaml from templates"""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    train_template_file = os.path.join(test_data_dir, 'train.template.yaml')
    tmpdir = tempfile.mkdtemp()
    train_dst_file = os.path.join(tmpdir, 'train.yaml')
    try:
      worker = 2
      pss = 1
      args_list = []
      args_list.append('--learning-rate=0.1')
      train._generate_train_yaml(train_template_file, train_dst_file, worker, pss, args_list)
      with open(os.path.join(test_data_dir, 'train.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(train_dst_file, 'r') as f:
        train_yaml = yaml.load(f)
      self.assertEqual(golden, train_yaml)

    finally:
      shutil.rmtree(tmpdir)

  def test_yaml_generation_two(self):
    """Test generating train yaml from templates"""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    train_template_file = os.path.join(test_data_dir, 'train.template.yaml')
    tmpdir = tempfile.mkdtemp()
    train_dst_file = os.path.join(tmpdir, 'train.yaml')
    try:
      worker = 0
      pss = 0
      args_list = []
      args_list.append('--learning-rate=0.1')
      train._generate_train_yaml(train_template_file, train_dst_file, worker, pss, args_list)
      with open(os.path.join(test_data_dir, 'train2.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(train_dst_file, 'r') as f:
        train_yaml = yaml.load(f)
      self.assertEqual(golden, train_yaml)

    finally:
      shutil.rmtree(tmpdir)

if __name__ == '__main__':
  unittest.main()